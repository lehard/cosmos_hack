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
	for ctx.Err() == nil {
		fence, ok, err := c.leases.Acquire(ctx, lease, c.opt.Holder, c.opt.TTL)
		if err != nil {
			return err
		}
		if !ok {
			pause(ctx, c.opt.TTL/2)
			continue
		}
		head := c.signal.Head()
		cursor, err := c.store.Cursor(ctx, name, part)
		if err != nil {
			return err
		}
		q := app.ReadQuery{AfterSeq: cursor, Limit: c.opt.Batch}
		if !scope.Global {
			q.Partition = &part
		}
		batch, err := c.store.Read(ctx, q)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			wait(ctx, c.signal, max(head, cursor), c.opt.TTL/2)
			continue
		}
		rq, err := handle(ctx, batch)
		if err != nil {
			return fmt.Errorf("потребитель %s: %w", name, err)
		}
		rq.Fence = &fence
		rq.Consumer = &app.CursorAdvance{Name: name, Partition: part, Seq: int64(batch[len(batch)-1].Seq)}
		if _, err := c.store.Append(ctx, rq); err != nil {
			if errors.Is(err, app.ErrFenced) {
				c.opt.Log.Warn("потребитель: аренда утрачена", "consumer", name, "lease", lease, "epoch", fence.Epoch)
				continue
			}
			return fmt.Errorf("потребитель %s: %w", name, err)
		}
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
