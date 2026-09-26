package security

import (
	"encoding/json"
	"time"

	"ant/internal/application/platform"
)

// IntegrityStatus — индикатор целостности на столах (AD-46, FR-73): ant забирает
// последний подписанный отчёт верификатора у хранителя; индикатор помечен «по
// данным сервера» и желтеет сам, если свежего отчёта нет дольше двух интервалов.
type IntegrityStatus struct {
	Status          string     `json:"status" enum:"ok,violated,stale,unknown" doc:"ok — последний отчёт верификатора «цело»; stale — свежего отчёта нет дольше двух интервалов."`
	CheckedAt       *time.Time `json:"checked_at,omitempty" doc:"Время отчёта верификатора."`
	ReportRef       string     `json:"report_ref,omitempty" doc:"Отпечаток отчёта верификатора."`
	IntervalSeconds int        `json:"interval_seconds" minimum:"1" doc:"Интервал проверок верификатора."`
	ServerSide      bool       `json:"server_side" doc:"Всегда true — показывается «по данным сервера»."`
}

// CriticalAction — запись журнала критических действий CA-‹n› (AD-28, FR-77):
// действие, объект, было → стало, кто, полномочие и клеймо с ревизией политики,
// основание, ссылка на подписанную запись основного журнала.
type CriticalAction struct {
	CARef         string            `json:"ca_ref" doc:"CA-‹n› — позиция в цепочке критических действий."`
	CANo          int64             `json:"ca_no" minimum:"1"`
	CAGroup       string            `json:"ca_group" enum:"product_decision,nc_decision,cause,risk_scope,control_change,authority,protected_data,admin_security"`
	ActionType    string            `json:"action_type" doc:"Тип основной записи (семейство.сущность.действие)."`
	Object        platform.DrillRef `json:"object"`
	ObjectRef     string            `json:"object_ref" doc:"Поток объекта (stream_ref)."`
	Before        string            `json:"before" doc:"Было."`
	After         string            `json:"after" doc:"Стало."`
	ActorID       string            `json:"actor_id,omitempty" doc:"Кто (псевдоним)."`
	AttestedBy    string            `json:"attested_by,omitempty" doc:"Заверитель бумажной подписи (AD-43)."`
	AuthorityID   string            `json:"authority_id,omitempty" doc:"Полномочие."`
	StampID       string            `json:"stamp_id,omitempty" doc:"Цифровое клеймо (FR-145)."`
	PolicySeq     *int64            `json:"policy_seq,omitempty" doc:"Ревизия политики."`
	BasisEventIDs []string          `json:"basis_event_ids" doc:"Основания."`
	MainEventID   string            `json:"main_event_id" doc:"Подписанная запись основного журнала."`
	MainCommit    string            `json:"main_commit" doc:"Обязательство основной записи (AD-44)."`
	RecordedAt    time.Time         `json:"recorded_at"`
	Cancels       string            `json:"cancels,omitempty" doc:"Отменяет CA-… (отмена — только новой записью с причиной)."`
	CancelReason  string            `json:"cancel_reason,omitempty"`
	CancelledBy   string            `json:"cancelled_by,omitempty" doc:"Отменено записью CA-…."`
	// Интерфейс 6 (стол Аудитора ИБ): название действия, кто словами и подписанты с классом ключа.
	ActionName   string                 `json:"action_name,omitempty" doc:"Название действия словами (перечень критических действий AD-28 или каталог типов)."`
	ActorDisplay string                 `json:"actor_display,omitempty" doc:"Кто — имя из справочника сотрудников (псевдоним кейса §4.6)."`
	Signers      []CriticalActionSigner `json:"signers,omitempty" doc:"Подписанты основной записи с классом ключа (AD-10, AD-14): кто подписал решение, которое записано в CA."`
}

// CriticalActionSigner — подписант основной записи критического действия:
// класс происхождения подписи (AD-10) и класс хранения ключа человека (Д-72).
type CriticalActionSigner struct {
	SignerID   string `json:"signer_id" doc:"Подписант: псевдоним сотрудника или источник (ant — заверение сервером)."`
	Display    string `json:"display,omitempty" doc:"Подписант словами."`
	KeyClass   string `json:"key_class" enum:"personal,device,server_attested,scenario,paper,partner,genesis" doc:"Класс подписи (AD-10): personal — ключ человека, server_attested — заверено сервером, paper — бумага с заверением, scenario — ключ сценария."`
	KeyStorage string `json:"key_storage,omitempty" enum:"hardware_token,software_browser" doc:"Класс хранения ключа человека (AD-14, Д-72): физический ключ или ключ в браузере под PIN."`
	KeyID      string `json:"key_id,omitempty" doc:"Ключ key_id@версия."`
}

// CriticalActionList — журнал критических действий.
type CriticalActionList struct {
	Items      []CriticalAction `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// SecurityEvent — событие шины безопасности (AD-24, FR-118): запись семейства security.
type SecurityEvent struct {
	EventID    string             `json:"event_id"`
	EventType  string             `json:"event_type" doc:"Тип записи шины безопасности: семейство security и выдача прав (policy.role.*, policy.authority.*)."`
	Seq        int64              `json:"seq" minimum:"1"`
	OccurredAt time.Time          `json:"occurred_at"`
	Severity   string             `json:"severity" enum:"info,warning,alarm"`
	SourceID   string             `json:"source_id,omitempty"`
	Summary    string             `json:"summary" doc:"Краткое описание по-русски."`
	Object     *platform.DrillRef `json:"object,omitempty"`
	CARef      string             `json:"ca_ref,omitempty" doc:"Критическое действие, если событие его породило."`
	PersonID   string             `json:"person_id,omitempty" doc:"Сотрудник, к которому относится событие (вход, отказ, допуск, присутствие, выдача прав)."`
	// PersonDisplay — сотрудник словами (интерфейс 6).
	PersonDisplay string `json:"person_display,omitempty" doc:"Сотрудник словами."`
	// WorkplaceID — пост (допуск, присутствие по СКУД, эпик 37).
	WorkplaceID string `json:"workplace_id,omitempty" doc:"Пост (допуск, присутствие по СКУД)."`
	// Data — data записи (для подписчиков шины; в API не выдаётся).
	Data json.RawMessage `json:"-"`
}

// SecurityEventList — лента шины безопасности.
type SecurityEventList struct {
	Items      []SecurityEvent `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

// VerifierCheckRow — строка отчёта верификатора (AD-9): что проверено и статус.
type VerifierCheckRow struct {
	Check   string `json:"check" doc:"Проверка: цепочки, подписи, момент подписи, права подписанта, реакции, source_seq, документы, задержка записи, сборка, проекции."`
	Status  string `json:"status" enum:"intact,rejected,unverifiable" doc:"цело / отвергнуто / не проверяемо."`
	Count   int    `json:"count" minimum:"0"`
	Details string `json:"details,omitempty"`
	CARef   string `json:"ca_ref,omitempty"`
	// Findings — место нарушения или оговорки (интерфейс 6): запись, цепочка, код.
	Findings []VerifierFinding `json:"findings,omitempty" doc:"Находки проверки: где нарушение или оговорка (первые 20)."`
}

// VerifierFinding — находка верификатора (contracts/internal/verifier/report.v1.json):
// место нарушения — цепочка и номер записи, CA-‹n›, код и текст по-русски.
type VerifierFinding struct {
	Code    string `json:"code" doc:"Код находки (chain_link_mismatch, checkpoint_mismatch.link, source_seq_gap.pending, …)."`
	Detail  string `json:"detail" doc:"Что и где — по-русски."`
	Chain   string `json:"chain,omitempty" enum:"main,ca" doc:"Цепочка."`
	Seq     int64  `json:"seq,omitempty" minimum:"0" doc:"Номер записи в цепочке."`
	CARef   string `json:"ca_ref,omitempty" doc:"Критическое действие CA-‹n›."`
	EventID string `json:"event_id,omitempty" doc:"Запись журнала (event_id), если известна ant: переход к записи."`
	Status  string `json:"status,omitempty" enum:"rejected,unverifiable,note" doc:"rejected — нарушение, unverifiable — не проверяемо, note — пояснение при «цело»."`
}

// VerifierReportSummary — отчёт верификатора в списке (AD-46).
type VerifierReportSummary struct {
	ReportDigest   string    `json:"report_digest" doc:"Отпечаток подписанного отчёта."`
	Verdict        string    `json:"verdict" enum:"intact,intact_with_reservations,violated" doc:"«цело» только при нуле «не проверяемо», иначе «цело с оговорками»."`
	CheckedUpToSeq int64     `json:"checked_up_to_seq" minimum:"0"`
	CheckedAt      time.Time `json:"checked_at"`
	VerifierBuild  string    `json:"verifier_build,omitempty" doc:"Хеш бинарника верификатора."`
	ServerSide     bool      `json:"server_side" doc:"Всегда true: получено ant у хранителя — «по данным сервера»."`
	// Headline — первая находка «нарушено» / «не проверяемо» для строки списка (интерфейс 6).
	Headline string `json:"headline,omitempty" doc:"Главная находка словами: где нарушение (для строки списка)."`
}

// VerifierReportList — отчёты верификатора.
type VerifierReportList struct {
	Items      []VerifierReportSummary `json:"items"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

// VerifierReport — отчёт верификатора целиком (contracts/internal/verifier/report.v1.json):
// классы подписей различаются; реестр бумажных решений для сверки с оригиналами.
type VerifierReport struct {
	Summary          VerifierReportSummary `json:"summary"`
	Checks           []VerifierCheckRow    `json:"checks"`
	SignatureClasses map[string]int        `json:"signature_classes" doc:"Число подписей по классам происхождения (personal, paper, partner, scenario, genesis, server_attested)."`
	PaperDecisions   []platform.DrillRef   `json:"paper_decisions" doc:"Реестр бумажных решений для сверки с оригиналами (AD-9)."`
	VirtualTime      bool                  `json:"virtual_time" doc:"Прогон в режиме scenario: временные проверки — «виртуальное время»."`
	SignedBy         string                `json:"signed_by" doc:"Ключ верификатора."`
}
