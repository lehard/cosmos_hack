package analytics

import (
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Глобальные проекции показателей (AD-45): записи вне потока изделия —
// состояние оборудования, остановки точки процесса и инциденты (гипотезы,
// причины, ошибки исполнителей). Свёртки — чистые функции
// (состояние, запись) → состояние; хранятся все переходы, а интервалы
// вычисляются из них заново, поэтому позднее событие встаёт на своё место,
// а повтор (тот же event_id) ничего не меняет.

// Transition — переход состояния оборудования или остановки.
type Transition struct {
	At      time.Time `json:"at"`
	EventID string    `json:"event_id"`
	// Down — оборудование простаивает: остановлено, прервано, неисправно или
	// остановлена точка процесса.
	Down bool   `json:"down"`
	Kind string `json:"kind,omitempty"`
	// SourceKind — вид источника (FR-140).
	SourceKind string `json:"source_kind,omitempty"`
}

// Equipment — проекция оборудования или остановки точки процесса.
type Equipment struct {
	// Key — equipment_id или `hold:‹hold_id›`.
	Key         string       `json:"key"`
	EquipmentID string       `json:"equipment_id,omitempty"`
	Station     string       `json:"station_id,omitempty"`
	Step        string       `json:"step_key,omitempty"`
	RunID       string       `json:"run_id,omitempty"`
	Transitions []Transition `json:"transitions"`
}

type equipmentData struct {
	EquipmentID string `json:"equipment_id"`
	Station     string `json:"station_id"`
	Execution   string `json:"execution"`
	Condition   string `json:"condition"`
}

type holdData struct {
	HoldID      string `json:"hold_id"`
	EquipmentID string `json:"equipment_id"`
	Step        string `json:"step_key"`
}

// EquipmentKeys — ключи проекции оборудования, которые меняет запись.
func EquipmentKeys(r kernel.Record) []string {
	switch r.Type {
	case catalog.EquipmentStateChanged:
		if d, ok := decode[equipmentData](r.Data); ok && d.EquipmentID != "" {
			return []string{d.EquipmentID}
		}
	case catalog.DecisionProcessHoldSet, catalog.DecisionProcessHoldReleased:
		if d, ok := decode[holdData](r.Data); ok && d.HoldID != "" {
			return []string{"hold:" + d.HoldID}
		}
	}
	return nil
}

// EquipmentStep — свёртка проекции оборудования по записи (FR-89: простой).
func EquipmentStep(prev Equipment, key string, r kernel.Record) Equipment {
	s := prev
	s.Key = key
	if slices.ContainsFunc(s.Transitions, func(t Transition) bool { return t.EventID == r.EventID }) {
		return s // повтор записи
	}
	if s.RunID == "" {
		s.RunID = r.RunID
	}
	tr := Transition{At: r.OccurredAt, EventID: r.EventID, SourceKind: sourceKind(r)}
	switch r.Type {
	case catalog.EquipmentStateChanged:
		d, _ := decode[equipmentData](r.Data)
		s.EquipmentID = d.EquipmentID
		if d.Station != "" {
			s.Station = d.Station
		}
		tr.Down = d.Execution == "stopped" || d.Execution == "interrupted" || d.Condition == "fault"
		tr.Kind = d.Execution
		if d.Condition == "fault" {
			tr.Kind = "fault"
		}
	case catalog.DecisionProcessHoldSet:
		d, _ := decode[holdData](r.Data)
		s.EquipmentID, s.Step = d.EquipmentID, d.Step
		tr.Down, tr.Kind = true, "process_hold"
	case catalog.DecisionProcessHoldReleased:
		tr.Kind = "process_hold_released"
	default:
		return s
	}
	s.Transitions = append(slices.Clone(s.Transitions), tr)
	slices.SortStableFunc(s.Transitions, func(a, b Transition) int {
		if !a.At.Equal(b.At) {
			return a.At.Compare(b.At)
		}
		return cmpStr(a.EventID, b.EventID)
	})
	return s
}

// Downtime — интервалы простоя из переходов: от первого перехода «простой»
// до первого следующего «работает»; незакрытый интервал длится до сих пор.
// Длительность вычисляет система по моментам источника.
func (e Equipment) Downtime() []Row {
	var out []Row
	var cur *Row
	for _, t := range e.Transitions {
		switch {
		case t.Down && cur == nil:
			// Ref — вид простоя: stopped, interrupted, fault, process_hold.
			dims := Dims{Run: e.RunID, Equipment: e.EquipmentID, Station: e.Station, Step: e.Step, Ref: t.Kind,
				Meaning: MeaningOther, DurationOrigin: OriginSystem}
			cur = &Row{Metric: RowEquipmentDowntime, At: t.At, Interval: true, Unit: UnitSec, Dims: dims, Value: 1}
			cur.Sources, cur.Kinds = addUnique(nil, t.EventID), addUnique(nil, t.SourceKind)
		case t.Down && cur != nil:
			cur.Sources, cur.Kinds = addUnique(cur.Sources, t.EventID), addUnique(cur.Kinds, t.SourceKind)
		case !t.Down && cur != nil:
			u := t.At
			cur.Until = &u
			cur.Sources, cur.Kinds = addUnique(cur.Sources, t.EventID), addUnique(cur.Kinds, t.SourceKind)
			out = append(out, *cur)
			cur = nil
		}
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}

// Incident — проекция инцидента для раздельного учёта (FR-87): гипотезы,
// выводы о причине и подтверждённые ошибки исполнителей.
type Incident struct {
	IncidentID string          `json:"incident_id"`
	RunID      string          `json:"run_id,omitempty"`
	Hypotheses []IncidentEntry `json:"hypotheses"`
	Causes     []IncidentEntry `json:"causes"`
	Errors     []IncidentEntry `json:"errors"`
}

// IncidentEntry — решение по инциденту.
type IncidentEntry struct {
	EventID string    `json:"event_id"`
	At      time.Time `json:"at"`
	NCIDs   []string  `json:"nc_ids,omitempty"`
	// Category — категория причины или гипотезы: incoming, equipment,
	// performer, handling, assembly, documentation, not_established.
	Category string `json:"category,omitempty"`
	// Conclusion — для вывода о причине: confirmed | not_established.
	Conclusion string `json:"conclusion,omitempty"`
	// Branch — для гипотезы: why_made | why_missed.
	Branch string `json:"branch,omitempty"`
	// Operator — для ошибки исполнителя.
	Operator   string `json:"operator,omitempty"`
	SourceKind string `json:"source_kind,omitempty"`
}

type incidentData struct {
	IncidentID string   `json:"incident_id"`
	NCIDs      []string `json:"nc_ids"`
	Category   string   `json:"category"`
	Conclusion string   `json:"conclusion"`
	Branch     string   `json:"branch"`
	Operator   string   `json:"operator_id"`
}

// IncidentKeys — ключ проекции инцидента, который меняет запись.
func IncidentKeys(r kernel.Record) []string {
	switch r.Type {
	case catalog.IncidentHypothesisRecorded, catalog.IncidentCauseConcluded, catalog.IncidentOperatorErrorConfirmed:
		if d, ok := decode[incidentData](r.Data); ok && d.IncidentID != "" {
			return []string{d.IncidentID}
		}
	}
	return nil
}

// IncidentStep — свёртка проекции инцидента по записи.
func IncidentStep(prev Incident, key string, r kernel.Record) Incident {
	s := prev
	s.IncidentID = key
	if s.RunID == "" {
		s.RunID = r.RunID
	}
	d, _ := decode[incidentData](r.Data)
	e := IncidentEntry{EventID: r.EventID, At: r.OccurredAt, NCIDs: slices.Sorted(slices.Values(d.NCIDs)), Category: d.Category,
		Conclusion: d.Conclusion, Branch: d.Branch, Operator: d.Operator, SourceKind: sourceKind(r)}
	add := func(list []IncidentEntry) []IncidentEntry {
		if slices.ContainsFunc(list, func(x IncidentEntry) bool { return x.EventID == r.EventID }) {
			return list
		}
		out := append(slices.Clone(list), e)
		slices.SortStableFunc(out, func(a, b IncidentEntry) int {
			if !a.At.Equal(b.At) {
				return a.At.Compare(b.At)
			}
			return cmpStr(a.EventID, b.EventID)
		})
		return out
	}
	switch r.Type {
	case catalog.IncidentHypothesisRecorded:
		s.Hypotheses = add(s.Hypotheses)
	case catalog.IncidentCauseConcluded:
		s.Causes = add(s.Causes)
	case catalog.IncidentOperatorErrorConfirmed:
		s.Errors = add(s.Errors)
	}
	return s
}
