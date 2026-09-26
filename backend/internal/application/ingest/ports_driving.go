package ingest

import (
	"context"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// Queries — ведущий порт чтения модуля ingest (AD-36): карантин и метрики
// приёма для стола администратора (FR-30, FR-41).
type Queries interface {
	// Quarantine — записи карантина по фильтру (ingest.quarantine.list).
	Quarantine(ctx context.Context, f QuarantineFilter) ([]QuarantineView, error)
	// QuarantineItem — одна запись карантина (ingest.quarantine.read).
	QuarantineItem(ctx context.Context, id string) (QuarantineView, error)
	// Stats — метрики приёма: задержка, дубли, отказы, карантин, полнота (ingest.stats.read, FR-41).
	Stats(ctx context.Context) (Stats, error)
}

// Commands — ведущий порт команд модуля ingest (AD-36, AD-39).
type Commands interface {
	// Ingest — приём одного сообщения: сырые байты DSSE-конверта события (или
	// самого события в профиле demo без подписи). Операция ingest.event.submit.
	Ingest(ctx context.Context, raw []byte) (Result, error)
	// IngestBatch — пачка edge-агента (FR-39): {source_id, sent_at, messages[]};
	// каждое сообщение обрабатывается отдельно. Операция ingest.batch.submit.
	IngestBatch(ctx context.Context, raw []byte) (BatchResult, error)
	// Reprocess — переобработка записи карантина администратором (FR-30):
	// ingest.message.reprocessed. Операция ingest.quarantine.reprocess.
	Reprocess(ctx context.Context, cmd platform.Command[ReprocessInput]) (ReprocessResult, error)
	// SubmitManual — ручной ввод как полноправный источник (FR-141): форма
	// стола или терминал. Операция ingest.manual.submit.
	SubmitManual(ctx context.Context, cmd platform.Command[ManualInput]) (Result, error)
	// ImportCSV — импорт журнала из CSV через адаптер источника (FR-141).
	// Операция ingest.import.submit.
	ImportCSV(ctx context.Context, cmd platform.Command[ImportInput]) (ImportResult, error)
}

// Outcome — итог приёма сообщения.
type Outcome string

// Итоги приёма.
const (
	OutcomeAccepted         Outcome = "accepted"
	OutcomeAcceptedWithFlag Outcome = "accepted_with_flag"
	OutcomeDuplicate        Outcome = "duplicate"
	OutcomeConflict         Outcome = "conflict"
	OutcomeQuarantined      Outcome = "quarantined"
)

// SignatureNotVerified — пометка принятого без проверки подписи (профиль demo,
// до эпика 05): сообщение принято, но подлинность источника не подтверждена.
const SignatureNotVerified = "подпись не проверялась"

// FlagView — флаг, записанный вместе с фактом (ingest.anomaly.flagged).
type FlagView struct {
	Flag     string `json:"flag"`
	Field    string `json:"field,omitempty"`
	RawValue string `json:"raw_value,omitempty"`
	SkewMS   int64  `json:"skew_ms,omitempty"`
}

// Result — ответ приёма на одно сообщение; повтор получает тот же ответ с
// Replayed = true (FR-31: «отправителю — такой же ответ, как в первый раз»).
type Result struct {
	Outcome Outcome `json:"outcome"`
	// Status — HTTP-статус по contracts/errors.yaml (202 — принято).
	Status int           `json:"status"`
	Code   errcodes.Code `json:"code,omitempty"`
	Field  string        `json:"field,omitempty"`
	Value  string        `json:"value,omitempty"`
	Detail string        `json:"detail,omitempty"`

	SourceID  string `json:"source_id,omitempty"`
	EventID   string `json:"event_id,omitempty"`
	EventType string `json:"event_type,omitempty"`
	SourceSeq int64  `json:"source_seq,omitempty"`
	// Seq — позиция факта в журнале (у дубля — прежняя).
	Seq       int64      `json:"seq,omitempty"`
	ItemID    string     `json:"item_id,omitempty"`
	Stream    string     `json:"stream,omitempty"`
	Partition int        `json:"partition"`
	Flags     []FlagView `json:"flags,omitempty"`
	// SignatureVerified — подпись проверена; иначе SignatureNote = «подпись не проверялась».
	SignatureVerified bool   `json:"signature_verified"`
	SignatureNote     string `json:"signature_note,omitempty"`
	QuarantineID      string `json:"quarantine_id,omitempty"`
	Replayed          bool   `json:"replayed,omitempty"`
}

// BatchResult — ответ на пачку: по элементу на сообщение в том же порядке.
type BatchResult struct {
	SourceID string   `json:"source_id,omitempty"`
	Results  []Result `json:"results"`
}

// ReprocessInput — переобработка записи карантина.
type ReprocessInput struct {
	QuarantineID string
	// Discard — не переобрабатывать, а закрыть запись («отброшено» с причиной).
	Discard bool
}

// ReprocessResult — итог переобработки (ingest.message.reprocessed).
type ReprocessResult struct {
	Receipt platform.Receipt `json:"receipt"`
	// Outcome — accepted | still_invalid | discarded.
	Outcome string `json:"outcome"`
	Result  Result `json:"result"`
}

// ManualInput — ручной ввод факта (FR-141, FR-140): тип и данные события,
// изделие или носитель; источник — терминал или форма стола.
type ManualInput struct {
	// SourceID — терминал или форма (`terminal-weld-2`, `desk:master`); пусто — `manual:‹сотрудник›`.
	SourceID      string
	EventType     string
	SchemaVersion int
	// OccurredAt — когда произошло по словам человека; пусто — доменное «сейчас».
	OccurredAt *time.Time
	ItemID     string
	// CarrierType, CarrierValue — носитель, если изделие указано меткой.
	CarrierType  string
	CarrierValue string
	// Data — поле data события (JSON-объект).
	Data map[string]any
	// Restored — внесено с бумаги задним числом («восстановлено с бумаги»).
	Restored bool
}

// ImportInput — импорт журнала CSV (FR-141).
type ImportInput struct {
	// SourceID — адаптер импорта (`import:weld-journal`).
	SourceID string
	FileName string
	Content  []byte
	// Defaults — значения по умолчанию для столбцов (event_type, schema_version…).
	Defaults map[string]string
}

// ImportRow — итог по строке импорта.
type ImportRow struct {
	Line   int    `json:"line"`
	Result Result `json:"result"`
}

// ImportResult — итог импорта (ingest.import.completed).
type ImportResult struct {
	Receipt    platform.Receipt `json:"receipt"`
	FileDigest string           `json:"file_digest"`
	Total      int              `json:"rows_total"`
	Accepted   int              `json:"rows_accepted"`
	Duplicate  int              `json:"rows_duplicate"`
	Rejected   int              `json:"rows_rejected"`
	Rows       []ImportRow      `json:"rows"`
}

// QuarantineView — запись карантина для стола администратора.
type QuarantineView struct {
	ID              string           `json:"id"`
	JournalSeq      int64            `json:"journal_seq"`
	SourceID        string           `json:"source_id"`
	EventID         string           `json:"event_id,omitempty"`
	EventType       string           `json:"event_type,omitempty"`
	SourceSeq       int64            `json:"source_seq,omitempty"`
	Code            errcodes.Code    `json:"code"`
	Title           string           `json:"title"`
	Field           string           `json:"field,omitempty"`
	Detail          string           `json:"detail,omitempty"`
	Fingerprint     string           `json:"fingerprint"`
	MaterialAddress string           `json:"material_address"`
	Status          QuarantineStatus `json:"status"`
	ReceivedAt      time.Time        `json:"received_at"`
	ResolvedBy      string           `json:"resolved_by,omitempty"`
}

// SourceStats — полнота источника (FR-41).
type SourceStats struct {
	SourceID       string `json:"source_id"`
	HighWater      int64  `json:"high_water"`
	Missing        int64  `json:"missing"`
	CompletenessBP int64  `json:"completeness_bp"`
	OpenGaps       int    `json:"open_gaps"`
}

// Stats — метрики приёма для стола администратора и /metrics (FR-41, AD-7):
// операционные счётчики, не проекции журнала.
type Stats struct {
	Accepted         int64            `json:"accepted"`
	AcceptedWithFlag int64            `json:"accepted_with_flag"`
	Duplicates       int64            `json:"duplicates"`
	Conflicts        int64            `json:"conflicts"`
	Quarantined      int64            `json:"quarantined"`
	Unverified       int64            `json:"unverified"`
	Rejects          map[string]int64 `json:"rejects"`
	QuarantineOpen   int64            `json:"quarantine_open"`
	// LatencyP50MS, LatencyMaxMS — задержка обработки приёма, мс.
	LatencyP50MS int64 `json:"latency_p50_ms"`
	LatencyMaxMS int64 `json:"latency_max_ms"`
	// DeliveryDelayMaxMS — наибольшая задержка доставки (received_at − occurred_at), мс.
	DeliveryDelayMaxMS int64         `json:"delivery_delay_max_ms"`
	Sources            []SourceStats `json:"sources"`
}

// Unimplemented — заглушка портов ingest: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализацию fixtures, чтобы новые
// операции контракта не ломали сборку (реализация переопределяет метод).
type Unimplemented struct{}

func (Unimplemented) Quarantine(context.Context, QuarantineFilter) ([]QuarantineView, error) {
	return nil, platform.NotImplemented("ingest.quarantine.list")
}

func (Unimplemented) QuarantineItem(context.Context, string) (QuarantineView, error) {
	return QuarantineView{}, platform.NotImplemented("ingest.quarantine.read")
}

func (Unimplemented) Stats(context.Context) (Stats, error) {
	return Stats{}, platform.NotImplemented("ingest.stats.read")
}

func (Unimplemented) Ingest(context.Context, []byte) (Result, error) {
	return Result{}, platform.NotImplemented("ingest.event.submit")
}

func (Unimplemented) IngestBatch(context.Context, []byte) (BatchResult, error) {
	return BatchResult{}, platform.NotImplemented("ingest.batch.submit")
}

func (Unimplemented) Reprocess(context.Context, platform.Command[ReprocessInput]) (ReprocessResult, error) {
	return ReprocessResult{}, platform.NotImplemented("ingest.quarantine.reprocess")
}

func (Unimplemented) SubmitManual(context.Context, platform.Command[ManualInput]) (Result, error) {
	return Result{}, platform.NotImplemented("ingest.manual.submit")
}

func (Unimplemented) ImportCSV(context.Context, platform.Command[ImportInput]) (ImportResult, error) {
	return ImportResult{}, platform.NotImplemented("ingest.import.submit")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
