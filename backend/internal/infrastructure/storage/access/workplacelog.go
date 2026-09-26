package access

import (
	"context"

	app "ant/internal/application/access"
	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	accessdom "ant/internal/domain/access"
)

// WorkplaceLog — записи потока поста `workplace:‹id›` из журнала (порт
// application/access.WorkplaceLog, UI-16): назначения, токен, допуск,
// отклонения присутствия — расшифрованные и повышенные тем же кодеком, что и
// у движка (AD-20), с фильтром момента и прогона (AD-22, AD-38).
type WorkplaceLog struct {
	Journal appjournal.JournalStore
	Codec   *engineapp.Codec
}

var _ app.WorkplaceLog = WorkplaceLog{}

// Stream — записи потока поста по возрастанию seq.
func (l WorkplaceLog) Stream(ctx context.Context, workplaceID string, m platform.Moment) ([]accessdom.Record, error) {
	var out []accessdom.Record
	after := int64(0)
	for {
		es, err := l.Journal.Read(ctx, appjournal.ReadQuery{Stream: "workplace:" + workplaceID, AfterSeq: after, Limit: readLimit, Moment: m, RunID: m.RunID})
		if err != nil {
			return nil, err
		}
		for _, e := range es {
			d, err := l.Codec.Decode(ctx, e)
			if err != nil {
				return nil, err
			}
			out = append(out, accessdom.Record{Seq: d.Record.Seq, Type: e.EventType, Data: d.Record.Data, OccurredAt: d.Record.OccurredAt,
				EventID: d.Record.EventID, Actor: d.Record.Actor})
			after = int64(e.Seq)
		}
		if len(es) < readLimit {
			return out, nil
		}
	}
}
