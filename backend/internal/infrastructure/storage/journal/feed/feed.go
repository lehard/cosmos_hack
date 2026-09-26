// Пакет feed — потребители журнала и подача работы воркерам (AD-45, AD-6):
// адаптеры портов application/journal.Consumer и application/engine.WorkFeed
// на Postgres (ключи consumer: postgres, work_feed: postgres).
//
// Слой: infrastructure/storage, модуль journal. Kafka с key = item_id —
// замена WorkFeed за тем же портом (AD-35, описание).
//
// Потребитель — копия-лидер по аренде с эпохой; курсор consumer_offsets
// сдвигается в той же транзакции Append, что и его выход, поэтому после
// падения ничего не применяется дважды. Ожидание нового — сигнал LISTEN/NOTIFY
// (только seq), с опросом раз в половину срока аренды на случай его потери.
package feed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"time"

	app "ant/internal/application/journal"
	jc "ant/internal/contracts/journal"
	store "ant/internal/infrastructure/storage/journal"
)

// Options — общие параметры потребителей и подачи работы.
type Options struct {
	// Holder — идентификатор копии (держатель аренд): имя хоста и роль.
	Holder string
	// TTL — срок аренды; продление — каждые TTL/2.
	TTL time.Duration
	// Batch — сколько записей (изделий) отдаётся за шаг.
	Batch int
	Log   *slog.Logger
}

func (o *Options) defaults() {
	if o.TTL <= 0 {
		o.TTL = 10 * time.Second
	}
	if o.Batch <= 0 {
		o.Batch = 256
	}
	if o.Log == nil {
		o.Log = slog.Default()
	}
}

// Consumer — адаптер порта Consumer (AD-45).
type Consumer struct {
	store  *store.Store
	leases *store.Leases
	signal app.Signal
	opt    Options
}

// NewConsumer создаёт потребителя над журналом, арендами и сигналом.
func NewConsumer(s *store.Store, l *store.Leases, sig app.Signal, opt Options) *Consumer {
	opt.defaults()
	return &Consumer{store: s, leases: l, signal: sig, opt: opt}
}

var _ app.Consumer = (*Consumer)(nil)

// ConsumerLease — имя аренды потребителя: глобальный — `consumer:‹имя›`,
// по партиции — `consumer:‹имя›:‹партиция›`.
func ConsumerLease(name string, scope app.Scope) string {
	if scope.Global {
		return "consumer:" + name
	}
	return fmt.Sprintf("consumer:%s:%d", name, scope.Partition)
}

// Consume отдаёт записи после курсора в handle, пока копия держит аренду;
// выход handle и сдвиг курсора — одна транзакция Append с Fence. Потеря
// аренды (ErrFenced) — не ошибка: копия ждёт и пробует взять аренду снова.
func (c *Consumer) Consume(ctx context.Context, name string, scope app.Scope, handle func(ctx context.Context, batch []jc.JournalEntry) (app.AppendRequest, error)) error {
	part := app.GlobalPartition
	if !scope.Global {
		if scope.Partition < 0 {
			return fmt.Errorf("потребитель %s: партиция %d", name, scope.Partition)
		}
		part = scope.Partition
	}
	lease := ConsumerLease(name, scope)
	// Уходя (остановка, смена лидера роли), копия отдаёт аренду сразу:
	// следующий лидер не ждёт её истечения.
	var held app.Fence
	defer func() {
		if held.Lease != "" {
			_ = c.leases.Release(context.WithoutCancel(ctx), held)
		}
	}()
	// Эпик 35 (узкое место MS-1): аренда продлевается не на каждой итерации,
	// а раз в TTL/3 — продление это запись с фиксацией в Postgres, а итераций
	// столько же, сколько сигналов журнала. Курсор после своей фиксации
	// известен копии (она единственный писатель, пока держит аренду), поэтому
	// перечитывается только при (пере)взятии аренды. Защита не ослабевает:
	// Append по-прежнему проверяет эпоху и срок аренды в своей транзакции.
	var (
		renewAt time.Time
		cursor  int64
	)
	for ctx.Err() == nil {
		if held.Lease == "" || !time.Now().Before(renewAt) {
			fence, ok, err := c.leases.Acquire(ctx, lease, c.opt.Holder, c.opt.TTL)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			if !ok {
				held = app.Fence{}
				pause(ctx, c.opt.TTL/2)
				continue
			}
			held, renewAt = fence, time.Now().Add(c.opt.TTL/3)
			if cursor, err = c.store.Cursor(ctx, name, part); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
		fence := held
		head := c.signal.Head()
		q := app.ReadQuery{AfterSeq: cursor, Limit: c.opt.Batch}
		if !scope.Global {
			q.Partition = &part
		}
		batch, err := c.store.Read(ctx, q)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if len(batch) == 0 {
			// Ждать не дольше срока продления: аренда не должна истечь во сне.
			wait(ctx, c.signal, max(head, cursor), min(c.opt.TTL/2, max(time.Until(renewAt), time.Millisecond)))
			continue
		}
		rq, err := handle(ctx, batch)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("потребитель %s: %w", name, err)
		}
		// Fence — аренда потребителя: она, а не аренда роли-лидера, защищает
		// курсор (Fence, поставленный handle, заменяется).
		next := int64(batch[len(batch)-1].Seq)
		rq.Fence = &fence
		rq.Consumer = &app.CursorAdvance{Name: name, Partition: part, Seq: next}
		if _, err := c.store.Append(ctx, rq); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if errors.Is(err, app.ErrFenced) {
				c.opt.Log.Warn("потребитель: аренда утрачена", "consumer", name, "lease", lease, "epoch", fence.Epoch)
				// Взять аренду и перечитать курсор заново на следующей итерации.
				held = app.Fence{}
				continue
			}
			return fmt.Errorf("потребитель %s: %w", name, err)
		}
		cursor = max(cursor, next)
	}
	return nil
}

// wait ждёт сигнала после seq не дольше d.
func wait(ctx context.Context, sig app.Signal, after int64, d time.Duration) {
	wctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	_, _ = sig.Wait(wctx, after)
}

func pause(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

var _ app.GroupConsumer = (*Consumer)(nil)

// ConsumeGroup — группа глобальных потребителей names одним проходом по
// журналу (эпик 35, app.GroupConsumer): у каждого имени своя аренда
// `consumer:‹имя›` и свой курсор; пачка читается после наименьшего курсора,
// выход handle и курсоры всех имён фиксируются одной транзакцией Append.
// Аренды продлеваются раз в TTL/3; не удалось взять хоть одну — копия отдаёт
// взятые и ждёт (группа работает целиком или не работает).
func (c *Consumer) ConsumeGroup(ctx context.Context, names []string, handle func(ctx context.Context, batch []jc.JournalEntry, from map[string]int64) (app.AppendRequest, error)) error {
	if len(names) == 0 {
		<-ctx.Done()
		return nil
	}
	held := map[string]app.Fence{}
	release := func() {
		for _, f := range held {
			_ = c.leases.Release(context.WithoutCancel(ctx), f)
		}
		clear(held)
	}
	defer release()
	var renewAt time.Time
	cursors := map[string]int64{}
	for ctx.Err() == nil {
		if len(held) == 0 || !time.Now().Before(renewAt) {
			ok := true
			for _, n := range names {
				f, got, err := c.leases.Acquire(ctx, ConsumerLease(n, app.Scope{Global: true}), c.opt.Holder, c.opt.TTL)
				if err != nil {
					if ctx.Err() != nil {
						return nil
					}
					return err
				}
				if !got {
					ok = false
					break
				}
				held[n] = f
			}
			if !ok {
				release()
				pause(ctx, c.opt.TTL/2)
				continue
			}
			renewAt = time.Now().Add(c.opt.TTL / 3)
			for _, n := range names {
				cur, err := c.store.Cursor(ctx, n, app.GlobalPartition)
				if err != nil {
					if ctx.Err() != nil {
						return nil
					}
					return err
				}
				cursors[n] = cur
			}
		}
		head := c.signal.Head()
		low := cursors[names[0]]
		for _, n := range names {
			low = min(low, cursors[n])
		}
		batch, err := c.store.Read(ctx, app.ReadQuery{AfterSeq: low, Limit: c.opt.Batch})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if len(batch) == 0 {
			wait(ctx, c.signal, max(head, low), min(c.opt.TTL/2, max(time.Until(renewAt), time.Millisecond)))
			continue
		}
		rq, err := handle(ctx, batch, maps.Clone(cursors))
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("группа потребителей %s…: %w", names[0], err)
		}
		next := int64(batch[len(batch)-1].Seq)
		fence := held[names[0]]
		rq.Fence = &fence
		rq.Consumer = nil
		rq.Cursors = rq.Cursors[:0]
		for _, n := range names {
			if cursors[n] < next {
				rq.Cursors = append(rq.Cursors, app.CursorAdvance{Name: n, Partition: app.GlobalPartition, Seq: next})
			}
		}
		if _, err := c.store.Append(ctx, rq); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if errors.Is(err, app.ErrFenced) {
				c.opt.Log.Warn("группа потребителей: аренда утрачена", "consumer", names[0], "epoch", fence.Epoch)
				release()
				continue
			}
			return fmt.Errorf("группа потребителей %s…: %w", names[0], err)
		}
		for _, n := range names {
			cursors[n] = max(cursors[n], next)
		}
	}
	return nil
}
