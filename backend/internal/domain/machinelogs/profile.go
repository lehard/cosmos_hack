package machinelogs

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"ant/internal/contracts/catalog"
)

// Профиль выполнения операции (FR-148, AD-29) — сводный объект одного
// выполнения операции на оборудовании для технолога вместо тысяч строк
// журнала: изделие, станок, программа, инструмент и его ресурс, начало и
// конец, режим, ручные изменения, предупреждения, сводки параметров против
// уставки и отклонения. Строится свёрткой изделия из записей выполнения
// (operation.run.*) и привязанных стадией событий оборудования
// (equipment.event.bound): привязку делает только межизделийная стадия.

// Способ привязки события оборудования к выполнению (equipment.event.bound).
const (
	BindingInterval = "interval"
	BindingContext  = "context"
)

// ProfileEvent — событие оборудования в профиле: классификация MTConnect,
// слой и способ привязки (в интервале выполнения / обстановка до начала).
type ProfileEvent struct {
	Event
	Class   Class  `json:"class"`
	Layer   Layer  `json:"layer"`
	Binding string `json:"binding"`
}

// Deviation — отклонение на выполнении (слой 4, FR-147): вид, параметр,
// значение против уставки, интервал; самостоятельный сигнал даже без
// дефекта (FR-121).
type Deviation struct {
	EventID   string     `json:"event_id"`
	Kind      string     `json:"kind"`
	Parameter string     `json:"parameter,omitempty"`
	Value     *Measure   `json:"value,omitempty"`
	Setpoint  *Tolerance `json:"setpoint,omitempty"`
	Start     time.Time  `json:"start"`
	End       *time.Time `json:"end,omitempty"`
}

// Profile — профиль выполнения операции (FR-148).
type Profile struct {
	Run
	// SpecialProcess — шаг выполнения — специальный процесс (FR-151, AD-17).
	SpecialProcess bool `json:"special_process"`

	ProgramRef      string `json:"program,omitempty"`
	ProgramRevision string `json:"program_revision,omitempty"`
	ToolID          string `json:"tool_id,omitempty"`
	ToolLifeUsed    *int64 `json:"tool_life_used,omitempty"`
	ToolLifeLimit   *int64 `json:"tool_life_limit,omitempty"`
	// ControllerMode — режим управления в начале выполнения; ManualMode —
	// во время выполнения был ручной режим (ручное изменение режима).
	ControllerMode string `json:"controller_mode,omitempty"`
	ManualMode     bool   `json:"manual_mode,omitempty"`

	Parameters []ParamSummary `json:"parameters,omitempty"`
	Events     []ProfileEvent `json:"events,omitempty"`
	Deviations []Deviation    `json:"deviations,omitempty"`
	Warnings   []ProfileEvent `json:"warnings,omitempty"`
	// Nonconformities, ViolationWindows — несоответствия, зарегистрированные
	// стадией по окну нарушения специального процесса (FR-151).
	Nonconformities  []string `json:"nonconformities,omitempty"`
	ViolationWindows []string `json:"violation_windows,omitempty"`
}

// Violation — нарушение режима специального процесса на выполнении
// (FR-151): стадия зарегистрировала несоответствие по окну нарушения.
func (p Profile) Violation() bool { return len(p.ViolationWindows) > 0 }

// ManualOverrides — ручные коррекции режима на выполнении (например,
// подача 130 %, FR-147).
func (p Profile) ManualOverrides() []Deviation {
	var out []Deviation
	for _, d := range p.Deviations {
		if d.Kind == DeviationManualOverride {
			out = append(out, d)
		}
	}
	return out
}

// Bound — данные записи equipment.event.bound (привязка стадии).
type Bound struct {
	OperationRunID    string          `json:"operation_run_id"`
	EquipmentID       string          `json:"equipment_id"`
	Binding           string          `json:"binding"`
	SubjectEventID    string          `json:"subject_event_id"`
	SubjectEventType  string          `json:"subject_event_type"`
	SubjectOccurredAt time.Time       `json:"subject_occurred_at"`
	SubjectSourceKind string          `json:"subject_source_kind,omitempty"`
	SubjectData       json.RawMessage `json:"subject_data"`
}

// Subject — привязанное событие оборудования в представлении домена.
func (b Bound) Subject() (Event, error) {
	e, err := ParseData(catalog.Type(b.SubjectEventType), b.SubjectData, b.SubjectOccurredAt.UTC())
	if err != nil {
		return Event{}, err
	}
	e.EventID, e.SourceKind = b.SubjectEventID, SourceKind(b.SubjectSourceKind)
	return e, nil
}

// ApplyBound добавляет привязанное событие к профилю (повтор того же
// события не меняет профиль) и пересчитывает производные поля.
func ApplyBound(p Profile, ev Event, binding string) Profile {
	ev.Data = nil
	for _, x := range p.Events {
		if x.EventID == ev.EventID {
			return p
		}
	}
	c, l := Classify(ev)
	events := append(slices.Clone(p.Events), ProfileEvent{Event: ev, Class: c, Layer: l, Binding: binding})
	slices.SortStableFunc(events, func(a, b ProfileEvent) int {
		if x := a.Start.Compare(b.Start); x != 0 {
			return x
		}
		return cmp.Compare(a.EventID, b.EventID)
	})
	p.Events = events
	return derive(p)
}

// derive — программа, инструмент, режим, сводки, отклонения и
// предупреждения профиля из событий (детерминированно, по порядку времени).
func derive(p Profile) Profile {
	p.ProgramRef, p.ProgramRevision, p.ToolID, p.ToolLifeUsed, p.ToolLifeLimit = p.Run.ProgramRef, "", "", nil, nil
	p.ControllerMode, p.ManualMode = "", false
	p.Parameters, p.Deviations, p.Warnings = nil, nil, nil
	for _, e := range p.Events {
		switch e.Type {
		case catalog.EquipmentProgramChanged:
			p.ProgramRef, p.ProgramRevision = e.ProgramRef, e.ProgramRevision
			if e.Binding == BindingInterval && e.Planned != nil && !*e.Planned {
				p.Deviations = append(p.Deviations, Deviation{EventID: e.EventID, Kind: DeviationProgramChange, Start: e.Start})
			}
		case catalog.EquipmentToolChanged:
			p.ToolID, p.ToolLifeUsed, p.ToolLifeLimit = e.ToolID, e.ToolLifeUsed, e.ToolLifeLimit
		case catalog.EquipmentStateChanged:
			if e.Binding == BindingContext || p.ControllerMode == "" {
				p.ControllerMode = e.ControllerMode
			}
			if e.Binding == BindingInterval && e.ControllerMode == ModeManual {
				p.ManualMode = true
			}
			if e.Binding == BindingInterval && (e.Condition == ConditionWarning || e.Condition == ConditionFault) {
				p.Warnings = append(p.Warnings, e)
			}
		case catalog.EquipmentCycleSummarized:
			if e.Binding == BindingInterval {
				p.Parameters = mergeParams(p.Parameters, e.Parameters)
			}
		case catalog.EquipmentDeviationDetected:
			if e.Binding == BindingInterval {
				p.Deviations = append(p.Deviations, Deviation{EventID: e.EventID, Kind: e.DeviationKind, Parameter: e.Parameter,
					Value: e.Value, Setpoint: e.Setpoint, Start: e.Start, End: e.End})
			}
		}
	}
	return p
}

// mergeParams объединяет сводки нескольких циклов одного выполнения:
// максимум — наибольший, минимум — наименьший, среднее — среднее средних
// в общем масштабе (целые, AD-4), время вне уставки — сумма.
func mergeParams(acc, add []ParamSummary) []ParamSummary {
	out := slices.Clone(acc)
	for _, a := range add {
		i := slices.IndexFunc(out, func(x ParamSummary) bool { return x.Parameter == a.Parameter })
		if i < 0 {
			out = append(out, a)
			continue
		}
		x := out[i]
		x.Max = pick(x.Max, a.Max, 1)
		x.Min = pick(x.Min, a.Min, -1)
		if x.Mean != nil && a.Mean != nil && x.Mean.Unit == a.Mean.Unit {
			s := max(x.Mean.Scale, a.Mean.Scale)
			m := Measure{Value: (x.Mean.Value*pow10(s-x.Mean.Scale) + a.Mean.Value*pow10(s-a.Mean.Scale)) / 2, Scale: s, Unit: a.Mean.Unit}
			x.Mean = &m
		} else if x.Mean == nil {
			x.Mean = a.Mean
		}
		if a.Setpoint != nil {
			x.Setpoint = a.Setpoint
		}
		if a.OutOfSetpointMs != nil {
			v := *a.OutOfSetpointMs
			if x.OutOfSetpointMs != nil {
				v += *x.OutOfSetpointMs
			}
			x.OutOfSetpointMs = &v
		}
		out[i] = x
	}
	slices.SortStableFunc(out, func(a, b ParamSummary) int { return cmp.Compare(a.Parameter, b.Parameter) })
	return out
}

// pick — из двух измерений большее (sign=1) или меньшее (sign=−1).
func pick(a, b *Measure, sign int) *Measure {
	switch {
	case a == nil:
		return b
	case b == nil || a.Unit != b.Unit:
		return a
	case Cmp(*b, *a)*sign > 0:
		return b
	}
	return a
}

// MarkViolation отмечает несоответствие окна нарушения на выполнении (FR-151).
func MarkViolation(p Profile, ncID, windowEventID string) Profile {
	if ncID != "" && !slices.Contains(p.Nonconformities, ncID) {
		p.Nonconformities = append(slices.Clone(p.Nonconformities), ncID)
		slices.Sort(p.Nonconformities)
	}
	if windowEventID != "" && !slices.Contains(p.ViolationWindows, windowEventID) {
		p.ViolationWindows = append(slices.Clone(p.ViolationWindows), windowEventID)
		slices.Sort(p.ViolationWindows)
	}
	return p
}

// decodeBound разбирает данные записи привязки.
func decodeBound(data json.RawMessage) (Bound, error) {
	var b Bound
	if err := json.Unmarshal(data, &b); err != nil {
		return b, fmt.Errorf("machinelogs: equipment.event.bound: %w", err)
	}
	return b, nil
}
