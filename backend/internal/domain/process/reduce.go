package process

import (
	"encoding/json"
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// Применение записи входа изделия к исполнителю (AD-5): какой факт или
// решение двигает токен на каком шаге (FR-44 — только факты и решения,
// предусмотренные версией; через закрывающую точку — только с решением
// человека).

// TypeConcessionGranted — выдача разрешения на отклонение (пачка правок
// контракта Д-24; до появления в каталоге сравнивается по строке).
const TypeConcessionGranted catalog.Type = "decision.concession.granted"

// Полномочия точек предъявления подпроцесса брака: решение по изделию
// (ЗТ-Р) закрывается decision.disposition.set, разрешение на отклонение —
// записью с разрешением.
const (
	authorityDisposition = "nc_disposition"
	authorityConcession  = "concession_approval"
)

type registeredData struct {
	ItemID       string   `json:"item_id"`
	Hash         string   `json:"process_version_hash"`
	LotIDs       []string `json:"lot_ids"`
	EntryStepKey string   `json:"entry_step_key"`
}

type runData struct {
	RunID       string     `json:"operation_run_id"`
	StepKey     string     `json:"step_key"`
	OperatorID  *string    `json:"operator_id"`
	EquipmentID string     `json:"equipment_id"`
	ReworkOf    string     `json:"rework_of"`
	Completion  string     `json:"completion"`
	StartedAt   *time.Time `json:"operation_started_at"`
	FinishedAt  *time.Time `json:"operation_finished_at"`
}

type movementData struct {
	StepKey         string `json:"step_key"`
	DestinationKind string `json:"destination_kind"`
}

type inspectionData struct {
	StepKey         string   `json:"step_key"`
	InspectionPoint string   `json:"inspection_point"`
	Method          string   `json:"method"`
	Outcome         string   `json:"outcome"`
	ProcessingState string   `json:"processing_state"`
	ZoneIDs         []string `json:"zone_ids"`
	OperationRunID  string   `json:"operation_run_id"`
}

type presentationData struct {
	StepKey        string `json:"step_key"`
	PresentationNo int    `json:"presentation_no"`
	Resolution     string `json:"resolution"`
	ConcessionID   string `json:"concession_id"`
}

type dispositionData struct {
	Disposition  string `json:"disposition"`
	ConcessionID string `json:"concession_id"`
}

type interventionData struct {
	ID       string   `json:"intervention_id"`
	ZoneIDs  []string `json:"zone_ids"`
	Rechecks []string `json:"recheck_event_ids"`
}

type waiverData struct {
	ZoneID string `json:"zone_id"`
	Extra  int    `json:"extra_allowed"`
}

type containmentData struct {
	Level string `json:"level"`
}

type obligationData struct {
	ObligationID string `json:"obligation_id"`
}

// signedDecision — решение человека с подписью: вид decision и
// происхождение personal, paper или scenario (AD-2: разрешающее действие не
// опирается только на записи server-attested).
func signedDecision(r kernel.Record) bool {
	return r.Kind == catalog.KindDecision && (r.Provenance == "personal" || r.Provenance == "paper" || r.Provenance == "scenario")
}

func decode[T any](r kernel.Record) (T, bool) {
	var v T
	if err := json.Unmarshal(r.Data, &v); err != nil {
		return v, false
	}
	return v, true
}

// apply — запись входа изделия (после срабатывания истёкших окон).
func (m *machine) apply(r kernel.Record) {
	switch r.Type {
	case catalog.ItemItemRegistered:
		m.register(r)
	case catalog.OperationRunStarted:
		m.runStarted(r)
	case catalog.OperationRunFinished:
		m.runFinished(r)
	case catalog.OperationRunPaused, catalog.OperationRunResumed:
		d, _ := decode[runData](r)
		if run, ok := m.s.Runs[d.RunID]; ok {
			if t := m.s.TokenAt(run.StepKey); t != nil && t.RunID == d.RunID && t.Phase != PhaseBlocked {
				tt := m.tok(t.ID)
				tt.Phase = PhaseInProgress
				if r.Type == catalog.OperationRunPaused {
					tt.Phase = PhasePaused
				}
			}
		}
	case catalog.OperationMovementSent:
		m.movementSent(r)
	case catalog.OperationMovementReceived:
		m.movementReceived(r)
	case catalog.ItemReleaseRecorded:
		m.releaseRecorded()
	case catalog.InspectionResultRecorded:
		m.inspection(r)
	case catalog.ItemPresentationRecorded:
		d, _ := decode[presentationData](r)
		if t := m.findToken(d.StepKey); t != nil {
			g := m.s.Gates[d.StepKey]
			if d.PresentationNo > g.Count {
				g.Count = d.PresentationNo
				g.Authority = requiredAuthority(m.d.ByStep(d.StepKey), g.Count)
				setMap(&m.s.Gates, d.StepKey, g)
				m.setVar(t, VarPresentationNo, Int(int64(g.Count)))
			}
		} else {
			m.refuse("unrouted", d.StepKey, errcodes.ProcessPreconditionFailed, "предъявление на шаге, где изделия нет")
		}
	case catalog.DecisionPresentationResolved:
		d, _ := decode[presentationData](r)
		m.noteDecision(r, d.StepKey)
		m.advanceGate(Presentation{StepKey: d.StepKey, Resolution: d.Resolution, DecisionEventID: r.EventID, PresentationNo: d.PresentationNo, ConcessionID: d.ConcessionID})
	case catalog.DecisionDispositionSet:
		m.disposition(r)
	case catalog.DecisionItemIsolated:
		m.s.Isolated = true
	case catalog.DecisionContainmentSet, catalog.DecisionContainmentApplied:
		if d, ok := decode[containmentData](r); ok {
			m.s.Containment = d.Level
		}
	case catalog.DecisionContainmentReleased:
		m.s.Containment = "none"
	case catalog.DecisionReworkLimitWaived:
		if d, ok := decode[waiverData](r); ok && signedDecision(r) {
			setMap(&m.s.Waivers, d.ZoneID, m.s.Waivers[d.ZoneID]+d.Extra)
		}
	case catalog.ItemInterventionOpened:
		d, _ := decode[interventionData](r)
		setMap(&m.s.Interventions, d.ID, Intervention{ID: d.ID, Zones: d.ZoneIDs, OpenedAt: r.OccurredAt, Open: true})
		for _, z := range d.ZoneIDs {
			zs := m.s.Zones[z]
			zs.Outdated = true
			setMap(&m.s.Zones, z, zs)
		}
	case catalog.ItemInterventionClosed:
		d, _ := decode[interventionData](r)
		if iv, ok := m.s.Interventions[d.ID]; ok {
			iv.Open = false
			setMap(&m.s.Interventions, d.ID, iv)
		}
	case catalog.ObligationDueReached:
		d, _ := decode[obligationData](r)
		for _, tm := range slices.Clone(m.s.Timers) {
			if tm.ObligationID == d.ObligationID && !tm.DueAt.After(r.OccurredAt) {
				m.fire(tm)
			}
		}
	case TypeConcessionGranted:
		m.noteDecision(r, "")
		for _, t := range m.s.Tokens {
			if n := m.d.Node(t.Node); n != nil && n.Presentation != nil && n.Presentation.Authority == authorityConcession {
				m.passGate(t.ID, r, nil)
				break
			}
		}
	}
	m.messages(r)
	m.recheckBlocked()
}

// register — запуск изделия (item.item.registered, Д-5): регистрация
// соответствует заданию из 1С; токен встаёт на стартовое событие-сообщение
// главного процесса (или на entry_step_key — вход в середину процесса).
func (m *machine) register(r kernel.Record) {
	if m.s.StartedAt != nil {
		return
	}
	d, _ := decode[registeredData](r)
	at := r.OccurredAt
	m.s.StartedAt, m.s.Lots = &at, d.LotIDs
	main := m.d.Scopes[m.d.Main]
	if main == nil || len(main.Starts) == 0 {
		return
	}
	start := main.Starts[0]
	for _, sid := range main.Starts {
		if n := m.d.Node(sid); n.Event == eventMessage && n.Props.TriggerEventType == "erp.order.received" {
			start = sid
		}
	}
	id := m.newToken(nil, nil, []string{r.EventID})
	if d.EntryStepKey != "" {
		if n := m.d.ByStep(d.EntryStepKey); n != nil && n.Scope == m.d.Main {
			start = n.ID
		}
	}
	m.enter(id, start)
}

// messages — промежуточные события-приёмы, ждущие записи этого типа (Д-5).
func (m *machine) messages(r kernel.Record) {
	for _, t := range slices.Clone(m.s.Tokens) {
		n := m.d.Node(t.Node)
		if n == nil || n.Type != NodeCatchEvent || n.Event != eventMessage {
			continue
		}
		if catalog.Type(n.Props.TriggerEventType) == r.Type || (n.Props.TriggerEventType == "erp.lot.received" && r.Type == catalog.GenealogyLinkAdded) || m.messageReady(n) {
			m.next(t.ID, "completed")
		}
	}
}

func strp(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// runStarted — начало операции: токен на шаге (или догон), предусловия,
// связь повторов (FR-47) и лимит доработок (FR-18).
func (m *machine) runStarted(r kernel.Record) {
	d, ok := decode[runData](r)
	if !ok || d.RunID == "" {
		setMap(&m.s.Errors, r.EventID, "operation.run.started без operation_run_id")
		return
	}
	if _, dup := m.s.Runs[d.RunID]; dup {
		return
	}
	n := m.d.ByStep(d.StepKey)
	start := r.OccurredAt
	run := Run{RunID: d.RunID, StepKey: d.StepKey, ReworkOf: d.ReworkOf, Equipment: d.EquipmentID, Operator: strp(d.OperatorID),
		StartedAt: start, Causes: []Cause{causeOf(r)}}
	if d.StartedAt != nil {
		run.StartedAt, run.SourceStart = d.StartedAt.UTC(), true
	}
	if n == nil {
		run.Detached = true
		setMap(&m.s.Runs, d.RunID, run)
		m.refuse("unrouted", d.StepKey, errcodes.ProcessPreconditionFailed, "шага %q нет в версии процесса изделия", d.StepKey)
		return
	}
	run.Node, run.Special = n.ID, n.Special()
	var t *Token
	if tk := m.s.TokenAt(d.StepKey); tk != nil {
		t = m.tok(tk.ID)
	} else if !m.leftByTimer(n) {
		t = m.catchUp(n.ID)
	}
	if t == nil {
		run.Detached = true
		setMap(&m.s.Runs, d.RunID, run)
		m.outOfRoute(r, n, d.RunID)
		return
	}
	prev := m.lastRun(n.ID)
	run.N = m.s.runsAt(n.ID, "") + 1
	if run.N > 1 {
		run.Zones = m.s.zonesFor(n)
		if run.ReworkOf == "" && prev != "" {
			run.ReworkOf, run.Inferred = prev, true
		}
	} else {
		run.Zones = slices.Clone(n.Zones)
		if len(run.Zones) == 0 {
			run.Zones = []string{"*"}
		}
	}
	setMap(&m.s.Runs, d.RunID, run)
	t = m.tok(t.ID)
	t.RunID, t.Phase = d.RunID, PhaseInProgress
	m.setVar(t, VarReworkCount, Int(int64(run.N-1)))
	// Д-8: окно until_started закрывается началом операции.
	m.s.Timers = slices.DeleteFunc(m.s.Timers, func(tm Timer) bool {
		return tm.Token == t.ID && tm.Host == n.ID && tm.Scope == timerScopeUntilStart
	})
	fails := m.s.checks(m.env, n, run.Operator, run.Equipment, run.Zones, run.StartedAt, false)
	m.breach(r, n, d.RunID, fails)
}

// breach — нарушения предусловий как реакции; блокирующее держит токен (AD-30:
// внешний факт принимается с сигналом нарушения, затем реакция и блок).
func (m *machine) breach(r kernel.Record, n *Node, runID string, fails []failure) {
	block := ""
	for _, f := range fails {
		key := runID + "|" + f.Kind + "|" + f.Subject
		if runID == "" {
			key = r.EventID + "|" + f.Kind + "|" + f.Subject
		}
		m.s.Breaches = append(m.s.Breaches, Breach{Key: key, StepKey: n.StepKey(), RunID: runID, Precondition: f.Kind, Mode: f.Mode,
			Subject: f.Subject, Detail: f.Detail, Causes: []Cause{causeOf(r)}})
		if f.Mode == preconditionBlock && block == "" {
			block = f.Kind
		}
	}
	if block != "" {
		if t := m.s.TokenAt(n.StepKey()); t != nil {
			tt := m.tok(t.ID)
			tt.Phase, tt.Block = PhaseBlocked, block
		}
	}
}

// outOfRoute — операция на шаге, где изделия нет. Если изделие ушло с шага
// по истёкшему окну (Д-8), это нарушение окна (time_window); иначе — факт
// вне маршрута (виден в State.Refusals, изделие не двигается, FR-44).
func (m *machine) outOfRoute(r kernel.Record, n *Node, runID string) {
	for i := len(m.s.History) - 1; i >= 0; i-- {
		h := m.s.History[i]
		if h.Node == n.ID && h.Via == "timer" {
			m.s.Breaches = append(m.s.Breaches, Breach{Key: runID + "|time_window|", StepKey: n.StepKey(), RunID: runID, Precondition: "time_window",
				Mode: preconditionBlock, Detail: "операция начата после истечения окна — изделие отправлено на повторную подготовку", Causes: []Cause{causeOf(r)}})
			m.refuse("time_window", n.StepKey(), errcodes.ProcessPreconditionFailed, "окно истекло до начала операции")
			return
		}
	}
	code := errcodes.ProcessPreconditionFailed
	if g := m.gateBefore(n); g != "" {
		code = errcodes.NonconformityGateWithoutSignature
	}
	m.refuse("unrouted", n.StepKey(), code, "изделие не на шаге %s (стоит на %v)", n.StepKey(), m.s.Steps())
}

// leftByTimer — изделие последний раз покинуло шаг по истёкшему окну (Д-8):
// догонять на него нельзя — это нарушение окна, а не пропуск данных.
func (m *machine) leftByTimer(n *Node) bool {
	for i := len(m.s.History) - 1; i >= 0; i-- {
		if h := m.s.History[i]; h.Node == n.ID {
			return h.Via == "timer"
		}
	}
	return false
}

// gateBefore — точка предъявления, на которой стоит изделие и за которой
// лежит шаг n (попытка пройти шлагбаум без подписи, FR-19).
func (m *machine) gateBefore(n *Node) string {
	for _, t := range m.s.Tokens {
		g := m.d.Node(t.Node)
		if g != nil && g.IsPresentationPoint() && m.reaches(g.ID, n.ID) {
			return g.StepKey()
		}
	}
	return ""
}

func (m *machine) lastRun(node string) string {
	var last Run
	for _, k := range sortedKeys(m.s.Runs) {
		r := m.s.Runs[k]
		if r.Node == node && !r.Detached && (last.RunID == "" || r.StartedAt.After(last.StartedAt)) {
			last = r
		}
	}
	return last.RunID
}

// runFinished — конец операции: завершённая операция уводит токен дальше,
// прерванная — оставляет на шаге в очереди.
func (m *machine) runFinished(r kernel.Record) {
	d, _ := decode[runData](r)
	run, ok := m.s.Runs[d.RunID]
	if !ok {
		// Конец без начала (пропущено событие, кейс §5.1): выполнение
		// восстанавливается от входа на шаг — интервал вычислен системой.
		t := m.findToken(d.StepKey)
		if t == nil || d.StepKey == "" {
			m.refuse("unrouted", d.StepKey, errcodes.ProcessPreconditionFailed, "конец операции %s без начала и шага", d.RunID)
			return
		}
		n := m.d.Node(t.Node)
		run = Run{RunID: d.RunID, StepKey: n.StepKey(), Node: n.ID, N: m.s.runsAt(n.ID, "") + 1, StartedAt: t.Since, Zones: m.s.zonesFor(n), Special: n.Special()}
		t.RunID = d.RunID
	}
	end := r.OccurredAt
	run.SourceEnd = false
	if d.FinishedAt != nil {
		end, run.SourceEnd = d.FinishedAt.UTC(), true
	}
	run.FinishedAt, run.Completion = &end, d.Completion
	run.Causes = append(run.Causes, causeOf(r))
	setMap(&m.s.Runs, d.RunID, run)
	t := m.s.TokenAt(run.StepKey)
	if t == nil || t.RunID != d.RunID {
		return
	}
	tt := m.tok(t.ID)
	if tt.Phase == PhaseBlocked {
		return
	}
	if d.Completion == "interrupted" {
		tt.Phase, tt.RunID = PhaseWaiting, ""
		return
	}
	m.complete(tt.ID)
}

// complete — задача выполнена: токен уходит по единственной стрелке.
func (m *machine) complete(id int) { m.next(id, "completed") }

// movementSent — отправка (FR-16, FR-130): факт с ключом шага-передачи
// завершает его, и следующий шаг-перемещение начинается «в перемещении»;
// без ключа — шаг-перемещение изделия переходит в «в перемещении».
func (m *machine) movementSent(r kernel.Record) {
	d, _ := decode[movementData](r)
	if d.StepKey != "" {
		if t := m.findToken(d.StepKey); t != nil {
			m.transit = true
			m.complete(t.ID)
			m.transit = false
			return
		}
	}
	for _, t := range m.s.Tokens {
		if n := m.d.Node(t.Node); n != nil && (n.Props.StepKind == stepKindMovement || n.Props.StepKind == stepKindStorage) {
			m.tok(t.ID).Phase = PhaseInTransit
			return
		}
	}
}

// movementReceived — приём на новом месте завершает шаг-перемещение или
// хранение; приём в изолятор — изделие физически в изоляторе (FR-55).
func (m *machine) movementReceived(r kernel.Record) {
	d, _ := decode[movementData](r)
	if d.DestinationKind == "isolator" {
		m.s.Isolated = true
	}
	if d.StepKey != "" {
		if t := m.findToken(d.StepKey); t != nil {
			m.complete(t.ID)
		}
		return
	}
	for _, t := range m.s.Tokens {
		n := m.d.Node(t.Node)
		if n == nil || (n.Props.StepKind != stepKindMovement && n.Props.StepKind != stepKindStorage) {
			continue
		}
		if d.DestinationKind == "isolator" && len(t.Stack) == 0 {
			continue
		}
		m.complete(t.ID)
		return
	}
}

// releaseRecorded — сдача на склад готовой продукции (item.release.recorded,
// Е-74 в описании узла «Сдача на склад готовой продукции»): закрывает шаг-
// перемещение, за которым процесс шлёт в 1С «выпуск годного» (erpAction
// release), как приём на склад (FR-44, FR-91). Шаг уже закрыт приёмом —
// запись только фиксирует выпуск, токен не двигает.
func (m *machine) releaseRecorded() {
	for _, t := range m.s.Tokens {
		n := m.d.Node(t.Node)
		if n == nil || (n.Props.StepKind != stepKindMovement && n.Props.StepKind != stepKindStorage) {
			continue
		}
		for _, id := range m.d.Next(n) {
			if nx := m.d.Node(id); nx != nil && nx.Type == NodeThrowEvent && nx.Props.ErpAction == "release" {
				m.complete(t.ID)
				return
			}
		}
	}
}

// inspection — результат контроля: зоны проверены или с дефектом (FR-20,
// FR-46); шаг контроля выполнен; переменные условий test.result и
// tools.accounted (rules.yaml).
func (m *machine) inspection(r kernel.Record) {
	d, _ := decode[inspectionData](r)
	for _, z := range d.ZoneIDs {
		zs := m.s.Zones[z]
		switch d.Outcome {
		case "no_defect_indicated":
			at := r.OccurredAt
			zs.CheckedAt, zs.CheckedBy, zs.Outdated, zs.Defect = &at, r.EventID, false, false
		case "defect_indicated":
			zs.Defect = true
		}
		setMap(&m.s.Zones, z, zs)
	}
	if d.Outcome == "defect_indicated" && len(d.ZoneIDs) > 0 {
		m.s.DefectZones = slices.Clone(d.ZoneIDs)
	}
	var t *Token
	if d.StepKey != "" {
		t = m.findToken(d.StepKey)
	} else if d.InspectionPoint != "" {
		for _, tk := range m.s.Tokens {
			if n := m.d.Node(tk.Node); n != nil && n.Props.InspectionPoint == d.InspectionPoint {
				t = m.tok(tk.ID)
				break
			}
		}
	}
	step := d.StepKey
	if t != nil {
		step = t.StepKey
	}
	if d.Outcome == "defect_indicated" && step != "" {
		for _, z := range d.ZoneIDs {
			key := step + "|" + z + "|" + itoa(m.s.Visits[m.d.Steps[step]])
			if !m.s.DefectSeen[key] {
				setMap(&m.s.DefectSeen, key, true)
				setMap(&m.s.Defects, step, m.s.Defects[step]+1)
			}
		}
	}
	if t == nil {
		return
	}
	n := m.d.Node(t.Node)
	if d.Method == "leak_test" || (n.Inspection != nil && n.Inspection.Method == "leak_test") {
		v := "invalid"
		switch {
		case d.ProcessingState != "" && d.ProcessingState != "completed":
		case d.Outcome == "no_defect_indicated":
			v = "tight"
		case d.Outcome == "defect_indicated":
			v = "leak"
		}
		m.setVar(t, VarTestResult, Str(v))
	}
	if n.Props.StepKind == stepKindHuman && !n.IsPresentationPoint() {
		m.setVar(t, VarToolsAccounted, Bool(d.Outcome == "no_defect_indicated"))
	}
	// Шаг контроля выполнен результатом: автоматизированный контроль или
	// контроль человеком без точки предъявления (там нужна подпись).
	if !n.IsPresentationPoint() && (n.Type == NodeServiceTask || n.Props.StepKind == stepKindAutomated || n.Props.StepKind == stepKindHuman) && t.Phase != PhaseBlocked {
		m.complete(t.ID)
	}
}

// noteDecision — решение человека во входе изделия (для проверки подписи).
func (m *machine) noteDecision(r kernel.Record, stepKey string) {
	setMap(&m.s.Decisions, r.EventID, Decision{EventID: r.EventID, Type: string(r.Type), StepKey: stepKey, Provenance: r.Provenance, Signed: signedDecision(r)})
}

// advanceGate — продвижение токена через точку предъявления (FR-19, FR-44):
// только по подписанному решению человека, один раз на решение, не при
// открытом вмешательстве для приёмки (FR-21).
func (m *machine) advanceGate(p Presentation) {
	if p.DecisionEventID == "" {
		m.refuse("gate", p.StepKey, errcodes.NonconformityGateWithoutSignature, "нет решения с подписью")
		return
	}
	if m.s.Applied[p.DecisionEventID] {
		return
	}
	dec, ok := m.s.Decisions[p.DecisionEventID]
	if !ok || !dec.Signed {
		m.refuse("gate", p.StepKey, errcodes.NonconformityGateWithoutSignature, "решение %s без подписи человека — точка не пройдена", p.DecisionEventID)
		return
	}
	n := m.d.ByStep(p.StepKey)
	if n == nil || !n.IsPresentationPoint() {
		m.refuse("gate", p.StepKey, errcodes.ProcessPreconditionFailed, "шаг %q — не точка предъявления", p.StepKey)
		return
	}
	t := m.findToken(p.StepKey)
	if t == nil {
		m.refuse("gate", p.StepKey, errcodes.ProcessPreconditionFailed, "изделие не на точке предъявления %s", p.StepKey)
		return
	}
	accept := p.Resolution == ResolutionAccept || p.Resolution == ResolutionAcceptWithConcession
	if accept {
		if iv := m.s.OpenInterventions(); len(iv) > 0 {
			m.refuse("gate", p.StepKey, errcodes.NonconformityInterventionOpen, "открыта запись вмешательства %s — приёмка запрещена", iv[0].ID)
			m.s.Breaches = append(m.s.Breaches, Breach{Key: p.DecisionEventID + "|open_intervention|" + iv[0].ID, StepKey: p.StepKey,
				Precondition: "open_intervention", Mode: preconditionBlock, Subject: iv[0].ID, Detail: "приёмка при открытом вмешательстве",
				Causes: []Cause{m.cause}})
			return
		}
	}
	g := m.s.Gates[p.StepKey]
	g.StepKey = p.StepKey
	if p.PresentationNo > g.Count {
		g.Count = p.PresentationNo
		g.Authority = requiredAuthority(n, g.Count)
	}
	g.Decisions, g.Last = append(slices.Clone(g.Decisions), p.DecisionEventID), p.Resolution
	setMap(&m.s.Gates, p.StepKey, g)
	m.setVar(t, VarDecision, Str(p.Resolution))
	m.passGate(t.ID, kernel.Record{EventID: p.DecisionEventID}, nil)
}

// passGate — подписанное решение закрывает точку: токен уходит дальше,
// основание закрывающей точки — решение (closing_basis сообщений в 1С).
func (m *machine) passGate(id int, dec kernel.Record, vars map[string]Value) {
	t := m.tok(id)
	if t == nil {
		return
	}
	setMap(&m.s.Applied, dec.EventID, true)
	for _, k := range sortedKeys(vars) {
		m.setVar(t, k, vars[k])
	}
	t.Basis = []string{dec.EventID}
	m.complete(id)
}

// disposition — решение по изделию (ЗТ-Р): вариант решения в переменную
// disposition; с разрешением на отклонение закрывает и точку разрешения.
func (m *machine) disposition(r kernel.Record) {
	d, _ := decode[dispositionData](r)
	m.noteDecision(r, "")
	if !signedDecision(r) {
		m.refuse("gate", "", errcodes.NonconformityGateWithoutSignature, "решение по изделию без подписи")
		return
	}
	vars := map[string]Value{VarDisposition: Str(d.Disposition)}
	order := []string{authorityDisposition}
	if d.ConcessionID != "" {
		order = []string{authorityConcession, authorityDisposition}
	}
	for _, auth := range order {
		for _, id := range m.d.Order {
			n := m.d.Nodes[id]
			if n.Presentation == nil || n.Presentation.Authority != auth {
				continue
			}
			if t := m.findToken(n.StepKey()); t != nil {
				m.passGate(t.ID, r, vars)
				return
			}
		}
	}
	// Решение пришло, когда изделие не на точке: вариант запоминается на
	// токенах подпроцесса брака.
	for i := range m.s.Tokens {
		if len(m.s.Tokens[i].Stack) > 0 {
			m.setVar(&m.s.Tokens[i], VarDisposition, Str(d.Disposition))
		}
	}
}

// recheckBlocked — снять блок, если предусловия теперь выполнены (разрешение
// сверх лимита, снятие сдерживания, повторная проверка зоны).
func (m *machine) recheckBlocked() {
	for _, t := range slices.Clone(m.s.Tokens) {
		if t.Phase != PhaseBlocked || t.RunID == "" {
			continue
		}
		run, ok := m.s.Runs[t.RunID]
		n := m.d.Node(t.Node)
		if !ok || n == nil {
			continue
		}
		blocking := false
		for _, f := range m.s.checks(m.env, n, run.Operator, run.Equipment, run.Zones, run.StartedAt, false) {
			if f.Mode == preconditionBlock {
				blocking = true
			}
		}
		if blocking {
			continue
		}
		tt := m.tok(t.ID)
		tt.Phase, tt.Block = PhaseInProgress, ""
		if run.FinishedAt != nil {
			if run.Completion == "interrupted" {
				tt.Phase, tt.RunID = PhaseWaiting, ""
			} else {
				m.complete(t.ID)
			}
		}
	}
}
