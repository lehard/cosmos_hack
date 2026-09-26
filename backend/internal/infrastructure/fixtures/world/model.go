package world

import (
	dom "ant/internal/domain/documents"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	ncapp "ant/internal/application/nonconformity"
	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
	"ant/internal/infrastructure/fixtures/loader"
)

// Model — развёрнутый мир сценария: изделия с маршрутом и осями статусов,
// история (записи журнала), несоответствия, области риска, 1С, источники —
// всё во времени; снимок на шаг курсора берут рендеры ответов.
type Model struct {
	Spec  *Spec
	clk   clock
	Steps []time.Time
	// shifts — шаблоны смен справочника (период «смена» показателей, FR-81);
	// пусто — смены по 8 ч с 00:00, как у live без графика.
	shifts []shiftPattern
	// names — названия зон и оборудования справочников (поля *_label карточки).
	names Names

	Items    []*Item
	itemByID map[string]*Item
	people   map[string]PersonRef
	policy   *Policy

	NCs       []*NC
	ncByID    map[string]*NC
	Incidents []*Incident
	ERP       []*ERPMsg
	Events    []*Event
	Tasks     []*Task
	// Loop — контур улучшений: предложения генераторов и меры (эпик 42).
	Loop Loop

	// BPMN действующей версии: узлы по step_key и по порядку, отпечаток XML (AD-17).
	Bpmn       map[string]*BpmnNode
	BpmnOrder  []*BpmnNode
	BpmnDigest string

	anchor *Item
	nextEv int
	nextER int

	// templates — шаблоны документов нормативного слоя; docs — документы мира
	// (render_documents.go, строятся один раз).
	templates dom.Templates
	docs      []*wdoc
}

// Item — изделие сценария.
type Item struct {
	ID         string // локальный: F-017
	Label      string // Ф-017
	Order      string
	ItemType   string
	Ring       string
	RingLot    string
	Launch     time.Time
	Spec       ItemSpec
	Background bool
	PlanLine   string
	Runs       []*OpRun
	moves      []move
	marks      []mark
}

// OpRun — выполнение операции (FR-47: повтор — новый id со ссылкой на прежнее).
type OpRun struct {
	ID        string // RUN-SV-017-1
	Label     string // СВ-017-1
	Item      *Item
	StepKey   string
	Kind      string // machining | welding | assembly | leak_test
	Equipment string
	Station   string
	Performer string
	Program   string
	From, To  time.Time
	ReworkOf  string
	Line      string
	// LateLog — журнал параметров источника пришёл с опозданием (S07) или
	// потерян (S04): время получения сводки цикла и признак потери.
	LogReceived time.Time
	LogLost     bool
	CurrentA    [2]int // мин–макс ток по журналу, А
}

type move struct {
	at   time.Time
	step string
	pos  string
	loc  string
}

type mark struct {
	at    time.Time
	axis  string // quality | disposition | containment | erp | incident:RS-01 | rework_done | review
	value string
}

// ItemState — снимок изделия на момент.
type ItemState struct {
	Exists      bool
	Step        string
	Position    string
	Location    string
	Line        string
	Quality     string
	Disposition string
	Containment string
	ERP         string
	Incidents   map[string]string
	ReworkDone  bool
	Review      string
	Summary     string
}

// Event — запись журнала мира (факт, реакция, решение, служебная; AD-2).
type Event struct {
	Seq        int64
	ID         string
	Type       string
	Kind       string // fact | reaction | decision | service
	Occurred   time.Time
	Recorded   time.Time
	Item       *Item
	Stream     string
	Source     string
	SourceKind string // manual_entry | machine | sensor | camera | external_system | import
	Provenance string // device | personal | server_attested | scenario
	StepKey    string
	Author     string
	Summary    string
	Params     map[string]string
	Step       int
	Late       bool
	CARef      string
	Entity     loader.Change
	// Materials — материалы наблюдения (иллюстрации, illustrations.go): адреса — в evidence_refs.
	Materials []*illustration
	// Reading — числа режима оборудования (сводка цикла, отклонение): в карточке НС.
	Reading *ncapp.NCParameterReading
}

// Build разворачивает описание мира: разбирает времена, строит маршруты
// изделий, наложения сюжета, записи журнала и шаги.
func Build(spec *Spec, pol *Policy, bpmnXML []byte) (*Model, error) {
	clk, err := newClock(spec.Month, spec.TZ)
	if err != nil {
		return nil, err
	}
	if err := clk.resolveTimes(reflect.ValueOf(spec).Elem()); err != nil {
		return nil, fmt.Errorf("%s: %w", spec.ID, err)
	}
	m := &Model{Spec: spec, clk: clk, itemByID: map[string]*Item{}, people: map[string]PersonRef{}, ncByID: map[string]*NC{}, policy: pol}
	for _, p := range spec.People {
		m.people[p.Person] = p
	}
	if m.Bpmn, m.BpmnOrder, err = ParseBpmn(bpmnXML); err != nil {
		return nil, fmt.Errorf("BPMN: %w", err)
	}
	m.BpmnDigest = Digest(bpmnXML)
	for i, s := range spec.Steps {
		if i > 0 && s.At.Time().Before(spec.Steps[i-1].At.Time()) {
			return nil, fmt.Errorf("шаг %d раньше шага %d", i, i-1)
		}
		m.Steps = append(m.Steps, s.At.Time())
	}
	if err := m.buildItems(); err != nil {
		return nil, err
	}
	if err := m.buildStory(); err != nil {
		return nil, err
	}
	m.finishEvents()
	return m, nil
}

// ─────────────────────────────── изделия ───────────────────────────────

const (
	enterprise = "ENT01"
	flangeType = "FL-100.00.000"
	ringType   = "FL-100.01.002"
)

// FullID — внутренний ID изделия: код_предприятия:локальный_id (AD-16).
func FullID(local string) string { return enterprise + ":" + local }

func labelOf(local string) string {
	switch {
	case strings.HasPrefix(local, "F-"):
		return "Ф-" + local[2:]
	case strings.HasPrefix(local, "R-"):
		return "К-" + local[2:]
	}
	return local
}

func lineOf(src string) string {
	if src == "IS-1" {
		return "LINE-FL-1"
	}
	return "LINE-FL-2"
}

func postOf(src string) string {
	if src == "IS-1" {
		return "WP-WELD-1"
	}
	return "WP-WELD-2"
}

func (m *Model) buildItems() error {
	orderOf := map[string]string{}
	for _, o := range m.Spec.Orders {
		ids, err := expandRange(o.Items)
		if err != nil {
			return err
		}
		for _, id := range ids {
			orderOf[id] = o.ID
		}
	}
	// кольца П-116 — по порядку фланцам без явного кольца; фоновый заказ — П-109.
	ringsOf := map[string][]string{}
	for _, l := range m.Spec.Lots {
		rs, err := expandRange(l.Rings)
		if err != nil {
			return err
		}
		ringsOf[l.ID] = rs
	}
	lotOfRing := map[string]string{}
	for lot, rs := range ringsOf {
		for _, r := range rs {
			lotOfRing[r] = lot
		}
	}
	p116, p109 := ringsOf["LOT-R-116"], ringsOf["LOT-R-109"]
	// фоновые выпущенные изделия
	bg, err := expandRange(m.Spec.Background.Items)
	if err != nil {
		return err
	}
	for i, id := range bg {
		it := &Item{ID: id, Label: labelOf(id), Order: orderOf[id], ItemType: flangeType, Background: true}
		if i < len(p109) {
			it.Ring = p109[i]
		}
		m.backgroundRoute(it, i)
		m.addItem(it)
	}
	for _, s := range m.Spec.Items {
		it := &Item{ID: s.ID, Label: labelOf(s.ID), Order: orderOf[s.ID], ItemType: flangeType, Spec: s}
		switch {
		case s.Ring != "":
			it.Ring = s.Ring
		case strings.HasPrefix(s.ID, "F-2"):
			n := len(bg) + (atoi(s.ID[2:]) - 201 - len(bg))
			if n < len(p109) {
				it.Ring = p109[n]
			}
		case !s.Weld.IsZero() && len(p116) > 0:
			it.Ring, p116 = p116[0], p116[1:]
		}
		m.addItem(it)
	}
	for _, it := range m.Items {
		it.RingLot = lotOfRing[it.Ring]
		if !it.Background {
			if err := m.mainRoute(it); err != nil {
				return err
			}
		}
		if it.Spec.Anchor {
			m.anchor = it
		}
	}
	for _, rw := range m.Spec.Rework {
		it := m.itemByID[rw.Item]
		if it == nil {
			return fmt.Errorf("rework: нет изделия %s", rw.Item)
		}
		m.reworkRoute(it, rw)
	}
	return nil
}

func (m *Model) addItem(it *Item) {
	m.Items = append(m.Items, it)
	m.itemByID[it.ID] = it
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func (it *Item) move(at time.Time, step, pos, loc string) {
	it.moves = append(it.moves, move{at: at, step: step, pos: pos, loc: loc})
}

func (it *Item) mark(at time.Time, axis, value string) {
	it.marks = append(it.marks, mark{at: at, axis: axis, value: value})
}

func minutes(n int) time.Duration { return time.Duration(n) * time.Minute }

// machiningWindow — мехобработка: Ф-001 — 18 09:00–10:02 (S01); остальные по
// 20 мин подряд с 18 10:05; фоновый заказ ЗП-0911 — 17.09.
func (m *Model) machiningWindow(it *Item) (time.Time, time.Time) {
	n := atoi(it.ID[2:])
	switch {
	case it.ID == "F-001":
		return m.clk.at(18, 9, 0), m.clk.at(18, 10, 2)
	case n >= 221:
		from := m.clk.at(17, 9, 0).Add(minutes((n - 221) * 20))
		return from, from.Add(minutes(20))
	default:
		from := m.clk.at(18, 10, 5).Add(minutes((n - 2) * 20))
		return from, from.Add(minutes(20))
	}
}

func (m *Model) shiftPerformer(t time.Time, a, b string) string {
	h := t.In(m.clk.loc).Hour()
	if h >= 8 && (h < 16 || (h == 16 && t.In(m.clk.loc).Minute() < 30)) {
		return a
	}
	return b
}

// mainRoute — маршрут изделия главного потока по вехам ItemSpec (AD-17: шаги — step_key).
func (m *Model) mainRoute(it *Item) error {
	s := it.Spec
	mcFrom, mcTo := m.machiningWindow(it)
	it.Launch = mcFrom.Add(-minutes(30))
	if it.ID == "F-001" || atoi(it.ID[2:]) < 200 {
		it.Launch = m.clk.at(18, 8, 30)
	}
	plan := "LINE-FL-1"
	if s.Src != "" {
		plan = lineOf(s.Src)
	} else if atoi(it.ID[2:])%2 == 0 {
		plan = "LINE-FL-2"
	}
	it.move(it.Launch, "machining.cnc", "in_queue", "WS-MC")
	it.PlanLine = plan
	it.mark(it.Launch, "erp", "accepted_into_work")
	it.mark(it.Launch, "quality", "not_inspected")
	op := m.shiftPerformer(mcFrom, "O17", "O18")
	it.Runs = append(it.Runs, &OpRun{ID: "RUN-MO-" + it.ID[2:] + "-1", Label: "МО-" + it.ID[2:] + "-1", Item: it, StepKey: "machining.cnc", Kind: "machining", Equipment: "CNC-1", Station: "WP-CNC-1", Performer: op, Program: "ЧПУ-ФЛ-100-01", From: mcFrom, To: mcTo, Line: plan})
	it.move(mcFrom, "machining.cnc", "in_progress", "WP-CNC-1")
	it.move(mcTo, "machining.kt2_camera", "at_inspection", "WP-CNC-1")
	it.move(mcTo.Add(minutes(10)), "machining.kt2_cmm", "at_inspection", "ST-CMM")
	it.move(mcTo.Add(minutes(25)), "machining.zt2_acceptance", "at_presentation_point", "WP-QC-MC")
	zt2 := mcTo.Add(minutes(30))
	if it.ID == "F-001" {
		zt2 = m.clk.at(18, 11, 0)
	}
	it.move(zt2, "machining.send_to_welding", "in_storage", "WH-MC")
	it.mark(zt2, "quality", "conforming")
	toWC := m.clk.at(21, 7, 50)
	it.move(toWC.Add(-minutes(10)), "welding.receive", "in_transit", "WS-WC")
	it.move(toWC, "welding.edge_prep", "in_queue", "WS-WC")
	it.mark(toWC.Add(time.Minute), "erp", "moved")
	if s.Weld.IsZero() {
		return nil
	}
	if !s.EdgePrep.IsZero() {
		it.move(s.EdgePrep.Time(), "welding.edge_prep", "in_progress", postOf(s.Src))
	}
	m.weld(it, 1, s.Weld, s.Src, s.Welder, "")
	if err := m.afterWeld(it, s.Xray, s.ZT3, s.ToAC, s.Asm, s.ZT4, s.Test, s.ZT5, s.KT5, s.ZT6, s.Rel); err != nil {
		return err
	}
	for _, mv := range s.Moves {
		it.move(mv.At.Time(), mv.Step, mv.Pos, mv.Loc)
	}
	return nil
}

// weld — выполнение сварки n на источнике src.
func (m *Model) weld(it *Item, n int, w Span, src, welder, reworkOf string) *OpRun {
	r := &OpRun{ID: fmt.Sprintf("RUN-SV-%s-%d", it.ID[2:], n), Label: fmt.Sprintf("СВ-%s-%d", it.ID[2:], n), Item: it, StepKey: "welding.weld", Kind: "welding",
		Equipment: src, Station: postOf(src), Performer: welder, Program: "ПС-4", From: w.From.Time(), To: w.To.Time(), ReworkOf: reworkOf, Line: lineOf(src)}
	it.Runs = append(it.Runs, r)
	it.move(r.From, "welding.weld", "in_progress", r.Station)
	it.mark(r.From, "quality", "not_inspected")
	it.move(r.To, "welding.kt3_camera", "at_inspection", "ST-WELD")
	it.move(r.To.Add(minutes(10)), "welding.kt3_radiography", "in_queue", "ST-WELD")
	return r
}

// afterWeld — рентген, ЗТ-3, рейс в СИЦ, сборка, испытание, окончательный контроль.
func (m *Model) afterWeld(it *Item, xray, zt3, toAC T, asm Span, zt4 T, test Span, zt5, kt5, zt6, rel T) error {
	if xray.IsZero() {
		return nil
	}
	x := xray.Time()
	it.move(x.Add(-minutes(20)), "welding.kt3_radiography", "at_inspection", "LAB-NDT")
	it.move(x, "welding.zt3_acceptance", "at_presentation_point", "WP-QC-WC")
	if zt3.IsZero() {
		return nil
	}
	z := zt3.Time()
	it.move(z, "welding.send_to_assembly", "in_storage", "WH-WC")
	it.mark(z, "quality", "conforming")
	if toAC.IsZero() {
		return nil
	}
	a := toAC.Time()
	it.move(a, "assembly.receive", "in_transit", "WS-AC")
	it.move(a.Add(minutes(10)), "assembly.seal_install", "in_queue", "WS-AC")
	it.mark(a.Add(minutes(11)), "erp", "moved")
	if asm.IsZero() {
		return nil
	}
	f, t := asm.From.Time(), asm.To.Time()
	n := len(it.Runs) + 1
	it.Runs = append(it.Runs, &OpRun{ID: fmt.Sprintf("RUN-SB-%s-%d", it.ID[2:], n), Label: fmt.Sprintf("СБ-%s-%d", it.ID[2:], n), Item: it, StepKey: "assembly.seal_install", Kind: "assembly", Equipment: "TW-1", Station: "WP-ASM-1", Performer: "A31", From: f, To: t})
	it.move(f, "assembly.seal_install", "in_progress", "WP-ASM-1")
	it.move(f.Add(t.Sub(f)/2), "assembly.torque", "in_progress", "WP-ASM-1")
	it.move(t, "assembly.zt4_acceptance", "at_presentation_point", "WP-QC-AC")
	z4 := zt4.Time()
	if zt4.IsZero() {
		z4 = t.Add(minutes(10))
	}
	if z4.After(m.Steps[len(m.Steps)-1]) && zt4.IsZero() {
		return nil
	}
	it.move(z4, "testing.leak_test", "in_queue", "WS-AC")
	if test.IsZero() {
		return nil
	}
	tf, tt := test.From.Time(), test.To.Time()
	it.Runs = append(it.Runs, &OpRun{ID: fmt.Sprintf("RUN-IS-%s-%d", it.ID[2:], n+1), Label: fmt.Sprintf("ИС-%s-%d", it.ID[2:], n+1), Item: it, StepKey: "testing.leak_test", Kind: "leak_test", Equipment: "LEAK-1", Station: "WP-LEAK-1", Performer: "T41", From: tf, To: tt})
	it.move(tf, "testing.leak_test", "in_progress", "WP-LEAK-1")
	it.move(tt, "testing.zt5_protocol", "at_presentation_point", "WP-QC-AC")
	z5 := zt5.Time()
	if zt5.IsZero() {
		z5 = tt.Add(minutes(15))
	}
	it.move(z5, "final.kt5_camera", "in_queue", "WS-QA")
	if kt5.IsZero() {
		return nil
	}
	it.move(kt5.Time(), "final.kt5_camera", "at_inspection", "WP-FINAL-1")
	it.move(kt5.Time().Add(minutes(10)), "final.zt6_acceptance", "at_presentation_point", "WP-FINAL-1")
	if zt6.IsZero() {
		return nil
	}
	it.move(zt6.Time(), "final.release_to_warehouse", "in_transit", "WS-QA")
	if !rel.IsZero() {
		it.move(rel.Time(), "final.released", "completed", "WH-FG")
	}
	return nil
}

// backgroundRoute — фоновый выпущенный фланец ЗП-0911 (сравнение сварщиков, кейс §5.2).
func (m *Model) backgroundRoute(it *Item, i int) {
	b := m.Spec.Background
	welder := b.Welders[i%len(b.Welders)]
	src := b.Sources[(i/2)%len(b.Sources)]
	day := i / 4
	var wf time.Time
	if welder == b.Welders[0] {
		wf = b.From.Time().Add(time.Duration(day)*24*time.Hour + minutes((i%4)*50))
	} else {
		wf = b.From.Time().Add(time.Duration(day)*24*time.Hour + minutes(8*60+(i%4)*50))
	}
	mc := wf.Add(-72 * time.Hour)
	it.Launch = mc.Add(-minutes(30))
	it.move(it.Launch, "machining.cnc", "in_queue", "WS-MC")
	it.PlanLine = lineOf(src)
	it.mark(it.Launch, "erp", "accepted_into_work")
	it.mark(it.Launch, "quality", "not_inspected")
	it.Runs = append(it.Runs, &OpRun{ID: "RUN-MO-" + it.ID[2:] + "-1", Label: "МО-" + it.ID[2:] + "-1", Item: it, StepKey: "machining.cnc", Kind: "machining", Equipment: "CNC-1", Station: "WP-CNC-1", Performer: "O17", Program: "ЧПУ-ФЛ-100-01", From: mc, To: mc.Add(minutes(20)), Line: lineOf(src)})
	it.move(mc, "machining.cnc", "in_progress", "WP-CNC-1")
	it.move(mc.Add(minutes(50)), "machining.send_to_welding", "in_storage", "WH-MC")
	it.mark(mc.Add(minutes(50)), "quality", "conforming")
	it.move(wf.Add(-2*time.Hour), "welding.edge_prep", "in_queue", "WS-WC")
	it.mark(wf.Add(-2*time.Hour), "erp", "moved")
	w := Span{From: T{t: wf}, To: T{t: wf.Add(minutes(40))}}
	m.weld(it, 1, w, src, welder, "")
	rel := wf.Add(30 * time.Hour)
	if max := b.ReleasedBy.Time(); rel.After(max) {
		rel = max.Add(-minutes(10 * (19 - i)))
	}
	it.move(wf.Add(3*time.Hour), "welding.send_to_assembly", "in_storage", "WH-WC")
	it.mark(wf.Add(3*time.Hour), "quality", "conforming")
	it.move(wf.Add(6*time.Hour), "assembly.seal_install", "in_progress", "WP-ASM-1")
	it.move(rel.Add(-2*time.Hour), "testing.leak_test", "in_progress", "WP-LEAK-1")
	it.move(rel.Add(-minutes(30)), "final.zt6_acceptance", "at_presentation_point", "WP-FINAL-1")
	it.move(rel, "final.released", "completed", "WH-FG")
	it.mark(rel.Add(time.Minute), "erp", "released")
}

// reworkRoute — переварка (S10-А): новое выполнение со ссылкой на прежнее (FR-47).
func (m *Model) reworkRoute(it *Item, rw ReworkSpec) {
	prev := ""
	for _, r := range it.Runs {
		if r.Kind == "welding" {
			prev = r.ID
		}
	}
	m.weld(it, 2, rw.Weld, "IS-1", "W21", prev)
	if rw.Xray.IsZero() {
		return
	}
	x := rw.Xray.Time()
	it.move(x.Add(-minutes(20)), "welding.kt3_radiography", "at_inspection", "LAB-NDT")
	it.move(x, "welding.zt3_acceptance", "at_presentation_point", "WP-QC-WC")
	if rw.ZT3.IsZero() {
		return
	}
	z := rw.ZT3.Time()
	it.move(z, "welding.send_to_assembly", "in_storage", "WH-WC")
	it.mark(z, "quality", "conforming")
	it.mark(z, "rework_done", "yes")
	it.mark(z, "containment", "none")
	if rw.ToAC.IsZero() {
		return
	}
	a := rw.ToAC.Time()
	it.move(a, "assembly.receive", "in_transit", "WS-AC")
	it.move(a.Add(minutes(10)), "assembly.seal_install", "in_queue", "WS-AC")
	if !rw.Asm.IsZero() {
		it.move(rw.Asm.From.Time(), "assembly.seal_install", "in_progress", "WP-ASM-1")
		it.move(rw.Asm.To.Time(), "assembly.zt4_acceptance", "at_presentation_point", "WP-QC-AC")
	}
	if !rw.ZT4.IsZero() {
		it.move(rw.ZT4.Time(), "testing.leak_test", "in_queue", "WS-AC")
	}
	if !rw.Test.IsZero() {
		it.move(rw.Test.From.Time(), "testing.leak_test", "in_progress", "WP-LEAK-1")
	}
}

// ─────────────────────────────── состояние ───────────────────────────────

func (it *Item) sortTimeline() {
	sort.SliceStable(it.moves, func(i, j int) bool { return it.moves[i].at.Before(it.moves[j].at) })
	sort.SliceStable(it.marks, func(i, j int) bool { return it.marks[i].at.Before(it.marks[j].at) })
}

// State — снимок изделия на момент t: последнее положение и последняя отметка
// каждой оси с моментом ≤ t (при равных моментах — записанная позже).
func (it *Item) State(t time.Time) ItemState {
	st := ItemState{Quality: "not_inspected", Disposition: "none", Containment: "none", ERP: "not_sent", Incidents: map[string]string{}}
	var best *move
	for i := range it.moves {
		mv := &it.moves[i]
		if !mv.at.After(t) && (best == nil || !mv.at.Before(best.at)) {
			best = mv
		}
	}
	if best != nil {
		st.Exists = true
		st.Step, st.Position, st.Location = best.step, best.pos, best.loc
	}
	// Линия: пост последней сварки; до первой — из плана запуска (процессная сессия §2.4).
	st.Line = it.PlanLine
	for _, r := range it.Runs {
		if r.Kind == "welding" && !r.From.After(t) {
			st.Line = r.Line
		}
	}
	last := map[string]*mark{}
	for i := range it.marks {
		mk := &it.marks[i]
		if mk.at.After(t) {
			continue
		}
		if b := last[mk.axis]; b == nil || !mk.at.Before(b.at) {
			last[mk.axis] = mk
		}
	}
	for axis, mk := range last {
		switch {
		case axis == "quality":
			st.Quality = mk.value
		case axis == "disposition":
			st.Disposition = mk.value
		case axis == "containment":
			st.Containment = mk.value
		case axis == "erp":
			st.ERP = mk.value
		case axis == "rework_done":
			st.ReworkDone = mk.value == "yes"
		case axis == "review":
			st.Review = mk.value
		case strings.HasPrefix(axis, "incident:"):
			st.Incidents[strings.TrimPrefix(axis, "incident:")] = mk.value
		}
	}
	st.Summary = summaryOf(st)
	return st
}

// summaryOf — сводный статус (словарь item_summary) из осей (PRD §3b): сам осью не является.
func summaryOf(s ItemState) string {
	incident := ""
	for _, id := range sortedKeys(s.Incidents) {
		v := s.Incidents[id]
		if v == "confirmed" || incident == "" || (incident == "excluded" && v != "excluded") {
			incident = v
		}
	}
	switch {
	case s.Disposition == "scrap":
		return "scrapped"
	case s.Disposition == "return_to_supplier":
		return "returned"
	case s.Disposition == "rework" && !s.ReworkDone:
		return "in_rework"
	case s.Disposition == "repair":
		return "in_repair"
	case s.Quality == "nonconforming" && s.Disposition == "none":
		return "pending_decision"
	case s.Quality == "signal":
		return "pending_decision"
	case incident == "confirmed" && s.Disposition == "none":
		return "nonconforming"
	case incident == "suspect" || incident == "unknown":
		return "suspect"
	case s.Containment == "item_hold" || s.Containment == "lot_hold":
		return "hold"
	case s.Quality == "unable_to_assess" || s.Containment == "additional_check":
		return "reinspection_required"
	case s.Quality == "accepted_with_concession":
		return "accepted_with_concession"
	case s.Position == "completed":
		return "released"
	case incident == "excluded":
		return "cleared"
	}
	return "in_process"
}

func sortedKeys[V any](mp map[string]V) []string {
	keys := make([]string, 0, len(mp))
	for k := range mp {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// Item — изделие по локальному id.
func (m *Model) Item(id string) *Item { return m.itemByID[id] }

// StepOf — шаг, на котором запись с моментом записи t уже видна.
func (m *Model) StepOf(t time.Time) int {
	for i, c := range m.Steps {
		if !c.Before(t) {
			return i
		}
	}
	return len(m.Steps)
}

func (m *Model) eventID(key string) string {
	return kernel.UUIDv5(constants.NsAnt, "fixtures/"+m.Spec.ID+"/"+key)
}
