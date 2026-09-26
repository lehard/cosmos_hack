package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// Rebuilder — `ant rebuild` (FR-115, FR-124, AD-45): проекции
// восстанавливаются из журнала той же свёрткой, что у воркера; журнал не
// меняется, реакции не пишутся и наружу ничего не уходит. `--item` —
// повтор изделия с «обработка остановлена» или пересборка его проекций.
type Rebuilder struct {
	Codec    *Codec
	Fold     engine.Folder
	Bundles  BundleSource
	Registry *Registry
	// Partitions — партиции, которые читает полная пересборка; пусто —
	// весь журнал (записи вне изделия лежат в партиции 0).
	Partitions []int
}

// RebuildReport — итог пересборки.
type RebuildReport struct {
	Items   int
	Globals int
	// Retried — изделие было «обработка остановлена»: записан повтор обработки.
	Retried bool
	// RebuildHash — хеш всех значений проекций (AD-6 rebuild_hash): пересборка
	// над одним журналом на 1 и N воркерах обязана дать тот же хеш.
	RebuildHash string
}

func (r *Rebuilder) defaults() (engine.Folder, BundleSource) {
	fold, bundles := r.Fold, r.Bundles
	if fold == nil {
		fold = engine.Fold
	}
	if bundles == nil {
		bundles = EmptyBundles{}
	}
	return fold, bundles
}

// RebuildItem — `ant rebuild --item ‹id›`: если изделие «обработка
// остановлена», записывает ops.processing.retried — воркер снимет отметку и
// свернёт изделие заново (AD-45); иначе заново строит проекции изделия и его
// вклады. reason — причина повтора (для журнала).
func (r *Rebuilder) RebuildItem(ctx context.Context, itemID, reason string) (RebuildReport, error) {
	in, err := r.Codec.LoadItem(ctx, itemID, 0)
	if err != nil {
		return RebuildReport{}, err
	}
	if len(in.Input) == 0 && !in.Stopped {
		return RebuildReport{}, fmt.Errorf("изделие %s: входа в журнале нет", itemID)
	}
	if in.Stopped {
		if reason == "" {
			reason = "ant rebuild --item: повтор обработки после исправления"
		}
		id := kernel.UUIDv5(constants.NsAnt, string(catalog.OpsProcessingRetried)+"\x1f"+in.FailureID)
		pend, err := r.Codec.Encode(ctx, Out{
			EventID: id, Type: catalog.OpsProcessingRetried, Kind: catalog.KindDecision, Stream: "item:" + itemID,
			ItemID: itemID, RunID: in.Last.RunID, OccurredAt: in.Last.OccurredAt, Causation: in.FailureID,
			Data: ev.OpsProcessingRetriedV1{FailureEventID: ev.UUID(in.FailureID), Reason: &ev.Reason{Text: ev.Text(reason)}},
		})
		if err != nil {
			return RebuildReport{}, err
		}
		if _, err := r.Codec.Store.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{pend}}); err != nil {
			return RebuildReport{}, err
		}
		return RebuildReport{Items: 1, Retried: true}, nil
	}
	values, err := r.itemValues(ctx, in)
	if err != nil {
		return RebuildReport{}, err
	}
	effects := []appjournal.Effect{}
	for _, p := range r.Registry.Items() {
		effects = append(effects, ProjectionReset{Name: p.Name, ItemID: itemID})
	}
	effects = append(effects, values...)
	effects = append(effects, Notify{Changes: itemChanges(itemID, in.Last.Seq, in.Last)})
	if _, err := r.Codec.Store.Append(ctx, appjournal.AppendRequest{Effects: effects}); err != nil {
		return RebuildReport{}, err
	}
	return RebuildReport{Items: 1, RebuildHash: hashEffects(values)}, nil
}

func (r *Rebuilder) itemValues(ctx context.Context, in ItemInput) ([]appjournal.Effect, error) {
	fold, bundles := r.defaults()
	b, _, err := bundles.Bundle(ctx, in.ItemID, in.Input)
	if err != nil {
		return nil, err
	}
	snap, rs, err := safeFold(fold, b, in.Input)
	if err != nil {
		return nil, &ProcessingError{ItemID: in.ItemID, Seq: in.Last.Seq, Err: err}
	}
	normalize(rs)
	return r.Registry.ItemEffects(in.ItemID, snap, rs)
}

// RebuildAll — `ant rebuild`: сбросить все проекции и построить их заново из
// журнала: проекции изделий и вклады — свёрткой каждого изделия, глобальные
// — переигрыванием журнала по seq; курсоры глобальных проекций встают на
// конец журнала. Изделия с «обработка остановлена» пропускаются (их
// повторяет `--item`). Запускать при остановленных ролях worker и projector.
func (r *Rebuilder) RebuildAll(ctx context.Context) (RebuildReport, error) {
	var all []jc.JournalEntry
	parts := make([]*int, 0, len(r.Partitions))
	for _, p := range r.Partitions {
		parts = append(parts, &p)
	}
	if len(parts) == 0 {
		parts = append(parts, nil)
	}
	for _, part := range parts {
		after := int64(0)
		for {
			page, err := r.Codec.Store.Read(ctx, appjournal.ReadQuery{Partition: part, AfterSeq: after, Limit: 1000})
			if err != nil {
				return RebuildReport{}, err
			}
			all = append(all, page...)
			if len(page) < 1000 {
				break
			}
			after = int64(page[len(page)-1].Seq)
		}
	}
	slices.SortFunc(all, func(a, b jc.JournalEntry) int { return a.Seq - b.Seq })
	all = slices.CompactFunc(all, func(a, b jc.JournalEntry) bool { return a.Seq == b.Seq })

	items := map[string]bool{}
	for _, e := range all {
		if e.ItemID != nil && *e.ItemID != "" {
			items[*e.ItemID] = true
		}
	}
	var hashed []appjournal.Effect
	reset := []appjournal.Effect{}
	for _, p := range r.Registry.Items() {
		reset = append(reset, ProjectionReset{Name: p.Name})
	}
	for _, g := range r.Registry.Globals() {
		reset = append(reset, ProjectionReset{Name: g.Name})
	}
	if _, err := r.Codec.Store.Append(ctx, appjournal.AppendRequest{Effects: reset}); err != nil {
		return RebuildReport{}, err
	}
	rep := RebuildReport{}
	for _, id := range slices.Sorted(maps.Keys(items)) {
		in, err := r.Codec.LoadItem(ctx, id, 0)
		if err != nil {
			return rep, err
		}
		if in.Stopped || len(in.Input) == 0 {
			continue
		}
		values, err := r.itemValues(ctx, in)
		if err != nil {
			return rep, err
		}
		if _, err := r.Codec.Store.Append(ctx, appjournal.AppendRequest{Effects: values}); err != nil {
			return rep, err
		}
		hashed = append(hashed, values...)
		rep.Items++
	}
	decoded := make([]Decoded, 0, len(all))
	for _, e := range all {
		d, err := r.Codec.Decode(ctx, e)
		if err != nil {
			return rep, &ProcessingError{Seq: int64(e.Seq), Err: err}
		}
		decoded = append(decoded, d)
	}
	var last int64
	if len(all) > 0 {
		last = int64(all[len(all)-1].Seq)
	}
	for _, g := range r.Registry.Globals() {
		state := map[string]json.RawMessage{}
		for _, d := range decoded {
			for _, k := range g.Keys(d.Record) {
				next, err := g.Step(k, state[k], d.Record)
				if err != nil {
					return rep, fmt.Errorf("проекция %s, seq %d: %w", g.Name, d.Record.Seq, err)
				}
				state[k] = next
			}
		}
		var effects []appjournal.Effect
		for _, k := range slices.Sorted(maps.Keys(state)) {
			effects = append(effects, ProjectionPut{Name: g.Name, Key: k, Value: state[k]})
		}
		rq := appjournal.AppendRequest{Effects: effects,
			Consumer: &appjournal.CursorAdvance{Name: ConsumerName(g.Name), Partition: appjournal.GlobalPartition, Seq: last}}
		if _, err := r.Codec.Store.Append(ctx, rq); err != nil {
			return rep, err
		}
		hashed = append(hashed, effects...)
		rep.Globals++
	}
	if _, err := r.Codec.Store.Append(ctx, appjournal.AppendRequest{Effects: []appjournal.Effect{
		Notify{Changes: []Change{{Entity: platform.EntityLiveMap, ID: "global", Seq: last}}}}}); err != nil {
		return rep, err
	}
	rep.RebuildHash = hashEffects(hashed)
	return rep, nil
}

// hashEffects — rebuild_hash: H(канонический список «проекция, ключ, значение»,
// отсортированный), вклады — как значения проекции вкладов (AD-6).
func hashEffects(effects []appjournal.Effect) string {
	var rows []string
	for _, e := range effects {
		switch x := e.(type) {
		case ProjectionPut:
			rows = append(rows, x.Name+"\x1f"+x.Key+"\x1f"+string(x.Value))
		case ContributionsReplace:
			b, _ := engine.Canonical(x.Rows)
			rows = append(rows, "contributions\x1f"+x.ItemID+"\x1f"+string(b))
		}
	}
	slices.Sort(rows)
	return engine.Hash([]byte(strings.Join(rows, "\x1e")))
}
