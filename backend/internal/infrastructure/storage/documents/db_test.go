package documents_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	app "ant/internal/application/documents"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	ncapp "ant/internal/application/nonconformity"
	"ant/internal/application/nonconformity/nctest"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	dom "ant/internal/domain/documents"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	storage "ant/internal/infrastructure/storage/documents"
	enginestore "ant/internal/infrastructure/storage/engine"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

// Позвоночник на своей БД (make dev-db): факты и решения через
// journal.Append → воркер сворачивает изделие настоящей свёрткой с
// нормативным слоем documents (встроенная копия normative/) → реакции
// document.version.drafted и document.route.closed, проекция documents.item —
// одной транзакцией (AD-45); решение режима 4 исполняется только после
// закрытия маршрута подписей (AD-43).
const parts = 2

type dbWorld struct {
	t       *testing.T
	ctx     context.Context
	journal *journalstore.Store
	leases  *journalstore.Leases
	engine  *enginestore.Store
	codec   *engineapp.Codec
	worker  *engineapp.WorkerService
	docs    *app.Service
	nc      *ncapp.Service
}

func newDBWorld(t *testing.T) *dbWorld {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)
	p := journaltest.NewDB(t).AppPool(t)
	clock := journaltest.SysClock{}
	w := &dbWorld{t: t, ctx: ctx, journal: journalstore.NewStore(p, clock), leases: journalstore.NewLeases(p, clock), engine: &enginestore.Store{Pool: p}}
	t.Cleanup(w.engine.Close)
	w.codec = &engineapp.Codec{Store: w.journal, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: dj.ZeroLink.String(), Partitions: parts}
	env, err := storage.SeedEnv(dom.VerificationDemo)
	if err != nil {
		t.Fatal(err)
	}
	reg := engineapp.NewRegistry()
	if err := app.RegisterProjections(reg); err != nil {
		t.Fatal(err)
	}
	bundles := app.Bundles{Env: env}
	w.worker = engineapp.NewWorker(engineapp.WorkerConfig{Codec: w.codec, Projections: reg, Bundles: bundles, Fold: nctest.DraftingFold})
	w.docs = app.NewService(app.WithDeps(app.Deps{Journal: w.journal, Codec: w.codec, Bundles: bundles, Fold: nctest.DraftingFold, Env: env}),
		app.WithConfig(app.Config{DomainBuild: dj.ZeroLink.String(), Partitions: parts}))
	w.nc = ncapp.NewService(ncapp.WithDeps(ncapp.Deps{Journal: w.journal, Codec: w.codec, Bundles: bundles, Fold: nctest.DraftingFold, Routes: ncapp.PendingRoutes{}}),
		ncapp.WithConfig(ncapp.Config{DomainBuild: dj.ZeroLink.String(), Partitions: parts}))
	return w
}

func (w *dbWorld) put(ps ...appjournal.Pending) {
	w.t.Helper()
	if _, err := w.journal.Append(w.ctx, appjournal.AppendRequest{Batch: ps}); err != nil {
		w.t.Fatal(err)
	}
}

// process — воркер сворачивает изделие до последней записи (под арендой партиции).
func (w *dbWorld) process(item string) {
	w.t.Helper()
	es, err := w.journal.Read(w.ctx, appjournal.ReadQuery{ItemID: item})
	if err != nil || len(es) == 0 {
		w.t.Fatalf("вход изделия: %d %v", len(es), err)
	}
	last := es[len(es)-1]
	part := kernel.PartitionOf(item, parts)
	f, ok, err := w.leases.Acquire(w.ctx, appjournal.PartitionLease(part), "documents-db-test", 30*time.Second)
	if err != nil || !ok {
		w.t.Fatalf("аренда партиции: %v %v", ok, err)
	}
	if err := w.worker.Process(w.ctx, engineapp.Partition{Number: part, Epoch: f.Epoch}, []engineapp.Work{{ItemID: item, UpToSeq: int64(last.Seq), Trigger: last}}); err != nil {
		w.t.Fatal(err)
	}
}

func (w *dbWorld) streams(t catalog.Type) []string {
	es, err := w.journal.Read(w.ctx, appjournal.ReadQuery{EventType: string(t), Limit: 1000})
	if err != nil {
		w.t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Stream)
	}
	return out
}

func as(person, role string) context.Context {
	return platform.WithPrincipal(context.Background(), platform.Principal{PersonID: person, Role: role})
}

func hdr() platform.CommandHeader { return platform.CommandHeader{CommandID: nctest.ID()} }

func TestPostgresSpine(t *testing.T) {
	w := newDBWorld(t)
	nctest.Partitions = parts
	const item = "FL:0201"
	at := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Millisecond)
	w.put(nctest.RecordP(catalog.OperationRunStarted, item, at, map[string]any{"operation_run_id": "RUN-1", "operation_code": "020",
		"step_key": "welding.weld", "equipment_id": "WELD-1", "operator_id": "W21"}, parts),
		nctest.RecordP(catalog.OperationRunFinished, item, at.Add(20*time.Minute), map[string]any{"operation_run_id": "RUN-1", "completion": "completed"}, parts),
		nctest.RecordP(catalog.InspectionResultRecorded, item, at.Add(30*time.Minute), map[string]any{"method": "camera", "phase": "after_operation",
			"outcome": "defect_indicated", "processing_state": "completed", "step_key": "welding.weld", "zone_ids": []string{"Z-1"}}, parts))
	w.process(item)

	l, err := w.nc.List(w.ctx, ncapp.NCFilter{ItemID: item}, platform.Moment{}, platform.Page{})
	if err != nil || len(l.Items) != 1 {
		t.Fatalf("несоответствия: %+v %v", l, err)
	}
	ncID := l.Items[0].NCID
	card, _ := w.nc.Card(w.ctx, ncID, platform.Moment{})
	ins := as("INS-01", "quality_inspector")
	if _, err := w.nc.Confirm(ins, ncID, ncapp.ConfirmNonconformity{CommandHeader: hdr(), SignalIDs: []string{card.Evidence.Signals[0].SignalID},
		Severity: "major", Reason: ncapp.NCReason{Text: "Прожог"}}); err != nil {
		t.Fatal(err)
	}
	w.process(item)
	if _, err := w.nc.GrantConcession(as("HQC-01", "head_of_qc"), ncapp.GrantConcession{CommandHeader: hdr(), ConcessionID: "CON-21", Title: "Катет шва",
		Kind: "use_as_is", ScopeItemIDs: []string{item}, Limit: 1, Reason: ncapp.NCReason{Text: "ТУ п. 4.2"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.nc.SetDisposition(ins, ncID, ncapp.SetDisposition{CommandHeader: hdr(), Disposition: "use_as_is", ConcessionID: "CON-21",
		Reason: ncapp.NCReason{Text: "в допуске"}}); err != nil {
		t.Fatal(err)
	}
	w.process(item)
	docID := "ncd-" + ncID
	if !slices.Contains(w.streams(catalog.DocumentVersionDrafted), "document:"+docID) {
		t.Fatal("нет document.version.drafted решения в журнале")
	}
	if c, _ := w.nc.Card(w.ctx, ncID, platform.Moment{}); c.Axes.Disposition != "none" {
		t.Fatalf("режим 4 исполнен без подписей: %+v", c.Axes)
	}
	d, err := w.docs.Document(w.ctx, docID, 0, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range []struct{ who, role string }{{"TEC-01", "technologist"}, {"HQC-01", "head_of_qc"}} {
		if _, err := w.docs.RecordSignature(as(s.who, s.role), docID, app.RecordSignature{CommandHeader: hdr(), Version: d.Version, Stage: i + 2}); err != nil {
			t.Fatalf("%s: %v", s.who, err)
		}
		w.process(item)
	}
	if !slices.Contains(w.streams(catalog.DocumentRouteClosed), "document:"+docID) {
		t.Fatal("нет document.route.closed в журнале")
	}
	if c, _ := w.nc.Card(w.ctx, ncID, platform.Moment{}); c.Axes.Disposition != "use_as_is" || c.ApprovalsStatus == nil || *c.ApprovalsStatus != "route_closed" {
		t.Fatalf("после закрытия маршрута: %+v %v", c.Axes, c.ApprovalsStatus)
	}
	// Проекция изделия: документы и счётчик «собрано из истории».
	raw, ok, err := w.engine.Get(w.ctx, app.ItemProjection, item)
	if err != nil || !ok {
		t.Fatalf("проекция documents.item: %v %v", ok, err)
	}
	var sum app.ItemSummary
	if err := json.Unmarshal(raw, &sum); err != nil || sum.CollectedFromHistory != 3 || sum.ManualEntries != 0 {
		t.Fatalf("проекция: %s %v", raw, err)
	}
}
