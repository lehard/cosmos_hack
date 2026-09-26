package journal_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	app "ant/internal/application/journal"
	"ant/internal/application/platform"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	store "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/clock"
	jt "ant/internal/infrastructure/storage/journal/journaltest"
)

// Две копии ant (два пула, свои соединения) пишут параллельно — цепочка одна,
// без вилки: seq подряд, каждое звено сходится с предыдущим (AD-6, AD-44).
func TestParallelAppendNoFork(t *testing.T) {
	d := jt.NewDB(t)
	copies := []*store.Store{store.NewStore(d.AppPool(t), jt.SysClock{}), store.NewStore(d.AppPool(t), jt.SysClock{})}
	const workers, rounds = 4, 15
	var wg sync.WaitGroup
	errs := make(chan error, len(copies)*workers)
	for ci, s := range copies {
		for w := range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for r := range rounds {
					item := fmt.Sprintf("ENT01:C%d-W%d", ci, w)
					rq := app.AppendRequest{Batch: []app.Pending{jt.Fact(item), jt.Fact(item)}}
					if r%5 == 0 {
						// Критическое действие — в той же транзакции, своя цепочка ca (AD-8).
						rq.Critical = []app.Pending{jt.Entry("decision.disposition.set", jc.JournalEntryEntryKindDecision, "ca", item, time.Now())}
					}
					if _, err := s.Append(context.Background(), rq); err != nil {
						errs <- err
						return
					}
				}
			}()
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	main := jt.ReadAll(t, copies[0], "main")
	if want := len(copies) * workers * rounds * 2; len(main) != want {
		t.Fatalf("записей %d, ожидалось %d", len(main), want)
	}
	head, err := dj.VerifyLinks(dj.ZeroLink, 0, main)
	if err != nil {
		t.Fatalf("вилка или разрыв цепочки: %v", err)
	}
	ca := jt.ReadAll(t, copies[1], "ca")
	if want := len(copies) * workers * (rounds / 5); len(ca) != want {
		t.Fatalf("записей ca %d, ожидалось %d", len(ca), want)
	}
	caHead, err := dj.VerifyLinks(dj.ZeroLink, 0, ca)
	if err != nil {
		t.Fatalf("цепочка ca: %v", err)
	}
	h, err := copies[1].Head(context.Background())
	if err != nil || h.MainLink != head.String() || h.CALink != caHead.String() || h.MainSeq != int64(len(main)) {
		t.Fatalf("головы %+v, ожидались %s / %s (%v)", h, head, caHead, err)
	}
	// committed_at и recorded_at не убывают по seq (AD-37); конверт открывается, commit сверен.
	for i := 1; i < len(main); i++ {
		if main[i].CommittedAt < main[i-1].CommittedAt || main[i].RecordedAt < main[i-1].RecordedAt {
			t.Fatalf("время убывает на seq %d", main[i].Seq)
		}
	}
	if _, err := copies[0].Open(context.Background(), main[0]); err != nil {
		t.Fatal(err)
	}
}

// Изменение, удаление и очистка журнала невозможны: у ant_app нет прав,
// у владельца и суперпользователя срабатывают триггеры (AD-2, FR-71).
func TestAppendOnly(t *testing.T) {
	d := jt.NewDB(t)
	s := store.NewStore(d.AppPool(t), jt.SysClock{})
	if _, err := s.Append(context.Background(), app.AppendRequest{Batch: []app.Pending{jt.Fact("ENT01:A-1")},
		ConcessionGrants: []app.ConcessionGrant{{ConcessionID: "CON-A", Limit: 1}}}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := d.AdminConn(t)
	attempt := func(role, sql string) error {
		tx, err := c.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if role != "" {
			if _, err := tx.Exec(ctx, "SET LOCAL ROLE "+role); err != nil {
				t.Fatal(err)
			}
		}
		_, err = tx.Exec(ctx, sql)
		return err
	}
	code := func(err error) string {
		var pe *pgconn.PgError
		if errors.As(err, &pe) {
			return pe.Code
		}
		return fmt.Sprint(err)
	}
	for _, sql := range []string{
		"UPDATE journal.entries SET source_id = 'forged' WHERE seq = 1",
		"DELETE FROM journal.entries WHERE seq = 1",
		"TRUNCATE journal.entries",
		"UPDATE journal.concession_ledger SET delta = 100",
	} {
		// Роль приложения: нет привилегии (42501).
		if got := code(attempt("ant_app", sql)); got != "42501" {
			t.Errorf("ant_app: %s → %s, ожидался отказ 42501", sql, got)
		}
		// Владелец таблиц: триггер append-only (AJ001).
		if got := code(attempt("ant_owner", sql)); got != "AJ001" {
			t.Errorf("ant_owner: %s → %s, ожидался AJ001", sql, got)
		}
	}
	// Роль приложения не выключает триггеры через session_replication_role.
	if got := code(attempt("ant_app", "SET session_replication_role = replica")); got != "42501" {
		t.Errorf("ant_app: session_replication_role → %s, ожидался 42501", got)
	}
	// Суперпользователь в режиме replica: триггеры ENABLE ALWAYS всё равно срабатывают.
	if got := code(attempt("", "SET LOCAL session_replication_role = replica; UPDATE journal.entries SET source_id = 'x'")); got != "AJ001" {
		t.Errorf("суперпользователь, replica: %s, ожидался AJ001", got)
	}
	// Верификатор только читает.
	if got := code(attempt("ant_verifier", "INSERT INTO journal.concession_ledger VALUES ('x', 1, 1)")); got != "42501" {
		t.Errorf("ant_verifier: INSERT → %s, ожидался 42501", got)
	}
	if err := attempt("ant_verifier", "SELECT count(*) FROM journal.entries"); err != nil {
		t.Errorf("ant_verifier: SELECT: %v", err)
	}
	e := jt.ReadAll(t, s, "main")
	if len(e) != 1 || e[0].SourceID != "test-source" {
		t.Fatalf("запись изменилась: %+v", e)
	}
}

// Копия, потерявшая аренду, не может писать: эпоха выросла у нового
// держателя, Append прежней копии — ErrFenced (AD-6, AD-44).
func TestFencedAfterLeaseLoss(t *testing.T) {
	d := jt.NewDB(t)
	clk := clock.NewFake(time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC))
	poolA, poolB := d.AppPool(t), d.AppPool(t)
	sA, sB := store.NewStore(poolA, clk), store.NewStore(poolB, clk)
	lA, lB := store.NewLeases(poolA, clk), store.NewLeases(poolB, clk)
	ctx := context.Background()
	lease := app.PartitionLease(1)
	fA, ok, err := lA.Acquire(ctx, lease, "copy-a", 10*time.Second)
	if err != nil || !ok || fA.Epoch != 1 {
		t.Fatalf("A: %+v %v %v", fA, ok, err)
	}
	if _, ok, _ := lB.Acquire(ctx, lease, "copy-b", 10*time.Second); ok {
		t.Fatal("B взял занятую аренду")
	}
	if _, err := sA.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Fact("ENT01:F-1")}, Fence: &fA}); err != nil {
		t.Fatalf("A с действующей арендой: %v", err)
	}
	// Продление своей действующей аренды — та же эпоха.
	if f, ok, _ := lA.Acquire(ctx, lease, "copy-a", 10*time.Second); !ok || f.Epoch != fA.Epoch {
		t.Fatalf("продление: %+v %v", f, ok)
	}
	// A «уснул»: аренда истекла по InfraClock, её взял B.
	clk.Advance(11 * time.Second)
	fB, ok, err := lB.Acquire(ctx, lease, "copy-b", 10*time.Second)
	if err != nil || !ok || fB.Epoch != 2 {
		t.Fatalf("B после истечения: %+v %v %v", fB, ok, err)
	}
	_, err = sA.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Fact("ENT01:F-1")}, Fence: &fA})
	if !errors.Is(err, app.ErrFenced) {
		t.Fatalf("A после потери аренды: %v, ожидался ErrFenced", err)
	}
	if pe, ok := app.Problem(err); !ok || pe.Code != "journal.fenced" {
		t.Fatalf("problem: %+v", pe)
	}
	if _, err := sB.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Fact("ENT01:F-1")}, Fence: &fB}); err != nil {
		t.Fatalf("B: %v", err)
	}
	// Истёкшая и не перехваченная аренда тоже не даёт писать.
	clk.Advance(11 * time.Second)
	if _, err := sB.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Fact("ENT01:F-1")}, Fence: &fB}); !errors.Is(err, app.ErrFenced) {
		t.Fatalf("B с истёкшей арендой: %v", err)
	}
	// Отданная аренда сразу доступна другому — с новой эпохой.
	fB2, _, _ := lB.Acquire(ctx, lease, "copy-b", 10*time.Second)
	if err := lB.Release(ctx, fB2); err != nil {
		t.Fatal(err)
	}
	if f, ok, _ := lA.Acquire(ctx, lease, "copy-a", 10*time.Second); !ok || f.Epoch != fB2.Epoch+1 {
		t.Fatalf("после Release: %+v %v", f, ok)
	}
	if n := len(jt.ReadAll(t, sA, "main")); n != 2 {
		t.Fatalf("записей %d, ожидалось 2 (запись без аренды не прошла)", n)
	}
}

// Проверки конкурентности AD-39 в транзакции записи.
func TestConcurrencyChecks(t *testing.T) {
	d := jt.NewDB(t)
	s := store.NewStore(d.AppPool(t), jt.SysClock{})
	ctx := context.Background()
	item := "ENT01:K-1"
	stream := dj.ItemStream(item)
	r1, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Fact(item)}})
	if err != nil {
		t.Fatal(err)
	}
	basis := r1.Seqs[0]
	// Реакция воркера не guard_relevant — версия потока не меняется.
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Entry("decision.nonconformity.drafted", jc.JournalEntryEntryKindReaction, stream, item, time.Now())}}); err != nil {
		t.Fatal(err)
	}
	decision := func() app.Pending {
		return jt.Entry("decision.disposition.set", jc.JournalEntryEntryKindDecision, stream, item, time.Now())
	}
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{decision()}, Checks: []app.Check{{Stream: stream, BasisSeq: basis}}}); err != nil {
		t.Fatalf("гард на актуальном basis_seq: %v", err)
	}
	// Решение на старом basis_seq — после него уже есть guard_relevant запись.
	_, err = s.Append(ctx, app.AppendRequest{Batch: []app.Pending{decision()}, Checks: []app.Check{{Stream: stream, BasisSeq: basis}}})
	if !errors.Is(err, app.ErrStaleState) {
		t.Fatalf("устаревший basis_seq: %v", err)
	}
	if pe, ok := app.Problem(err); !ok || pe.Code != "journal.stale_state" || pe.Params["stream"] != stream {
		t.Fatalf("problem: %+v", pe)
	}
	// Необработанный вход изделия: курсора воркера нет — вход не обработан.
	h, _ := s.Head(ctx)
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{decision()}, Checks: []app.Check{{Stream: stream, BasisSeq: h.MainSeq, ItemProcessed: true}}}); !errors.Is(err, app.ErrStaleState) {
		t.Fatalf("необработанный вход: %v", err)
	}
	part := kernel.PartitionOf(item, 4)
	if _, err := s.Append(ctx, app.AppendRequest{Consumer: &app.CursorAdvance{Name: app.WorkerConsumer, Partition: part, Seq: h.MainSeq}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{decision()}, Checks: []app.Check{{Stream: stream, BasisSeq: h.MainSeq, ItemProcessed: true}}}); err != nil {
		t.Fatalf("вход обработан: %v", err)
	}
	// Политика изменилась после policy_seq.
	pol := jt.Entry("policy.role.assigned", jc.JournalEntryEntryKindDecision, "policy:global", "", time.Now())
	pr, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{pol}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{decision()}, Checks: []app.Check{{PolicyStream: "policy:global", PolicySeq: pr.Seqs[0] - 1}}}); !errors.Is(err, app.ErrStalePolicy) {
		t.Fatalf("политика: %v", err)
	}
	// Лимит разрешения на отклонение: 2 изделия; третий расход — отказ, расход атомарен с решением.
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{decision()}, ConcessionGrants: []app.ConcessionGrant{{ConcessionID: "CON-1", Limit: 2}}}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var okN, exhaustedN int
	var mu sync.Mutex
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{decision()}, Checks: []app.Check{{ConcessionID: "CON-1", Consume: 1}}})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				okN++
			case errors.Is(err, app.ErrConcessionExhausted):
				exhaustedN++
			default:
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if okN != 2 || exhaustedN != 3 {
		t.Fatalf("расход лимита: принято %d, отказов %d (ожидалось 2 и 3)", okN, exhaustedN)
	}
	// Повтор event_id — ErrDuplicate.
	p := jt.Fact(item)
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{p}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{p}}); !errors.Is(err, app.ErrDuplicate) {
		t.Fatalf("повтор event_id: %v", err)
	}
	// Отказ проверки — ни одной записи из пачки (транзакция откатилась).
	if _, err := dj.VerifyLinks(dj.ZeroLink, 0, jt.ReadAll(t, s, "main")); err != nil {
		t.Fatal(err)
	}
}

// Время: recorded_at не убывает (сценарий), отставание часов копии в пределах
// допуска поглощается, больше — отказ (AD-37).
func TestTimeMonotonic(t *testing.T) {
	d := jt.NewDB(t)
	clk := clock.NewFake(time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC))
	s := store.NewStore(d.AppPool(t), clk)
	ctx := context.Background()
	p := jt.Fact("ENT01:T-1")
	p.Entry.RecordedAt = "2026-09-26T12:00:00.000Z"
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{p}}); err != nil {
		t.Fatal(err)
	}
	p = jt.Fact("ENT01:T-1")
	p.Entry.RecordedAt = "2026-09-26T11:59:59.999Z"
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{p}}); !errors.Is(err, app.ErrTimeRegression) {
		t.Fatalf("убывание recorded_at: %v", err)
	}
	clk.Advance(-time.Second)
	p = jt.Fact("ENT01:T-1")
	p.Entry.RecordedAt = "2026-09-26T12:00:01.000Z"
	r, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{p}})
	if err != nil || !r.Committed.Equal(time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("отставание часов в допуске: %v %v", r.Committed, err)
	}
	clk.Advance(-10 * time.Second)
	p = jt.Fact("ENT01:T-1")
	p.Entry.RecordedAt = "2026-09-26T12:00:02.000Z"
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{p}}); !errors.Is(err, app.ErrTimeRegression) {
		t.Fatalf("откат часов больше допуска: %v", err)
	}
}

// Чтение на момент: «что мы знали» — префикс по seq до recorded_at ≤ T;
// «как было» — occurred_at ≤ T по всему известному, включая позднее
// пришедшие записи о прошлом (AD-22, AD-37).
func TestReadMoment(t *testing.T) {
	d := jt.NewDB(t)
	s := store.NewStore(d.AppPool(t), jt.SysClock{})
	ctx := context.Background()
	t0 := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	item := "ENT01:M-1"
	mk := func(occurred, recorded time.Time) app.Pending {
		p := jt.Entry("inspection.result.recorded", jc.JournalEntryEntryKindFact, dj.ItemStream(item), item, occurred)
		p.Entry.RecordedAt = dj.FormatTime(recorded)
		return p
	}
	// seq1: случилось 8:00, записано 8:01; seq2: случилось 9:00, записано 9:01;
	// seq3: досылка — случилось 8:30, записано 10:00.
	for _, p := range []app.Pending{mk(t0, t0.Add(time.Minute)), mk(t0.Add(time.Hour), t0.Add(61*time.Minute)), mk(t0.Add(30*time.Minute), t0.Add(2*time.Hour))} {
		if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{p}}); err != nil {
			t.Fatal(err)
		}
	}
	at := func(axis platform.Axis, asOf time.Time) []int {
		es, err := s.Read(ctx, app.ReadQuery{Stream: dj.ItemStream(item), Moment: platform.Moment{Axis: axis, AsOf: &asOf}})
		if err != nil {
			t.Fatal(err)
		}
		var seqs []int
		for _, e := range es {
			seqs = append(seqs, e.Seq)
		}
		return seqs
	}
	check := func(name string, got []int, want ...int) {
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s: %v, ожидалось %v", name, got, want)
		}
	}
	check("знали на 9:30", at(platform.AxisRecorded, t0.Add(90*time.Minute)), 1, 2)
	check("было на 9:30", at(platform.AxisOccurred, t0.Add(90*time.Minute)), 1, 2, 3)
	check("знали на 8:45", at(platform.AxisRecorded, t0.Add(45*time.Minute)), 1)
	check("было на 8:45", at(platform.AxisOccurred, t0.Add(45*time.Minute)), 1, 3)
	check("знали на 7:00", at(platform.AxisRecorded, t0.Add(-time.Hour)))
	last, err := s.Read(ctx, app.ReadQuery{Backward: true, Limit: 1})
	if err != nil || len(last) != 1 || last[0].Seq != 3 {
		t.Fatalf("последняя запись: %v %v", last, err)
	}
	p := 0
	if es, _ := s.Read(ctx, app.ReadQuery{Partition: &p}); kernel.PartitionOf(item, 4) != 0 && len(es) != 0 {
		t.Fatalf("фильтр партиции: %d записей", len(es))
	}
}

// Миграции повторно ничего не применяют (роль migrate идемпотентна).
func TestMigrateIdempotent(t *testing.T) {
	d := jt.NewDB(t)
	applied, err := jt.MigrateAgain(d)
	if err != nil || len(applied) != 0 {
		t.Fatalf("повтор миграций: %v %v", applied, err)
	}
}
