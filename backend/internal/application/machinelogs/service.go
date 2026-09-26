package machinelogs

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	ml "ant/internal/domain/machinelogs"
)

// ProjectionReader — ведомый порт чтения проекций (engineapp.ProjectionStore):
// значение проекции по ключу; запись — только эффектами движка (AD-45).
type ProjectionReader interface {
	Get(ctx context.Context, name, key string) (json.RawMessage, bool, error)
}

// ItemStates — ведомый порт состояния изделия на момент (AD-22): та же
// свёртка, что у воркера, над префиксом входа (engineapp.StateQueries).
type ItemStates interface {
	Item(ctx context.Context, itemID string, m platform.Moment) (engineapp.ItemAt, error)
}

// Service — реализация live ведущих портов модуля machinelogs (AD-36):
// операции чтения над проекциями модуля; состояние на момент в прошлом —
// свёрткой изделия (профиль) или повтором журнала оборудования (состояние).
// Без проекций (Store == nil: выгрузка OpenAPI, тесты API) операции
// отвечают 501.
type Service struct {
	Unimplemented
	Store  ProjectionReader
	States ItemStates
	Env    ml.Env
}

// NewService создаёт реализацию live без хранилища (операции — 501).
func NewService() *Service { return &Service{} }

// NewLiveService — реализация live над проекциями и запросами на момент.
func NewLiveService(store ProjectionReader, states ItemStates, env ml.Env) *Service {
	return &Service{Store: store, States: states, Env: env}
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// maxTimelineDays — предел окна журнала оборудования за один запрос.
const maxTimelineDays = 31

// statusReplayDays — сколько суток журнала повторять для состояния
// оборудования на момент в прошлом.
const statusReplayDays = 7

func get[T any](ctx context.Context, s ProjectionReader, name, k string) (T, bool, error) {
	var v T
	raw, ok, err := s.Get(ctx, name, k)
	if err != nil || !ok {
		return v, ok, err
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, false, fmt.Errorf("проекция %s[%s]: %w", name, k, err)
	}
	return v, true, nil
}

func notFound(object, id string) error {
	e := platform.Fail(errcodes.ApiNotFound, "object", object, "id", id)
	e.Detail = object + " «" + id + "» не найден"
	return e
}

// visible — запись известна на момент m (ось «как было» — occurred_at, «что
// мы знали» — recorded_at, AD-22).
func visible(m platform.Moment, occurred, recorded time.Time) bool {
	if m.AsOf == nil {
		return true
	}
	if m.Axis == platform.AxisRecorded {
		return !recorded.After(*m.AsOf)
	}
	return !occurred.After(*m.AsOf)
}

// Equipment — оборудование прогона (machinelogs.equipment.list).
func (s *Service) Equipment(ctx context.Context, stationID string, m platform.Moment) (EquipmentList, error) {
	if s.Store == nil {
		return s.Unimplemented.Equipment(ctx, stationID, m)
	}
	idx, _, err := get[EquipmentIndex](ctx, s.Store, ProjectionEquipment, key(m.RunID, indexKey))
	if err != nil {
		return EquipmentList{}, err
	}
	out := EquipmentList{Items: []EquipmentState{}}
	for _, id := range idx.IDs {
		st, ok, err := s.status(ctx, id, m)
		if err != nil {
			return EquipmentList{}, err
		}
		if !ok || (stationID != "" && st.StationID != stationID) {
			continue
		}
		out.Items = append(out.Items, equipmentView(st))
	}
	return out, nil
}

// EquipmentByID — карточка оборудования (machinelogs.equipment.read).
func (s *Service) EquipmentByID(ctx context.Context, equipmentID string, m platform.Moment) (EquipmentState, error) {
	if s.Store == nil {
		return s.Unimplemented.EquipmentByID(ctx, equipmentID, m)
	}
	st, ok, err := s.status(ctx, equipmentID, m)
	if err != nil {
		return EquipmentState{}, err
	}
	if !ok {
		return EquipmentState{}, notFound("Оборудование", equipmentID)
	}
	return equipmentView(st), nil
}

// status — состояние оборудования: текущее из проекции или на момент —
// повтором журнала оборудования за statusReplayDays суток (AD-22).
func (s *Service) status(ctx context.Context, id string, m platform.Moment) (ml.Status, bool, error) {
	cur, ok, err := get[ml.Status](ctx, s.Store, ProjectionEquipment, key(m.RunID, id))
	if err != nil || !ok || m.AsOf == nil {
		return cur, ok, err
	}
	evs, err := s.events(ctx, id, m.AsOf.AddDate(0, 0, -statusReplayDays), *m.AsOf, m)
	if err != nil {
		return ml.Status{}, false, err
	}
	st := ml.Status{EquipmentID: id, RunID: m.RunID, Title: cur.Title, Kind: cur.Kind, StationID: cur.StationID,
		SpecialProcess: cur.SpecialProcess, Verification: cur.Verification, VerifiedTill: cur.VerifiedTill}
	for _, e := range evs {
		st = ml.ApplyStatus(st, s.Env, kernel.Record{EventID: e.EventID, Seq: e.Seq, Type: e.Type, RunID: e.RunID,
			SourceKind: e.SourceKind, OccurredAt: e.OccurredAt, RecordedAt: e.RecordedAt, Data: e.Data})
	}
	return st, len(evs) > 0 || ok, nil
}

// events — события оборудования, пересекающие [from, to] и известные на момент m.
func (s *Service) events(ctx context.Context, id string, from, to time.Time, m platform.Moment) ([]ml.Event, error) {
	var out []ml.Event
	for d := from.UTC().Truncate(24 * time.Hour); !d.After(to); d = d.AddDate(0, 0, 1) {
		tl, _, err := get[DayTimeline](ctx, s.Store, ProjectionTimeline, key(m.RunID, id, day(d)))
		if err != nil {
			return nil, err
		}
		for _, e := range tl.Events {
			if e.Start.After(to) || e.Until().Before(from) || !visible(m, e.OccurredAt, e.RecordedAt) {
				continue
			}
			out = append(out, e)
		}
	}
	return out, nil
}

// Timeline — журнал оборудования на окне (machinelogs.timeline.read).
func (s *Service) Timeline(ctx context.Context, equipmentID string, from, to time.Time, m platform.Moment) (EquipmentTimeline, error) {
	if s.Store == nil {
		return s.Unimplemented.Timeline(ctx, equipmentID, from, to, m)
	}
	if to.Before(from) {
		return EquipmentTimeline{}, platform.Fail(errcodes.ApiValidationFailed, "field", "to", "reason", "конец окна раньше начала")
	}
	if to.Sub(from) > maxTimelineDays*24*time.Hour {
		return EquipmentTimeline{}, platform.Fail(errcodes.ApiValidationFailed, "field", "to", "reason", fmt.Sprintf("окно больше %d суток", maxTimelineDays))
	}
	evs, err := s.events(ctx, equipmentID, from, to, m)
	if err != nil {
		return EquipmentTimeline{}, err
	}
	out := EquipmentTimeline{EquipmentID: equipmentID, From: from, To: to, Rows: []EquipmentEventRow{}}
	for _, e := range evs {
		c, l := ml.Classify(e)
		out.Rows = append(out.Rows, eventRow(ml.ProfileEvent{Event: e, Class: c, Layer: l}))
	}
	return out, nil
}

// RunProfile — профиль выполнения операции (machinelogs.run_profile.read, FR-148).
func (s *Service) RunProfile(ctx context.Context, runID string, m platform.Moment) (RunProfile, error) {
	if s.Store == nil {
		return s.Unimplemented.RunProfile(ctx, runID, m)
	}
	ref, ok, err := get[RunRef](ctx, s.Store, ProjectionRunIndex, key(m.RunID, runID))
	if err != nil {
		return RunProfile{}, err
	}
	if !ok || ref.ItemID == "" {
		return RunProfile{}, notFound("Выполнение операции", runID)
	}
	var p ml.Profile
	found := false
	if m.AsOf != nil && s.States != nil {
		at, err := s.States.Item(ctx, ref.ItemID, m)
		if err != nil {
			return RunProfile{}, err
		}
		p, found = at.Snapshot.Machinelogs.Profile(runID)
	} else {
		runs, _, err := get[ItemRuns](ctx, s.Store, ProjectionItemRuns, ref.ItemID)
		if err != nil {
			return RunProfile{}, err
		}
		i := slices.IndexFunc(runs.Runs, func(x ml.Profile) bool { return x.RunID == runID })
		if found = i >= 0; found {
			p = runs.Runs[i]
		}
	}
	if !found {
		return RunProfile{}, notFound("Выполнение операции", runID)
	}
	return profileView(p), nil
}

// Violations — окна нарушений специального процесса (machinelogs.violation.list, FR-151).
func (s *Service) Violations(ctx context.Context, m platform.Moment) (ViolationList, error) {
	if s.Store == nil {
		return s.Unimplemented.Violations(ctx, m)
	}
	v, _, err := get[Violations](ctx, s.Store, ProjectionViolations, key(m.RunID))
	if err != nil {
		return ViolationList{}, err
	}
	out := ViolationList{Items: []ViolationWindow{}}
	for _, w := range v.Windows {
		if w.EquipmentID == "" || !visible(m, w.OccurredAt, w.RecordedAt) {
			continue
		}
		end := w.End
		vw := ViolationWindow{EquipmentID: w.EquipmentID, StepKey: w.StepKey, WindowStart: w.Start, WindowEnd: &end,
			DeviationEventIDs: nonNilStr(w.DeviationIDs), OperationRunIDs: nonNilStr(w.RunIDs),
			Items: []platform.DrillRef{}, Nonconformities: []platform.DrillRef{}}
		for _, it := range w.ItemIDs {
			vw.Items = append(vw.Items, platform.DrillRef{Entity: platform.EntityItem, ID: it})
		}
		for _, n := range w.NCs {
			vw.Nonconformities = append(vw.Nonconformities, platform.DrillRef{Entity: platform.EntityNonconformity, ID: n.NcID})
		}
		out.Items = append(out.Items, vw)
	}
	return out, nil
}

func nonNilStr(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

// ——— представление для интерфейса ———

func orUnknown(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}

func equipmentView(st ml.Status) EquipmentState {
	v := EquipmentState{EquipmentID: st.EquipmentID, Title: st.Title, StationID: st.StationID,
		Execution: orUnknown(st.Execution), ControllerMode: orUnknown(st.ControllerMode), Condition: orUnknown(st.Condition),
		ProgramRef: st.ProgramRef, ToolID: st.ToolID, CurrentRunID: st.CurrentRunID, SpecialProcess: st.SpecialProcess,
		Verification: EquipmentVerification{Status: "unknown"}, Warnings: []EquipmentWarning{}, SourceKind: st.SourceKind}
	if v.Title == "" {
		v.Title = st.EquipmentID
	}
	if st.ProgramRevision != "" {
		v.ProgramRef += " ред. " + st.ProgramRevision
	}
	v.ToolLifeUsed, v.ToolLifeLimit = intPtr(st.ToolLifeUsed), intPtr(st.ToolLifeLimit)
	switch st.Verification {
	case "valid":
		v.Verification.Status = "valid"
	case "invalid":
		v.Verification.Status = "expired"
	}
	if t, err := time.Parse("2006-01-02", st.VerifiedTill); err == nil {
		v.Verification.ValidTill = &t
	}
	for _, w := range st.Warnings {
		v.Warnings = append(v.Warnings, EquipmentWarning{Kind: warningKind(w.Kind), Text: deviationText(w.Kind, w.Parameter, w.Value, w.Setpoint),
			Since: w.Since, EventID: w.EventID})
	}
	if !st.UpdatedAt.IsZero() {
		t := st.UpdatedAt
		v.UpdatedAt = &t
	}
	return v
}

func intPtr(p *int64) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

func warningKind(k string) string {
	switch k {
	case ml.DeviationOutOfSetpoint, ml.DeviationOverload, ml.DeviationToolLife, ml.DeviationManualOverride, ml.DeviationProgramChange, ml.DeviationAlarm:
		return k
	}
	return "other"
}

func profileView(p ml.Profile) RunProfile {
	v := RunProfile{OperationRunID: p.RunID, ItemID: p.ItemID, StepKey: p.StepKey, EquipmentID: p.EquipmentID,
		OperatorID: p.OperatorID, StartedAt: p.StartedAt, FinishedAt: p.FinishedAt, IntervalOrigin: p.Origin,
		ProgramRef: p.ProgramRef, ToolID: p.ToolID, Parameters: []CycleParameter{}, Events: []EquipmentEventRow{},
		SpecialProcess: p.SpecialProcess, Violation: p.Violation()}
	if v.IntervalOrigin == "" {
		v.IntervalOrigin = ml.OriginSourceReported
	}
	if p.ProgramRevision != "" {
		v.ProgramRef += " ред. " + p.ProgramRevision
	}
	for _, ps := range p.Parameters {
		v.Parameters = append(v.Parameters, parameterView(ps))
	}
	for _, e := range p.Events {
		v.Events = append(v.Events, eventRow(e))
	}
	return v
}

func parameterView(ps ml.ParamSummary) CycleParameter {
	c := CycleParameter{Parameter: ps.Parameter, InRange: ps.InRange()}
	m := ps.Max
	if m == nil {
		m = ps.Mean
	}
	if m != nil {
		val := m.Value
		c.Value, c.Scale, c.Unit = &val, m.Scale, m.Unit
	}
	if ps.Setpoint != nil {
		c.Setpoint = toleranceText(*ps.Setpoint)
		if c.Unit == "" {
			if b := bound(ps.Setpoint); b != nil {
				c.Unit, c.Scale = b.Unit, b.Scale
			}
		}
	}
	return c
}

func bound(t *ml.Tolerance) *ml.Measure {
	for _, m := range []*ml.Measure{t.Nominal, t.Upper, t.Lower} {
		if m != nil {
			return m
		}
	}
	return nil
}

func eventRow(e ml.ProfileEvent) EquipmentEventRow {
	row := EquipmentEventRow{EventID: e.EventID, EventType: string(e.Type), Layer: string(e.Layer), Seq: e.Seq,
		OccurredAt: e.Start, EndedAt: e.End, Summary: summary(e.Event), SourceKind: e.SourceKind,
		Params: map[string]string{"class": string(e.Class)}}
	if e.Binding != "" {
		row.Params["binding"] = e.Binding
	}
	if e.DeviationKind != "" {
		row.Params["deviation_kind"] = e.DeviationKind
	}
	if e.Parameter != "" {
		row.Params["parameter"] = e.Parameter
	}
	if e.Value != nil {
		row.Params["value"] = measureText(*e.Value)
	}
	if e.Setpoint != nil {
		row.Params["setpoint"] = toleranceText(*e.Setpoint)
	}
	return row
}

// ——— тексты (русский, термины словаря продукта) ———

var executionText = map[string]string{"running": "работает", "idle": "ожидает", "stopped": "остановлено", "setup": "наладка", "interrupted": "прервано", "unknown": "неизвестно"}
var modeText = map[string]string{"automatic": "автомат", "manual": "ручной", "manual_data_input": "ручной ввод", "unknown": "неизвестно"}
var conditionText = map[string]string{"normal": "норма", "warning": "предупреждение", "fault": "неисправность", "unknown": "неизвестно"}

var parameterText = map[string]string{"current": "Ток", "voltage": "Напряжение", "wire_feed": "Подача проволоки",
	"feed_override": "Коррекция подачи", "spindle_override": "Коррекция шпинделя", "spindle_load": "Нагрузка шпинделя",
	"feed_rate": "Подача", "spindle_speed": "Обороты шпинделя", "heat_input": "Тепловложение", "temperature": "Температура"}

func paramName(p string) string {
	if t, ok := parameterText[p]; ok {
		return t
	}
	return p
}

// measureText — «180,0 A» (значение × 10^(−scale) без float).
func measureText(m ml.Measure) string {
	v := m.Value
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	s := fmt.Sprint(v)
	if m.Scale > 0 {
		for len(s) <= m.Scale {
			s = "0" + s
		}
		s = s[:len(s)-m.Scale] + "," + s[len(s)-m.Scale:]
	}
	unit := m.Unit
	if unit != "%" && unit != "" {
		unit = " " + unit
	} else if unit == "%" {
		unit = " %"
	}
	return sign + s + unit
}

func toleranceText(t ml.Tolerance) string {
	switch {
	case t.Lower != nil && t.Upper != nil:
		return measureText(*t.Lower) + " … " + measureText(*t.Upper)
	case t.Upper != nil:
		return "не выше " + measureText(*t.Upper)
	case t.Lower != nil:
		return "не ниже " + measureText(*t.Lower)
	case t.Nominal != nil:
		return measureText(*t.Nominal)
	}
	return ""
}

func deviationText(kind, param string, v *ml.Measure, sp *ml.Tolerance) string {
	var b strings.Builder
	switch kind {
	case ml.DeviationOutOfSetpoint:
		b.WriteString(paramName(param) + " вне уставки")
	case ml.DeviationOverload:
		b.WriteString("Перегрузка: " + strings.ToLower(paramName(param)))
	case ml.DeviationToolLife:
		b.WriteString("Ресурс инструмента на исходе")
	case ml.DeviationManualOverride:
		b.WriteString("Ручное изменение режима")
		if param != "" {
			b.WriteString(": " + strings.ToLower(paramName(param)))
		}
	case ml.DeviationProgramChange:
		b.WriteString("Внеплановая смена программы")
	case ml.DeviationAlarm:
		b.WriteString("Авария")
	default:
		b.WriteString("Отклонение")
	}
	if v != nil {
		b.WriteString(" " + measureText(*v))
	}
	if sp != nil {
		if t := toleranceText(*sp); t != "" {
			b.WriteString(" при уставке " + t)
		}
	}
	return b.String()
}

func summary(e ml.Event) string {
	switch e.Type {
	case catalog.EquipmentStateChanged:
		return "Режим: " + executionText[orUnknown(e.Execution)] + ", управление: " + modeText[orUnknown(e.ControllerMode)] +
			", исправность: " + conditionText[orUnknown(e.Condition)]
	case catalog.EquipmentProgramChanged:
		s := "Программа " + e.ProgramRef
		if e.ProgramRevision != "" {
			s += " ред. " + e.ProgramRevision
		}
		if e.Planned != nil && !*e.Planned {
			s += " (внеплановая смена)"
		}
		return s
	case catalog.EquipmentToolChanged:
		s := "Инструмент " + e.ToolID
		if e.ToolLifeUsed != nil && e.ToolLifeLimit != nil {
			s += fmt.Sprintf(", ресурс %d/%d", *e.ToolLifeUsed, *e.ToolLifeLimit)
		}
		return s
	case catalog.EquipmentCycleSummarized:
		parts := []string{}
		for _, p := range e.Parameters {
			pv := parameterView(p)
			t := paramName(p.Parameter)
			if p.Max != nil {
				t += " макс. " + measureText(*p.Max)
			}
			if pv.InRange != nil && !*pv.InRange {
				t += " — вне уставки"
			}
			parts = append(parts, t)
		}
		return "Сводка цикла: " + strings.Join(parts, "; ")
	case catalog.EquipmentDeviationDetected:
		return deviationText(e.DeviationKind, e.Parameter, e.Value, e.Setpoint)
	}
	return string(e.Type)
}
