package nonconformity

import (
	"maps"
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// Сдерживание процесса (FR-49) — отдельный объект от сдерживания изделий:
// стоп точки процесса (станок, инструмент, программа, операция) и
// критическая остановка живут в потоке `equipment:‹точка›`; снимает их только
// уполномоченный человек, после снятия действует «точка чистоты» — первые N
// изделий, прошедших точку, получают усиленный контроль.

// Hold — остановка точки процесса.
type Hold struct {
	ID          string    `json:"id"`
	Level       string    `json:"level"`
	EquipmentID string    `json:"equipment_id,omitempty"`
	ToolID      string    `json:"tool_id,omitempty"`
	ProgramRef  string    `json:"program_ref,omitempty"`
	StepKey     string    `json:"step_key,omitempty"`
	IncidentID  string    `json:"incident_id,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	SetEventID  string    `json:"set_event_id"`
	SetAt       time.Time `json:"set_at"`
	Stream      string    `json:"stream"`
	// Released, ReleasedEventID — снятие; CleanPoint — N изделий точки
	// чистоты, CleanRuns — выполнения, уже получившие усиленный контроль.
	Released        bool     `json:"released,omitempty"`
	ReleasedEventID string   `json:"released_event_id,omitempty"`
	CleanPoint      int      `json:"clean_point,omitempty"`
	CleanRuns       []string `json:"clean_runs,omitempty"`
}

// StageState — состояние модуля nonconformity в межизделийной стадии (AD-42):
// остановки точек процесса и счётчики точки чистоты.
type StageState struct {
	Holds map[string]Hold `json:"holds,omitempty"`
}

// Stage — функция модуля nonconformity в межизделийной стадии (AD-42):
// остановки точек процесса и точка чистоты — адресованные записи
// decision.clean_point.assigned в поток изделия (occurred_at — наибольший
// среди причин). Регистрация несоответствий окна нарушения специального
// процесса — RegisterWindowNC через порт machinelogs.Registrar.
func Stage(s StageState, r kernel.Record) (StageState, []kernel.Addressed) {
	switch r.Type {
	case catalog.DecisionProcessHoldSet:
		var d ProcessHoldSetData
		if decode(r, &d) && d.HoldID != "" {
			s = s.clone()
			s.Holds[d.HoldID] = Hold{ID: d.HoldID, Level: d.Level, EquipmentID: d.EquipmentID, ToolID: d.ToolID,
				ProgramRef: d.ProgramRef, StepKey: d.StepKey, IncidentID: d.IncidentID, Reason: d.Reason.Text,
				SetEventID: r.EventID, SetAt: r.OccurredAt, Stream: r.Stream}
		}
	case catalog.DecisionProcessHoldReleased:
		var d ProcessHoldReleasedData
		if decode(r, &d) {
			if h, ok := s.Holds[d.HoldID]; ok && !h.Released {
				s = s.clone()
				h.Released, h.ReleasedEventID, h.CleanPoint = true, r.EventID, d.CleanPointItems
				s.Holds[d.HoldID] = h
			}
		}
	case catalog.OperationRunStarted:
		return s.cleanPoint(r)
	}
	return s, nil
}

// cleanPoint — выполнение операции на точке процесса после снятия остановки:
// первые N изделий — усиленный контроль (FR-49).
func (s StageState) cleanPoint(r kernel.Record) (StageState, []kernel.Addressed) {
	if r.ItemID == "" || len(s.Holds) == 0 {
		return s, nil
	}
	var d struct {
		OperationRunID string `json:"operation_run_id"`
		StepKey        string `json:"step_key"`
		EquipmentID    string `json:"equipment_id"`
		ProgramRef     string `json:"program_ref"`
	}
	if !decode(r, &d) || d.OperationRunID == "" {
		return s, nil
	}
	var out []kernel.Addressed
	for _, id := range slices.Sorted(maps.Keys(s.Holds)) {
		h := s.Holds[id]
		if !h.Released || len(h.CleanRuns) >= h.CleanPoint || slices.Contains(h.CleanRuns, d.OperationRunID) || !h.matches(d.EquipmentID, d.StepKey, d.ProgramRef) {
			continue
		}
		s = s.clone()
		h.CleanRuns = append(h.CleanRuns, d.OperationRunID)
		s.Holds[id] = h
		data := CleanPointData{HoldID: id, OperationRunID: d.OperationRunID, EquipmentID: d.EquipmentID, Ordinal: len(h.CleanRuns), Of: h.CleanPoint}
		a, err := kernel.NewAddressed(Module, catalog.DecisionCleanPointAssigned, "item:"+r.ItemID, id+"|"+d.OperationRunID, data,
			kernel.Record{EventID: h.ReleasedEventID, OccurredAt: time.Time{}}, r)
		if err != nil {
			panic(err)
		}
		out = append(out, a)
	}
	return s, out
}

// matches — выполнение относится к точке процесса остановки: оборудование,
// иначе программа, иначе шаг.
func (h Hold) matches(equipmentID, stepKey, programRef string) bool {
	switch {
	case h.EquipmentID != "":
		return h.EquipmentID == equipmentID
	case h.ProgramRef != "":
		return h.ProgramRef == programRef
	case h.StepKey != "":
		return h.StepKey == stepKey
	}
	return false
}

// Active — остановка действует.
func (h Hold) Active() bool { return !h.Released }

func (s StageState) clone() StageState {
	out := StageState{Holds: make(map[string]Hold, len(s.Holds))}
	for _, k := range slices.Sorted(maps.Keys(s.Holds)) {
		v := s.Holds[k]
		v.CleanRuns = slices.Clone(v.CleanRuns)
		out.Holds[k] = v
	}
	return out
}

// HoldStream — поток точки процесса остановки (AD-39): оборудование, иначе
// инструмент, программа или шаг — всё в виде потока `equipment:‹точка›`.
func HoldStream(d ProcessHoldSetData) string {
	switch {
	case d.EquipmentID != "":
		return "equipment:" + d.EquipmentID
	case d.ToolID != "":
		return "equipment:tool/" + d.ToolID
	case d.ProgramRef != "":
		return "equipment:program/" + d.ProgramRef
	case d.StepKey != "":
		return "equipment:step/" + d.StepKey
	}
	return ""
}

// HoldGuard — гард остановки точки процесса (FR-49): установка — точка
// указана и такой остановки ещё нет; снятие — остановка действует.
func HoldGuard(s StageState, cmd kernel.Command) error {
	switch p := cmd.Payload.(type) {
	case ProcessHoldSetData:
		if HoldStream(p) == "" {
			return kernel.Refuse(errcodes.ApiValidationFailed, "field", "equipment_id", "reason", "укажите оборудование, инструмент, программу или шаг")
		}
		if h, ok := s.Holds[p.HoldID]; ok && h.Active() {
			return kernel.Refuse(errcodes.NonconformityInvalidTransition, "action", "остановить точку процесса", "nc_id", p.HoldID, "status", "уже остановлена")
		}
	case ProcessHoldReleasedData:
		if h, ok := s.Holds[p.HoldID]; !ok || !h.Active() {
			return kernel.Refuse(errcodes.NonconformityProcessHoldNotActive, "hold_id", p.HoldID)
		}
	}
	return nil
}
