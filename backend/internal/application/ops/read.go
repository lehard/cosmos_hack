package ops

import (
	"context"
	"encoding/json"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/ops"
)

// readPage — страница чтения журнала при обходе по типу записи.
const readPage = 1000

// records — все записи типа t основной цепочки (индекс по event_type),
// разобранные кодеком; stream — фильтр потока (пусто — все).
func records(ctx context.Context, j appjournal.JournalStore, c *engineapp.Codec, t catalog.Type, stream string) ([]engineapp.Decoded, error) {
	var out []engineapp.Decoded
	after := int64(0)
	for {
		page, err := j.Read(ctx, appjournal.ReadQuery{EventType: string(t), Stream: stream, AfterSeq: after, Limit: readPage})
		if err != nil {
			return nil, err
		}
		for _, e := range page {
			d, err := c.Decode(ctx, e)
			if err != nil {
				return nil, err
			}
			out = append(out, d)
		}
		if len(page) < readPage {
			return out, nil
		}
		after = int64(page[len(page)-1].Seq)
	}
}

// last — последняя запись типа t в потоке stream (nil — нет).
func last(ctx context.Context, j appjournal.JournalStore, c *engineapp.Codec, t catalog.Type, stream string) (*engineapp.Decoded, error) {
	page, err := j.Read(ctx, appjournal.ReadQuery{EventType: string(t), Stream: stream, Backward: true, Limit: 1})
	if err != nil || len(page) == 0 {
		return nil, err
	}
	d, err := c.Decode(ctx, page[0])
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// failuresAndRetries — сбои обработки и повторы из журнала (AD-45).
func failuresAndRetries(ctx context.Context, j appjournal.JournalStore, c *engineapp.Codec) ([]dom.Failure, []dom.Retry, error) {
	fs, err := records(ctx, j, c, catalog.OpsProcessingFailed, "")
	if err != nil {
		return nil, nil, err
	}
	rs, err := records(ctx, j, c, catalog.OpsProcessingRetried, "")
	if err != nil {
		return nil, nil, err
	}
	failures := make([]dom.Failure, 0, len(fs))
	for _, d := range fs {
		var x ev.OpsProcessingFailedV1
		if err := json.Unmarshal(d.Record.Data, &x); err != nil {
			return nil, nil, err
		}
		failures = append(failures, dom.Failure{EventID: d.Record.EventID, Seq: d.Record.Seq, ItemID: d.Record.ItemID,
			Consumer: x.Consumer, FailedSeq: int64(x.FailedSeq), Error: x.Error, At: d.Record.RecordedAt})
	}
	retries := make([]dom.Retry, 0, len(rs))
	for _, d := range rs {
		var x ev.OpsProcessingRetriedV1
		if err := json.Unmarshal(d.Record.Data, &x); err != nil {
			return nil, nil, err
		}
		retries = append(retries, dom.Retry{EventID: d.Record.EventID, Seq: d.Record.Seq, ItemID: d.Record.ItemID, FailureEventID: string(x.FailureEventID)})
	}
	return failures, retries, nil
}

// integrationRecords — записи ops.integration.degraded (последняя по системе решает).
func integrationRecords(ctx context.Context, j appjournal.JournalStore, c *engineapp.Codec) ([]dom.IntegrationRecord, error) {
	ds, err := records(ctx, j, c, catalog.OpsIntegrationDegraded, "")
	if err != nil {
		return nil, err
	}
	out := make([]dom.IntegrationRecord, 0, len(ds))
	for _, d := range ds {
		r, err := integrationRecord(d)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func integrationRecord(d engineapp.Decoded) (dom.IntegrationRecord, error) {
	var x ev.OpsIntegrationDegradedV1
	if err := json.Unmarshal(d.Record.Data, &x); err != nil {
		return dom.IntegrationRecord{}, err
	}
	r := dom.IntegrationRecord{Seq: d.Record.Seq, System: string(x.System), State: string(x.State), At: d.Record.RecordedAt}
	if x.Detail != nil {
		r.Detail = *x.Detail
	}
	return r, nil
}
