package machinelogs_test

import (
	"context"
	"slices"
	"testing"
	"time"

	crossapp "ant/internal/application/crossitem"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	mlapp "ant/internal/application/machinelogs"
	"ant/internal/application/machinelogs/mltest"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	dcross "ant/internal/domain/crossitem"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	ml "ant/internal/domain/machinelogs"
	enginestore "ant/internal/infrastructure/storage/engine"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/feed"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

// Сквозной путь machinelogs на своей БД (make dev-db): журнал эпика 04,
// хранение движка эпика 07, стадия, воркер с настоящей свёрткой и проектор;
// операции чтения — живая реализация над проекциями в Postgres.

const partitions = 4

var t0 = time.Date(2026, 9, 26, 7, 0, 0, 0, time.UTC)

// Заготовка функции-намерения nonconformity (эпик 21): модуль и тип —
// переменные (emitcheck: запись от имени nonconformity строит только тест).
var (
	ncModule kernel.Module = "nonconformity"
	ncType                 = catalog.DecisionNonconformityRegistered
)

func fakeRegistrar(q ml.NCRequest) (kernel.Addressed, error) {
	return kernel.NewAddressed(ncModule, ncType, "item:"+q.ItemID, q.Key(),
		map[string]any{"nc_id": q.NCID, "violation_window_event_id": q.WindowEventID, "operation_run_id": q.OperationRunID, "step_key": q.StepKey},
		q.CauseRecords()...)
}

type core struct {
	journal *journalstore.Store
	engine  *enginestore.Store
	codec   *engineapp.Codec
	reg     *engineapp.Registry
	svc     *mlapp.Service
}

// run — факты сценария в журнал, стадия, воркер (до обработки всех изделий),
// проектор; stage — шаг стадии (nil — сборка crossitem.Fold).
func run(t *testing.T, sc mltest.Scenario, stage func(dcross.Stage, kernel.Record) (dcross.Stage, []kernel.Addressed)) *core {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	p := journaltest.NewDB(t).AppPool(t)
	clock := journaltest.SysClock{}
	c := &core{journal: journalstore.NewStore(p, clock), engine: &enginestore.Store{Pool: p}, reg: engineapp.NewRegistry()}
	t.Cleanup(c.engine.Close)
	leases, listener := journalstore.NewLeases(p, clock), journalstore.NewListener(p, nil)
	go func() { _ = listener.Run(ctx) }()
	c.codec = &engineapp.Codec{Store: c.journal, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1",
		DomainBuild: dj.ZeroLink.String(), Partitions: partitions}
	if err := mlapp.RegisterProjections(c.reg, ml.Env{}); err != nil {
		t.Fatal(err)
	}
	for _, o := range sc.Facts {
		pd, err := c.codec.Encode(ctx, o)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{pd}}); err != nil {
			t.Fatal(err)
		}
	}
	// Стадия (роль crossitem) — одна пачка всего журнала.
	sr := &crossapp.StageRunner{Codec: c.codec, Store: c.engine, Fold: stage}
	_, rq, err := sr.Apply(ctx, dcross.Stage{}, journaltest.ReadAll(t, c.journal, "main"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.journal.Append(ctx, rq); err != nil {
		t.Fatal(err)
	}
	// Воркер (роль worker) — пока не свёрнуты все изделия сценария.
	wf := feed.NewWorkFeed(c.journal, leases, listener, partitions, feed.Options{Holder: "ml-test", TTL: 3 * time.Second})
	wk := engineapp.NewWorker(engineapp.WorkerConfig{Feed: wf, Codec: c.codec, Projections: c.reg, Refresh: 200 * time.Millisecond})
	wctx, wcancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { _ = wk.Run(wctx); close(done) }()
	items := map[string]bool{}
	for it := range sc.Runs {
		items[it] = true
	}
	for len(items) > 0 {
		for it := range items {
			if _, ok, err := c.engine.Get(ctx, mlapp.ProjectionItemRuns, it); err == nil && ok {
				delete(items, it)
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("воркер не свернул изделия: %v", items)
		case <-time.After(50 * time.Millisecond):
		}
	}
	wcancel()
	<-done
	// Проектор (роль projector): глобальные проекции по журналу.
	pr := &engineapp.Projector{Codec: c.codec, Store: c.engine, Registry: c.reg}
	all := journaltest.ReadAll(t, c.journal, "main")
	for _, g := range c.reg.Globals() {
		rq, err := pr.Apply(ctx, g, all)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.journal.Append(ctx, rq); err != nil {
			t.Fatal(err)
		}
	}
	c.svc = mlapp.NewLiveService(c.engine, engineapp.StateQueries{Codec: c.codec}, ml.Env{})
	return c
}

// FR-147, FR-148: ручная подача 130 % видна как отклонение в профиле
// выполнения операции — на Postgres.
func TestPostgresRunProfileFeed130(t *testing.T) {
	c := run(t, mltest.CNCTwoDeviations(t0, ""), nil)
	ctx := context.Background()
	p, err := c.svc.RunProfile(ctx, "RUN-C-2", platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range p.Events {
		if e.Layer == "deviation" && e.Params["deviation_kind"] == "manual_override" && e.Params["value"] == "130 %" {
			found = true
		}
	}
	if !found {
		t.Fatalf("подача 130 %% в профиле: %+v", p.Events)
	}
	// Профиль на момент (та же свёртка над префиксом журнала, AD-22).
	at := t0.Add(27 * time.Minute)
	was, err := c.svc.RunProfile(ctx, "RUN-C-2", platform.Moment{AsOf: &at})
	if err != nil || was.FinishedAt != nil {
		t.Fatalf("профиль до конца выполнения: %+v %v", was, err)
	}
	eq, err := c.svc.EquipmentByID(ctx, "CNC-1", platform.Moment{})
	if err != nil || eq.ToolLifeUsed == nil || *eq.ToolLifeUsed != 73 {
		t.Fatalf("оборудование: %+v %v", eq, err)
	}
}

// FR-151: «сварка вне режима» — все 6 изделий окна получают несоответствие,
// 3 из них без найденного дефекта — на Postgres.
func TestPostgresSpecialProcessSixItems(t *testing.T) {
	sc := mltest.WeldingOutOfRegime(t0, "")
	f := ml.StageWith(ml.StagePorts{Register: fakeRegistrar})
	stage := func(s dcross.Stage, r kernel.Record) (dcross.Stage, []kernel.Addressed) {
		return dcross.Settle(s, r, func(s dcross.Stage, r kernel.Record) (dcross.Stage, []kernel.Addressed) {
			var a []kernel.Addressed
			s.Machinelogs, a = f(s.Machinelogs, r)
			return s, a
		})
	}
	c := run(t, sc, stage)
	ctx := context.Background()
	ncs, err := c.journal.Read(ctx, appjournal.ReadQuery{EventType: string(catalog.DecisionNonconformityRegistered)})
	if err != nil {
		t.Fatal(err)
	}
	var items []string
	for _, e := range ncs {
		items = append(items, *e.ItemID)
	}
	slices.Sort(items)
	if !slices.Equal(items, sc.WindowItems) {
		t.Fatalf("несоответствия изделиям окна: %v", items)
	}
	noDefect := 0
	for _, it := range items {
		if !slices.Contains(sc.WithDefect, it) {
			noDefect++
		}
	}
	if noDefect != 3 {
		t.Fatalf("без найденного дефекта: %d", noDefect)
	}
	v, err := c.svc.Violations(ctx, platform.Moment{})
	if err != nil || len(v.Items) != 1 || len(v.Items[0].Items) != 6 || len(v.Items[0].Nonconformities) != 6 {
		t.Fatalf("окно нарушения: %+v %v", v, err)
	}
	p, err := c.svc.RunProfile(ctx, sc.Runs[sc.WindowItems[4]], platform.Moment{})
	if err != nil || !p.Violation {
		t.Fatalf("профиль выполнения в окне: %+v %v", p, err)
	}
}
