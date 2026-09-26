package machinelogs

import (
	"encoding/json"
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Состояние оборудования (стол мастера «Люди и оборудование», FR-148):
// последнее известное по классификации MTConnect — режим работы, режим
// управления, исправность, программа, инструмент и его ресурс, текущее
// выполнение, действующие предупреждения; справочные сведения — название,
// поверка. Чистая свёртка записей оборудования (проекция machinelogs).

// Warning — действующее предупреждение по оборудованию.
type Warning struct {
	Kind      string     `json:"kind"`
	Parameter string     `json:"parameter,omitempty"`
	Value     *Measure   `json:"value,omitempty"`
	Setpoint  *Tolerance `json:"setpoint,omitempty"`
	Since     time.Time  `json:"since"`
	EventID   string     `json:"event_id,omitempty"`
}

// Status — состояние оборудования.
type Status struct {
	EquipmentID     string    `json:"equipment_id"`
	RunID           string    `json:"run_id,omitempty"`
	Title           string    `json:"title,omitempty"`
	Kind            string    `json:"kind,omitempty"`
	StationID       string    `json:"station_id,omitempty"`
	Execution       string    `json:"execution,omitempty"`
	ControllerMode  string    `json:"controller_mode,omitempty"`
	Condition       string    `json:"condition,omitempty"`
	ProgramRef      string    `json:"program_ref,omitempty"`
	ProgramRevision string    `json:"program_revision,omitempty"`
	ToolID          string    `json:"tool_id,omitempty"`
	ToolLifeUsed    *int64    `json:"tool_life_used,omitempty"`
	ToolLifeLimit   *int64    `json:"tool_life_limit,omitempty"`
	CurrentRunID    string    `json:"current_run_id,omitempty"`
	SpecialProcess  bool      `json:"special_process,omitempty"`
	Verification    string    `json:"verification,omitempty"`
	VerifiedTill    string    `json:"verified_till,omitempty"`
	Warnings        []Warning `json:"warnings,omitempty"`
	SourceKind      string    `json:"source_kind,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitzero"`
	LastSeq         int64     `json:"last_seq,omitempty"`
}

// maxWarnings — сколько последних предупреждений держит состояние.
const maxWarnings = 5

// EquipmentOf — оборудование, к состоянию которого относится запись; пусто —
// запись не про оборудование.
func EquipmentOf(r kernel.Record) string {
	switch {
	case IsEquipmentFact(r.Type), r.Type == catalog.ReferenceEquipmentDefined, r.Type == catalog.ReferenceEquipmentVerified,
		r.Type == catalog.OperationRunStarted, r.Type == catalog.OperationRunIntervalResolved:
		var d struct {
			EquipmentID string `json:"equipment_id"`
		}
		if json.Unmarshal(r.Data, &d) == nil {
			return d.EquipmentID
		}
	}
	return ""
}

// ApplyStatus применяет запись к состоянию оборудования (чистая функция).
func ApplyStatus(st Status, env Env, r kernel.Record) Status {
	if st.EquipmentID == "" {
		st.EquipmentID, st.RunID = EquipmentOf(r), r.RunID
	}
	switch r.Type {
	case catalog.ReferenceEquipmentDefined:
		var d struct {
			Name       string `json:"name"`
			Kind       string `json:"kind"`
			LocationID string `json:"location_id"`
		}
		if json.Unmarshal(r.Data, &d) == nil {
			st.Title, st.Kind = d.Name, d.Kind
			if st.StationID == "" {
				st.StationID = d.LocationID
			}
		}
		return st
	case catalog.ReferenceEquipmentVerified:
		var d struct {
			Result     string `json:"result"`
			ValidUntil string `json:"valid_until"`
		}
		if json.Unmarshal(r.Data, &d) == nil {
			st.Verification, st.VerifiedTill = d.Result, d.ValidUntil
		}
		return st
	case catalog.OperationRunStarted:
		var d struct {
			ID      string `json:"operation_run_id"`
			StepKey string `json:"step_key"`
		}
		if json.Unmarshal(r.Data, &d) == nil {
			st.CurrentRunID = d.ID
			st.SpecialProcess = st.SpecialProcess || env.Special(d.StepKey)
		}
		return st
	case catalog.OperationRunIntervalResolved:
		var d struct {
			ID          string  `json:"operation_run_id"`
			IntervalEnd *string `json:"interval_end"`
		}
		if json.Unmarshal(r.Data, &d) == nil && d.IntervalEnd != nil && st.CurrentRunID == d.ID {
			st.CurrentRunID = ""
		}
		return st
	}
	ev, ok, err := ParseEvent(r)
	if !ok || err != nil {
		return st
	}
	if ev.StationID != "" {
		st.StationID = ev.StationID
	}
	if ev.SourceKind != "" {
		st.SourceKind = ev.SourceKind
	}
	if !ev.OccurredAt.Before(st.UpdatedAt) {
		st.UpdatedAt = ev.OccurredAt
	}
	st.LastSeq = max(st.LastSeq, r.Seq)
	switch ev.Type {
	case catalog.EquipmentStateChanged:
		st.Execution, st.ControllerMode, st.Condition = ev.Execution, ev.ControllerMode, ev.Condition
		if ev.Condition == ConditionWarning || ev.Condition == ConditionFault {
			kind := DeviationAlarm
			if ev.Parameter != "" && ev.Setpoint != nil {
				kind = DeviationOutOfSetpoint
			}
			st.Warnings = addWarning(st.Warnings, Warning{Kind: kind, Parameter: ev.Parameter, Value: ev.Value, Setpoint: ev.Setpoint, Since: ev.Start, EventID: ev.EventID})
		}
		if ev.Execution != ExecutionRunning {
			st.CurrentRunID = ""
		}
	case catalog.EquipmentProgramChanged:
		st.ProgramRef, st.ProgramRevision = ev.ProgramRef, ev.ProgramRevision
		if ev.Planned != nil && !*ev.Planned {
			st.Warnings = addWarning(st.Warnings, Warning{Kind: DeviationProgramChange, Since: ev.Start, EventID: ev.EventID})
		}
	case catalog.EquipmentToolChanged:
		st.ToolID, st.ToolLifeUsed, st.ToolLifeLimit = ev.ToolID, ev.ToolLifeUsed, ev.ToolLifeLimit
		st.Warnings = slices.DeleteFunc(slices.Clone(st.Warnings), func(w Warning) bool { return w.Kind == DeviationToolLife })
	case catalog.EquipmentDeviationDetected:
		st.Warnings = addWarning(st.Warnings, Warning{Kind: ev.DeviationKind, Parameter: ev.Parameter, Value: ev.Value, Setpoint: ev.Setpoint, Since: ev.Start, EventID: ev.EventID})
	case catalog.EquipmentCycleSummarized:
		if closesWindow(ev) {
			// Цикл в пределах уставки: прежние выходы за уставку и ручные
			// коррекции больше не действуют (предупреждение ресурса — до замены инструмента).
			st.Warnings = slices.DeleteFunc(slices.Clone(st.Warnings), func(w Warning) bool { return w.Kind != DeviationToolLife })
		}
	}
	return st
}

func addWarning(ws []Warning, w Warning) []Warning {
	out := slices.DeleteFunc(slices.Clone(ws), func(x Warning) bool { return x.EventID == w.EventID && w.EventID != "" })
	out = append(out, w)
	if len(out) > maxWarnings {
		out = out[len(out)-maxWarnings:]
	}
	return out
}
