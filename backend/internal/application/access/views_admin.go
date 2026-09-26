package access

import "time"

// Формы администрирования доступа (PRD §3a «Администратор», «Аудитор ИБ»;
// FR-78…FR-85, FR-145; AD-11, AD-15). Сотрудники — только псевдонимы (кейс §4.6).

// AccessReason — основание: код и текст (defs.reason).
type AccessReason struct {
	Code string `json:"code,omitempty" maxLength:"64" doc:"Машинный код основания."`
	Text string `json:"text" maxLength:"2000" doc:"Основание по-русски."`
}

// AccessRoleGrant — роль сотрудника в области со сроком.
type AccessRoleGrant struct {
	RoleID     string     `json:"role_id"`
	Scope      string     `json:"scope" doc:"Область: здание → цех → участок → рабочее место."`
	ValidFrom  time.Time  `json:"valid_from"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
}

// AccessPerson — сотрудник: псевдоним, учётная запись, роли (FR-78).
type AccessPerson struct {
	PersonID      string            `json:"person_id" doc:"Псевдоним сотрудника."`
	DisplayName   string            `json:"display_name"`
	OrgUnit       string            `json:"org_unit,omitempty"`
	AccountStatus string            `json:"account_status" enum:"none,pending,active,blocked" doc:"Учётная запись: нет, ждёт активации, действует, заблокирована (FR-128)."`
	Login         string            `json:"login,omitempty"`
	Roles         []AccessRoleGrant `json:"roles"`
	PolicySeq     int64             `json:"policy_seq" doc:"Версия политики, на которой построен ответ (AD-39)."`
}

// AccessPersonList — сотрудники.
type AccessPersonList struct {
	Items      []AccessPerson `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// AccessRole — роль политики с действиями (x-ant-action id с `*`) и наследованием.
type AccessRole struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	CaseRole bool     `json:"case_role" doc:"Одна из пяти ролей кейса."`
	Inherits []string `json:"inherits"`
	Actions  []string `json:"actions" doc:"Действия ‹модуль›.‹объект›.‹действие›; * — любой сегмент."`
}

// AccessAuthority — полномочие (особое право поверх роли, FR-50, FR-78).
type AccessAuthority struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// AccessRoleList — роли и полномочия действующей политики.
type AccessRoleList struct {
	Items       []AccessRole      `json:"items"`
	Authorities []AccessAuthority `json:"authorities"`
	StampKinds  []string          `json:"stamp_kinds" doc:"Виды контроля для цифровых клейм (FR-145)."`
	PolicySeq   int64             `json:"policy_seq"`
}

// AccessGrantEntry — строка истории выдачи прав (журнал выдачи прав у Аудитора ИБ, AD-15).
type AccessGrantEntry struct {
	EventID           string    `json:"event_id"`
	Seq               int64     `json:"seq"`
	Kind              string    `json:"kind" enum:"role,authority,stamp,qualification,account,audit,sod_rule" doc:"Что выдано или отозвано."`
	Action            string    `json:"action" enum:"granted,revoked,set" doc:"Выдано, отозвано, установлено."`
	PersonID          string    `json:"person_id,omitempty"`
	SubjectID         string    `json:"subject_id" doc:"Роль, полномочие, клеймо, квалификация."`
	Scope             string    `json:"scope,omitempty"`
	At                time.Time `json:"at"`
	By                string    `json:"by" doc:"Кто выдал (псевдоним)."`
	SecondSignatureBy string    `json:"second_signature_by,omitempty" doc:"Вторая подпись независимой стороны (AD-11)."`
	DocumentID        string    `json:"document_id,omitempty" doc:"Документ выдачи с маршрутом подписей."`
	CARef             string    `json:"ca_ref,omitempty" doc:"Критическое действие CA-‹n› (AD-28)."`
}

// AccessGrantHistory — история выдачи прав.
type AccessGrantHistory struct {
	Items      []AccessGrantEntry `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// AccessStamp — цифровое клеймо контролёра (FR-145): одно на вид контроля, по приказу, с областью и сроком.
type AccessStamp struct {
	StampID        string     `json:"stamp_id"`
	PersonID       string     `json:"person_id"`
	InspectionKind string     `json:"inspection_kind" doc:"Вид контроля (stamp_kinds политики)."`
	Scope          string     `json:"scope"`
	OrderRef       string     `json:"order_ref" doc:"Приказ."`
	ValidFrom      time.Time  `json:"valid_from"`
	ValidUntil     *time.Time `json:"valid_until,omitempty"`
	Status         string     `json:"status" enum:"active,expired,revoked"`
}

// AccessStampList — клейма.
type AccessStampList struct {
	Items []AccessStamp `json:"items"`
}

// AccessAssignment — назначение на пост в смене (FR-81): исполнителей назначает
// мастер, контролёра — документом «запрос мастера → согласование начальника ОТК» (PRD §11.18).
type AccessAssignment struct {
	WorkplaceID        string `json:"workplace_id"`
	ShiftID            string `json:"shift_id"`
	PersonID           string `json:"person_id"`
	AssigneeRole       string `json:"assignee_role" enum:"performer,quality_inspector"`
	ApprovalDocumentID string `json:"approval_document_id,omitempty" doc:"Документ согласования начальника ОТК — для контролёра обязателен."`
	Admitted           bool   `json:"admitted" doc:"Допуск к рабочему месту действует (барьер 2)."`
	QualificationOK    bool   `json:"qualification_ok" doc:"Квалификация действует на дату смены (FR-80)."`
}

// AccessAssignmentList — назначения смены.
type AccessAssignmentList struct {
	Items    []AccessAssignment `json:"items"`
	BasisSeq int64              `json:"basis_seq"`
}

// WorkplaceCandidate — кандидат на пост в смене (FR-80, FR-81; интерфейс 6,
// «Смена → Назначить на пост»): сотрудник с ролью исполнителя или контролёра в
// области поста и вердикт квалификации на дату смены — те же условия, что
// проверяет гард access.assignment.set (GuardAssignment).
type WorkplaceCandidate struct {
	PersonID string `json:"person_id"`
	Display  string `json:"display" doc:"Сотрудник словами (псевдоним кейса §4.6)."`
	Role     string `json:"role" enum:"performer,quality_inspector" doc:"Роль назначения (assignee_role команды access.assignment.set)."`
	// QualificationVerdict — ok — действует; expiring — истекает в 30 дней; expired — истекла; missing — нет в области поста.
	QualificationVerdict string     `json:"qualification_verdict" enum:"ok,expiring,expired,missing" doc:"Квалификация на дату смены: ok — действует, expiring — истекает в 30 дней, expired — истекла, missing — нет в области поста. Контролёру квалификация не нужна (ok) — нужен документ согласования."`
	QualificationID      string     `json:"qualification_id,omitempty" doc:"Квалификация, по которой вердикт."`
	ValidUntil           *time.Time `json:"valid_until,omitempty" doc:"Срок квалификации."`
	Why                  string     `json:"why" doc:"Почему так — по-русски: срок, область, что нужно для назначения."`
	// Allowed — гард назначения пропустит (исполнитель с действующей квалификацией; контролёр — после согласования).
	Allowed       bool `json:"allowed" doc:"Назначение пройдёт гард access.assignment.set (контролёру — после документа согласования начальника ОТК)."`
	NeedsApproval bool `json:"needs_approval,omitempty" doc:"Контролёр: назначение — по документу «запрос мастера → согласование начальника ОТК» (PRD §11.18)."`
	AssignedHere  bool `json:"assigned_here,omitempty" doc:"Уже назначен на этот пост в этой смене."`
	// AssignedElsewhere — пост, на который сотрудник уже назначен в этой смене.
	AssignedElsewhere string `json:"assigned_elsewhere,omitempty" doc:"Уже назначен в этой смене на другой пост (workplace_id)."`
}

// WorkplaceCandidateList — кандидаты на пост в смене: сначала допустимые.
type WorkplaceCandidateList struct {
	WorkplaceID string               `json:"workplace_id"`
	ShiftID     string               `json:"shift_id,omitempty" doc:"Смена, на дату которой проверены квалификации; пусто — текущая."`
	Items       []WorkplaceCandidate `json:"items"`
	BasisSeq    int64                `json:"basis_seq"`
}

// AccessQualification — квалификация или аттестация со сроком (FR-80).
type AccessQualification struct {
	PersonID        string     `json:"person_id"`
	QualificationID string     `json:"qualification_id"`
	Scope           string     `json:"scope,omitempty"`
	CertificateRef  string     `json:"certificate_ref,omitempty"`
	ValidFrom       time.Time  `json:"valid_from"`
	ValidUntil      *time.Time `json:"valid_until,omitempty"`
	Status          string     `json:"status" enum:"valid,expiring,expired,revoked"`
}

// AccessQualificationList — квалификации.
type AccessQualificationList struct {
	Items []AccessQualification `json:"items"`
}

// AccessAuditParameters — параметры аудита policy.audit.* (AD-8, AD-15):
// принадлежат Аудитору ИБ, это данные журнала, а не конфигурация.
type AccessAuditParameters struct {
	CriticalTypes          []string   `json:"critical_types" doc:"Перечень критических типов записей."`
	CheckpointIntervalS    int        `json:"checkpoint_interval_s" minimum:"1" doc:"Интервал контрольных точек, с."`
	CheckpointMaxGapS      int        `json:"checkpoint_max_gap_s" minimum:"1" doc:"Предельный разрыв контрольных точек, с."`
	KeeperKeyFingerprint   string     `json:"keeper_key_fingerprint" doc:"Отпечаток ключа хранителя (streebog256:…)."`
	SecurityBusSubscribers []string   `json:"security_bus_subscribers" doc:"Подписчики шины безопасности (AD-24)."`
	SetBy                  string     `json:"set_by,omitempty"`
	SetAt                  *time.Time `json:"set_at,omitempty"`
	PolicySeq              int64      `json:"policy_seq"`
}
