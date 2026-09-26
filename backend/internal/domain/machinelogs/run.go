package machinelogs

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Происхождение интервала выполнения («Соглашения/Длительности»).
const (
	OriginSourceReported = "source_reported"
	OriginSystemComputed = "system_computed"
)

// Run — выполнение операции над изделием на оборудовании (FR-121): кто, что,
// где и интервал. Интервал берётся из operation.run.interval_resolved
// (вычисляет process, эпик 17); пока его нет — из начала и конца выполнения,
// как их сообщил источник (origin = source_reported). Второго алгоритма
// интервалов здесь нет (AD-42): при появлении interval_resolved он главнее.
type Run struct {
	RunID         string     `json:"operation_run_id"`
	ItemID        string     `json:"item_id"`
	ScenarioRun   string     `json:"run_id,omitempty"`
	StepKey       string     `json:"step_key,omitempty"`
	OperationCode string     `json:"operation_code,omitempty"`
	EquipmentID   string     `json:"equipment_id,omitempty"`
	StationID     string     `json:"station_id,omitempty"`
	OperatorID    *string    `json:"operator_id"`
	ProgramRef    string     `json:"program_ref,omitempty"`
	ReworkOf      string     `json:"rework_of,omitempty"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	Origin        string     `json:"interval_origin"`
	// Resolved — интервал задан operation.run.interval_resolved (главнее
	// сообщённого источником).
	Resolved bool `json:"resolved,omitempty"`
	// Causes — записи выполнения (начало, конец, интервал) для причин
	// адресованных записей стадии.
	Causes []Cause `json:"causes,omitempty"`
}

// Cause — ссылка на запись-причину: id и время (AD-3, AD-37).
type Cause struct {
	EventID    string    `json:"event_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Record — причина как запись для kernel.NewAddressed.
func (c Cause) Record() kernel.Record {
	return kernel.Record{EventID: c.EventID, OccurredAt: c.OccurredAt}
}

// Closed — интервал выполнения закрыт (конец известен).
func (r Run) Closed() bool { return r.FinishedAt != nil }

// Overlaps — интервал выполнения пересекается с [start, end] (границы
// включены; открытое выполнение — до бесконечности).
func (r Run) Overlaps(start, end time.Time) bool {
	if r.StartedAt.After(end) {
		return false
	}
	return r.FinishedAt == nil || !r.FinishedAt.Before(start)
}

// Contains — момент t в интервале выполнения (закрытого).
func (r Run) Contains(t time.Time) bool {
	return r.FinishedAt != nil && !t.Before(r.StartedAt) && !t.After(*r.FinishedAt)
}

type runData struct {
	OperationRunID      string     `json:"operation_run_id"`
	OperationCode       string     `json:"operation_code"`
	StepKey             string     `json:"step_key"`
	StationID           string     `json:"station_id"`
	EquipmentID         string     `json:"equipment_id"`
	OperatorID          *string    `json:"operator_id"`
	ProgramRef          string     `json:"program_ref"`
	ReworkOf            string     `json:"rework_of"`
	OperationStartedAt  *time.Time `json:"operation_started_at"`
	OperationFinishedAt *time.Time `json:"operation_finished_at"`
	IntervalStart       *time.Time `json:"interval_start"`
	IntervalEnd         *time.Time `json:"interval_end"`
	IntervalOrigin      string     `json:"interval_origin"`
}

// IsRunRecord — запись выполнения операции, которую читает machinelogs.
func IsRunRecord(t catalog.Type) bool {
	return t == catalog.OperationRunStarted || t == catalog.OperationRunFinished || t == catalog.OperationRunIntervalResolved
}

// RunID — operation_run_id записи выполнения; пусто — не запись выполнения.
func RunID(r kernel.Record) string {
	if !IsRunRecord(r.Type) {
		return ""
	}
	var d struct {
		ID string `json:"operation_run_id"`
	}
	if json.Unmarshal(r.Data, &d) != nil {
		return ""
	}
	return d.ID
}

// ApplyRun применяет запись выполнения к выполнению (чистая функция).
func ApplyRun(run Run, r kernel.Record) (Run, error) {
	var d runData
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return run, fmt.Errorf("machinelogs: данные %s: %w", r.Type, err)
	}
	if run.RunID == "" {
		run.RunID, run.Origin = d.OperationRunID, OriginSourceReported
	}
	if run.ItemID == "" {
		run.ItemID = r.ItemID
	}
	if run.ScenarioRun == "" {
		run.ScenarioRun = r.RunID
	}
	if d.EquipmentID != "" {
		run.EquipmentID = d.EquipmentID
	}
	switch r.Type {
	case catalog.OperationRunStarted:
		run.StepKey, run.OperationCode, run.StationID = d.StepKey, d.OperationCode, d.StationID
		run.OperatorID, run.ProgramRef, run.ReworkOf = d.OperatorID, d.ProgramRef, d.ReworkOf
		if !run.Resolved {
			run.StartedAt = r.OccurredAt.UTC()
			if d.OperationStartedAt != nil {
				run.StartedAt = d.OperationStartedAt.UTC()
			}
		}
	case catalog.OperationRunFinished:
		if !run.Resolved {
			end := r.OccurredAt.UTC()
			if d.OperationFinishedAt != nil {
				end = d.OperationFinishedAt.UTC()
			}
			run.FinishedAt = &end
			if d.OperationStartedAt != nil && run.StartedAt.IsZero() {
				run.StartedAt = d.OperationStartedAt.UTC()
			}
		}
	case catalog.OperationRunIntervalResolved:
		run.Resolved = true
		if d.IntervalStart != nil {
			run.StartedAt = d.IntervalStart.UTC()
		}
		if d.IntervalEnd != nil {
			end := d.IntervalEnd.UTC()
			run.FinishedAt = &end
		}
		if d.IntervalOrigin != "" {
			run.Origin = d.IntervalOrigin
		}
	}
	run.Causes = addCause(run.Causes, Cause{EventID: r.EventID, OccurredAt: r.OccurredAt.UTC()})
	return run, nil
}

func addCause(cs []Cause, c Cause) []Cause {
	for _, x := range cs {
		if x.EventID == c.EventID {
			return cs
		}
	}
	return append(slices.Clone(cs), c)
}
