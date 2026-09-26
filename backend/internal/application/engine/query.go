package engine

import (
	"context"
	"errors"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// ItemAt — состояние изделия на момент (AD-22): та же свёртка, что у
// воркера, над префиксом входа; ничего не пишет и не отправляет.
type ItemAt struct {
	ItemID    string
	Moment    platform.Moment
	Snapshot  engine.Snapshot
	Reactions []kernel.Reaction
	StateHash string
	// Stopped — «обработка остановлена» (AD-45): состояние показано по
	// входу, но воркер его не сворачивает до `ant rebuild --item`.
	Stopped bool
}

// StateQueries — запросы состояния на момент (AD-22, FR-4, FR-124): «как
// было» (occurred_at ≤ T по всему известному) и «что мы знали» (префикс по
// seq с recorded_at ≤ T). Прогон «по другой версии правил» — тот же запрос с
// другим Bundles.
type StateQueries struct {
	Codec   *Codec
	Fold    engine.Folder
	Bundles BundleSource
}

// Item — состояние изделия на момент m.
func (q StateQueries) Item(ctx context.Context, itemID string, m platform.Moment) (ItemAt, error) {
	fold, bundles := q.Fold, q.Bundles
	if fold == nil {
		fold = engine.Fold
	}
	if bundles == nil {
		bundles = EmptyBundles{}
	}
	in, err := q.Codec.LoadItem(ctx, itemID, 0)
	if err != nil {
		return ItemAt{}, err
	}
	if len(in.Input) == 0 {
		e := platform.Fail(errcodes.ApiNotFound, "item_id", itemID)
		e.Detail = "Изделие " + itemID + " не найдено в журнале"
		return ItemAt{}, e
	}
	dm := engine.Moment{Axis: engine.AxisOccurred}
	if m.Axis == platform.AxisRecorded {
		dm.Axis = engine.AxisRecorded
	}
	if m.AsOf != nil {
		dm.At = *m.AsOf
	}
	prefix := engine.Prefix(in.Input, dm)
	bundle, _, err := bundles.Bundle(ctx, itemID, prefix)
	if err != nil {
		return ItemAt{}, err
	}
	snap, rs, err := safeFold(fold, bundle, prefix)
	if err != nil {
		return ItemAt{}, errors.Join(errors.New("свёртка на момент"), err)
	}
	normalize(rs)
	h, err := engine.StateHash(snap, rs)
	if err != nil {
		return ItemAt{}, err
	}
	return ItemAt{ItemID: itemID, Moment: m, Snapshot: snap, Reactions: rs, StateHash: h, Stopped: in.Stopped}, nil
}
