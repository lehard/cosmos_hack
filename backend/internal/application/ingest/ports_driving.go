package ingest

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля ingest (AD-36).
type Queries interface {
	// Quarantine — карантин сообщений (ingest.quarantine.list, FR-30).
	Quarantine(ctx context.Context, f QuarantineFilter, p platform.Page) (QuarantineList, error)
	// QuarantineEntry — сообщение в карантине с содержимым (ingest.quarantine.read).
	QuarantineEntry(ctx context.Context, quarantineID string) (QuarantineEntry, error)
	// Sources — источники событий (ingest.source.list, AD-7).
	Sources(ctx context.Context, p platform.Page) (SourceList, error)
	// Metrics — метрики приёма (ingest.metrics.read, FR-41).
	Metrics(ctx context.Context) (IngestMetrics, error)
}

// Commands — ведущий порт команд модуля ingest: приём фактов и решения по карантину.
type Commands interface {
	// SubmitBatch — приём пачки подписанных конвертов (ingest.batch.submit).
	SubmitBatch(ctx context.Context, in IngestBatch) (IngestResult, error)
	// SubmitEvent — одно событие ручного ввода или терминала (ingest.event.submit).
	SubmitEvent(ctx context.Context, in ManualEvent) (IngestOutcome, error)
	// Import — импорт CSV / Excel (ingest.import.submit, FR-141).
	Import(ctx context.Context, in ImportFile) (ImportResult, error)
	// Reprocess — переобработка из карантина (ingest.message.reprocess).
	Reprocess(ctx context.Context, quarantineID string, in ReprocessMessage) (platform.Receipt, error)
}

// Unimplemented — заглушка портов ingest: каждая операция отвечает 501.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Quarantine(context.Context, QuarantineFilter, platform.Page) (QuarantineList, error) {
	return QuarantineList{}, ni("ingest.quarantine.list")
}
func (Unimplemented) QuarantineEntry(context.Context, string) (QuarantineEntry, error) {
	return QuarantineEntry{}, ni("ingest.quarantine.read")
}
func (Unimplemented) Sources(context.Context, platform.Page) (SourceList, error) {
	return SourceList{}, ni("ingest.source.list")
}
func (Unimplemented) Metrics(context.Context) (IngestMetrics, error) {
	return IngestMetrics{}, ni("ingest.metrics.read")
}
func (Unimplemented) SubmitBatch(context.Context, IngestBatch) (IngestResult, error) {
	return IngestResult{}, ni("ingest.batch.submit")
}
func (Unimplemented) SubmitEvent(context.Context, ManualEvent) (IngestOutcome, error) {
	return IngestOutcome{}, ni("ingest.event.submit")
}
func (Unimplemented) Import(context.Context, ImportFile) (ImportResult, error) {
	return ImportResult{}, ni("ingest.import.submit")
}
func (Unimplemented) Reprocess(context.Context, string, ReprocessMessage) (platform.Receipt, error) {
	return platform.Receipt{}, ni("ingest.message.reprocess")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
