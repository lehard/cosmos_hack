package machinelogs

import (
	"cmp"
	"encoding/json"
	"maps"
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
)

// Функции machinelogs в межизделийной стадии (AD-42, AD-29):
//
//  1. Временная линия оборудования: события оборудования приходят без
//     изделия (поток equipment:‹id›) и копятся по оборудованию.
//  2. Привязка к выполнению операции: когда интервал выполнения закрыт,
//     события его оборудования в интервале (и обстановка до начала —
//     последние программа, инструмент, состояние) адресуются в поток изделия
//     записью equipment.event.bound; позднее событие, попавшее в закрытый
//     интервал, привязывается при поступлении. Edge-агент operation_run_id не
//     подставляет — привязывает только стадия.
//  3. Специальный процесс (FR-151): параметр вне уставки на оборудовании
//     образует окно нарушения; когда окно закрыто, стадия пишет
//     equipment.violation.window_resolved и для каждого выполнения шага со
//     специальным процессом в окне — запрос несоответствия через порт
//     Registrar (функция-намерение модуля nonconformity) — даже если дефект
//     при контроле не найден.
//
// Всё — чистые функции записи стадии (AD-4): время — только из записей.

// Retention — сколько стадия помнит события оборудования и закрытые
// выполнения и окна после самого позднего известного момента (по времени
// записей, не по часам): поздние данные в этих пределах ещё привязываются.
const Retention = 72 * time.Hour

// StageEnv — нормативная часть стадии: шаги-специальные процессы (FR-151).
// Пустой список — DefaultSpecialSteps (до передачи признака эпиком 17).
type StageEnv = Env

// NCRequest — запрос регистрации несоответствия выполнению операции в окне
// нарушения режима специального процесса (FR-151).
type NCRequest struct {
	// NCID — детерминированный id группового несоответствия окна (UUIDv5
	// окна): один на все выполнения окна (FR-151, решение комиссии — одно).
	NCID           string    `json:"nc_id"`
	ItemID         string    `json:"item_id"`
	OperationRunID string    `json:"operation_run_id"`
	StepKey        string    `json:"step_key,omitempty"`
	EquipmentID    string    `json:"equipment_id"`
	RunID          string    `json:"run_id,omitempty"`
	WindowEventID  string    `json:"violation_window_event_id"`
	WindowStart    time.Time `json:"window_start"`
	WindowEnd      time.Time `json:"window_end"`
	// Causes — отклонения окна и записи выполнения.
	Causes []Cause `json:"causes"`
	// Err — почему порт не зарегистрировал несоответствие (пусто — не вызывался).
	Err string `json:"error,omitempty"`
}

// Key — ключ идемпотентности адресованной записи несоответствия.
func (q NCRequest) Key() string { return q.WindowEventID + "|" + q.OperationRunID }

// CauseRecords — причины как записи для kernel.NewAddressed.
func (q NCRequest) CauseRecords() []kernel.Record {
	out := make([]kernel.Record, 0, len(q.Causes))
	for _, c := range q.Causes {
		out = append(out, c.Record())
	}
	return out
}

// Registrar — порт к функции-намерению модуля nonconformity «зарегистрировать
// несоответствие окна нарушения специального процесса» (FR-151, AD-40):
// строит адресованную запись decision.nonconformity.registered в поток
// изделия. machinelogs стоит в композиции раньше nonconformity и не может
// импортировать его, поэтому функцию подставляет сборка стадии
// (domain/crossitem): StageWith(StagePorts{Register: nonconformity.…}).
// Пока модуля nonconformity нет (эпик 21), порт пуст: запросы копятся в
// StageState.Pending и видны в окнах нарушений.
type Registrar func(req NCRequest) (kernel.Addressed, error)

// StagePorts — нормативная часть и порты стадии machinelogs.
type StagePorts struct {
	Env      StageEnv
	Register Registrar
}

// StageRun — выполнение операции в стадии и уже привязанные к нему события.
type StageRun struct {
	Run
	BoundIDs []string `json:"bound,omitempty"`
}

// Window — окно нарушения режима на оборудовании (FR-151): от начала первого
// выхода параметра за уставку до конца последнего перекрывающегося.
type Window struct {
	// ID — event_id первого отклонения окна.
	ID           string     `json:"id"`
	RunID        string     `json:"run_id,omitempty"`
	EquipmentID  string     `json:"equipment_id"`
	Start        time.Time  `json:"start"`
	End          *time.Time `json:"end,omitempty"`
	DeviationIDs []string   `json:"deviation_ids"`
	Causes       []Cause    `json:"causes"`
	// EventID — id записи equipment.violation.window_resolved (после закрытия).
	EventID string `json:"event_id,omitempty"`
	// Runs — выполнения окна, по которым запрошено несоответствие.
	Runs []string `json:"runs,omitempty"`
}

// StageState — состояние модуля machinelogs в межизделийной стадии (AD-42):
// выполнения, временные линии оборудования, окна нарушений и запросы
// несоответствий без порта. Сериализуется в проекцию стадии.
type StageState struct {
	Runs    map[string]StageRun `json:"runs,omitempty"`
	Tracks  map[string][]Event  `json:"tracks,omitempty"`
	Windows map[string]Window   `json:"windows,omitempty"`
	Pending []NCRequest         `json:"pending,omitempty"`
	// Latest — самый поздний известный момент (по записям) — отсчёт Retention.
	Latest time.Time `json:"latest,omitzero"`
}

// Stage — функция модуля machinelogs, подключаемая к межизделийной стадии
// (AD-42, точка подключения domain/crossitem.Modules): нормативная часть по
// умолчанию, порт несоответствий пуст (до эпика 21 — заготовка).
func Stage(s StageState, r kernel.Record) (StageState, []kernel.Addressed) {
	return StageWith(StagePorts{})(s, r)
}

// StageWith — функция стадии с портами p.
func StageWith(p StagePorts) func(StageState, kernel.Record) (StageState, []kernel.Addressed) {
	return func(s StageState, r kernel.Record) (StageState, []kernel.Addressed) {
		st := &stage{s: s.clone(), p: p}
		switch {
		case IsRunRecord(r.Type):
			st.run(r)
		case IsEquipmentFact(r.Type):
			st.event(r)
		default:
			return s, nil
		}
		st.prune()
		return st.s, st.out
	}
}

type stage struct {
	s   StageState
	p   StagePorts
	out []kernel.Addressed
}

func (s StageState) clone() StageState {
	s.Runs, s.Tracks, s.Windows = maps.Clone(s.Runs), maps.Clone(s.Tracks), maps.Clone(s.Windows)
	if s.Runs == nil {
		s.Runs = map[string]StageRun{}
	}
	if s.Tracks == nil {
		s.Tracks = map[string][]Event{}
	}
	if s.Windows == nil {
		s.Windows = map[string]Window{}
	}
	s.Pending = slices.Clone(s.Pending)
	return s
}

// trackKey — временная линия оборудования в пространстве прогона (AD-38).
func trackKey(runID, equipmentID string) string { return runID + "|" + equipmentID }

func (st *stage) seen(t time.Time) {
	if t.After(st.s.Latest) {
		st.s.Latest = t
	}
}

// run — запись выполнения: интервал, привязка при закрытии, окна.
func (st *stage) run(r kernel.Record) {
	id := RunID(r)
	if id == "" {
		return
	}
	cur := st.s.Runs[id]
	run, err := ApplyRun(cur.Run, r)
	if err != nil {
		return
	}
	cur.Run = run
	st.s.Runs[id] = cur
	st.seen(run.StartedAt)
	if run.FinishedAt != nil {
		st.seen(*run.FinishedAt)
	}
	if run.Closed() && run.EquipmentID != "" {
		st.bindRun(id)
	}
	for _, wid := range st.windowIDs() {
		st.windowRun(wid, id)
	}
}

// event — факт оборудования: временная линия, поздняя привязка, окна.
func (st *stage) event(r kernel.Record) {
	ev, _, err := ParseEvent(r)
	if err != nil {
		return
	}
	ev.Data = r.Data
	key := trackKey(r.RunID, ev.EquipmentID)
	tr := st.s.Tracks[key]
	if slices.ContainsFunc(tr, func(x Event) bool { return x.EventID == ev.EventID }) {
		return
	}
	tr = append(slices.Clone(tr), ev)
	slices.SortStableFunc(tr, eventOrder)
	st.s.Tracks[key] = tr
	st.seen(ev.Until())

	// Позднее событие в уже закрытом интервале — привязка сразу.
	for _, id := range slices.Sorted(maps.Keys(st.s.Runs)) {
		run := st.s.Runs[id]
		if run.Closed() && run.EquipmentID == ev.EquipmentID && run.ScenarioRun == r.RunID && run.Overlaps(ev.Start, ev.Until()) {
			st.bind(id, ev, BindingInterval)
		}
	}
	// Окна нарушения режима (FR-151).
	switch {
	case ev.Type == catalog.EquipmentDeviationDetected && ViolatesRegime(ev.DeviationKind):
		st.deviation(r, ev)
	case closesWindow(ev):
		for _, wid := range st.windowIDs() {
			w := st.s.Windows[wid]
			if w.End == nil && w.EquipmentID == ev.EquipmentID && w.RunID == r.RunID && !ev.Start.Before(w.Start) {
				end := ev.Start
				w.End = &end
				w.Causes = addCause(w.Causes, Cause{EventID: ev.EventID, OccurredAt: ev.OccurredAt})
				st.s.Windows[wid] = w
				st.resolve(wid)
			}
		}
	}
}

func eventOrder(a, b Event) int {
	if x := a.Start.Compare(b.Start); x != 0 {
		return x
	}
	return cmp.Compare(a.EventID, b.EventID)
}

// closesWindow — событие подтверждает нормальный режим: исправность «норма»
// или сводка цикла, все параметры которой в пределах уставки.
func closesWindow(ev Event) bool {
	switch ev.Type {
	case catalog.EquipmentStateChanged:
		return ev.Condition == ConditionNormal && ev.Execution == ExecutionRunning
	case catalog.EquipmentCycleSummarized:
		known := false
		for _, p := range ev.Parameters {
			in := p.InRange()
			if in == nil {
				continue
			}
			if !*in {
				return false
			}
			known = true
		}
		return known
	}
	return false
}

// deviation — выход параметра за уставку: новое окно или расширение
// перекрывающегося открытого/незакрытого окна того же оборудования.
func (st *stage) deviation(r kernel.Record, ev Event) {
	c := Cause{EventID: ev.EventID, OccurredAt: ev.OccurredAt}
	for _, wid := range st.windowIDs() {
		w := st.s.Windows[wid]
		if w.EquipmentID != ev.EquipmentID || w.RunID != r.RunID || w.EventID != "" {
			continue
		}
		if w.End != nil && ev.Start.After(*w.End) {
			continue
		}
		if ev.Start.Before(w.Start) {
			w.Start = ev.Start
		}
		w.End = maxEnd(w.End, ev.End)
		w.DeviationIDs = appendSorted(w.DeviationIDs, ev.EventID)
		w.Causes = addCause(w.Causes, c)
		st.s.Windows[wid] = w
		if w.End != nil {
			st.resolve(wid)
		}
		return
	}
	w := Window{ID: ev.EventID, RunID: r.RunID, EquipmentID: ev.EquipmentID, Start: ev.Start, End: ev.End,
		DeviationIDs: []string{ev.EventID}, Causes: []Cause{c}}
	st.s.Windows[w.ID] = w
	if w.End != nil {
		st.resolve(w.ID)
	}
}

func maxEnd(a, b *time.Time) *time.Time {
	switch {
	case a == nil:
		return nil
	case b == nil:
		return nil
	case b.After(*a):
		return b
	}
	return a
}

func appendSorted(xs []string, x string) []string {
	if slices.Contains(xs, x) {
		return xs
	}
	out := append(slices.Clone(xs), x)
	slices.Sort(out)
	return out
}

func (st *stage) windowIDs() []string { return slices.Sorted(maps.Keys(st.s.Windows)) }

// resolve — окно закрыто: запись окна и несоответствия выполнений в нём.
func (st *stage) resolve(wid string) {
	w := st.s.Windows[wid]
	if w.End == nil || w.EventID != "" {
		return
	}
	var runs, items []string
	var step string
	for _, id := range slices.Sorted(maps.Keys(st.s.Runs)) {
		if run := st.s.Runs[id]; st.inWindow(w, run.Run) {
			runs = append(runs, id)
			items = appendSorted(items, run.ItemID)
			if step == "" {
				step = run.StepKey
			} else if step != run.StepKey {
				step = "-"
			}
		}
	}
	data := windowData{EquipmentID: w.EquipmentID, WindowStart: ts(w.Start), WindowEnd: ts(*w.End),
		DeviationEventIDs: w.DeviationIDs, AffectedOperationRunIDs: nonNil(runs), AffectedItemIDs: items}
	if step != "" && step != "-" {
		data.StepKey = step
	}
	stream := "equipment:" + w.EquipmentID
	a, err := kernel.NewAddressed(Module, catalog.EquipmentViolationWindowResolved, stream, w.ID, data, causes(w.Causes)...)
	if err != nil {
		return
	}
	w.EventID = WindowEventID(stream, w.ID)
	st.s.Windows[wid] = w
	st.out = append(st.out, a)
	for _, id := range runs {
		st.windowRun(wid, id)
	}
}

// inWindow — выполнение шага со специальным процессом на оборудовании окна
// пересекается с окном (FR-151).
func (st *stage) inWindow(w Window, run Run) bool {
	return w.End != nil && run.EquipmentID == w.EquipmentID && run.ScenarioRun == w.RunID &&
		run.ItemID != "" && !run.StartedAt.IsZero() && run.IsSpecial(st.p.Env) && run.Overlaps(w.Start, *w.End)
}

// windowRun — несоответствие выполнению id в закрытом окне wid (один раз).
func (st *stage) windowRun(wid, id string) {
	w := st.s.Windows[wid]
	run := st.s.Runs[id].Run
	if w.EventID == "" || slices.Contains(w.Runs, id) || !st.inWindow(w, run) {
		return
	}
	w.Runs = appendSorted(w.Runs, id)
	st.s.Windows[wid] = w
	q := NCRequest{ItemID: run.ItemID, OperationRunID: id, StepKey: run.StepKey, EquipmentID: w.EquipmentID, RunID: w.RunID,
		WindowEventID: w.EventID, WindowStart: w.Start, WindowEnd: *w.End}
	// Групповое несоответствие (NC-G1, S05): один id на окно нарушения —
	// решение комиссии одно на все изделия окна и приходит каждому из них;
	// запись в поток каждого изделия — своя (ключ окна и выполнения).
	q.NCID = "NC-SP-" + kernel.UUIDv5(constants.NsAnt, "machinelogs.special_process\x1f"+w.EventID)
	q.Causes = append(slices.Clone(w.Causes), run.Causes...)
	slices.SortStableFunc(q.Causes, func(a, b Cause) int { return cmp.Compare(a.EventID, b.EventID) })
	if st.p.Register == nil {
		st.s.Pending = append(st.s.Pending, q)
		return
	}
	a, err := st.p.Register(q)
	if err != nil {
		q.Err = err.Error()
		st.s.Pending = append(st.s.Pending, q)
		return
	}
	st.out = append(st.out, a)
}

// bindRun — привязка событий оборудования к закрытому выполнению id:
// события в интервале и обстановка до начала.
func (st *stage) bindRun(id string) {
	run := st.s.Runs[id].Run
	tr := st.s.Tracks[trackKey(run.ScenarioRun, run.EquipmentID)]
	ctx := map[catalog.Type]Event{}
	for _, ev := range tr {
		if run.Overlaps(ev.Start, ev.Until()) {
			st.bind(id, ev, BindingInterval)
			continue
		}
		if ev.Until().Before(run.StartedAt) {
			switch ev.Type {
			case catalog.EquipmentProgramChanged, catalog.EquipmentToolChanged, catalog.EquipmentStateChanged:
				ctx[ev.Type] = ev // последнее до начала — по порядку временной линии
			}
		}
	}
	for _, t := range slices.Sorted(maps.Keys(ctx)) {
		st.bind(id, ctx[t], BindingContext)
	}
}

// bind — запись привязки события ev к выполнению id в поток его изделия.
func (st *stage) bind(id string, ev Event, binding string) {
	sr := st.s.Runs[id]
	if sr.ItemID == "" || slices.Contains(sr.BoundIDs, ev.EventID) {
		return
	}
	data := boundData{OperationRunID: id, EquipmentID: ev.EquipmentID, Binding: binding, SubjectEventID: ev.EventID,
		SubjectEventType: string(ev.Type), SubjectOccurredAt: ts(ev.OccurredAt), SubjectSourceKind: ev.SourceKind,
		SubjectData: ev.Data}
	if len(data.SubjectData) == 0 {
		data.SubjectData = []byte("{}")
	}
	cs := append(slices.Clone(sr.Causes), Cause{EventID: ev.EventID, OccurredAt: ev.OccurredAt})
	a, err := kernel.NewAddressed(Module, catalog.EquipmentEventBound, "item:"+sr.ItemID, id+"|"+ev.EventID, data, causes(cs)...)
	if err != nil {
		return
	}
	sr.BoundIDs = appendSorted(sr.BoundIDs, ev.EventID)
	st.s.Runs[id] = sr
	st.out = append(st.out, a)
}

// prune — забыть то, что старше Retention от самого позднего момента:
// события оборудования, закрытые выполнения и разрешённые окна.
func (st *stage) prune() {
	if st.s.Latest.IsZero() {
		return
	}
	edge := st.s.Latest.Add(-Retention)
	for _, k := range slices.Sorted(maps.Keys(st.s.Tracks)) {
		tr := slices.DeleteFunc(slices.Clone(st.s.Tracks[k]), func(e Event) bool { return e.Until().Before(edge) })
		if len(tr) == 0 {
			delete(st.s.Tracks, k)
		} else {
			st.s.Tracks[k] = tr
		}
	}
	for _, k := range slices.Sorted(maps.Keys(st.s.Runs)) {
		if r := st.s.Runs[k]; r.FinishedAt != nil && r.FinishedAt.Before(edge) {
			delete(st.s.Runs, k)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(st.s.Windows)) {
		if w := st.s.Windows[k]; w.EventID != "" && w.End.Before(edge) {
			delete(st.s.Windows, k)
		}
	}
}

// WindowEventID — event_id записи окна нарушения: та же формула, что у
// адресованных записей стадии (crossitem.AddressedID): UUIDv5(NS_ANT,
// «stage» ‖ эмитент ‖ тип ‖ поток ‖ ключ). Нужен несоответствию окна до
// записи окна в журнал.
func WindowEventID(stream, key string) string {
	return kernel.UUIDv5(constants.NsAnt, "stage\x1f"+string(Module)+"\x1f"+string(catalog.EquipmentViolationWindowResolved)+"\x1f"+stream+"\x1f"+key)
}

func causes(cs []Cause) []kernel.Record {
	out := make([]kernel.Record, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Record())
	}
	return out
}

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

// timeLayout — время в данных записей: RFC 3339 UTC, ровно три знака после
// секунд («Соглашения/Время»).
const timeLayout = "2006-01-02T15:04:05.000Z"

func ts(t time.Time) string { return t.UTC().Format(timeLayout) }

// windowData — данные equipment.violation.window_resolved v1.
type windowData struct {
	EquipmentID             string   `json:"equipment_id"`
	StepKey                 string   `json:"step_key,omitempty"`
	WindowStart             string   `json:"window_start"`
	WindowEnd               string   `json:"window_end"`
	DeviationEventIDs       []string `json:"deviation_event_ids"`
	AffectedOperationRunIDs []string `json:"affected_operation_run_ids"`
	AffectedItemIDs         []string `json:"affected_item_ids,omitempty"`
}

// boundData — данные equipment.event.bound v1.
type boundData struct {
	OperationRunID    string          `json:"operation_run_id"`
	EquipmentID       string          `json:"equipment_id"`
	Binding           string          `json:"binding"`
	SubjectEventID    string          `json:"subject_event_id"`
	SubjectEventType  string          `json:"subject_event_type"`
	SubjectOccurredAt string          `json:"subject_occurred_at"`
	SubjectSourceKind string          `json:"subject_source_kind,omitempty"`
	SubjectData       json.RawMessage `json:"subject_data"`
}
