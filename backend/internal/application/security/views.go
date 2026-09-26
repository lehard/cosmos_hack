package security

import (
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
}

// CriticalActionList — журнал критических действий.
type CriticalActionList struct {
	Items      []CriticalAction `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// SecurityEvent — событие шины безопасности (AD-24, FR-118): запись семейства security.
type SecurityEvent struct {
	EventID    string             `json:"event_id"`
	EventType  string             `json:"event_type" doc:"Тип записи семейства security."`
	Seq        int64              `json:"seq" minimum:"1"`
	OccurredAt time.Time          `json:"occurred_at"`
	Severity   string             `json:"severity" enum:"info,warning,alarm"`
	SourceID   string             `json:"source_id,omitempty"`
	Summary    string             `json:"summary" doc:"Краткое описание по-русски."`
	Object     *platform.DrillRef `json:"object,omitempty"`
	CARef      string             `json:"ca_ref,omitempty" doc:"Критическое действие, если событие его породило."`
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
}

// VerifierReportSummary — отчёт верификатора в списке (AD-46).
type VerifierReportSummary struct {
	ReportDigest   string    `json:"report_digest" doc:"Отпечаток подписанного отчёта."`
	Verdict        string    `json:"verdict" enum:"intact,intact_with_reservations,violated" doc:"«цело» только при нуле «не проверяемо», иначе «цело с оговорками»."`
	CheckedUpToSeq int64     `json:"checked_up_to_seq" minimum:"0"`
	CheckedAt      time.Time `json:"checked_at"`
	VerifierBuild  string    `json:"verifier_build,omitempty" doc:"Хеш бинарника верификатора."`
	ServerSide     bool      `json:"server_side" doc:"Всегда true: получено ant у хранителя — «по данным сервера»."`
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
