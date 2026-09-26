package access

import (
	"time"

	"ant/internal/application/platform"
)

// RegisterPerson — завести сотрудника с псевдонимом (access.person.registered, FR-78).
type RegisterPerson struct {
	platform.CommandHeader
	PersonID    string `json:"person_id" maxLength:"64" doc:"Псевдоним (кейс §4.6); соответствие человеку хранится отдельно."`
	DisplayName string `json:"display_name" minLength:"1" maxLength:"256"`
	OrgUnit     string `json:"org_unit,omitempty" maxLength:"256"`
}

// ActivateAccount — активировать учётную запись с начальной ролью (access.account.activated, FR-128).
type ActivateAccount struct {
	platform.CommandHeader
	Login         string `json:"login" minLength:"1" maxLength:"128"`
	InitialRoleID string `json:"initial_role_id,omitempty" maxLength:"64"`
}

// GrantPolicy — выдать роль, полномочие или цифровое клеймо (policy.role.assigned,
// policy.authority.granted, policy.stamp.issued; AD-11, AD-15, FR-145):
// привилегированная выдача — документом с маршрутом и второй подписью независимой стороны.
type GrantPolicy struct {
	platform.CommandHeader
	PersonID       string         `json:"person_id" maxLength:"64"`
	Kind           string         `json:"kind" enum:"role,authority,stamp"`
	RoleID         string         `json:"role_id,omitempty" maxLength:"64" doc:"Для kind=role."`
	AuthorityID    string         `json:"authority_id,omitempty" maxLength:"64" doc:"Для kind=authority."`
	StampID        string         `json:"stamp_id,omitempty" maxLength:"64" doc:"Для kind=stamp."`
	InspectionKind string         `json:"inspection_kind,omitempty" maxLength:"64" doc:"Вид контроля клейма."`
	Scope          string         `json:"scope" maxLength:"256"`
	Limits         map[string]any `json:"limits,omitempty" doc:"Рамки полномочия: программы, типы изделий, тяжесть, виды решений."`
	OrderRef       string         `json:"order_ref,omitempty" maxLength:"256" doc:"Приказ (для клейма обязателен)."`
	ValidFrom      time.Time      `json:"valid_from"`
	ValidUntil     *time.Time     `json:"valid_until,omitempty"`
	DocumentID     string         `json:"document_id,omitempty" maxLength:"128" doc:"Документ выдачи с маршрутом подписей (AD-13, AD-43)."`
}

// RevokePolicy — отозвать роль, полномочие или клеймо (policy.role.unassigned,
// policy.authority.revoked, policy.stamp.revoked; защитное действие).
type RevokePolicy struct {
	platform.CommandHeader
	PersonID      string       `json:"person_id" maxLength:"64"`
	Kind          string       `json:"kind" enum:"role,authority,stamp"`
	SubjectID     string       `json:"subject_id" maxLength:"64" doc:"Роль, полномочие или клеймо."`
	Scope         string       `json:"scope,omitempty" maxLength:"256"`
	EffectiveFrom time.Time    `json:"effective_from"`
	Reason        AccessReason `json:"reason"`
}

// SetAssignment — назначить на пост в смене (access.assignment.set, FR-81).
type SetAssignment struct {
	platform.CommandHeader
	WorkplaceID        string `json:"workplace_id" maxLength:"128"`
	ShiftID            string `json:"shift_id" maxLength:"128"`
	PersonID           string `json:"person_id" maxLength:"64"`
	AssigneeRole       string `json:"assignee_role" enum:"performer,quality_inspector"`
	ApprovalDocumentID string `json:"approval_document_id,omitempty" maxLength:"128" doc:"Для контролёра — документ согласования начальника ОТК (PRD §11.18)."`
}

// ClearAssignment — снять с поста (access.assignment.cleared).
type ClearAssignment struct {
	platform.CommandHeader
	WorkplaceID string        `json:"workplace_id" maxLength:"128"`
	ShiftID     string        `json:"shift_id" maxLength:"128"`
	PersonID    string        `json:"person_id" maxLength:"64"`
	Reason      *AccessReason `json:"reason,omitempty"`
}

// GrantQualification — выдать квалификацию (access.qualification.granted, FR-80).
type GrantQualification struct {
	platform.CommandHeader
	QualificationID string     `json:"qualification_id" maxLength:"128"`
	Scope           string     `json:"scope,omitempty" maxLength:"256"`
	CertificateRef  string     `json:"certificate_ref,omitempty" maxLength:"256"`
	ValidFrom       time.Time  `json:"valid_from"`
	ValidUntil      *time.Time `json:"valid_until,omitempty"`
}

// RevokeQualification — отозвать квалификацию (access.qualification.revoked).
type RevokeQualification struct {
	platform.CommandHeader
	QualificationID string       `json:"qualification_id" maxLength:"128"`
	EffectiveFrom   time.Time    `json:"effective_from"`
	Reason          AccessReason `json:"reason"`
}

// SetAuditParameters — параметры аудита (policy.audit.parameters_set): только
// Аудитор ИБ, подпись уровня 2 (AD-8, AD-15).
type SetAuditParameters struct {
	platform.CommandHeader
	CriticalTypes          []string `json:"critical_types,omitempty"`
	CheckpointIntervalS    int      `json:"checkpoint_interval_s" minimum:"1"`
	CheckpointMaxGapS      int      `json:"checkpoint_max_gap_s" minimum:"1"`
	KeeperKeyFingerprint   string   `json:"keeper_key_fingerprint" maxLength:"128"`
	SecurityBusSubscribers []string `json:"security_bus_subscribers,omitempty"`
}

// AdmitWorkplace — допуск к рабочему месту (барьер 2, AD-15): СКУД в зоне ∧ роль
// в области ∧ квалификация на дату ∧ назначение в смене ∧ токен и PIN
// (access.workplace.admitted).
type AdmitWorkplace struct {
	platform.CommandHeader
	ShiftID        string   `json:"shift_id,omitempty" maxLength:"128"`
	ChecksEventIDs []string `json:"checks_event_ids,omitempty" doc:"Записи проверок допуска (проход СКУД, токен)."`
}

// ReleaseWorkplace — снять допуск (access.workplace.released).
type ReleaseWorkplace struct {
	platform.CommandHeader
	WorkplaceSessionID string `json:"workplace_session_id" format:"uuid"`
}

// ConfirmStep — исполнитель подтверждает шаг ТП у рабочего места (operator.step.confirmed, FR-137, уровень подписи 1).
type ConfirmStep struct {
	platform.CommandHeader
	ItemID         string `json:"item_id,omitempty" maxLength:"128"`
	OperationRunID string `json:"operation_run_id,omitempty" maxLength:"128"`
	StepKey        string `json:"step_key" maxLength:"128"`
	TPStep         string `json:"tp_step,omitempty" maxLength:"256" doc:"Шаг технологического процесса."`
}

// ReportDeviation — исполнитель сообщает об отклонении или подозрении на дефект (operator.deviation.reported).
type ReportDeviation struct {
	platform.CommandHeader
	ItemID         string `json:"item_id,omitempty" maxLength:"128"`
	OperationRunID string `json:"operation_run_id,omitempty" maxLength:"128"`
	ZoneID         string `json:"zone_id,omitempty" maxLength:"128"`
	Description    string `json:"description" minLength:"1" maxLength:"2000"`
}

// RequestInspection — исполнитель запрашивает контроль (operator.inspection.requested).
type RequestInspection struct {
	platform.CommandHeader
	ItemID         string `json:"item_id,omitempty" maxLength:"128"`
	OperationRunID string `json:"operation_run_id,omitempty" maxLength:"128"`
	StepKey        string `json:"step_key" maxLength:"128"`
}
