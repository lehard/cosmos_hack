package process

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// Исполнитель BPMN — чистая функция «версия + вход изделия → положение
// токенов + обязательства» (AD-17, FR-11). machine — шаг исполнителя над
// копией состояния: переходы токенов по узлам, шлюзы И/ИЛИ/исключающий,
// события, таймеры-окна, подпроцессы со стеком вызовов и возвратом в точку
// вызова, условия на своём языке, циклы.

// maxSteps — предел переходов за одну запись: защита от цикла без задач
// (шлюзы по кругу) — токен останавливается с ошибкой, свёртка не виснет.
const maxSteps = 5000

// catchUpDepth — предел длины догона по факту дальнего шага.
const catchUpDepth = 40

type machine struct {
	s     *State
	env   Env
	d     *Definition
	at    time.Time
	cause Cause
	steps int
	// transit — запись отправила изделие (operation.movement.sent): следующий
	// шаг-перемещение начинается «в перемещении».
	transit bool
}

func (m *machine) tok(id int) *Token {
	for i := range m.s.Tokens {
		if m.s.Tokens[i].ID == id {
			return &m.s.Tokens[i]
		}
	}
	return nil
}

func (m *machine) newToken(stack []Frame, vars map[string]Value, basis []string) int {
	m.s.NextToken++
	t := Token{ID: m.s.NextToken, Stack: slices.Clone(stack), Vars: cloneVars(vars), Basis: slices.Clone(basis), Phase: PhaseWaiting, Since: m.at}
	m.s.Tokens = append(m.s.Tokens, t)
	return t.ID
}

func cloneVars(v map[string]Value) map[string]Value {
	out := make(map[string]Value, len(v))
	for _, k := range sortedKeys(v) {
		out[k] = v[k]
	}
	return out
}

func (m *machine) remove(id int, via string) {
	m.closeVisit(id, via)
	m.disarm(id, "")
	m.s.Tokens = slices.DeleteFunc(m.s.Tokens, func(t Token) bool { return t.ID == id })
}

func (m *machine) closeVisit(id int, via string) {
	t := m.tok(id)
	if t == nil || t.Node == "" {
		return
	}
	for i := len(m.s.History) - 1; i >= 0; i-- {
		h := &m.s.History[i]
		if h.Node == t.Node && h.Left == nil {
			at := m.at
			h.Left, h.Via = &at, via
			return
		}
	}
}

func (m *machine) setVar(t *Token, name string, v Value) {
	if t.Vars == nil {
		t.Vars = map[string]Value{}
	}
	t.Vars[name] = v
}

// vars — переменные условий токена плюс план контроля (Env.Plan).
func (m *machine) vars(t *Token) map[string]Value {
	out := cloneVars(t.Vars)
	if _, ok := out[VarPlanRadiography]; !ok {
		out[VarPlanRadiography] = Bool(m.env.planRadiography())
	}
	return out
}

// enter — токен входит в узел.
func (m *machine) enter(id int, nodeID string) {
	m.steps++
	t := m.tok(id)
	if t == nil {
		return
	}
	if m.steps > maxSteps {
		t.Phase, t.Block = PhaseStuck, "цикл без задач: превышен предел переходов"
		setMap(&m.s.Errors, "loop:"+nodeID, "цикл без задач на узле "+nodeID)
		return
	}
	n := m.d.Node(nodeID)
	if n == nil {
		t.Phase, t.Block = PhaseStuck, "нет узла "+nodeID
		return
	}
	t.Node, t.StepKey, t.Since, t.Phase, t.RunID, t.Block = n.ID, n.StepKey(), m.at, PhaseWaiting, "", ""
	setMap(&m.s.Visits, n.ID, m.s.Visits[n.ID]+1)
	m.s.History = append(m.s.History, Visit{StepKey: n.StepKey(), Node: n.ID, Entered: m.at})
	switch n.Type {
	case NodeStartEvent, NodeBoundaryEvent:
		m.next(id, "completed")
	case NodeEndEvent:
		m.end(id, n)
	case NodeThrowEvent:
		if n.Event == eventMessage {
			m.throw(t, n)
		}
		m.next(id, "completed")
	case NodeCatchEvent:
		switch n.Event {
		case eventTimer:
			m.arm(t, n, n)
		case eventMessage:
			if m.messageReady(n) {
				m.next(id, "completed")
			}
		}
	case NodeExclusiveGateway:
		m.exclusive(id, n)
	case NodeParallelGateway:
		m.parallel(id, n)
	case NodeInclusiveGateway:
		m.inclusive(id, n)
	case NodeCallActivity, NodeSubProcess:
		scope := n.CalledElement
		if n.Type == NodeSubProcess {
			scope = n.ID
		}
		sc := m.d.Scopes[scope]
		if sc == nil || len(sc.Starts) == 0 {
			t.Phase, t.Block = PhaseStuck, "нет вызываемого процесса "+scope
			return
		}
		m.s.NextInst++
		t.Stack = append(t.Stack, Frame{Call: n.ID, Inst: m.s.NextInst})
		m.enter(id, sc.Starts[0])
	default: // задачи
		for _, b := range n.Boundaries {
			m.arm(t, m.d.Node(b), n)
		}
		if n.IsPresentationPoint() {
			g := m.s.Gates[n.StepKey()]
			g.StepKey = n.StepKey()
			g.Count++
			g.Authority = requiredAuthority(n, g.Count)
			setMap(&m.s.Gates, n.StepKey(), g)
			m.setVar(t, VarPresentationNo, Int(int64(g.Count)))
		}
		if n.Props.StepKind == stepKindMovement && m.pendingTransit(n) {
			t.Phase = PhaseInTransit
		}
	}
}

// requiredAuthority — полномочие подписи предъявления номер no (FR-19): с
// каждым повторным — полномочие более высокого уровня (repeatAuthority).
func requiredAuthority(n *Node, no int) string {
	if n.Presentation == nil {
		return ""
	}
	if no >= 2 && n.Presentation.RepeatAuthority != "" {
		return n.Presentation.RepeatAuthority
	}
	return n.Presentation.Authority
}

// pendingTransit — перемещение уже отправлено предыдущим шагом-передачей.
func (m *machine) pendingTransit(n *Node) bool {
	_ = n
	return m.transit
}

// next — токен покидает текущий узел по единственной исходящей стрелке.
func (m *machine) next(id int, via string) {
	t := m.tok(id)
	if t == nil {
		return
	}
	n := m.d.Node(t.Node)
	m.closeVisit(id, via)
	m.disarm(id, t.Node)
	if n == nil || len(n.Out) == 0 {
		t.Phase, t.Block = PhaseStuck, "нет исходящей стрелки"
		return
	}
	m.enter(id, m.d.Flows[n.Out[0]].Target)
}

// end — конечное событие: terminate завершает все токены изделия (Д-4);
// конец вызванного процесса возвращает токен в точку вызова (AD-17) с итогом
// ant:properties/@outcome в переменной nc.outcome.
func (m *machine) end(id int, n *Node) {
	t := m.tok(id)
	if n.Event == eventTerminate {
		for _, o := range slices.Clone(m.s.Tokens) {
			m.remove(o.ID, "terminated")
		}
		m.finish("terminated", n)
		return
	}
	if len(t.Stack) == 0 {
		m.remove(id, "completed")
		if len(m.s.Tokens) == 0 {
			m.finish("released", n)
		}
		return
	}
	top := t.Stack[len(t.Stack)-1]
	for _, o := range m.s.Tokens {
		if o.ID != id && len(o.Stack) > 0 && o.Stack[len(o.Stack)-1] == top {
			m.remove(id, "completed")
			return
		}
	}
	m.closeVisit(id, "completed")
	t = m.tok(id)
	t.Stack = t.Stack[:len(t.Stack)-1]
	if n.Props.Outcome != "" {
		m.setVar(t, VarNCOutcome, Str(n.Props.Outcome))
	}
	if len(t.Stack) == 0 {
		m.s.Isolated = false
	}
	call := m.d.Node(top.Call)
	t.Node, t.StepKey = call.ID, call.StepKey()
	m.next(id, "completed")
}

func (m *machine) finish(outcome string, n *Node) {
	at := m.at
	m.s.Completed, m.s.Outcome, m.s.EndStep, m.s.CompletedAt = true, outcome, n.StepKey(), &at
	m.s.Timers = nil
	m.s.Isolated = false
}

// exclusive — исключающий шлюз: первая по порядку документа ветка с
// истинным условием, иначе ветка по умолчанию; нет такой — токен стоит
// («нет данных», не догадка).
func (m *machine) exclusive(id int, n *Node) {
	if len(n.Out) <= 1 {
		m.next(id, "completed")
		return
	}
	t := m.tok(id)
	vars := m.vars(t)
	for _, fid := range n.Out {
		f := m.d.Flows[fid]
		if fid != n.Default && f.Cond != nil && f.Cond.Eval(vars) {
			m.take(id, f)
			return
		}
	}
	if n.Default != "" {
		m.take(id, m.d.Flows[n.Default])
		return
	}
	t.Phase, t.Block = PhaseStuck, "ни одно условие развилки не выполнено"
}

func (m *machine) take(id int, f *Flow) {
	m.closeVisit(id, "completed")
	m.enter(id, f.Target)
}

// parallel — параллельный шлюз: слияние ждёт токены по всем входящим
// стрелкам своего кадра вызова; развилка порождает токен на каждую ветку.
func (m *machine) parallel(id int, n *Node) {
	if len(n.In) > 1 {
		need := len(n.In)
		if m.alone(n) {
			// Других токенов изделия нет — ждать слиянию некого: изделие
			// вошло в процесс после ветвления (entry_step_key регистрации,
			// эпик 16), ветви, которые оно не проходило, не запирают слияние.
			need = 0
		}
		k, ok := m.join(id, n, need)
		if !ok {
			return
		}
		id = k
	}
	m.split(id, n, n.Out)
}

// alone — все токены изделия стоят на узле n (ни один не может прийти ещё).
func (m *machine) alone(n *Node) bool {
	for _, o := range m.s.Tokens {
		if o.Node != n.ID {
			return false
		}
	}
	return true
}

// join — слияние: токены на шлюзе в том же кадре вызова сливаются в токен с
// меньшим id (устойчиво к порядку прихода); need > 0 — сколько нужно.
func (m *machine) join(id int, n *Node, need int) (int, bool) {
	t := m.tok(id)
	var here []int
	for _, o := range m.s.Tokens {
		if o.Node == n.ID && slices.Equal(o.Stack, t.Stack) {
			here = append(here, o.ID)
		}
	}
	if need > 0 && len(here) < need {
		return 0, false
	}
	slices.Sort(here)
	keep := here[0]
	vars, basis := map[string]Value{}, []string{}
	for _, h := range here {
		o := m.tok(h)
		for _, k := range sortedKeys(o.Vars) {
			vars[k] = o.Vars[k]
		}
		basis = append(basis, o.Basis...)
	}
	for _, h := range here[1:] {
		m.remove(h, "merged")
	}
	k := m.tok(keep)
	k.Vars, k.Basis = vars, slices.Compact(sortedStrings(basis))
	return keep, true
}

func sortedStrings(xs []string) []string {
	out := slices.Clone(xs)
	slices.Sort(out)
	return out
}

func (m *machine) split(id int, n *Node, outs []string) {
	if len(outs) == 0 {
		m.tok(id).Phase = PhaseStuck
		return
	}
	t := m.tok(id)
	stack, vars, basis := slices.Clone(t.Stack), cloneVars(t.Vars), slices.Clone(t.Basis)
	m.closeVisit(id, "completed")
	var spawned []int
	for range outs[1:] {
		spawned = append(spawned, m.newToken(stack, vars, basis))
	}
	m.enter(id, m.d.Flows[outs[0]].Target)
	for i, sid := range spawned {
		m.enter(sid, m.d.Flows[outs[i+1]].Target)
	}
}

// inclusive — включающий шлюз: развилка — все ветки с истинным условием
// (нет — по умолчанию); слияние ждёт, пока ни один другой токен того же
// кадра не может дойти до шлюза.
func (m *machine) inclusive(id int, n *Node) {
	if len(n.In) > 1 {
		t := m.tok(id)
		for _, o := range m.s.Tokens {
			if o.ID != id && o.Node != n.ID && slices.Equal(o.Stack, t.Stack) && m.reaches(o.Node, n.ID) {
				return
			}
		}
		id, _ = m.join(id, n, 0)
	}
	t := m.tok(id)
	vars := m.vars(t)
	var outs []string
	for _, fid := range n.Out {
		f := m.d.Flows[fid]
		if fid != n.Default && (f.Cond == nil && f.CondText == "" || f.Cond != nil && f.Cond.Eval(vars)) {
			outs = append(outs, fid)
		}
	}
	if len(outs) == 0 && n.Default != "" {
		outs = []string{n.Default}
	}
	if len(outs) == 0 {
		t.Phase, t.Block = PhaseStuck, "ни одно условие развилки не выполнено"
		return
	}
	m.split(id, n, outs)
}

// reaches — от узла from есть путь до to в той же области.
func (m *machine) reaches(from, to string) bool {
	seen := map[string]bool{}
	q := []string{from}
	for len(q) > 0 {
		id := q[0]
		q = q[1:]
		if id == to {
			return true
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		n := m.d.Node(id)
		if n == nil {
			continue
		}
		q = append(q, m.d.Next(n)...)
		q = append(q, n.Boundaries...)
	}
	return false
}

// throw — событие-сообщение «в 1С» (AD-18): реакция operation.message.thrown.
func (m *machine) throw(t *Token, n *Node) {
	basis := slices.Clone(t.Basis)
	if len(basis) == 0 && m.cause.EventID != "" {
		basis = []string{m.cause.EventID}
	}
	m.s.Thrown = append(m.s.Thrown, Thrown{StepKey: n.StepKey(), Node: n.ID, MessageRef: n.MessageRef, ErpAction: n.Props.ErpAction,
		Visit: m.s.Visits[n.ID], Basis: basis, Causes: []Cause{m.cause}})
}

// messageReady — промежуточное событие-приём уже выполнено входом изделия
// (Д-5): сообщение из 1С о партиях — это партии в регистрации изделия.
func (m *machine) messageReady(n *Node) bool {
	switch n.Props.TriggerEventType {
	case "erp.lot.received":
		return len(m.s.Lots) > 0
	case "erp.order.received":
		return m.s.StartedAt != nil
	}
	return false
}

// ObligationID — id срока таймера для notifications (AD-4): UUIDv5 изделия,
// узла таймера и номера посещения — одинаков у воркера и верификатора.
func ObligationID(itemID, node string, visit int) string {
	return kernel.UUIDv5(constants.NsAnt, "process.timer\x1f"+itemID+"\x1f"+node+"\x1f"+strconv.Itoa(visit))
}

// arm — взвести таймер-окно (Д-8): граничный — на задаче host, промежуточный — сам узел.
func (m *machine) arm(t *Token, timer, host *Node) {
	if timer == nil || timer.Event != eventTimer || timer.TimerDur <= 0 {
		return
	}
	tm := Timer{ObligationID: ObligationID(m.s.ItemID, timer.ID, m.s.Visits[host.ID]), Token: t.ID, Node: timer.ID, StepKey: timer.StepKey(),
		Scope: timer.Props.TimerScope, ArmedAt: m.at, DueAt: m.at.Add(timer.TimerDur)}
	if timer.ID != host.ID {
		tm.Host = host.ID
	}
	m.s.Timers = append(m.s.Timers, tm)
}

// disarm — снять таймеры токена: все (node == "") или привязанные к узлу node.
func (m *machine) disarm(id int, node string) {
	m.s.Timers = slices.DeleteFunc(m.s.Timers, func(tm Timer) bool {
		return tm.Token == id && (node == "" || tm.Host == node || tm.Node == node)
	})
}

// fireDue — сработать таймеры со сроком строго раньше before (окно
// истекло до записи): порядок — по сроку, затем по id.
func (m *machine) fireDue(before time.Time) {
	for {
		idx := -1
		for i, tm := range m.s.Timers {
			if !tm.DueAt.Before(before) {
				continue
			}
			if idx < 0 || tm.DueAt.Before(m.s.Timers[idx].DueAt) || tm.DueAt.Equal(m.s.Timers[idx].DueAt) && tm.ObligationID < m.s.Timers[idx].ObligationID {
				idx = i
			}
		}
		if idx < 0 {
			return
		}
		m.fire(m.s.Timers[idx])
	}
}

// fire — окно истекло (AD-17 «таймер — окно истекло»): граничный
// прерывающий таймер уводит токен с задачи по своей ветке (например, на
// повторную подготовку кромок, FR-11), непрерывающий — порождает токен.
func (m *machine) fire(tm Timer) {
	m.s.Timers = slices.DeleteFunc(m.s.Timers, func(x Timer) bool { return x.ObligationID == tm.ObligationID })
	saved := m.at
	m.at = tm.DueAt
	defer func() { m.at = saved }()
	t := m.tok(tm.Token)
	if t == nil {
		return
	}
	timer := m.d.Node(tm.Node)
	if tm.Host == "" {
		if t.Node == tm.Node {
			m.next(t.ID, "completed")
		}
		return
	}
	if t.Node != tm.Host || timer == nil {
		return
	}
	if timer.CancelActivity {
		m.closeVisit(t.ID, "timer")
		m.disarm(t.ID, tm.Host)
		t.Node, t.StepKey = timer.ID, timer.StepKey()
		setMap(&m.s.Visits, timer.ID, m.s.Visits[timer.ID]+1)
		m.s.History = append(m.s.History, Visit{StepKey: timer.StepKey(), Node: timer.ID, Entered: m.at})
		m.next(t.ID, "completed")
		return
	}
	nid := m.newToken(t.Stack, t.Vars, t.Basis)
	m.enter(nid, timer.ID)
}

// catchUp — факт пришёл с дальнего шага target: догнать токен, если путь от
// его узла до target не проходит точек предъявления, контроля человеком,
// вызовов подпроцессов, приёма событий и слияний (FR-44: без решения
// закрывающую точку не пройти). Пропущенные шаги — «нет данных» (Gap).
func (m *machine) catchUp(target string) *Token {
	tn := m.d.Node(target)
	if tn == nil {
		return nil
	}
	ids := make([]int, 0, len(m.s.Tokens))
	for _, t := range m.s.Tokens {
		ids = append(ids, t.ID)
	}
	for _, id := range ids {
		t := m.tok(id)
		cur := m.d.Node(t.Node)
		if cur == nil || cur.Scope != tn.Scope || !m.passable(cur) || t.Phase == PhaseBlocked {
			continue
		}
		path := m.pathTo(t, target)
		if path == nil {
			continue
		}
		m.jump(id, path)
		return m.tokenAtNode(target)
	}
	return nil
}

func (m *machine) tokenAtNode(node string) *Token {
	for i := range m.s.Tokens {
		if m.s.Tokens[i].Node == node {
			return &m.s.Tokens[i]
		}
	}
	return nil
}

// passable — узел можно пройти догоном без данных.
func (m *machine) passable(n *Node) bool {
	switch {
	case n.IsPresentationPoint(), n.IsHumanControl():
		return false
	case n.Type == NodeCallActivity, n.Type == NodeSubProcess, n.Type == NodeCatchEvent, n.Type == NodeEndEvent:
		return false
	case (n.Type == NodeParallelGateway || n.Type == NodeInclusiveGateway) && len(n.In) > 1:
		return false
	}
	return true
}

// pathTo — стрелки от узла токена до target (поиск в ширину, по порядку документа).
func (m *machine) pathTo(t *Token, target string) []string {
	type step struct {
		node string
		path []string
	}
	vars := m.vars(t)
	seen := map[string]bool{t.Node: true}
	q := []step{{node: t.Node}}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		if len(cur.path) > catchUpDepth {
			continue
		}
		n := m.d.Node(cur.node)
		for _, fid := range n.Out {
			f := m.d.Flows[fid]
			if n.Type == NodeExclusiveGateway && len(n.Out) > 1 && fid != n.Default && (f.Cond == nil || !f.Cond.Eval(vars)) {
				continue
			}
			p := append(slices.Clone(cur.path), fid)
			if f.Target == target {
				return p
			}
			tn := m.d.Node(f.Target)
			if seen[f.Target] || !m.passable(tn) {
				continue
			}
			seen[f.Target] = true
			q = append(q, step{node: f.Target, path: p})
		}
	}
	return nil
}

// jump — провести токен по пути догона: шаги без данных отмечаются Gap,
// сообщения «в 1С» на пути отправляются, развилки порождают токены.
func (m *machine) jump(id int, path []string) {
	if t := m.tok(id); t != nil && m.leaveVia(t) == "skipped" {
		m.s.Gaps = append(m.s.Gaps, Gap{StepKey: t.StepKey, EventID: m.cause.EventID})
	}
	for i, fid := range path {
		f := m.d.Flows[fid]
		t := m.tok(id)
		if t == nil {
			return
		}
		if i == len(path)-1 {
			m.closeVisit(id, m.leaveVia(t))
			m.disarm(id, t.Node)
			m.enter(id, f.Target)
			return
		}
		m.closeVisit(id, m.leaveVia(t))
		m.disarm(id, t.Node)
		n := m.d.Node(f.Target)
		t.Node, t.StepKey, t.Since, t.Phase, t.RunID = n.ID, n.StepKey(), m.at, PhaseWaiting, ""
		setMap(&m.s.Visits, n.ID, m.s.Visits[n.ID]+1)
		m.s.History = append(m.s.History, Visit{StepKey: n.StepKey(), Node: n.ID, Entered: m.at})
		switch {
		case n.IsTask():
			m.s.Gaps = append(m.s.Gaps, Gap{StepKey: n.StepKey(), EventID: m.cause.EventID})
		case n.Type == NodeThrowEvent && n.Event == eventMessage:
			m.throw(t, n)
		case n.Type == NodeParallelGateway || n.Type == NodeInclusiveGateway:
			next := path[i+1]
			stack, vars, basis := slices.Clone(t.Stack), cloneVars(t.Vars), slices.Clone(t.Basis)
			for _, o := range n.Out {
				if o == next {
					continue
				}
				nid := m.newToken(stack, vars, basis)
				m.enter(nid, m.d.Flows[o].Target)
			}
		}
	}
}

// leaveVia — как покинут узел при догоне: задача без её факта — skipped.
func (m *machine) leaveVia(t *Token) string {
	if n := m.d.Node(t.Node); n != nil && n.IsTask() && t.Phase != PhaseInProgress {
		return "skipped"
	}
	return "completed"
}

// findToken — токен на шаге stepKey; нет — догон; нет пути — nil.
func (m *machine) findToken(stepKey string) *Token {
	if t := m.s.TokenAt(stepKey); t != nil {
		return m.tok(t.ID)
	}
	n := m.d.ByStep(stepKey)
	if n == nil {
		return nil
	}
	return m.catchUp(n.ID)
}

func (m *machine) refuse(kind, stepKey string, code errcodes.Code, format string, args ...any) {
	m.s.Refusals = append(m.s.Refusals, Refusal{Kind: kind, StepKey: stepKey, EventID: m.cause.EventID, Code: string(code), Detail: fmt.Sprintf(format, args...)})
}
