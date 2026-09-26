package ingest

import (
	"context"

	app "ant/internal/application/ingest"
	"ant/internal/application/platform"
)

// Adapter — реализация fixtures ведущих портов модуля ingest (AD-36):
// карантин, источники и метрики приёма — из мира заготовок. Приём на
// заготовках журнал не пишет: события сценария уже есть в мире заготовок,
// поэтому каждое присланное сообщение — «повтор» (duplicate), без записи.
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Quarantine — карантин сообщений (ingest.quarantine.list) с фильтром.
func (Adapter) Quarantine(ctx context.Context, f app.QuarantineFilter, _ platform.Page) (app.QuarantineList, error) {
	v, err := respond[app.QuarantineList](ctx, "ingest.quarantine.list",
		map[string]string{"source_id": f.SourceID, "code": f.Code, "state": f.State}, nil)
	if err != nil {
		return v, err
	}
	out := v.Items[:0]
	for _, x := range v.Items {
		if (f.SourceID != "" && x.SourceID != f.SourceID) || (f.Code != "" && x.ProblemCode != f.Code) || (f.State != "" && x.State != f.State) {
			continue
		}
		out = append(out, x)
	}
	v.Items = out
	return v, nil
}

// QuarantineEntry — сообщение в карантине (ingest.quarantine.read).
func (Adapter) QuarantineEntry(ctx context.Context, quarantineID string) (app.QuarantineEntry, error) {
	return respond[app.QuarantineEntry](ctx, "ingest.quarantine.read", map[string]string{"quarantine_id": quarantineID}, nil)
}

// Sources — источники событий (ingest.source.list).
func (Adapter) Sources(ctx context.Context, _ platform.Page) (app.SourceList, error) {
	return respond[app.SourceList](ctx, "ingest.source.list", nil, nil)
}

// Metrics — метрики приёма (ingest.metrics.read).
func (Adapter) Metrics(ctx context.Context) (app.IngestMetrics, error) {
	return respond[app.IngestMetrics](ctx, "ingest.metrics.read", nil, nil)
}

// SubmitBatch — пачка событий (ingest.batch.submit): на заготовках каждый
// конверт — повтор уже известного миру события, в журнал ничего не пишется.
func (Adapter) SubmitBatch(_ context.Context, in app.IngestBatch) (app.IngestResult, error) {
	out := app.IngestResult{Duplicates: len(in.Envelopes), Items: make([]app.IngestOutcome, len(in.Envelopes))}
	for i := range in.Envelopes {
		out.Items[i] = duplicate(i)
	}
	return out, nil
}

// SubmitEvent — одно событие ручного ввода (ingest.event.submit): повтор, без записи.
func (Adapter) SubmitEvent(context.Context, app.ManualEvent) (app.IngestOutcome, error) {
	return duplicate(0), nil
}

// Import — импорт CSV / Excel (ingest.import.submit): на заготовках строки
// не разбираются и не записываются.
func (Adapter) Import(_ context.Context, in app.ImportFile) (app.ImportResult, error) {
	return app.ImportResult{SourceID: "import/" + in.FileName, FileDigest: "streebog256:" + zeros, Rows: []app.IngestOutcome{}}, nil
}

// Reprocess — переобработать сообщение из карантина (ingest.message.reprocess).
func (Adapter) Reprocess(ctx context.Context, quarantineID string, in app.ReprocessMessage) (platform.Receipt, error) {
	return decide(ctx, "ingest.message.reprocess", "quarantine", quarantineID, in.CommandMeta())
}

const zeros = "0000000000000000000000000000000000000000000000000000000000000000"

func duplicate(i int) app.IngestOutcome {
	return app.IngestOutcome{Index: i, Status: "duplicate", Flags: []string{}}
}
