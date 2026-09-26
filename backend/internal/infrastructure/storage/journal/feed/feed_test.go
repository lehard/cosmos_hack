package feed_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"ant/internal/application/engine"
	app "ant/internal/application/journal"
	"ant/internal/application/platform"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	store "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/feed"
	jt "ant/internal/infrastructure/storage/journal/journaltest"
)

// Потребитель: выход (проекция) и курсор — одна транзакция; после
// перезапуска ничего не применяется дважды; сигнал LISTEN/NOTIFY будит
// ожидающего (AD-45, AD-6).
func TestConsumerExactlyOnce(t *testing.T) {
	d := jt.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	// Таблица проекции тестового модуля: своя схема, полный доступ ant_app.
	adm := d.AdminConn(t)
	if _, err := adm.Exec(ctx, `CREATE SCHEMA proj; CREATE TABLE proj.seen (seq bigint PRIMARY KEY);
GRANT USAGE ON SCHEMA proj TO ant_app; GRANT ALL ON proj.seen TO ant_app`); err != nil {
		t.Fatal(err)
	}
	pool := d.AppPool(t)
	s := store.NewStore(pool, jt.SysClock{})
	leases := store.NewLeases(pool, jt.SysClock{})
	sig := store.NewListener(pool, nil)
	go func() { _ = sig.Run(ctx) }()

	const total = 30
	for i := range total {
		if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Fact(fmt.Sprintf("ENT01:P-%d", i%3))}}); err != nil {
			t.Fatal(err)
		}
	}
	// run — копия потребителя работает, пока курсор не дойдёт до target, и «падает».
	run := func(target int64) {
		cctx, ccancel := context.WithCancel(ctx)
		defer ccancel()
		c := feed.NewConsumer(s, leases, sig, feed.Options{Holder: "copy-1", TTL: 2 * time.Second, Batch: 7})
		errc := make(chan error, 1)
		go func() {
			errc <- c.Consume(cctx, "test.projection", app.Scope{Global: true}, func(ctx context.Context, batch []jc.JournalEntry) (app.AppendRequest, error) {
				return app.AppendRequest{Project: func(ctx context.Context, _ app.AppendResult) error {
					tx, ok := store.Tx(ctx)
					if !ok {
						return errors.New("нет транзакции в Project")
					}
					for _, e := range batch {
						if _, err := tx.Exec(ctx, "INSERT INTO proj.seen (seq) VALUES ($1)", e.Seq); err != nil {
							return err
						}
					}
					return nil
				}}, nil
			})
		}()
		for {
			cur, err := s.Cursor(ctx, "test.projection", app.GlobalPartition)
			if err != nil {
				t.Error(err)
				return
			}
			if cur >= target {
				break
			}
			select {
			case err := <-errc:
				t.Errorf("потребитель остановился: %v", err)
				return
			case <-ctx.Done():
				t.Errorf("курсор %d не дошёл до %d", cur, target)
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
		ccancel()
		if err := <-errc; err != nil {
			t.Error(err)
		}
	}
	run(10) // «упал» после части записей
	run(total)
	// Новые записи будят потребителя через сигнал.
	done := make(chan struct{})
	go func() {
		defer close(done)
		run(total + 5)
	}()
	time.Sleep(300 * time.Millisecond)
	for i := range 5 {
		if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Fact(fmt.Sprintf("ENT01:Q-%d", i))}}); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("потребитель не проснулся по сигналу")
	}
	var n, maxSeq int64
	if err := adm.QueryRow(ctx, "SELECT count(*), COALESCE(max(seq), 0) FROM proj.seen").Scan(&n, &maxSeq); err != nil {
		t.Fatal(err)
	}
	cur, _ := s.Cursor(ctx, "test.projection", app.GlobalPartition)
	if n != total+5 || maxSeq != total+5 || cur != total+5 {
		t.Fatalf("применено %d (max %d), курсор %d; ожидалось %d", n, maxSeq, cur, total+5)
	}
}

// Вторая копия потребителя не работает, пока аренду держит первая: глобальный
// потребитель — одна копия-лидер (AD-45).
func TestConsumerSingleLeader(t *testing.T) {
	d := jt.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := d.AppPool(t)
	s := store.NewStore(pool, jt.SysClock{})
	leases := store.NewLeases(pool, jt.SysClock{})
	sig := store.NewListener(pool, nil)
	go func() { _ = sig.Run(ctx) }()
	if _, _, err := leases.Acquire(ctx, feed.ConsumerLease("x", app.Scope{Global: true}), "copy-1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Fact("ENT01:L-1")}}); err != nil {
		t.Fatal(err)
	}
	cctx, ccancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer ccancel()
	called := false
	c := feed.NewConsumer(s, leases, sig, feed.Options{Holder: "copy-2", TTL: time.Second})
	_ = c.Consume(cctx, "x", app.Scope{Global: true}, func(context.Context, []jc.JournalEntry) (app.AppendRequest, error) {
		called = true
		return app.AppendRequest{}, nil
	})
	if called {
		t.Fatal("вторая копия обработала записи при чужой аренде")
	}
}

// WorkFeed: партиции делятся между копиями, изделия с необработанным входом
// отдаются по возрастанию последнего триггера; реакции воркера триггером не
// являются; курсор с Fence партиции сдвигается в Append воркера (AD-5, AD-6).
func TestWorkFeed(t *testing.T) {
	d := jt.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := d.AppPool(t)
	s := store.NewStore(pool, jt.SysClock{})
	leases := store.NewLeases(pool, jt.SysClock{})
	sig := store.NewListener(pool, nil)
	go func() { _ = sig.Run(ctx) }()
	const P = 4
	w1 := feed.NewWorkFeed(s, leases, sig, P, feed.Options{Holder: "w1", TTL: 2 * time.Second})
	w2 := feed.NewWorkFeed(s, leases, sig, P, feed.Options{Holder: "w2", TTL: 2 * time.Second})
	p1, err := w1.Partitions(ctx)
	if err != nil || len(p1) != P {
		t.Fatalf("одна копия: %v %v", p1, err)
	}
	// Вторая копия: сначала w1 отдаёт лишние, затем w2 берёт.
	if _, err := w2.Partitions(ctx); err != nil {
		t.Fatal(err)
	}
	p1, _ = w1.Partitions(ctx)
	p2, _ := w2.Partitions(ctx)
	if len(p1) != P/2 || len(p2) != P/2 {
		t.Fatalf("распределение: w1 %v, w2 %v", p1, p2)
	}
	owned := map[int]bool{}
	for _, p := range append(append([]any{}, anyParts(p1)...), anyParts(p2)...) {
		n := p.(int)
		if owned[n] {
			t.Fatalf("партиция %d у двух копий", n)
		}
		owned[n] = true
	}
	// Записи в партиции изделия.
	byPart := map[int]string{}
	for i := 0; len(byPart) < P && i < 1000; i++ {
		item := fmt.Sprintf("ENT01:W-%d", i)
		if _, ok := byPart[kernel.PartitionOf(item, P)]; !ok {
			byPart[kernel.PartitionOf(item, P)] = item
		}
	}
	part := p1[0]
	item := byPart[part.Number]
	mk := func(eventType string, kind jc.JournalEntryEntryKind) app.Pending {
		p := jt.Entry(eventType, kind, dj.ItemStream(item), item, time.Now())
		p.Entry.Partition = part.Number
		return p
	}
	r, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{mk("inspection.result.recorded", jc.JournalEntryEntryKindFact), mk("inspection.result.recorded", jc.JournalEntryEntryKindFact)}})
	if err != nil {
		t.Fatal(err)
	}
	works, err := w1.Next(ctx, part)
	if err != nil || len(works) != 1 || works[0].ItemID != item || works[0].UpToSeq != r.Seqs[1] {
		t.Fatalf("работа: %+v %v", works, err)
	}
	// Воркер записал реакцию и сдвинул курсор с Fence партиции.
	fence := w1.Fence(part)
	if _, err := s.Append(ctx, app.AppendRequest{
		Batch:    []app.Pending{mk("decision.nonconformity.drafted", jc.JournalEntryEntryKindReaction)},
		Fence:    &fence,
		Consumer: &app.CursorAdvance{Name: app.WorkerConsumer, Partition: part.Number, Seq: works[0].UpToSeq},
	}); err != nil {
		t.Fatal(err)
	}
	// Реакция не триггер: работы нет, Next ждёт сигнала и возвращает пусто.
	nctx, ncancel := context.WithTimeout(ctx, 3*time.Second)
	works, err = w1.Next(nctx, part)
	ncancel()
	if len(works) != 0 || (err != nil && !errors.Is(err, context.DeadlineExceeded)) {
		t.Fatalf("после реакции: %+v %v", works, err)
	}
	// Копия w2 не может писать в партицию w1.
	other := app.Fence{Lease: app.PartitionLease(part.Number), Epoch: part.Epoch + 1}
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{mk("decision.nonconformity.drafted", jc.JournalEntryEntryKindReaction)}, Fence: &other}); !errors.Is(err, app.ErrFenced) {
		t.Fatalf("чужая эпоха: %v", err)
	}
}

func anyParts(ps []engine.Partition) []any {
	out := make([]any, len(ps))
	for i, p := range ps {
		out[i] = p.Number
	}
	return out
}

// Живые обновления: подписка application/journal над журналом и сигналом
// отдаёт изменения сущностей по seq и догоняет после afterSeq (AD-21, AD-6).
func TestSubscribe(t *testing.T) {
	d := jt.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := d.AppPool(t)
	s := store.NewStore(pool, jt.SysClock{})
	sig := store.NewListener(pool, nil)
	go func() { _ = sig.Run(ctx) }()
	svc := app.NewServiceWith(s, sig)
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Fact("ENT01:S-1")}}); err != nil {
		t.Fatal(err)
	}
	sub, err := svc.Subscribe(ctx, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()
	ch, err := sub.Next(ctx)
	if err != nil || ch.Entity != platform.EntityItem || ch.ID != "ENT01:S-1" || ch.Seq != 1 {
		t.Fatalf("первое изменение: %+v %v", ch, err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(200 * time.Millisecond)
		_, _ = s.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Entry("policy.role.assigned", jc.JournalEntryEntryKindDecision, "global", "", time.Now()), jt.Fact("ENT01:S-2")}})
	}()
	ch, err = sub.Next(ctx)
	wg.Wait()
	if err != nil || ch.ID != "ENT01:S-2" || ch.Seq != 3 {
		t.Fatalf("изменение по сигналу: %+v %v", ch, err)
	}
}
