package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/domain/engine/enginetest"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	storage "ant/internal/infrastructure/storage/engine"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/feed"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

const partitions = 4

// core — ядро на Postgres, собранное как в cmd/ant: журнал (эпик 04),
// хранение движка, воркер с пустышкой enginetest, публикатор SSE и
// live-реализация journal с делегированием подписки.
type core struct {
	pool     *pgxpool.Pool
	journal  *journalstore.Store
	leases   *journalstore.Leases
	listener *journalstore.Listener
	engine   *storage.Store
	codec    *engineapp.Codec
	live     *engineapp.LiveUpdates
	service  *appjournal.Service
}

func newCore(t *testing.T, ctx context.Context) *core {
	t.Helper()
	p := pool(t)
	clock := journaltest.SysClock{}
	c := &core{pool: p, journal: journalstore.NewStore(p, clock), leases: journalstore.NewLeases(p, clock),
		listener: journalstore.NewListener(p, nil), engine: &storage.Store{Pool: p}}
	t.Cleanup(c.engine.Close)
	c.codec = &engineapp.Codec{Store: c.journal, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1",
		DomainBuild: dj.ZeroLink.String(), Partitions: partitions}
	c.live = engineapp.NewLiveUpdates(engineapp.LiveConfig{Log: c.engine})
	c.service = appjournal.NewServiceWith(c.journal, c.listener, appjournal.WithLive(c.live))
	go func() { _ = c.listener.Run(ctx) }()
	go func() { _ = c.live.Run(ctx) }()
	return c
}

// fact — результат контроля изделия, как его записал бы приём (эпик 06).
func (c *core) fact(t *testing.T, ctx context.Context, item, name, outcome string) (appjournal.AppendResult, error) {
	t.Helper()
	id := kernel.UUIDv5(constants.NsAnt, "slice-test\x1f"+name)
	p, err := c.codec.Encode(ctx, engineapp.Out{EventID: id, Type: catalog.InspectionResultRecorded, Kind: catalog.KindFact,
		Stream: "item:" + item, ItemID: item, OccurredAt: time.Now(), Correlation: id, Data: map[string]string{"outcome": outcome}})
	if err != nil {
		t.Fatal(err)
	}
	return c.journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}})
}

// Сквозной путь на Postgres (эпики 04 + 07): факт через Append → воркер
// (аренда партиции, курсор engine.worker) сворачивает изделие пустышкой →
// реакция, проекция изделия, вклады, журнал изменений и курсор записаны одной
// транзакцией Append → SSE-сообщение (сущность, id, seq) не позже 2 с.
func TestSliceFactToSSE(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := newCore(t, ctx)
	const item = "ENT01:I-42"
	part := kernel.PartitionOf(item, partitions)

	sub, err := c.service.Subscribe(ctx, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	started := time.Now()
	res, err := c.fact(t, ctx, item, "f1", "defect_indicated")
	if err != nil {
		t.Fatal(err)
	}
	seq := res.Seqs[0]

	// Решение по изделию до обработки его входа — 409 (AD-39, Check.ItemProcessed
	// по курсору воркера journal.WorkerConsumer).
	guard := appjournal.AppendRequest{Checks: []appjournal.Check{{Stream: "item:" + item, BasisSeq: seq, ItemProcessed: true}}}
	if _, err := c.journal.Append(ctx, guard); !errors.Is(err, appjournal.ErrStaleState) {
		t.Fatalf("до свёртки ждали journal.stale_state: %v", err)
	}

	wf := feed.NewWorkFeed(c.journal, c.leases, c.listener, partitions, feed.Options{Holder: "slice-test", TTL: 3 * time.Second})
	worker := engineapp.NewWorker(engineapp.WorkerConfig{Feed: wf, Codec: c.codec, Fold: enginetest.Fold, Refresh: time.Second})
	go func() { _ = worker.Run(ctx) }()

	sctx, scancel := context.WithTimeout(ctx, 2*time.Second)
	defer scancel()
	var got appjournal.Change
	for got.Entity != platform.EntityItem {
		if got, err = sub.Next(sctx); err != nil {
			t.Fatalf("SSE не пришло за 2 с (FR-2): %v", err)
		}
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("событие → SSE %v > 2 с", elapsed)
	}
	if got.ID != item || got.Seq != seq || got.Mode != platform.ModeLive {
		t.Fatalf("сообщение SSE: %+v, ждали item %s seq %d", got, item, seq)
	}

	// Реакция пустышки с basis_seq факта — в журнале.
	rs, err := c.journal.Read(ctx, appjournal.ReadQuery{ItemID: item, EventType: string(catalog.QualitySignalRaised)})
	if err != nil || len(rs) != 1 {
		t.Fatalf("реакция: %d %v", len(rs), err)
	}
	r := rs[0]
	if r.BasisSeq == nil || int64(*r.BasisSeq) != seq || r.Partition != part || r.CausationID == nil {
		t.Fatalf("реакция: basis=%v partition=%d causation=%v", r.BasisSeq, r.Partition, r.CausationID)
	}
	// Проекция изделия, курсор воркера и журнал изменений — та же транзакция:
	// видны вместе с реакцией (committed_at реакции не раньше записи курсора).
	raw, ok, err := c.engine.Get(ctx, engineapp.ItemStateProjection, item)
	if err != nil || !ok {
		t.Fatalf("проекция изделия: %v %v", ok, err)
	}
	var st engineapp.ItemState
	if err := json.Unmarshal(raw, &st); err != nil || st.BasisSeq != seq || st.Reactions != 1 || st.StateHash == "" {
		t.Fatalf("проекция изделия: %s %v", raw, err)
	}
	cursor, err := c.journal.Cursor(ctx, appjournal.WorkerConsumer, part)
	if err != nil || cursor != seq {
		t.Fatalf("курсор воркера партиции %d: %d %v", part, cursor, err)
	}
	// Гард видел поток изделия на seq реакции — вход обработан, 409 нет.
	guard.Checks[0].BasisSeq = int64(r.Seq)
	if _, err := c.journal.Append(ctx, guard); err != nil {
		t.Fatalf("после свёртки проверка ItemProcessed проходит: %v", err)
	}

	// Позднее событие → новая версия реакции «пересмотрен из-за записи» (FR-32)
	// и следующее сообщение SSE с seq позднего факта.
	res2, err := c.fact(t, ctx, item, "f2", "defect_indicated")
	if err != nil {
		t.Fatal(err)
	}
	sctx2, scancel2 := context.WithTimeout(ctx, 2*time.Second)
	defer scancel2()
	for got.Entity != platform.EntityItem || got.Seq != res2.Seqs[0] {
		if got, err = sub.Next(sctx2); err != nil {
			t.Fatalf("второе сообщение SSE: %v", err)
		}
	}
	rs, _ = c.journal.Read(ctx, appjournal.ReadQuery{ItemID: item, EventType: string(catalog.QualitySignalRaised)})
	if len(rs) != 2 || rs[1].Version == nil || *rs[1].Version != 2 || rs[1].Supersedes == nil || *rs[1].Supersedes != rs[0].EventID {
		t.Fatalf("новая версия слота: %d записей", len(rs))
	}

	// Переподключение с Last-Event-ID = seq первого факта догоняет второе изменение.
	again, err := c.service.Subscribe(ctx, seq, "")
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	cctx, ccancel := context.WithTimeout(ctx, time.Second)
	defer ccancel()
	if ch, err := again.Next(cctx); err != nil || ch.Seq != res2.Seqs[0] {
		t.Fatalf("догон по Last-Event-ID: %+v %v", ch, err)
	}
}

// Атомарность Append (AD-45): сбой выхода потребителя откатывает записи,
// эффекты движка и курсор вместе.
func TestAppendEffectsAtomic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c := newCore(t, ctx)
	id := kernel.UUIDv5(constants.NsAnt, "slice-test-atomic")
	p, err := c.codec.Encode(ctx, engineapp.Out{EventID: id, Type: catalog.InspectionResultRecorded, Kind: catalog.KindFact,
		Stream: "item:ENT01:I-9", ItemID: "ENT01:I-9", OccurredAt: time.Now(), Data: map[string]string{"outcome": "defect_indicated"}})
	if err != nil {
		t.Fatal(err)
	}
	boom := errors.New("сбой выхода")
	rq := appjournal.AppendRequest{
		Batch: []appjournal.Pending{p},
		Effects: []appjournal.Effect{
			engineapp.ProjectionPut{Name: engineapp.ItemStateProjection, Key: "ENT01:I-9", ItemID: "ENT01:I-9", Value: json.RawMessage(`{}`)},
			engineapp.Notify{Changes: []engineapp.Change{{Entity: platform.EntityItem, ID: "ENT01:I-9", Seq: 1}}},
		},
		Consumer: &appjournal.CursorAdvance{Name: "test", Partition: appjournal.GlobalPartition, Seq: 1},
		Project:  func(context.Context, appjournal.AppendResult) error { return boom },
	}
	if _, err := c.journal.Append(ctx, rq); !errors.Is(err, boom) {
		t.Fatalf("ждали сбой выхода: %v", err)
	}
	if es, _ := c.journal.Read(ctx, appjournal.ReadQuery{}); len(es) != 0 {
		t.Fatalf("записи откатились: %d", len(es))
	}
	if _, ok, _ := c.engine.Get(ctx, engineapp.ItemStateProjection, "ENT01:I-9"); ok {
		t.Fatal("проекция откатилась")
	}
	if n, _ := c.engine.Tail(ctx); n != 0 {
		t.Fatalf("журнал изменений откатился: %d", n)
	}
	if cur, _ := c.journal.Cursor(ctx, "test", appjournal.GlobalPartition); cur != 0 {
		t.Fatalf("курсор откатился: %d", cur)
	}
	// Эффект, который никто не применяет, — ошибка записи, а не тихий пропуск.
	rq.Project = nil
	rq.Effects = append(rq.Effects, unknownEffect{})
	if _, err := c.journal.Append(ctx, rq); !errors.Is(err, appjournal.ErrInvalidEntry) {
		t.Fatalf("неизвестный эффект: %v", err)
	}
	// Без сбоя — всё вместе.
	rq.Effects = rq.Effects[:2]
	if _, err := c.journal.Append(ctx, rq); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := c.engine.Get(ctx, engineapp.ItemStateProjection, "ENT01:I-9"); !ok {
		t.Fatal("проекция записана")
	}
	if cur, _ := c.journal.Cursor(ctx, "test", appjournal.GlobalPartition); cur != 1 {
		t.Fatalf("курсор: %d", cur)
	}
}

type unknownEffect struct{}

func (unknownEffect) EffectKind() string { return "test.unknown" }
