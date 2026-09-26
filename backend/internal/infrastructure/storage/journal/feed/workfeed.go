package feed

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"

	"ant/internal/application/engine"
	app "ant/internal/application/journal"
	store "ant/internal/infrastructure/storage/journal"
)

// WorkFeed — адаптер порта engine.WorkFeed (AD-6, AD-35, ключ work_feed:
// postgres): P фиксированных партиций hash(item_id) mod P с арендой
// `partition:‹n›` и эпохой; семантика consumer group Kafka с key = item_id.
//
// Распределение: копия держит не больше ⌈P / число живых копий⌉
// партиций; лишние отдаёт при следующем Partitions, свободные и истёкшие
// берёт — так изменение числа воркеров перераспределяет партиции, а упавший
// воркер отдаёт свои по истечении аренды (InfraClock).
type WorkFeed struct {
	store  *store.Store
	leases *store.Leases
	signal app.Signal
	parts  int
	opt    Options

	mu   sync.Mutex
	held map[int]app.Fence
}

// NewWorkFeed создаёт подачу работы для P партиций.
func NewWorkFeed(s *store.Store, l *store.Leases, sig app.Signal, partitions int, opt Options) *WorkFeed {
	opt.defaults()
	return &WorkFeed{store: s, leases: l, signal: sig, parts: partitions, opt: opt, held: map[int]app.Fence{}}
}

var _ engine.WorkFeed = (*WorkFeed)(nil)

const (
	partitionPrefix = "partition:"
	memberPrefix    = "worker:"
)

// Partitions продлевает свои аренды, отдаёт лишние, берёт свободные и
// возвращает партиции с эпохами. Вызывать не реже TTL/2.
func (w *WorkFeed) Partitions(ctx context.Context) ([]engine.Partition, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Присутствие копии — своя аренда `worker:‹держатель›`: новая копия
	// видна остальным раньше, чем получит хоть одну партицию.
	if _, _, err := w.leases.Acquire(ctx, memberPrefix+w.opt.Holder, w.opt.Holder, w.opt.TTL); err != nil {
		return nil, err
	}
	members, err := w.leases.Live(ctx, memberPrefix)
	if err != nil {
		return nil, err
	}
	live, err := w.leases.Live(ctx, partitionPrefix)
	if err != nil {
		return nil, err
	}
	holders := map[string]bool{w.opt.Holder: true}
	for _, m := range members {
		holders[m.Holder] = true
	}
	taken := map[int]bool{}
	var mine []int
	for _, l := range live {
		n, err := strconv.Atoi(strings.TrimPrefix(l.Name, partitionPrefix))
		if err != nil || n < 0 || n >= w.parts {
			continue
		}
		holders[l.Holder] = true
		taken[n] = true
		if l.Holder == w.opt.Holder {
			mine = append(mine, n)
		}
	}
	target := (w.parts + len(holders) - 1) / len(holders)
	next := map[int]app.Fence{}
	slices.Sort(mine)
	for i, n := range mine {
		if i >= target {
			// Лишняя партиция — отдать новой копии.
			if f, ok := w.held[n]; ok {
				_ = w.leases.Release(ctx, f)
			}
			continue
		}
		f, ok, err := w.leases.Acquire(ctx, store.PartitionLeaseName(n), w.opt.Holder, w.opt.TTL)
		if err != nil {
			return nil, err
		}
		if ok {
			next[n] = f
		}
	}
	for n := 0; n < w.parts && len(next) < target; n++ {
		if taken[n] {
			continue
		}
		f, ok, err := w.leases.Acquire(ctx, store.PartitionLeaseName(n), w.opt.Holder, w.opt.TTL)
		if err != nil {
			return nil, err
		}
		if ok {
			next[n] = f
		}
	}
	w.held = next
	out := make([]engine.Partition, 0, len(next))
	for _, n := range slices.Sorted(maps.Keys(next)) {
		out = append(out, engine.Partition{Number: n, Epoch: next[n].Epoch})
	}
	return out, nil
}

// Next — изделия партиции с необработанным входом после курсора воркера
// (WorkerConsumer), по возрастанию последнего триггера. Нет работы — ждёт
// сигнала «есть новое» не дольше TTL/2 и возвращает пустой список: воркеру
// пора продлить аренды (Partitions).
func (w *WorkFeed) Next(ctx context.Context, p engine.Partition) ([]engine.Work, error) {
	head := w.signal.Head()
	cursor, err := w.store.Cursor(ctx, app.WorkerConsumer, p.Number)
	if err != nil {
		return nil, err
	}
	items, err := w.store.PendingItems(ctx, p.Number, cursor, w.opt.Batch)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		wait(ctx, w.signal, max(head, cursor), w.opt.TTL/2)
		if items, err = w.store.PendingItems(ctx, p.Number, cursor, w.opt.Batch); err != nil {
			return nil, err
		}
	}
	out := make([]engine.Work, 0, len(items))
	for _, it := range items {
		out = append(out, engine.Work{ItemID: it.ItemID, UpToSeq: it.UpToSeq, Trigger: it.Trigger})
	}
	return out, ctx.Err()
}

// Fence — аренда партиции для Append воркера (ErrFenced при потере).
func (w *WorkFeed) Fence(p engine.Partition) app.Fence {
	return app.Fence{Lease: store.PartitionLeaseName(p.Number), Epoch: p.Epoch}
}
