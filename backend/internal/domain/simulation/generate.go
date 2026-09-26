package simulation

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Params — параметры запуска генератора.
type Params struct {
	// RunID — пространство имён прогона (AD-38), шаблон ^[a-z0-9][a-z0-9-]{0,62}$.
	RunID string
	// Seed — seed генератора; 0 — seed определения прогона.
	Seed int64
	// Now — доменное «сейчас» на старте прогона (AD-37); нулевое — без сдвига дат.
	Now time.Time
}

var reRunID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// draft — событие источника до нумерации и шума.
type draft struct {
	at       time.Time // когда случилось на самом деле
	skew     time.Duration
	deliver  time.Time // когда доставлено; нулевое — сразу (at + задержка доставки)
	source   string
	typ      string
	ver      int
	item     string
	lot      string
	carrier  string
	level    string
	data     map[string]any
	mutate   *Mutation
	label    string
	scenario string
	order    int
	lost     bool
	seqReset bool
	dups     []time.Time
	conflict map[string]any
	// background — событие маршрута фона: к нему применяется случайный шум.
	background bool
	reordered  bool

	// после нумерации
	n        int
	eventID  string
	seq      int64
	occurred time.Time
}

// gen — состояние генератора одного плана.
type gen struct {
	b      Bundle
	p      Params
	seed   int64
	shift  time.Duration
	ids    *IDMap
	drafts []*draft
	acts   []Action
	order  int
	items  map[string]*ItemTruth
	plans  map[string]*ItemPlan
	welds  []WeldTruth
	// streams — окна потоков журнала оборудования: сводки сварок в окне даёт поток.
	streams []streamWindow
	// outOfSetpoint — посты, у которых сварка шагом карточки уже выдала
	// отклонение «ток вне уставки», а ток в уставку ещё не вернулся.
	outOfSetpoint map[string]bool
	errs          []string
}

type streamWindow struct {
	equipment string
	from, to  time.Time
}

// Generate строит план прогона по определениям и seed (FR-104, AD-26, AD-38).
// Функция чистая: тот же вход — тот же план на любой машине (AD-4).
func Generate(b Bundle, p Params) (*Plan, error) {
	if !reRunID.MatchString(p.RunID) {
		return nil, fmt.Errorf("run_id %q не по шаблону ^[a-z0-9][a-z0-9-]{0,62}$ (AD-38)", p.RunID)
	}
	anchor, err := ParseTime(b.Run.Anchor)
	if err != nil {
		return nil, fmt.Errorf("прогон %s: anchor: %w", b.Run.ID, err)
	}
	end, err := ParseTime(b.Run.End)
	if err != nil {
		return nil, fmt.Errorf("прогон %s: end: %w", b.Run.ID, err)
	}
	seed := p.Seed
	if seed == 0 {
		seed = b.Run.Seed
	}
	g := &gen{b: b, p: p, seed: seed, shift: WeekShift(anchor, p.Now), items: map[string]*ItemTruth{}, plans: map[string]*ItemPlan{}}
	g.ids = NewIDMap(p.RunID, seed, b.World)

	g.collectStreams()
	g.ordersAndLots()
	g.route()
	for i := range b.Scenarios {
		g.scenario(&b.Scenarios[i])
	}
	if len(g.errs) > 0 {
		return nil, fmt.Errorf("определения прогона %s: %s", b.Run.ID, strings.Join(g.errs, "; "))
	}
	g.noise()
	plan := &Plan{RunID: p.RunID, RunDef: b.Run.ID, Seed: seed, Shift: g.shift,
		Start: anchor.Add(g.shift), End: end.Add(g.shift), IDs: g.ids}
	if err := g.number(plan); err != nil {
		return nil, err
	}
	g.actions(plan)
	g.live(plan)
	g.truth(plan)
	return plan, nil
}

// live — живая часть прогона (Д-85): каждое решение с From — только руками
// (остановка до нажатия), кроме решений машины или лаборатории (Auto);
// история до From — demo-signer.
func (g *gen) live(plan *Plan) {
	lp := g.b.Run.Live
	if lp == nil {
		return
	}
	plan.LiveFrom = g.t(lp.From)
	for i := range plan.Actions {
		a := &plan.Actions[i]
		if a.Kind == ActionDecision && !a.At.Before(plan.LiveFrom) && a.Refusal == "" && !a.Auto {
			a.Stop = true
		}
	}
}

func (g *gen) fail(format string, a ...any) { g.errs = append(g.errs, fmt.Sprintf(format, a...)) }

// t — момент определения со сдвигом прогона.
func (g *gen) t(s string) time.Time {
	t, err := ParseTime(s)
	if err != nil {
		g.fail("%v", err)
		return time.Time{}
	}
	return t.Add(g.shift)
}

func (g *gen) add(d *draft) *draft {
	g.order++
	d.order = g.order
	if d.ver == 0 {
		d.ver = 1
	}
	if _, ok := g.b.World.Sources[d.source]; !ok {
		g.fail("источник %q не описан в мире %s (%s %s)", d.source, g.b.World.ID, d.typ, d.label)
	}
	g.drafts = append(g.drafts, d)
	return d
}

func (g *gen) act(a Action) {
	g.acts = append(g.acts, a)
}

// collectStreams — окна потоков журнала оборудования (S07): сводки сварок
// в этих окнах генерирует поток, а не маршрут.
func (g *gen) collectStreams() {
	for _, sc := range g.b.Scenarios {
		for _, st := range sc.Steps {
			if st.Stream != nil {
				g.streams = append(g.streams, streamWindow{equipment: st.Stream.Equipment, from: g.t(st.Stream.From), to: g.t(st.Stream.To)})
			}
		}
	}
}

func (g *gen) inStream(equipment string, t time.Time) bool {
	for _, w := range g.streams {
		if w.equipment == equipment && !t.Before(w.from) && t.Before(w.to) {
			return true
		}
	}
	return false
}

// scenario — шаги карточки: факты, решения, служебные шаги, потоки.
func (g *gen) scenario(sc *ScenarioDef) {
	for i := range sc.Steps {
		st := &sc.Steps[i]
		switch {
		case st.Stream != nil:
			g.stream(sc.ID, st)
		case st.Weld != nil:
			g.stepWeld(sc.ID, st)
		case st.Fact != "":
			g.stepFact(sc.ID, st)
		case st.Decision != "":
			g.stepDecision(sc.ID, st)
		case st.Stand != nil:
			a := Action{Kind: ActionStand, At: g.t(st.At), Scenario: sc.ID, Label: st.Label, Note: st.Note, Stand: st.Stand}
			g.act(a)
			// «до какого момента» — по часам прогона: сбой снимает отдельный
			// шаг в этот момент (часы stand-а — InfraClock, виртуального
			// времени прогона они не знают)
			if st.Stand.Until != "" && !st.Stand.Clear {
				label := ""
				if st.Label != "" {
					label = st.Label + "/clear"
				}
				g.act(Action{Kind: ActionStand, At: g.t(st.Stand.Until), Scenario: sc.ID, Label: label, Note: "снять сбой: " + st.Stand.Fault,
					Stand: &StandAction{Stand: st.Stand.Stand, Clear: true}})
			}
		case st.Tamper != nil:
			g.act(Action{Kind: ActionTamper, At: g.t(st.At), Scenario: sc.ID, Label: st.Label, Note: st.Note, Tamper: st.Tamper})
		case st.Route != "":
			g.routeRef(sc.ID, st)
		case st.System != "":
			// вывод системы — проверяется утверждениями expected, источник его не шлёт
		default:
			g.fail("%s: шаг %q без факта, решения и служебного действия", sc.ID, st.At)
		}
	}
}

// stepWeld — сварка шагом карточки (FR-47: новое выполнение со ссылкой на прежнее).
func (g *gen) stepWeld(scenario string, st *Step) {
	w := st.Weld
	r := g.b.World.Route
	line, ln := g.lineOf(w.Station)
	if line == "" {
		g.fail("%s: сварка %s: пост %s не привязан к линии", scenario, w.Run, w.Station)
		return
	}
	start := g.t(st.At)
	m := w.Minutes
	if m == 0 {
		m = r.WeldMin
	}
	wt := WeldTruth{Item: w.Item, Run: w.Run, Station: w.Station, Welder: w.Welder, Shift: g.shiftOf(start), Start: start,
		End: start.Add(time.Duration(m) * time.Minute), CurrentMin: r.CurrentNominal - 2, CurrentMax: r.CurrentNominal + 3, Rework: w.ReworkOf}
	if len(w.Current) == 2 {
		wt.CurrentMin, wt.CurrentMax = w.Current[0], w.Current[1]
	}
	wt.ArcFrom, wt.ArcTo = wt.Start, wt.End
	g.welds = append(g.welds, wt)
	runID := "{local:" + w.Run + "}"
	program := w.Program
	if program == "" {
		program = r.Program
	}
	body := map[string]any{"operation_code": "SV", "operation_run_id": runID, "step_key": "welding.weld",
		"equipment_id": g.ids.Equipment(w.Station), "station_id": "ST-WELD", "program_ref": program}
	if w.ReworkOf != "" {
		body["rework_of"] = "{local:" + w.ReworkOf + "}"
	}
	actor := g.ids.Person(w.Welder)
	if !w.NoConfirm {
		g.act(Action{Kind: ActionDecision, At: start.Add(-2 * time.Minute), Scenario: scenario, Label: w.Run + "/confirm", Note: "Режим по карте сверен",
			Operation: "access.operator.confirm_step", Role: "performer", Actor: actor, Item: w.Item, Params: map[string]any{"workplace_id": ln.Workplace},
			Body: map[string]any{"step_key": "welding.weld", "item_id": "{item:" + w.Item + "}", "tp_step": "Режим по карте сверен"}})
	}
	note := st.Note
	if note == "" {
		note = "Сварка " + w.Run + " — «Начать»"
	}
	g.act(Action{Kind: ActionDecision, At: start, Scenario: scenario, Label: w.Run + "/start", Note: note, Operation: "process.operation.start",
		Role: "performer", Actor: actor, Item: w.Item, Params: map[string]any{"item_id": "{item:" + w.Item + "}"}, Body: body})
	cycles := g.weldCycles(wt, ln.WeldingSource, runID, NewRand(g.seed, "weld/"+w.Run), false)
	for _, d := range cycles {
		d.scenario = scenario
	}
	g.weldDeviation(scenario, wt, ln.WeldingSource, cycles)
	g.act(Action{Kind: ActionDecision, At: wt.End, Scenario: scenario, Label: w.Run + "/finish", Note: "Сварка " + w.Run + " (" + w.Item + ") — «Выполнено»", Operation: "process.operation.finish",
		Role: "performer", Actor: actor, Item: w.Item, Params: map[string]any{"run_id": runID}, Body: map[string]any{"completion": "completed"}})
}

// weldDeviation — отклонение «ток вне уставки» в момент сварки (показ SHOW-IS2,
// шаг 5; FR-148, FR-151): источник поста выдаёт equipment.deviation.detected в
// ту же минуту, что и первую сводку цикла вне уставки, — предупреждение у
// оборудования и задачи мастеру и руководителю появляются сразу, а не только
// по опоздавшему журналу. Конца отклонения источник не знает (пост
// останавливают, в уставку ток не возвращается) — ended_at нет, окно нарушения
// остаётся открытым. Одно отклонение на эпизод: пока ток поста не вернулся в
// уставку, следующая сварка на нём нового отклонения не даёт.
func (g *gen) weldDeviation(scenario string, w WeldTruth, source string, cycles []*draft) {
	r := g.b.World.Route
	var first *draft
	var peak any
	for _, d := range cycles {
		ps, _ := d.data["parameters"].([]any)
		if len(ps) == 0 {
			continue
		}
		p, _ := ps[0].(map[string]any)
		if _, out := p["out_of_setpoint_ms"]; out && first == nil {
			first, peak = d, p["max"]
		}
	}
	if first == nil {
		// ток в уставке — эпизод (если был) закончился
		delete(g.outOfSetpoint, w.Station)
		return
	}
	if g.outOfSetpoint[w.Station] {
		return
	}
	if g.outOfSetpoint == nil {
		g.outOfSetpoint = map[string]bool{}
	}
	g.outOfSetpoint[w.Station] = true
	data := map[string]any{"equipment_id": g.ids.Equipment(w.Station), "station_id": "ST-WELD", "deviation_kind": "out_of_setpoint",
		"parameter": "current", "value": peak, "started_at": first.data["window_start"],
		"setpoint": map[string]any{"nominal": measurement(int64(r.CurrentNominal), 0, "A"),
			"lower": measurement(int64(r.CurrentNominal-r.CurrentTol), 0, "A"), "upper": measurement(int64(r.CurrentNominal+r.CurrentTol), 0, "A")},
		"code": "I_OUT_OF_SETPOINT"}
	g.add(&draft{at: first.at, source: source, typ: "equipment.deviation.detected", scenario: scenario, data: data,
		label: fmt.Sprintf("%s/deviation/%s", w.Station, w.Run)})
}

// routeRef — строка карточки, которую даёт маршрут изделия: проверяем, что
// маршрут её действительно строит (карточка и прогон не разошлись).
func (g *gen) routeRef(scenario string, st *Step) {
	item, stage, ok := strings.Cut(st.Route, ":")
	switch item {
	case "order":
		if !slices.ContainsFunc(g.b.Run.Orders, func(o OrderPlan) bool { return o.ID == stage }) {
			g.fail("%s: строка маршрута %q: задания нет в прогоне", scenario, st.Route)
		}
		return
	case "lot":
		for _, l := range strings.Split(stage, ",") {
			if !slices.ContainsFunc(g.b.Run.Lots, func(x LotPlan) bool { return x.ID == l }) {
				g.fail("%s: строка маршрута %q: партии %s нет в прогоне", scenario, st.Route, l)
			}
		}
		return
	}
	p, found := g.plans[item]
	if !ok || !found {
		g.fail("%s: строка маршрута %q: изделия нет в прогоне", scenario, st.Route)
		return
	}
	for _, s := range strings.Split(stage, ",") {
		i := stageIndex(s)
		if i < 0 || i > stageIndex(p.Until) || slices.Contains(p.Skip, s) {
			g.fail("%s: строка маршрута %q: этап %s не строится маршрутом изделия (until %s, skip %v)", scenario, st.Route, s, p.Until, p.Skip)
		}
	}
}

func (g *gen) stepFact(scenario string, st *Step) {
	d := &draft{at: g.t(st.At), source: st.Source, typ: st.Fact, ver: st.SchemaVersion, item: st.Item, lot: st.Lot,
		carrier: st.Carrier, level: st.Level, label: st.Label, scenario: scenario, mutate: st.Mutate,
		data: shiftMap(st.Data, g.shift)}
	if st.Skew != "" {
		sk, err := ParseOffset(st.Skew)
		if err != nil {
			g.fail("%s: skew %q: %v", scenario, st.Skew, err)
		}
		d.skew = sk
	}
	if st.Deliver != "" {
		d.deliver = g.t(st.Deliver)
	}
	for _, s := range st.Duplicates {
		d.dups = append(d.dups, g.t(s))
	}
	if st.Conflict != nil {
		d.conflict = shiftMap(st.Conflict, g.shift)
	}
	d.lost, d.seqReset = st.Lost, st.SeqReset
	g.add(d)
}

func (g *gen) stepDecision(scenario string, st *Step) {
	a := Action{Kind: ActionDecision, At: g.t(st.At), Scenario: scenario, Label: st.Label, Note: st.Note,
		Operation: st.Decision, Role: st.Role, Actor: g.ids.Person(st.Actor), Stop: st.Stop || slices.Contains(g.b.Run.Stops, st.Label),
		Params: shiftMap(st.Params, g.shift), Body: shiftMap(st.Body, g.shift), Item: st.Item, Auto: st.Auto}
	if st.Refusal != "" {
		a.Refusal = g.ids.Code(st.Refusal)
	}
	g.act(a)
}

func shiftMap(m map[string]any, d time.Duration) map[string]any {
	if m == nil {
		return nil
	}
	return ShiftValue(m, d).(map[string]any)
}

// number — нумерация событий (N, event_id, source_seq), метки, конверты,
// доставка с повторами и конфликтами (AD-38: event_id ставит генератор).
func (g *gen) number(plan *Plan) error {
	slices.SortStableFunc(g.drafts, func(a, b *draft) int {
		if c := a.at.Compare(b.at); c != 0 {
			return c
		}
		return a.order - b.order
	})
	for i, d := range g.drafts {
		d.n = i + 1
		d.eventID = g.ids.EventID(d.n)
		d.occurred = d.at.Add(d.skew)
		if d.label != "" {
			if _, dup := g.ids.Labels[d.label]; dup {
				return fmt.Errorf("метка события %q повторяется", d.label)
			}
			g.ids.Labels[d.label] = d.eventID
		}
	}
	// source_seq — по порядку возникновения у источника; потерянные тоже
	// получают номер (разрыв source_seq виден приёму и верификатору, AD-7, AD-9).
	bySource := map[string][]*draft{}
	for _, d := range g.drafts {
		bySource[d.source] = append(bySource[d.source], d)
	}
	for _, key := range slices.Sorted(maps.Keys(bySource)) {
		ds := bySource[key]
		base := g.b.World.Sources[key].SeqStart
		if base == 0 {
			base = 1
		}
		if anchor, idx, ok := g.seqAnchor(key, ds); ok {
			base = anchor - int64(idx)
			if base < 1 {
				return fmt.Errorf("источник %s: номер первой потерянной записи %d меньше числа записей до неё", key, anchor)
			}
		}
		next := base
		for _, d := range ds {
			if d.seqReset {
				next = 1
			}
			d.seq = next
			next++
		}
	}
	lat := map[string]*Rand{}
	var em []Emission
	order := 0
	for _, d := range g.drafts {
		if d.lost {
			continue
		}
		body, contract, err := g.envelope(d, d.data)
		if err != nil {
			return err
		}
		r, ok := lat[d.source]
		if !ok {
			r = NewRand(g.seed, "latency/"+d.source)
			lat[d.source] = r
		}
		deliver := d.deliver
		delivery := DeliveryNormal
		if deliver.IsZero() {
			deliver = d.at.Add(time.Duration(r.Between(300, 1500)) * time.Millisecond)
		} else if deliver.Sub(d.at) > 5*time.Minute {
			delivery = DeliveryLate
		}
		if d.reordered {
			delivery = DeliveryReordered
		}
		order++
		base := Emission{N: d.n, Order: order, DeliverAt: deliver, SourceKey: d.source, SourceID: g.ids.SourceID(g.b.World.Sources[d.source].ID),
			SourceSeq: d.seq, EventID: d.eventID, EventType: d.typ, OccurredAt: d.occurred, TrueAt: d.at, Item: d.item,
			Scenario: d.scenario, Label: d.label, Delivery: delivery, Contract: contract, Event: body,
			Quarantine: d.mutate != nil && d.mutate.Quarantine}
		em = append(em, base)
		for _, at := range d.dups {
			order++
			dup := base
			dup.Order, dup.DeliverAt, dup.Delivery = order, at, DeliveryDuplicate
			em = append(em, dup)
		}
		if d.conflict != nil {
			cb, _, err := g.envelope(d, d.conflict)
			if err != nil {
				return err
			}
			order++
			c := base
			c.Order, c.Delivery, c.Event = order, DeliveryConflict, cb
			c.DeliverAt = deliver.Add(time.Minute)
			em = append(em, c)
		}
	}
	slices.SortStableFunc(em, func(a, b Emission) int {
		if c := a.DeliverAt.Compare(b.DeliverAt); c != 0 {
			return c
		}
		return a.Order - b.Order
	})
	plan.Emissions = em
	return nil
}

// seqAnchor — привязка номеров источника к номеру первой потерянной записи (S04: разрыв 4711–4745).
func (g *gen) seqAnchor(key string, ds []*draft) (int64, int, bool) {
	for _, sc := range g.b.Scenarios {
		for _, st := range sc.Steps {
			if st.Stream == nil || st.Stream.Source != key || st.Stream.Lost == nil || st.Stream.Lost.FirstSeq == 0 {
				continue
			}
			for i, d := range ds {
				if d.lost {
					return st.Stream.Lost.FirstSeq, i, true
				}
			}
		}
	}
	return 0, 0, false
}

// envelope — исходное событие (конверт v1, contracts/events/common/envelope.v1.json).
func (g *gen) envelope(d *draft, data map[string]any) (json.RawMessage, bool, error) {
	src := g.b.World.Sources[d.source]
	exp, err := g.ids.ExpandMap(data, false)
	if err != nil {
		return nil, false, err
	}
	if exp == nil {
		exp = map[string]any{}
	}
	ev := map[string]any{
		"event_id":       d.eventID,
		"event_type":     d.typ,
		"schema_version": d.ver,
		"source_id":      g.ids.SourceID(src.ID),
		"source_seq":     d.seq,
		"source_kind":    src.Kind,
		"reliability":    src.Reliability,
		"occurred_at":    FormatTime(d.occurred),
		"correlation_id": d.eventID,
		"causation_id":   nil,
		"run_id":         g.ids.RunID,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost",
			"signers": []any{"scenario." + strings.ToLower(src.ID) + "@1"}},
		"data": exp,
	}
	if d.item != "" && d.carrier != "none" {
		ev["item_ref"] = g.itemRef(d)
	}
	contract := true
	if m := d.mutate; m != nil {
		for _, p := range m.Drop {
			dropPath(ev, p)
		}
		for _, p := range slices.Sorted(maps.Keys(m.Set)) {
			v, err := g.ids.Expand(ShiftValue(m.Set[p], g.shift), false)
			if err != nil {
				return nil, false, err
			}
			setPath(ev, p, v)
		}
		contract = !m.Invalid
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return nil, false, fmt.Errorf("событие %d (%s): %w", d.n, d.typ, err)
	}
	return b, contract, nil
}

// itemRef — носитель изделия на момент события (AD-16, AD-41): до
// перемаркировки — бирка на таре, после — DataMatrix.
func (g *gen) itemRef(d *draft) map[string]any {
	level := d.level
	if level == "" {
		level = "unique"
	}
	kind, value := "dpm_datamatrix", "DM:"+d.item
	switch d.carrier {
	case "tag":
		kind, value = "tag_qr", "TAG:"+d.item
	case "dm":
	case "":
		if it, ok := g.items[d.item]; ok && (it.RemarkedAt.IsZero() || d.at.Before(it.RemarkedAt)) {
			kind, value = "tag_qr", "TAG:"+d.item
		}
	default:
		kind, value = "post_context", d.carrier
	}
	return map[string]any{"carrier_type": kind, "value": g.ids.Local(value), "identification_level": level}
}

func dropPath(m map[string]any, p string) {
	parts := strings.Split(p, "/")
	for i, k := range parts {
		if i == len(parts)-1 {
			delete(m, k)
			return
		}
		next, ok := m[k].(map[string]any)
		if !ok {
			return
		}
		m = next
	}
}

func setPath(m map[string]any, p string, v any) {
	parts := strings.Split(p, "/")
	for i, k := range parts {
		if i == len(parts)-1 {
			m[k] = v
			return
		}
		next, ok := m[k].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[k] = next
		}
		m = next
	}
}

// actions — шаги по времени; command_id — по номеру шага.
func (g *gen) actions(plan *Plan) {
	slices.SortStableFunc(g.acts, func(a, b Action) int { return a.At.Compare(b.At) })
	for i := range g.acts {
		g.acts[i].Seq = i + 1
		// метки событий известны — подставляем их в решения заранее
		if p, err := g.ids.ExpandMap(g.acts[i].Params, false); err == nil {
			g.acts[i].Params = p
		}
		if b, err := g.ids.ExpandMap(g.acts[i].Body, false); err == nil {
			g.acts[i].Body = b
		}
	}
	plan.Actions = g.acts
}

// truth — «истина» и счётчики доставки.
func (g *gen) truth(plan *Plan) {
	var tr Truth
	for _, id := range slices.Sorted(maps.Keys(g.items)) {
		tr.Items = append(tr.Items, *g.items[id])
	}
	slices.SortStableFunc(g.welds, func(a, b WeldTruth) int { return a.Start.Compare(b.Start) })
	tr.Welds = g.welds
	per := map[string]*SourceTruth{}
	get := func(key string) *SourceTruth {
		s, ok := per[key]
		if !ok {
			s = &SourceTruth{Key: key, SourceID: g.ids.SourceID(g.b.World.Sources[key].ID)}
			per[key] = s
		}
		return s
	}
	for _, d := range g.drafts {
		s := get(d.source)
		s.Counters.Records++
		if s.FirstSeq == 0 || d.seq < s.FirstSeq {
			s.FirstSeq = d.seq
		}
		s.LastSeq = max(s.LastSeq, d.seq)
		if d.lost {
			s.Counters.Lost++
			s.LostSeqs = append(s.LostSeqs, d.seq)
			continue
		}
		if d.skew != 0 {
			s.Counters.Skewed++
			s.SkewSec = int(d.skew / time.Second)
		}
		if d.mutate != nil && d.mutate.Invalid {
			s.Counters.Invalid++
		}
		if d.mutate != nil && d.mutate.Quarantine {
			s.Counters.Quarantine++
		}
	}
	for _, e := range plan.Emissions {
		s := get(e.SourceKey)
		s.Counters.Deliveries++
		switch e.Delivery {
		case DeliveryDuplicate:
			s.Counters.Duplicates++
		case DeliveryConflict:
			s.Counters.Conflicts++
		default:
			s.Counters.Unique++
			if e.Delivery == DeliveryLate {
				s.Counters.Late++
			}
			if e.Delivery == DeliveryReordered {
				s.Counters.Reordered++
			}
		}
	}
	for _, k := range slices.Sorted(maps.Keys(per)) {
		tr.Sources = append(tr.Sources, *per[k])
		tr.Totals.Add(per[k].Counters)
	}
	for _, a := range plan.Actions {
		if a.Kind == ActionDecision {
			tr.Totals.Decisions++
		}
	}
	plan.Truth = tr
}
