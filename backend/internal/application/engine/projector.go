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

// GlobalPartition — партиция курсора глобального потребителя (охват global, AD-45).
const GlobalPartition = -1

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

// Apply — выход глобальной проекции на пачку записей: новые значения
// затронутых ключей, изменения для SSE и сбои обработки (AD-45).
func (p *Projector) Apply(ctx context.Context, g GlobalProjection, batch []jc.JournalEntry) (appjournal.AppendRequest, error) {
	var rq appjournal.AppendRequest
	cache := map[string]json.RawMessage{}
	changed := map[string]Change{}
	for _, e := range batch {
		d, err := p.Codec.Decode(ctx, e)
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
