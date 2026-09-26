package engine

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"sync"

	appjournal "ant/internal/application/journal"
	jc "ant/internal/contracts/journal"
)

// ConsumerName — имя потребителя глобальной проекции в consumer_offsets.
func ConsumerName(projection string) string { return "projector:" + projection }

// Projector — роль projector (AD-45): глобальные проекции — одна копия-лидер
// по аренде; у каждой проекции свой курсор; проекция и курсор — в одной
// транзакции Append. Ошибка проекции на записи изделия →
// ops.processing.failed, проекция продолжает.
type Projector struct {
	Consumer appjournal.Consumer
	Codec    *Codec
	Store    ProjectionStore
	Registry *Registry
	Log      *slog.Logger
}

// Run исполняет все глобальные проекции под арендой лидера fence до отмены ctx.
func (p *Projector) Run(ctx context.Context, fence appjournal.Fence) error {
	log := p.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	if gc, ok := p.Consumer.(appjournal.GroupConsumer); ok {
		return p.runGroup(ctx, fence, gc)
	}
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for _, g := range p.Registry.Globals() {
		wg.Go(func() {
			err := p.Consumer.Consume(ctx, ConsumerName(g.Name), appjournal.Scope{Global: true}, func(ctx context.Context, batch []jc.JournalEntry) (appjournal.AppendRequest, error) {
				rq, err := p.Apply(ctx, g, batch)
				rq.Fence = &fence
				return rq, err
			})
			if err != nil && ctx.Err() == nil {
				log.Error("проекция остановлена", "projection", g.Name, "err", err)
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return errors.Join(errs...)
}

// runGroup — все глобальные проекции одним потребителем-группой (эпик 35):
// пачка журнала читается и декодируется один раз, выход всех проекций и их
// курсоры — одна транзакция Append. Раньше каждая проекция (их десятки)
// читала, декодировала и фиксировала каждую запись сама — это было главной
// нагрузкой на пул соединений и Postgres в прогоне MS-1. Смысл тот же: у
// каждой проекции свой курсор, запись отдаётся проекции только после него.
func (p *Projector) runGroup(ctx context.Context, fence appjournal.Fence, gc appjournal.GroupConsumer) error {
	globals := p.Registry.Globals()
	names := make([]string, len(globals))
	for i, g := range globals {
		names[i] = ConsumerName(g.Name)
	}
	return gc.ConsumeGroup(ctx, names, func(ctx context.Context, batch []jc.JournalEntry, from map[string]int64) (appjournal.AppendRequest, error) {
		decoded := make([]decodedEntry, len(batch))
		for i, e := range batch {
			decoded[i].entry = e
			decoded[i].d, decoded[i].err = p.Codec.Decode(ctx, e)
		}
		var rq appjournal.AppendRequest
		for i, g := range globals {
			after := from[names[i]]
			k := 0
			for k < len(decoded) && int64(decoded[k].entry.Seq) <= after {
				k++
			}
			if k == len(decoded) {
				continue
			}
			r, err := p.applyDecoded(ctx, g, decoded[k:])
			if err != nil {
				return rq, err
			}
			rq.Batch = append(rq.Batch, r.Batch...)
			rq.Effects = append(rq.Effects, r.Effects...)
		}
		rq.Fence = &fence
		return rq, nil
	})
}

// decodedEntry — запись журнала и результат её декодирования (один раз на группу).
type decodedEntry struct {
	entry jc.JournalEntry
	d     Decoded
	err   error
}

// Apply — выход глобальной проекции на пачку записей: новые значения
// затронутых ключей, изменения для SSE и сбои обработки (AD-45).
func (p *Projector) Apply(ctx context.Context, g GlobalProjection, batch []jc.JournalEntry) (appjournal.AppendRequest, error) {
	decoded := make([]decodedEntry, len(batch))
	for i, e := range batch {
		decoded[i].entry = e
		decoded[i].d, decoded[i].err = p.Codec.Decode(ctx, e)
	}
	return p.applyDecoded(ctx, g, decoded)
}

func (p *Projector) applyDecoded(ctx context.Context, g GlobalProjection, batch []decodedEntry) (appjournal.AppendRequest, error) {
	var rq appjournal.AppendRequest
	cache := map[string]json.RawMessage{}
	changed := map[string]Change{}
	for _, x := range batch {
		e, d, err := x.entry, x.d, x.err
		if err == nil {
			err = p.step(ctx, g, d, cache, changed)
		}
		if err == nil {
			continue
		}
		if e.ItemID == nil {
			// Запись вне изделия: сбой проекции — в лог, проекция продолжает.
			if p.Log != nil {
				p.Log.Error("проекция: запись пропущена", "projection", g.Name, "seq", e.Seq, "err", err)
			}
			continue
		}
		f, ferr := FailureRequest(ctx, p.Codec, ConsumerName(g.Name), *e.ItemID, int64(e.Seq), e, err)
		if ferr != nil {
			return rq, ferr
		}
		rq.Batch = append(rq.Batch, f.Batch...)
		rq.Effects = append(rq.Effects, f.Effects...)
	}
	var changes []Change
	for _, k := range slices.Sorted(maps.Keys(changed)) {
		rq.Effects = append(rq.Effects, ProjectionPut{Name: g.Name, Key: k, Value: cache[k]})
		if c := changed[k]; c.Entity != "" {
			changes = append(changes, c)
		}
	}
	if len(changes) > 0 {
		rq.Effects = append(rq.Effects, Notify{Changes: changes})
	}
	return rq, nil
}

func (p *Projector) step(ctx context.Context, g GlobalProjection, d Decoded, cache map[string]json.RawMessage, changed map[string]Change) error {
	for _, k := range g.Keys(d.Record) {
		prev, ok := cache[k]
		if !ok && p.Store != nil {
			v, found, err := p.Store.Get(ctx, g.Name, k)
			if err != nil {
				return err
			}
			if found {
				prev = v
			}
		}
		next, err := g.Step(k, prev, d.Record)
		if err != nil {
			return err
		}
		cache[k] = next
		c := Change{Seq: d.Record.Seq, RunID: d.Record.RunID, ReceivedAt: d.Record.ReceivedAt}
		if g.Entity != nil {
			if ent, id, ok := g.Entity(k); ok {
				c.Entity, c.ID = ent, id
			}
		}
		changed[k] = c
	}
	return nil
}
