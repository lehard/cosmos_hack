package analysis

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Инциденты и версии области риска в межизделийной стадии (AD-42, FR-61, FR-62):
// функция Stage подключена к domain/crossitem.Fold точкой подключения
// (crossitem.Modules → analysis.Stage), код стадии не правится.
//
// Правила:
//   - инцидент открывает подтверждённое несоответствие: общий фактор —
//     оборудование операции (или партия компонента для входного дефекта);
//   - первая версия области — все изделия, прошедшие ту же операцию через
//     любой общий фактор (оборудование, исполнитель, программа, инструмент) в
//     окне от последней подтверждённо годной детали через то же оборудование
//     до обнаружения (консервативно);
//   - правило только расширяет область (новое выполнение на том же
//     оборудовании, сборка с компонентом из области, выдача той же партии) и
//     никогда не исключает изделие (AD-27: исключение — разрешающее действие);
//   - сужение — только решением человека с основанием (incident.scope.narrowed,
//     incident.item.assessed «исключено»), каждая правка — новая версия;
//   - статус изделия в инциденте — две оси (что известно / что делать),
//     адресованная запись incident.membership.changed изделию (AD-30, AD-42).

// Факторы области (перечисление контракта common_factor в ответах API).
const (
	FactorMachine       = "machine"
	FactorTool          = "tool"
	FactorFixture       = "fixture"
	FactorProgram       = "program"
	FactorPerformer     = "performer"
	FactorMaterialBatch = "material_batch"
)

// Статусы изделия в инциденте (ось incident словаря статусов) и действия (FR-62).
const (
	StatusConfirmed = "confirmed"
	StatusSuspect   = "suspect"
	StatusExcluded  = "excluded"
	StatusUnknown   = "unknown"
	ActionObserve   = "observe"
	ActionCheck     = "check"
	ActionBlock     = "block"
	ActionRelease   = "release"
)

// Изменения версии области (поле change incident.scope.computed).
const (
	ChangeComputed = "computed"
	ChangeExpanded = "expanded"
	ChangeNarrowed = "narrowed"
)

// Местонахождение изделия для разбивки области (FR-61).
const (
	LocInProduction = "in_production"
	LocMovedOn      = "moved_on"
	LocAssembled    = "assembled"
	LocShipped      = "shipped"
)

// FactorRef — общий фактор и его значение.
type FactorRef struct {
	Factor string `json:"factor"`
	Value  string `json:"value"`
}

// StageRun — выполнение операции изделия в индексе стадии.
type StageRun struct {
	RunID     string     `json:"run_id"`
	StepKey   string     `json:"step_key"`
	Equipment string     `json:"equipment,omitempty"`
	Operator  string     `json:"operator,omitempty"`
	Program   string     `json:"program,omitempty"`
	Tool      string     `json:"tool,omitempty"`
	Started   time.Time  `json:"started"`
	Finished  *time.Time `json:"finished,omitempty"`
	EventID   string     `json:"event_id"`
	FinishID  string     `json:"finish_id,omitempty"`
}

// StageDefect — находка дефекта изделия (для входного дефекта и партии).
type StageDefect struct {
	EventID    string    `json:"event_id"`
	At         time.Time `json:"at"`
	Phase      string    `json:"phase,omitempty"`
	Components []string  `json:"components,omitempty"`
	LotID      string    `json:"lot_id,omitempty"`
}

// StageItem — изделие в индексе стадии: выполнения, находки, генеалогия,
// выпуск. Индекс — заготовка за портами Genealogy и EquipmentTimeline, пока
// генеалогия (эпик 18) и временная линия оборудования (эпик 23) не отдают
// своё состояние функциям модулей стадии.
type StageItem struct {
	Runs          []StageRun    `json:"runs,omitempty"`
	Defects       []StageDefect `json:"defects,omitempty"`
	Parents       []string      `json:"parents,omitempty"`
	Components    []string      `json:"components,omitempty"`
	ComponentLots []string      `json:"component_lots,omitempty"`
	Released      bool          `json:"released,omitempty"`
}

// ToolMark — установка инструмента на оборудование.
type ToolMark struct {
	At   time.Time `json:"at"`
	Tool string    `json:"tool"`
}

// Member — изделие в области: что известно, что делать, через какой компонент попало.
type Member struct {
	Status  string `json:"status"`
	Action  string `json:"action"`
	Via     string `json:"via,omitempty"`
	Version int    `json:"version"`
}

// Incident — инцидент и текущая версия области риска в стадии.
type Incident struct {
	ID             string            `json:"id"`
	Label          string            `json:"label"`
	Factor         string            `json:"factor"`
	Value          string            `json:"value"`
	StepKey        string            `json:"step_key,omitempty"`
	Factors        []FactorRef       `json:"factors,omitempty"`
	WindowStart    time.Time         `json:"window_start"`
	WindowEnd      time.Time         `json:"window_end"`
	KnownGoodEvent string            `json:"known_good_event,omitempty"`
	KnownGoodItem  string            `json:"known_good_item,omitempty"`
	Version        int               `json:"version"`
	Emitted        int               `json:"emitted"`
	Members        map[string]Member `json:"members,omitempty"`
	NCs            []string          `json:"ncs,omitempty"`
	Closed         bool              `json:"closed,omitempty"`
	CauseConcluded bool              `json:"cause_concluded,omitempty"`
}

// StageState — состояние модуля analysis в межизделийной стадии (AD-42):
// индекс изделий (заготовка портов генеалогии и оборудования) и инциденты.
type StageState struct {
	Items     map[string]StageItem  `json:"items,omitempty"`
	Tools     map[string][]ToolMark `json:"tools,omitempty"`
	Lots      map[string]string     `json:"lots,omitempty"`
	Incidents map[string]Incident   `json:"incidents,omitempty"`
}

// Genealogy — порт генеалогии и местонахождения изделий для области риска
// (AD-42: генеалогией владеет crossitem, эпик 18). До подключения генеалогии
// стадии к функциям модулей — индекс StageState (заготовка).
type Genealogy interface {
	// Parents — сборки, в которые вошло изделие.
	Parents(itemID string) []string
	// LotOf — партия компонента-экземпляра; пусто — неизвестна.
	LotOf(componentID string) string
	// Location — местонахождение изделия относительно операции stepKey.
	Location(itemID, stepKey string) string
}

// EquipmentTimeline — порт временной линии оборудования (AD-42, эпик 23):
// инструмент на оборудовании в момент.
type EquipmentTimeline interface {
	ToolAt(equipmentID string, at time.Time) string
}

var (
	_ Genealogy         = StageState{}
	_ EquipmentTimeline = StageState{}
)

// Parents — сборки изделия по индексу стадии.
func (s StageState) Parents(itemID string) []string { return s.Items[itemID].Parents }

// LotOf — партия компонента по индексу стадии.
func (s StageState) LotOf(componentID string) string { return s.Lots[componentID] }

// ToolAt — инструмент на оборудовании в момент at по индексу стадии.
func (s StageState) ToolAt(equipmentID string, at time.Time) string {
	tool := ""
	for _, t := range s.Tools[equipmentID] {
		if !t.At.After(at) {
			tool = t.Tool
		}
	}
	return tool
}

// Location — местонахождение изделия относительно операции stepKey (FR-61):
// отгружено, собрано, ушло дальше по маршруту или в производстве.
func (s StageState) Location(itemID, stepKey string) string {
	it := s.Items[itemID]
	return locationOf(it.Released, len(it.Parents) > 0 || len(it.Components) > 0, stageRuns(it.Runs), stepKey)
}

// runSpan — выполнение для определения «ушло дальше».
type runSpan struct {
	step    string
	started time.Time
}

func stageRuns(rs []StageRun) []runSpan {
	out := make([]runSpan, 0, len(rs))
	for _, r := range rs {
		out = append(out, runSpan{r.StepKey, r.Started})
	}
	return out
}

// locationOf — общая для стадии и запросов классификация местонахождения.
func locationOf(released, assembled bool, runs []runSpan, stepKey string) string {
	switch {
	case released:
		return LocShipped
	case assembled:
		return LocAssembled
	}
	var last time.Time
	found := false
	for _, r := range runs {
		if r.step == stepKey && (!found || r.started.After(last)) {
			last, found = r.started, true
		}
	}
	for _, r := range runs {
		if found && r.step != stepKey && r.started.After(last) {
			return LocMovedOn
		}
	}
	return LocInProduction
}

// Location — местонахождение изделия по состоянию свёртки изделия (для ответов API).
func Location(s State, stepKey string) string {
	runs := make([]runSpan, 0, len(s.Runs))
	for _, r := range s.Runs {
		runs = append(runs, runSpan{r.StepKey, r.Started})
	}
	return locationOf(s.Released, s.Assembled, runs, stepKey)
}

func (s StageState) init() StageState {
	if s.Items == nil {
		s.Items = map[string]StageItem{}
	}
	if s.Tools == nil {
		s.Tools = map[string][]ToolMark{}
	}
	if s.Lots == nil {
		s.Lots = map[string]string{}
	}
	if s.Incidents == nil {
		s.Incidents = map[string]Incident{}
	}
	return s
}

// Stage — функция модуля analysis в межизделийной стадии (AD-42): вход —
// запись стадии (факт без изделия, решение над инцидентом, запись изделия с
// пометкой publish_stage); выход — адресованные записи incident.* в потоки
// инцидентов и изделий, occurred_at — наибольший среди причин (запись-триггер).
// Без нового факта стадия не реагирует на свои адресованные записи
// (crossitem.Settle).
func Stage(s StageState, r kernel.Record) (StageState, []kernel.Addressed) {
	s = s.init()
	switch r.Type {
	case catalog.OperationRunStarted:
		return s.onRunStarted(r)
	case catalog.OperationRunFinished:
		d, ok := decodeAs[runFinishedData](r)
		if ok && r.ItemID != "" {
			at := r.OccurredAt
			if d.FinishedAt != nil {
				at = d.FinishedAt.UTC()
			}
			s.updateRun(r.ItemID, d.OperationRunID, func(x *StageRun) { x.Finished, x.FinishID = &at, r.EventID })
		}
	case catalog.OperationRunIntervalResolved:
		d, ok := decodeAs[intervalData](r)
		if ok && r.ItemID != "" {
			s.updateRun(r.ItemID, d.OperationRunID, func(x *StageRun) {
				if x.Equipment == "" {
					x.Equipment = deref(d.EquipmentID)
				}
				if !d.IntervalStart.IsZero() {
					x.Started = d.IntervalStart.UTC()
				}
				if d.IntervalEnd != nil {
					end := d.IntervalEnd.UTC()
					x.Finished = &end
				}
			})
		}
	case catalog.InspectionResultRecorded:
		d, ok := decodeAs[inspectionData](r)
		if ok && r.ItemID != "" && d.Outcome == "defect_indicated" {
			df := StageDefect{EventID: r.EventID, At: r.OccurredAt, Phase: d.Phase, LotID: deref(d.LotID)}
			for _, x := range d.Defects {
				df.Components = appendUnique(df.Components, deref(x.ComponentRef))
			}
			it := s.Items[r.ItemID]
			it.Defects = append(slices.Clone(it.Defects), df)
			s.Items[r.ItemID] = it
		}
	case catalog.DecisionNonconformityConfirmed:
		return s.onNonconformity(r)
	case catalog.ItemAssemblyRecorded:
		return s.onAssembly(r)
	case catalog.GenealogyLotIssued:
		return s.onLotIssued(r)
	case catalog.ItemReleaseRecorded:
		if r.ItemID != "" {
			it := s.Items[r.ItemID]
			it.Released = true
			s.Items[r.ItemID] = it
		}
	case catalog.EquipmentToolChanged:
		if e, ok := EquipmentEventOf(r); ok {
			s.Tools[e.EquipmentID] = append(slices.Clone(s.Tools[e.EquipmentID]), ToolMark{At: e.OccurredAt, Tool: e.ToolID})
		}
	case catalog.IncidentScopeNarrowed, catalog.IncidentScopeExpanded, catalog.IncidentItemAssessed,
		catalog.IncidentIncidentClosed, catalog.IncidentCauseConcluded:
		return s.onDecision(r)
	}
	return s, nil
}

func (s StageState) updateRun(itemID, runID string, f func(*StageRun)) {
	it, ok := s.Items[itemID]
	if !ok {
		return
	}
	it.Runs = slices.Clone(it.Runs)
	for i := range it.Runs {
		if it.Runs[i].RunID == runID {
			f(&it.Runs[i])
		}
	}
	s.Items[itemID] = it
}

// onRunStarted — выполнение в индекс; правило расширения: новое выполнение
// той же операции на оборудовании открытого инцидента после начала окна —
// изделие под подозрением (защитное, AD-27).
func (s StageState) onRunStarted(r kernel.Record) (StageState, []kernel.Addressed) {
	d, ok := decodeAs[runStartedData](r)
	if !ok || r.ItemID == "" || d.OperationRunID == "" {
		return s, nil
	}
	at := r.OccurredAt
	if d.StartedAt != nil {
		at = d.StartedAt.UTC()
	}
	run := StageRun{RunID: d.OperationRunID, StepKey: d.StepKey, Equipment: deref(d.EquipmentID), Operator: deref(d.OperatorID),
		Program: deref(d.ProgramRef), Started: at, EventID: r.EventID}
	run.Tool = s.ToolAt(run.Equipment, at)
	it := s.Items[r.ItemID]
	it.Runs = append(slices.Clone(it.Runs), run)
	s.Items[r.ItemID] = it
	if run.Equipment == "" {
		return s, nil
	}
	var out []kernel.Addressed
	for _, id := range s.openIncidents() {
		inc := s.Incidents[id]
		if inc.Factor != FactorMachine || inc.Value != run.Equipment || inc.StepKey != run.StepKey || !run.Started.After(inc.WindowStart) {
			continue
		}
		if m, in := inc.Members[r.ItemID]; in && m.Status != StatusExcluded {
			continue
		}
		changed := inc.AutoAdd(r.ItemID, StatusSuspect, "")
		if len(changed) == 0 {
			continue
		}
		if run.Started.After(inc.WindowEnd) {
			inc.WindowEnd = run.Started
		}
		out = append(out, s.version(&inc, ChangeComputed, "", changed, nil, []string{r.EventID}, r)...)
		s.Incidents[id] = inc
	}
	return s, out
}

// onNonconformity — подтверждённое несоответствие: изделие «подтверждено» в
// инциденте своего фактора (присоединение или расширение) либо новый инцидент.
func (s StageState) onNonconformity(r kernel.Record) (StageState, []kernel.Addressed) {
	d, ok := decodeAs[ncConfirmedData](r)
	if !ok || r.ItemID == "" || d.NcID == "" {
		return s, nil
	}
	it := s.Items[r.ItemID]
	if lot := s.incomingLot(it, r.OccurredAt); lot != "" {
		return s.join(r, d.NcID, FactorMaterialBatch, lot, "", func() Incident { return s.lotIncident(r, d.NcID, lot) })
	}
	run := lastRunWithEquipment(it.Runs, r.OccurredAt)
	if run == nil {
		// Фактор операции не установлен — инцидент по данным не открыть;
		// разбор несоответствия (FR-58) покажет нехватку сведений.
		return s, nil
	}
	return s.join(r, d.NcID, FactorMachine, run.Equipment, run.StepKey, func() Incident { return s.machineIncident(r, d.NcID, *run) })
}

// join — изделие несоответствия в открытом инциденте того же фактора
// (подтверждено, при необходимости — расширение новой версией) или новый инцидент.
func (s StageState) join(r kernel.Record, ncID, factor, value, step string, open func() Incident) (StageState, []kernel.Addressed) {
	for _, id := range s.openIncidents() {
		inc := s.Incidents[id]
		if inc.Factor != factor || inc.Value != value || inc.StepKey != step {
			continue
		}
		inc.NCs = appendUnique(slices.Clone(inc.NCs), ncID)
		m, in := inc.Members[r.ItemID]
		var out []kernel.Addressed
		switch {
		case in && m.Status == StatusConfirmed:
		case in && m.Status != StatusExcluded:
			// Статус внутри той же версии: «под подозрением» → «подтверждено».
			inc.Members = maps.Clone(inc.Members)
			inc.Members[r.ItemID] = Member{Status: StatusConfirmed, Action: ActionBlock, Via: m.Via, Version: inc.Version}
			out = append(out, s.membership(&inc, r.ItemID, r))
		default:
			changed := inc.AutoAdd(r.ItemID, StatusConfirmed, "")
			out = append(out, s.version(&inc, ChangeComputed, "", changed, nil, []string{r.EventID}, r)...)
		}
		s.Incidents[id] = inc
		return s, out
	}
	inc := open()
	if len(inc.Members) == 0 {
		return s, nil
	}
	opened, err := kernel.NewAddressed(Module, catalog.IncidentIncidentOpened, "incident:"+inc.ID, inc.ID, incidentOpened(inc, r.EventID), r)
	if err != nil {
		return s, nil
	}
	changed := slices.Sorted(maps.Keys(inc.Members))
	out := []kernel.Addressed{opened}
	out = append(out, s.version(&inc, ChangeComputed, "", changed, nil, []string{r.EventID}, r)...)
	s.Incidents[inc.ID] = inc
	return s, out
}

// IncidentID — детерминированный id инцидента от записи, открывшей его:
// `RS-‹8 hex›` (UUIDv5, AD-4).
func IncidentID(triggerEventID string) string {
	u := kernel.UUIDv5(constants.NsAnt, "incident\x1f"+triggerEventID)
	return "RS-" + strings.ToUpper(u[:8])
}

// machineIncident — первая версия области по оборудованию (FR-61): все
// выполнения той же операции через любой общий фактор в окне от последней
// подтверждённо годной детали через то же оборудование до обнаружения.
func (s StageState) machineIncident(r kernel.Record, ncID string, run StageRun) Incident {
	inc := Incident{ID: IncidentID(r.EventID), Label: "Инцидент " + run.Equipment, Factor: FactorMachine, Value: run.Equipment,
		StepKey: run.StepKey, WindowEnd: r.OccurredAt, NCs: []string{ncID}, Members: map[string]Member{}}
	for _, f := range []FactorRef{{FactorMachine, run.Equipment}, {FactorPerformer, run.Operator}, {FactorProgram, run.Program}, {FactorTool, run.Tool}} {
		if f.Value != "" {
			inc.Factors = append(inc.Factors, f)
		}
	}
	// Последняя подтверждённо годная деталь через то же оборудование — изделие
	// выпущено (прошло весь маршрут) и его выполнение закончилось до начала
	// выполнения с дефектом.
	var good *time.Time
	for _, id := range slices.Sorted(maps.Keys(s.Items)) {
		it := s.Items[id]
		if !it.Released {
			continue
		}
		for _, k := range it.Runs {
			if k.StepKey != run.StepKey || k.Equipment != run.Equipment || k.Finished == nil || !k.Finished.Before(run.Started) {
				continue
			}
			if good == nil || k.Finished.After(*good) {
				f := *k.Finished
				good, inc.KnownGoodEvent, inc.KnownGoodItem = &f, k.FinishID, id
				if inc.KnownGoodEvent == "" {
					inc.KnownGoodEvent = k.EventID
				}
			}
		}
	}
	if good != nil {
		inc.WindowStart = *good
	}
	earliest := r.OccurredAt
	for _, id := range slices.Sorted(maps.Keys(s.Items)) {
		for _, k := range s.Items[id].Runs {
			if k.StepKey != run.StepKey || k.Started.After(inc.WindowEnd) || !sharesFactor(k, inc.Factors) {
				continue
			}
			if k.Finished != nil && !k.Finished.After(inc.WindowStart) {
				continue
			}
			if k.Started.Before(earliest) {
				earliest = k.Started
			}
			inc.Members[id] = Member{Status: StatusSuspect, Action: ActionCheck}
		}
	}
	if good == nil {
		// Подтверждённо годной детали нет — окно от первого известного выполнения.
		inc.WindowStart = earliest
	}
	inc.Members[r.ItemID] = Member{Status: StatusConfirmed, Action: ActionBlock}
	return inc
}

// lotIncident — область по партии компонента для входного дефекта (FR-61):
// все изделия, в которые вошли компоненты той же партии.
func (s StageState) lotIncident(r kernel.Record, ncID, lot string) Incident {
	inc := Incident{ID: IncidentID(r.EventID), Label: "Входной брак партии " + lot, Factor: FactorMaterialBatch, Value: lot,
		Factors: []FactorRef{{FactorMaterialBatch, lot}}, WindowStart: r.OccurredAt, WindowEnd: r.OccurredAt,
		NCs: []string{ncID}, Members: map[string]Member{}}
	for _, id := range slices.Sorted(maps.Keys(s.Items)) {
		if s.usesLot(id, lot) {
			inc.Members[id] = Member{Status: StatusSuspect, Action: ActionCheck}
			for _, k := range s.Items[id].Runs {
				if k.Started.Before(inc.WindowStart) {
					inc.WindowStart = k.Started
				}
			}
		}
	}
	inc.Members[r.ItemID] = Member{Status: StatusConfirmed, Action: ActionBlock}
	return inc
}

// usesLot — в изделие вошли компоненты партии lot.
func (s StageState) usesLot(itemID, lot string) bool {
	it := s.Items[itemID]
	if slices.Contains(it.ComponentLots, lot) {
		return true
	}
	for _, c := range it.Components {
		if s.Lots[c] == lot {
			return true
		}
	}
	return false
}

// incomingLot — партия, к которой относится входной дефект изделия: находка
// входного контроля или дефект в теле компонента (FR-58, FR-59).
func (s StageState) incomingLot(it StageItem, at time.Time) string {
	for i := len(it.Defects) - 1; i >= 0; i-- {
		d := it.Defects[i]
		if d.At.After(at) {
			continue
		}
		if d.LotID != "" && (d.Phase == "incoming" || len(d.Components) > 0) {
			return d.LotID
		}
		for _, c := range d.Components {
			if l := s.Lots[c]; l != "" {
				return l
			}
		}
		if (d.Phase == "incoming" || len(d.Components) > 0) && len(it.ComponentLots) == 1 {
			return it.ComponentLots[0]
		}
		return ""
	}
	return ""
}

func lastRunWithEquipment(runs []StageRun, at time.Time) *StageRun {
	var best *StageRun
	for i := range runs {
		r := &runs[i]
		if r.Equipment == "" || r.Started.After(at) {
			continue
		}
		if best == nil || r.Started.After(best.Started) {
			best = r
		}
	}
	return best
}

// sharesFactor — выполнение проходило через хотя бы один общий фактор.
func sharesFactor(k StageRun, fs []FactorRef) bool {
	for _, f := range fs {
		switch {
		case f.Factor == FactorMachine && k.Equipment == f.Value,
			f.Factor == FactorPerformer && k.Operator == f.Value,
			f.Factor == FactorProgram && k.Program == f.Value,
			f.Factor == FactorTool && k.Tool == f.Value:
			return true
		}
	}
	return false
}

// onAssembly — генеалогия в индекс; распространение области вверх по дереву
// сборки (AD-42): сборка с компонентом из области — под подозрением.
func (s StageState) onAssembly(r kernel.Record) (StageState, []kernel.Addressed) {
	d, ok := decodeAs[assemblyData](r)
	if !ok || d.AssemblyItemID == "" {
		return s, nil
	}
	asm := s.Items[d.AssemblyItemID]
	comp := deref(d.ComponentItemID)
	lot := deref(d.ComponentLotID)
	asm.Components = appendUnique(slices.Clone(asm.Components), comp)
	asm.ComponentLots = appendUnique(slices.Clone(asm.ComponentLots), lot)
	s.Items[d.AssemblyItemID] = asm
	if comp != "" {
		c := s.Items[comp]
		c.Parents = appendUnique(slices.Clone(c.Parents), d.AssemblyItemID)
		s.Items[comp] = c
		if lot != "" {
			s.Lots[comp] = lot
		} else if l := s.Lots[comp]; l != "" {
			lot = l
		}
	}
	var out []kernel.Addressed
	for _, id := range s.openIncidents() {
		inc := s.Incidents[id]
		if m, in := inc.Members[d.AssemblyItemID]; in && m.Status != StatusExcluded {
			continue
		}
		via := ""
		if m, in := inc.Members[comp]; comp != "" && in && m.Status != StatusExcluded {
			via = comp
		} else if !(inc.Factor == FactorMaterialBatch && lot != "" && inc.Value == lot) {
			continue
		}
		changed := inc.AutoAdd(d.AssemblyItemID, StatusSuspect, via)
		out = append(out, s.version(&inc, ChangeComputed, "", changed, nil, []string{r.EventID}, r)...)
		s.Incidents[id] = inc
	}
	return s, out
}

// onLotIssued — партия выдана изделиям: индекс и расширение области партии.
func (s StageState) onLotIssued(r kernel.Record) (StageState, []kernel.Addressed) {
	d, ok := decodeAs[lotIssuedData](r)
	if !ok || d.LotID == "" {
		return s, nil
	}
	items := slices.Clone(d.ItemIDs)
	slices.Sort(items)
	for _, id := range items {
		it := s.Items[id]
		it.ComponentLots = appendUnique(slices.Clone(it.ComponentLots), d.LotID)
		s.Items[id] = it
	}
	var out []kernel.Addressed
	for _, id := range s.openIncidents() {
		inc := s.Incidents[id]
		if inc.Factor != FactorMaterialBatch || inc.Value != d.LotID {
			continue
		}
		var changed []string
		for _, it := range items {
			changed = append(changed, inc.AutoAdd(it, StatusSuspect, "")...)
		}
		out = append(out, s.version(&inc, ChangeComputed, "", changed, nil, []string{r.EventID}, r)...)
		s.Incidents[id] = inc
	}
	return s, out
}

// AutoAdd — изменение статуса правилом системы: только защитное (AD-27) —
// добавить в область, поднять «под подозрением» до «подтверждено», вернуть
// исключённое при новом воздействии. Исключить правило не может никогда:
// статус «исключено» здесь недопустим, понижение «подтверждено» → «под
// подозрением» не делается. Возвращает изменившиеся изделия.
func (inc *Incident) AutoAdd(itemID, status, via string) []string {
	if status == StatusExcluded || itemID == "" {
		return nil // FR-61, AD-27: incident.auto_exclude_forbidden
	}
	m, in := inc.Members[itemID]
	if in && (m.Status == status || m.Status == StatusConfirmed) {
		return nil
	}
	action := ActionCheck
	if status == StatusConfirmed {
		action = ActionBlock
	}
	inc.Members = maps.Clone(inc.Members)
	if inc.Members == nil {
		inc.Members = map[string]Member{}
	}
	inc.Members[itemID] = Member{Status: status, Action: action, Via: via}
	return []string{itemID}
}

// onDecision — решения человека над инцидентом (AD-27, FR-61, FR-62).
func (s StageState) onDecision(r kernel.Record) (StageState, []kernel.Addressed) {
	incID := incidentOf(r)
	inc, ok := s.Incidents[incID]
	if !ok {
		return s, nil
	}
	var out []kernel.Addressed
	switch r.Type {
	case catalog.IncidentScopeNarrowed:
		d, err := kernel.Decode[ev.IncidentScopeNarrowedV1](r)
		if err != nil || inc.Closed || len(d.EvidenceEventIds) == 0 || strings.TrimSpace(string(d.Reason.Text)) == "" {
			// Сужение без основания не применяется (FR-61: «ни одно изделие не
			// выходит из области без записи основания»); гард api его отклоняет.
			return s, nil
		}
		action := ActionObserve
		if d.ReleaseContainment != nil && *d.ReleaseContainment {
			action = ActionRelease
		}
		var removed []string
		for _, it := range sortedItems(d.ItemIds) {
			if m, in := inc.Members[it]; in && m.Status != StatusExcluded {
				inc.Members = maps.Clone(inc.Members)
				inc.Members[it] = Member{Status: StatusExcluded, Action: action, Via: m.Via}
				removed = append(removed, it)
			}
		}
		out = s.version(&inc, ChangeNarrowed, r.EventID, removed, removed, evidenceIDs(r.EventID, d.EvidenceEventIds), r)
	case catalog.IncidentScopeExpanded:
		d, err := kernel.Decode[ev.IncidentScopeExpandedV1](r)
		if err != nil || inc.Closed {
			return s, nil
		}
		var added []string
		for _, it := range sortedItems(d.ItemIds) {
			if m, in := inc.Members[it]; !in || m.Status == StatusExcluded {
				inc.Members = maps.Clone(inc.Members)
				if inc.Members == nil {
					inc.Members = map[string]Member{}
				}
				inc.Members[it] = Member{Status: StatusSuspect, Action: ActionCheck}
				added = append(added, it)
			}
		}
		out = s.version(&inc, ChangeExpanded, r.EventID, added, nil, []string{r.EventID}, r)
	case catalog.IncidentItemAssessed:
		d, err := kernel.Decode[ev.IncidentItemAssessedV1](r)
		it := string(d.ItemID)
		m, in := inc.Members[it]
		if err != nil || inc.Closed || !in || len(d.EvidenceEventIds) == 0 {
			return s, nil
		}
		inc.Members = maps.Clone(inc.Members)
		if d.Assessment == ev.IncidentItemAssessedV1AssessmentExcluded {
			if m.Status == StatusExcluded {
				return s, nil
			}
			// Исключение по проверке — сужение области с основанием: новая версия.
			inc.Members[it] = Member{Status: StatusExcluded, Action: ActionObserve, Via: m.Via}
			out = s.version(&inc, ChangeNarrowed, r.EventID, []string{it}, []string{it}, evidenceIDs(r.EventID, d.EvidenceEventIds), r)
		} else if m.Status != StatusConfirmed {
			inc.Members[it] = Member{Status: StatusConfirmed, Action: ActionBlock, Via: m.Via, Version: inc.Version}
			out = append(out, s.membership(&inc, it, r))
		}
	case catalog.IncidentIncidentClosed:
		inc.Closed = true
	case catalog.IncidentCauseConcluded:
		inc.CauseConcluded = true
	}
	s.Incidents[incID] = inc
	return s, out
}

// version — новая версия области: incident.scope.computed в поток инцидента
// и incident.membership.changed изменившимся изделиям (AD-42). Нет изменений —
// версии нет.
func (s StageState) version(inc *Incident, change, decisionID string, changed, removed, basis []string, r kernel.Record) []kernel.Addressed {
	if len(changed) == 0 {
		return nil
	}
	inc.Version++
	var added []string
	for _, id := range changed {
		m := inc.Members[id]
		m.Version = inc.Version
		inc.Members[id] = m
		if m.Status != StatusExcluded {
			added = append(added, id)
		}
	}
	d := scopeComputedData{IncidentID: inc.ID, ScopeVersion: inc.Version, WindowStart: ts(inc.WindowStart), WindowEnd: ts(inc.WindowEnd),
		LastKnownGoodEventID: inc.KnownGoodEvent, LastKnownGoodItemID: inc.KnownGoodItem, AddedItemIDs: added, RemovedItemIDs: removed,
		Size: inc.Size(), Breakdown: s.breakdown(*inc), Basis: sortedUnique(uuidsOnly(append(slices.Clone(basis), r.EventID))),
		Change: change, DecisionEventID: decisionID}
	if !isUUID(d.LastKnownGoodEventID) {
		d.LastKnownGoodEventID = ""
	}
	sc, err := kernel.NewAddressed(Module, catalog.IncidentScopeComputed, "incident:"+inc.ID, "v"+strconv.Itoa(inc.Version), d, r)
	if err != nil {
		return nil
	}
	out := []kernel.Addressed{sc}
	for _, id := range sortedUnique(changed) {
		out = append(out, s.membership(inc, id, r))
	}
	return out
}

// membership — адресованная запись изделию: статус в инциденте по двум осям.
func (s StageState) membership(inc *Incident, itemID string, r kernel.Record) kernel.Addressed {
	inc.Emitted++
	m := inc.Members[itemID]
	d := ev.IncidentMembershipChangedV1{IncidentID: ev.ObjectID(inc.ID), ScopeVersion: max(inc.Version, 1),
		Status: ev.AxisIncident(m.Status), Action: ev.DictIncidentAction(m.Action)}
	if m.Via != "" {
		via := ev.ItemID(m.Via)
		d.ViaAssemblyOf = &via
	}
	a, _ := kernel.NewAddressed(Module, catalog.IncidentMembershipChanged, "item:"+itemID, inc.ID+"/m"+strconv.Itoa(inc.Emitted), d, r)
	return a
}

// Size — размер области: изделия, не исключённые с основанием.
func (inc Incident) Size() int {
	n := 0
	for _, id := range slices.Sorted(maps.Keys(inc.Members)) {
		if inc.Members[id].Status != StatusExcluded {
			n++
		}
	}
	return n
}

// Breakdown — разбивка области по местонахождению (FR-61).
type Breakdown struct {
	InProduction int `json:"in_production"`
	MovedOn      int `json:"moved_on"`
	Assembled    int `json:"assembled"`
	Shipped      int `json:"shipped"`
}

// Add — изделие в разбивку.
func (b *Breakdown) Add(loc string) {
	switch loc {
	case LocShipped:
		b.Shipped++
	case LocAssembled:
		b.Assembled++
	case LocMovedOn:
		b.MovedOn++
	default:
		b.InProduction++
	}
}

func (s StageState) breakdown(inc Incident) Breakdown {
	var b Breakdown
	for _, id := range slices.Sorted(maps.Keys(inc.Members)) {
		if inc.Members[id].Status != StatusExcluded {
			b.Add(s.Location(id, inc.StepKey))
		}
	}
	return b
}

func (s StageState) openIncidents() []string {
	var out []string
	for _, id := range slices.Sorted(maps.Keys(s.Incidents)) {
		if !s.Incidents[id].Closed {
			out = append(out, id)
		}
	}
	return out
}

// incidentOf — incident_id из data решения над инцидентом.
func incidentOf(r kernel.Record) string {
	d, _ := decodeAs[struct {
		IncidentID string `json:"incident_id"`
	}](r)
	return d.IncidentID
}

func sortedItems(xs []ev.ItemID) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		out = append(out, string(x))
	}
	return sortedUnique(out)
}

func evidenceIDs(decision string, xs []ev.UUID) []string {
	out := []string{decision}
	for _, x := range xs {
		out = append(out, string(x))
	}
	return out
}

// scopeComputedData — data incident.scope.computed v1 (FR-61).
type scopeComputedData struct {
	IncidentID           string    `json:"incident_id"`
	ScopeVersion         int       `json:"scope_version"`
	WindowStart          string    `json:"window_start"`
	WindowEnd            string    `json:"window_end"`
	LastKnownGoodEventID string    `json:"last_known_good_event_id,omitempty"`
	LastKnownGoodItemID  string    `json:"last_known_good_item_id,omitempty"`
	AddedItemIDs         []string  `json:"added_item_ids,omitempty"`
	RemovedItemIDs       []string  `json:"removed_item_ids,omitempty"`
	Size                 int       `json:"size"`
	Breakdown            Breakdown `json:"breakdown"`
	Basis                []string  `json:"basis"`
	Change               string    `json:"change,omitempty"`
	DecisionEventID      string    `json:"decision_event_id,omitempty"`
}

// commonFactorOf — значение common_factor incident.incident.opened.
func commonFactorOf(f string) ev.IncidentIncidentOpenedV1CommonFactor {
	switch f {
	case FactorMachine:
		return ev.IncidentIncidentOpenedV1CommonFactorEquipment
	case FactorTool:
		return ev.IncidentIncidentOpenedV1CommonFactorTool
	case FactorFixture:
		return ev.IncidentIncidentOpenedV1CommonFactorFixture
	case FactorProgram:
		return ev.IncidentIncidentOpenedV1CommonFactorProgram
	case FactorPerformer:
		return ev.IncidentIncidentOpenedV1CommonFactorOperator
	case FactorMaterialBatch:
		return ev.IncidentIncidentOpenedV1CommonFactorLot
	}
	return ev.IncidentIncidentOpenedV1CommonFactorOther
}

// FactorOfCommon — фактор ответа API по common_factor записи.
func FactorOfCommon(c string) string {
	switch c {
	case "equipment":
		return FactorMachine
	case "tool":
		return FactorTool
	case "fixture":
		return FactorFixture
	case "program":
		return FactorProgram
	case "operator":
		return FactorPerformer
	case "lot", "heat", "charge", "supplier":
		return FactorMaterialBatch
	}
	return ""
}

// incidentOpenedData — data incident.incident.opened v1 (step_key — операция фактора).
type incidentOpenedData struct {
	IncidentID      string   `json:"incident_id"`
	CommonFactor    string   `json:"common_factor"`
	FactorRef       string   `json:"factor_ref,omitempty"`
	StepKey         string   `json:"step_key,omitempty"`
	TriggerEventIDs []string `json:"trigger_event_ids"`
}

func incidentOpened(inc Incident, trigger string) incidentOpenedData {
	return incidentOpenedData{IncidentID: inc.ID, CommonFactor: string(commonFactorOf(inc.Factor)), FactorRef: inc.Value,
		StepKey: inc.StepKey, TriggerEventIDs: uuidsOnly([]string{trigger})}
}
