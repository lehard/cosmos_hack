package journal

import (
	"context"

	app "ant/internal/application/journal"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля journal: SSE-канал живых обновлений
// (AD-21, FR-2) и чтение журнала событий.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_ = c
	httpapi.RegisterStream(api, func(ctx context.Context, after int64, runID string) (func(context.Context) (httpapi.EntityChanged, error), func(), error) {
		sub, err := q.Subscribe(ctx, after, runID)
		if err != nil {
			return nil, nil, err
		}
		next := func(ctx context.Context) (httpapi.EntityChanged, error) {
			ch, err := sub.Next(ctx)
			return httpapi.EntityChanged{Entity: ch.Entity, ID: ch.ID, Seq: ch.Seq, RunID: ch.RunID, Mode: ch.Mode}, err
		}
		return next, sub.Close, nil
	})
}
