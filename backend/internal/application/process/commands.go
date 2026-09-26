package process

import "ant/internal/application/platform"

// DraftVersion — черновик версии процесса из редактора технолога (FR-22, FR-25):
// в журнал не пишется до отправки на кворум.
type DraftVersion struct {
	platform.CommandHeader
	BaseVersionID string `json:"base_version_id,omitempty" maxLength:"128"`
	Label         string `json:"label" maxLength:"64"`
	BpmnXML       string `json:"bpmn_xml" minLength:"1" doc:"BPMN 2.0 XML со свойствами urn:ant:bpmn-ext:1; проверка при загрузке — FR-13."`
}

// SubmitVersion — отправить версию на утверждение кворумом (normative.version.submitted, FR-23):
// создаёт лист утверждения с маршрутом подписей.
type SubmitVersion struct {
	platform.CommandHeader
	Note string `json:"note,omitempty" maxLength:"2000"`
}

// ActivateVersion — ввести версию в действие после закрытия маршрута кворума
// (normative.version.activated; необратимое, критическое — control_change).
type ActivateVersion struct {
	platform.CommandHeader
	RouteClosedEventID string `json:"route_closed_event_id" format:"uuid" doc:"document.route.closed листа утверждения."`
}

// RetireVersion — вывести версию (normative.version.retired; критическое — control_change).
type RetireVersion struct {
	platform.CommandHeader
	ReasonText string `json:"reason" minLength:"1" maxLength:"1000"`
}

// StartOperation — начать операцию (operation.run.started, терминал исполнителя FR-137);
// предусловия FR-17 проверяет гард, нарушение — operation.precondition.failed.
type StartOperation struct {
	platform.CommandHeader
	OperationRunID string `json:"operation_run_id" maxLength:"128" doc:"Новый id выполнения; повтор операции — новый id + rework_of (FR-47)."`
	OperationCode  string `json:"operation_code" maxLength:"64"`
	StepKey        string `json:"step_key" maxLength:"128"`
	StationID      string `json:"station_id,omitempty" maxLength:"128"`
	EquipmentID    string `json:"equipment_id,omitempty" maxLength:"128"`
	ProgramRef     string `json:"program_ref,omitempty" maxLength:"256"`
	ReworkOf       string `json:"rework_of,omitempty" maxLength:"128"`
	GroupID        string `json:"group_id,omitempty" maxLength:"128" doc:"Групповая операция (FR-15)."`
}

// PauseOperation — приостановить операцию (operation.run.paused).
type PauseOperation struct {
	platform.CommandHeader
	PauseReason string `json:"pause_reason" enum:"setup,failure,waiting,shift_end,other,unknown"`
	Note        string `json:"note,omitempty" maxLength:"1000"`
}

// ResumeOperation — продолжить операцию (operation.run.resumed).
type ResumeOperation struct {
	platform.CommandHeader
}

// FinishOperation — завершить операцию (operation.run.finished).
type FinishOperation struct {
	platform.CommandHeader
	Completion string `json:"completion" enum:"completed,interrupted"`
}

// SendMovement — отправить изделие (operation.movement.sent, FR-16).
type SendMovement struct {
	platform.CommandHeader
	FromLocationID string `json:"from_location_id" maxLength:"128"`
	ToLocationID   string `json:"to_location_id" maxLength:"128"`
	ContainerID    string `json:"container_id,omitempty" maxLength:"128"`
	StepKey        string `json:"step_key,omitempty" maxLength:"128"`
}

// ReceiveMovement — подтвердить приём изделия (operation.movement.received):
// в том числе физическое перемещение в изолятор (destination_kind = isolator).
type ReceiveMovement struct {
	platform.CommandHeader
	FromLocationID      string `json:"from_location_id,omitempty" maxLength:"128"`
	ToLocationID        string `json:"to_location_id" maxLength:"128"`
	DestinationKind     string `json:"destination_kind" enum:"station,workshop,warehouse,isolator,other"`
	InspectionOnReceipt string `json:"inspection_on_receipt" enum:"no_damage,damage_found,not_inspected"`
	StepKey             string `json:"step_key,omitempty" maxLength:"128"`
}
