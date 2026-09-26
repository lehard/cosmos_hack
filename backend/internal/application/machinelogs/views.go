package machinelogs

import (
	"time"

	"ant/internal/application/platform"
)

// Классификация — как в MTConnect (AD-29): измерение, событие, условие исправности;
// четыре слоя: что делал станок, чем, как шёл процесс, отклонения. Не SCADA:
// без миллисекундной телеметрии (PRD §11.6).

// EquipmentWarning — предупреждение по оборудованию: отклонение, ресурс инструмента, поверка.
type EquipmentWarning struct {
	Kind    string    `json:"kind" enum:"out_of_setpoint,overload,tool_life_warning,manual_override,unplanned_program_change,alarm,verification_due,other"`
	Text    string    `json:"text" doc:"Подпись: «ресурс инструмента 73/75»."`
	Since   time.Time `json:"since"`
	EventID string    `json:"event_id,omitempty"`
}

// EquipmentVerification — поверка оборудования на дату (FR-17, AD-31).
type EquipmentVerification struct {
	Status    string     `json:"status" enum:"valid,expiring,expired,unknown"`
	ValidTill *time.Time `nullable:"true" json:"valid_till,omitempty"`
}

// EquipmentState — оборудование в списке и карточке (стол мастера «Люди и оборудование», FR-148).
type EquipmentState struct {
	EquipmentID    string                `json:"equipment_id"`
	Title          string                `json:"title"`
	StationID      string                `json:"station_id,omitempty"`
	Execution      string                `json:"execution" enum:"running,idle,stopped,setup,interrupted,unknown"`
	ControllerMode string                `json:"controller_mode" enum:"automatic,manual,manual_data_input,unknown"`
	Condition      string                `json:"condition" enum:"normal,warning,fault,unknown"`
	ProgramRef     string                `json:"program_ref,omitempty"`
	ToolID         string                `json:"tool_id,omitempty"`
	ToolLifeUsed   *int                  `nullable:"true" json:"tool_life_used,omitempty" minimum:"0"`
	ToolLifeLimit  *int                  `nullable:"true" json:"tool_life_limit,omitempty" minimum:"0"`
	CurrentRunID   string                `json:"current_run_id,omitempty" doc:"Текущее выполнение операции (привязку делает только межизделийная стадия, AD-29)."`
	SpecialProcess bool                  `json:"special_process" doc:"Выполняет специальный процесс (FR-151)."`
	Verification   EquipmentVerification `json:"verification"`
	Warnings       []EquipmentWarning    `json:"warnings"`
	SourceKind     string                `json:"source_kind,omitempty" enum:"manual_entry,machine,sensor,camera,external_system,import" doc:"Источник данных (FR-140)."`
	UpdatedAt      *time.Time            `json:"updated_at,omitempty"`
}

// EquipmentList — оборудование.
type EquipmentList struct {
	Items []EquipmentState `json:"items"`
}

// CycleParameter — параметр сводки цикла против уставки (FR-147, FR-148).
type CycleParameter struct {
	Parameter string `json:"parameter"`
	Value     *int64 `nullable:"true" json:"value" doc:"Значение = value × 10^(−scale); null — нет данных."`
	Scale     int    `json:"scale"`
	Unit      string `json:"unit"`
	Setpoint  string `json:"setpoint,omitempty" doc:"Уставка и допуск текстом с единицей."`
	InRange   *bool  `nullable:"true" json:"in_range" doc:"В пределах уставки; null — оценка невозможна."`
}

// EquipmentEventRow — запись одного из четырёх слоёв (AD-29) на шкале времени.
type EquipmentEventRow struct {
	EventID    string            `json:"event_id"`
	EventType  string            `json:"event_type"`
	Layer      string            `json:"layer" enum:"what,with_what,how,deviation" doc:"Что делал станок / чем / как шёл процесс / отклонения."`
	Seq        int64             `json:"seq"`
	OccurredAt time.Time         `json:"occurred_at"`
	EndedAt    *time.Time        `json:"ended_at,omitempty"`
	Summary    string            `json:"summary"`
	Params     map[string]string `json:"params,omitempty"`
	SourceKind string            `json:"source_kind,omitempty" enum:"manual_entry,machine,sensor,camera,external_system,import"`
}

// EquipmentTimeline — журнал оборудования на окне (FR-121, AD-29).
type EquipmentTimeline struct {
	EquipmentID string              `json:"equipment_id"`
	From        time.Time           `json:"from"`
	To          time.Time           `json:"to"`
	Rows        []EquipmentEventRow `json:"rows"`
}

// RunProfile — профиль выполнения операции (FR-148): проекция machinelogs —
// оборудование, программа, инструмент, сводки циклов, отклонения на окне выполнения.
type RunProfile struct {
	OperationRunID string              `json:"operation_run_id"`
	ItemID         string              `json:"item_id"`
	StepKey        string              `json:"step_key"`
	EquipmentID    string              `json:"equipment_id,omitempty"`
	OperatorID     *string             `nullable:"true" json:"operator_id" doc:"Псевдоним исполнителя; null — неизвестно."`
	StartedAt      time.Time           `json:"started_at"`
	FinishedAt     *time.Time          `nullable:"true" json:"finished_at"`
	IntervalOrigin string              `json:"interval_origin" enum:"source_reported,system_computed" doc:"Происхождение интервала («Длительности»)."`
	ProgramRef     string              `json:"program_ref,omitempty"`
	ToolID         string              `json:"tool_id,omitempty"`
	Parameters     []CycleParameter    `json:"parameters"`
	Events         []EquipmentEventRow `json:"events"`
	SpecialProcess bool                `json:"special_process"`
	Violation      bool                `json:"violation" doc:"Нарушение режима на специальном процессе (FR-151)."`
}

// ViolationWindow — окно нарушения режима специального процесса (FR-151): стадия
// регистрирует несоответствие для всех изделий окна, даже без найденного дефекта.
type ViolationWindow struct {
	EquipmentID       string              `json:"equipment_id"`
	StepKey           string              `json:"step_key"`
	WindowStart       time.Time           `json:"window_start"`
	WindowEnd         *time.Time          `nullable:"true" json:"window_end" doc:"null — окно не закрыто."`
	DeviationEventIDs []string            `json:"deviation_event_ids"`
	OperationRunIDs   []string            `json:"operation_run_ids"`
	Items             []platform.DrillRef `json:"items"`
	Nonconformities   []platform.DrillRef `json:"nonconformities"`
}

// ViolationList — окна нарушений.
type ViolationList struct {
	Items []ViolationWindow `json:"items"`
}
