package ingest

import (
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/crypto"
)

// Формы приёма событий (FR-26…FR-41, FR-123, FR-140, FR-141; AD-7, AD-20, AD-41;
// кейс §4.3–4.7). Ворота системы: подпись источника → схема → пять случаев
// изменения контракта → идемпотентность → source_seq → карантин.

// IngestBatch — пачка подписанных конвертов DSSE от edge-агента, шлюза,
// терминала (AD-46: пачки edge-агента идут этой операцией). Конверт несёт
// исходное событие; журнал хранит исходные байты без изменений (AD-20).
type IngestBatch struct {
	SourceID  string                `json:"source_id" maxLength:"128" doc:"Источник пачки (устройство, шлюз, терминал); в прогоне — ‹run_id›/‹источник› (AD-38)."`
	Envelopes []crypto.DsseEnvelope `json:"envelopes" minItems:"1" maxItems:"1000" doc:"Подписанные конверты в порядке source_seq."`
	SentAt    *time.Time            `json:"sent_at,omitempty" doc:"Время отправки пачки по часам источника: по нему приём оценивает расхождение часов источника (FR-33); задержка досылки буфера сдвигом часов не считается."`
}

// IngestOutcome — итог по одному конверту пачки (FR-29…FR-31).
type IngestOutcome struct {
	Index            int      `json:"index" minimum:"0" doc:"Позиция конверта в пачке."`
	EventID          *string  `json:"event_id,omitempty"`
	Status           string   `json:"status" enum:"accepted,duplicate,quarantined,rejected" doc:"accepted — записан; duplicate — тот же payload уже есть (повтор не меняет показатели, AD-7); quarantined — в карантине с кодом; rejected — отказ без карантина."`
	Seq              *int64   `json:"seq,omitempty" doc:"Позиция записи в журнале (для accepted)."`
	Code             *string  `json:"code,omitempty" doc:"Код из contracts/errors.yaml (ingest.*): неизвестная версия, нет поля, подпись…"`
	QuarantineID     *string  `json:"quarantine_id,omitempty"`
	Flags            []string `json:"flags" doc:"Флаги аномалий (ingest.anomaly.flagged): future_timestamp, clock_skew, sequence_violation, unknown_enum_value, unknown_defect_type, late_write."`
	SignatureChecked bool     `json:"signature_checked" doc:"false — подпись не проверялась (профиль demo до эпика 05, заметка эпика 06)."`
}

// IngestResult — итог приёма пачки.
type IngestResult struct {
	Accepted    int             `json:"accepted" minimum:"0"`
	Duplicates  int             `json:"duplicates" minimum:"0"`
	Quarantined int             `json:"quarantined" minimum:"0"`
	Rejected    int             `json:"rejected" minimum:"0"`
	Items       []IngestOutcome `json:"items"`
}

// ManualEvent — одно событие ручного ввода или терминала участка (FR-137,
// FR-141): подписанный конверт; source_kind = ручной ввод (FR-140).
type ManualEvent struct {
	Envelope crypto.DsseEnvelope `json:"envelope"`
}

// ImportFile — импорт CSV / Excel (FR-141): такие же источники, как устройства,
// с проверкой входов, дублей и привязки.
type ImportFile struct {
	Format   string `json:"format" enum:"csv,xlsx"`
	FileName string `json:"file_name" maxLength:"256"`
	Content  string `json:"content" contentEncoding:"base64" doc:"Содержимое файла, base64; сохраняется в хранилище материалов по адресу H(байты)."`
	Mapping  string `json:"mapping" maxLength:"128" doc:"Шаблон сопоставления колонок с типом события."`
	DryRun   bool   `json:"dry_run,omitempty" doc:"Только проверить, не записывать."`
}

// ImportResult — итог импорта (ingest.import.completed).
type ImportResult struct {
	SourceID      string          `json:"source_id"`
	FileDigest    string          `json:"file_digest" doc:"Отпечаток файла streebog256:…"`
	RowsTotal     int             `json:"rows_total" minimum:"0"`
	RowsAccepted  int             `json:"rows_accepted" minimum:"0"`
	RowsDuplicate int             `json:"rows_duplicate" minimum:"0"`
	RowsRejected  int             `json:"rows_rejected" minimum:"0"`
	Rows          []IngestOutcome `json:"rows" doc:"Итог по строкам (index — номер строки)."`
}

// QuarantineEntry — сообщение в карантине (FR-30): содержимое хранится вне
// журнала в хранилище материалов, факт помещения — служебная запись (AD-2).
type QuarantineEntry struct {
	QuarantineID    string    `json:"quarantine_id"`
	SourceID        string    `json:"source_id"`
	SourceSeq       *int64    `json:"source_seq,omitempty"`
	EventID         *string   `json:"event_id,omitempty"`
	EventType       *string   `json:"event_type,omitempty"`
	Fingerprint     string    `json:"fingerprint" doc:"Отпечаток канонического payload."`
	MaterialAddress string    `json:"material_address" doc:"Адрес содержимого в хранилище материалов."`
	ProblemCode     string    `json:"problem_code" doc:"Код из contracts/errors.yaml."`
	Detail          *string   `json:"detail,omitempty"`
	QuarantinedAt   time.Time `json:"quarantined_at"`
	State           string    `json:"state" enum:"open,accepted,still_invalid,discarded"`
	Content         *string   `json:"content,omitempty" doc:"Исходное содержимое (только в чтении одной записи и при правах)."`
	BasisSeq        int64     `json:"basis_seq" doc:"seq, на котором построен ответ (для basis_seq команды переобработки, AD-39)."`
}

// QuarantineList — карантин сообщений.
type QuarantineList struct {
	Items      []QuarantineEntry `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// IngestReason — основание решения: код и текст.
type IngestReason struct {
	Code *string `json:"code,omitempty"`
	Text string  `json:"text" minLength:"1" maxLength:"2000"`
}

// ReprocessMessage — переобработать сообщение из карантина (ingest.message.reprocessed):
// после появления повышателя или исправления — принять, оставить или отбросить.
type ReprocessMessage struct {
	platform.CommandHeader
	Discard bool         `json:"discard,omitempty" doc:"Отбросить без повторной проверки."`
	Reason  IngestReason `json:"reason"`
}

// SourceView — источник событий: вид, ключ, непрерывность source_seq (AD-7, AD-9).
type SourceView struct {
	SourceID       string     `json:"source_id"`
	SourceKind     string     `json:"source_kind" doc:"Вид источника (ручной ввод, станок, датчик, камера, внешняя система, импорт)."`
	KeyRef         *string    `json:"key_ref,omitempty" doc:"Ключ устройства key_id@версия (акт ввода)."`
	State          string     `json:"state" enum:"active,disabled,loss_suspected,unknown_key"`
	LastSeq        *int64     `nullable:"true" json:"last_seq" doc:"Последний source_seq."`
	GapCount       int        `json:"gap_count" minimum:"0" doc:"Дыр в source_seq (номер нет ни в журнале, ни в карантине)."`
	LastReceivedAt *time.Time `nullable:"true" json:"last_received_at"`
	ClockSkewMs    *int64     `json:"clock_skew_ms,omitempty" doc:"Оценка расхождения часов источника (FR-33)."`
	Quarantined    int        `json:"quarantined" minimum:"0"`
	BasisSeq       int64      `json:"basis_seq" doc:"seq, на котором построен ответ (для basis_seq команд над источником, AD-39)."`
}

// SourceList — источники событий.
type SourceList struct {
	Items      []SourceView `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

// IngestMetrics — метрики приёма (FR-41): задержка, дубли, отказы, объём
// карантина, полнота; операционные, не проекции (AD-7).
type IngestMetrics struct {
	WindowFrom         time.Time `json:"window_from"`
	WindowTo           time.Time `json:"window_to"`
	Received           int64     `json:"received" minimum:"0"`
	Accepted           int64     `json:"accepted" minimum:"0"`
	Duplicates         int64     `json:"duplicates" minimum:"0"`
	Rejected           int64     `json:"rejected" minimum:"0"`
	Quarantined        int64     `json:"quarantined" minimum:"0"`
	QuarantineOpen     int64     `json:"quarantine_open" minimum:"0"`
	LatencyP50Ms       int64     `json:"latency_p50_ms" minimum:"0" doc:"Приём → запись, мс."`
	LatencyP95Ms       int64     `json:"latency_p95_ms" minimum:"0"`
	EventToScreenP95Ms int64     `json:"event_to_screen_p95_ms" minimum:"0" doc:"ant_event_to_sse_seconds, бюджет FR-2 ≤ 2 с."`
	CompletenessBP     int       `json:"completeness_bp" minimum:"0" maximum:"10000" doc:"Полнота: доля ожидаемых source_seq, которые есть, б. п."`
}

// QuarantineFilter — фильтр карантина.
type QuarantineFilter struct {
	SourceID string
	Code     string
	State    string
}
