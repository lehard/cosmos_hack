package quality_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	appquality "ant/internal/application/quality"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	enginestore "ant/internal/infrastructure/storage/engine"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
	storage "ant/internal/infrastructure/storage/quality"
)

// Сквозной путь модуля quality на своей БД (make dev-db): факты контроля и
// паспорт анализатора через journal.Append → воркер сворачивает изделие
// настоящей свёрткой с нормативным слоем (встроенная копия normative/) →
// реакции quality.*, проекция quality.item и вклады показателей — одной
// транзакцией (AD-45) → живые операции quality читают проекции.
type dbWorld struct {
	t       *testing.T
	ctx     context.Context
	journal *journalstore.Store
	leases  *journalstore.Leases
	engine  *enginestore.Store
	codec   *engineapp.Codec
	reg     *engineapp.Registry
	worker  *engineapp.WorkerService
	n       int
}

const parts = 2

func newDBWorld(t *testing.T) *dbWorld {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	p := journaltest.NewDB(t).AppPool(t)
	clock := journaltest.SysClock{}
	w := &dbWorld{t: t, ctx: ctx, journal: journalstore.NewStore(p, clock), leases: journalstore.NewLeases(p, clock), engine: &enginestore.Store{Pool: p}}
	t.Cleanup(w.engine.Close)
	w.codec = &engineapp.Codec{Store: w.journal, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1",
		DomainBuild: dj.ZeroLink.String(), Partitions: parts}
	w.reg = engineapp.NewRegistry()
	if err := appquality.Register(w.reg); err != nil {
		t.Fatal(err)
	}
	env, err := storage.SeedEnv("seed-v1")
	if err != nil {
		t.Fatal(err)
	}
	w.worker = engineapp.NewWorker(engineapp.WorkerConfig{Codec: w.codec, Projections: w.reg,
		Bundles: appquality.Bundles{Env: env, Passports: storage.JournalPassports{Journal: w.journal, Codec: w.codec}}})
	return w
}

// put — факт или решение в журнал, как его записал бы приём (эпик 06).
func (w *dbWorld) put(typ catalog.Type, stream, item string, data any) int64 {
	w.t.Helper()
	w.n++
	info, _ := catalog.Lookup(typ)
	id := kernel.UUIDv5(constants.NsAnt, fmt.Sprintf("quality-db-test\x1f%s\x1f%d", w.t.Name(), w.n))
	p, err := w.codec.Encode(w.ctx, engineapp.Out{EventID: id, Type: typ, Kind: info.Kind, Stream: stream, ItemID: item,
		OccurredAt: time.Date(2026, 9, 26, 8, w.n, 0, 0, time.UTC), Correlation: id, Data: data})
	if err != nil {
		w.t.Fatal(err)
	}
	res, err := w.journal.Append(w.ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}})
	if err != nil {
		w.t.Fatal(err)
	}
	return res.Seqs[0]
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
	f, ok, err := w.leases.Acquire(w.ctx, appjournal.PartitionLease(part), "quality-db-test", 30*time.Second)
	if err != nil || !ok {
		w.t.Fatalf("аренда партиции: %v %v", ok, err)
	}
	if err := w.worker.Process(w.ctx, engineapp.Partition{Number: part, Epoch: f.Epoch}, []engineapp.Work{{ItemID: item, UpToSeq: int64(last.Seq), Trigger: last}}); err != nil {
		w.t.Fatal(err)
	}
}

// project — глобальные проекции модуля по всему журналу (роль projector).
func (w *dbWorld) project() {
	w.t.Helper()
	all, err := w.journal.Read(w.ctx, appjournal.ReadQuery{Limit: 1000})
	if err != nil {
		w.t.Fatal(err)
	}
	pr := &engineapp.Projector{Codec: w.codec, Store: w.engine, Registry: w.reg}
	for _, g := range w.reg.Globals() {
		rq, err := pr.Apply(w.ctx, g, all)
		if err != nil {
			w.t.Fatal(err)
		}
		if _, err := w.journal.Append(w.ctx, rq); err != nil {
			w.t.Fatal(err)
		}
	}
}

func (w *dbWorld) records(item string, typ catalog.Type) []jc.JournalEntry {
	w.t.Helper()
	es, err := w.journal.Read(w.ctx, appjournal.ReadQuery{ItemID: item, EventType: string(typ)})
	if err != nil {
		w.t.Fatal(err)
	}
	return es
}

func (w *dbWorld) metric(name string) int64 {
	w.t.Helper()
	var v int64
	if err := w.engine.Pool.QueryRow(w.ctx, `SELECT COALESCE(SUM(value), 0) FROM engine.contributions WHERE metric = $1`, name).Scan(&v); err != nil {
		w.t.Fatal(err)
	}
	return v
}

func camera(outcome string, q, c int, defects ...map[string]any) map[string]any {
	d := map[string]any{"method": "camera", "phase": "after_operation", "outcome": outcome, "processing_state": "completed",
		"step_key": "welding.kt3_camera", "inspection_point": "KT-3", "zone_ids": []string{"W-1"}, "observation_quality_bp": q, "analyzer_confidence_bp": c,
		"stages":   []any{map[string]any{"stage": "localize", "version": "vqc-weld 2.3.1", "confidence_bp": c}},
		"versions": map[string]any{"recipe_ref": "kt3-weld@1", "analyzer_version": "vqc-weld 2.3.1", "contract_version": "1.0"}}
	if len(defects) > 0 {
		d["defects"] = defects
	}
	return d
}

func TestQualityOnPostgres(t *testing.T) {
	w := newDBWorld(t)
	w.put(catalog.AnalyzerPassportAdmitted, "analyzer_passport:PP-KT3", "", map[string]any{"passport_id": "PP-KT3", "stage": "active", "trust_level": 3,
		"recipe_ref": "kt3-weld@1", "document_id": "DOC-PP-KT3", "versions": map[string]any{"recipe_ref": "kt3-weld@1", "analyzer_version": "vqc-weld 2.3.1", "contract_version": "1.0"}})

	// Изделие А: плохой кадр — «оценка невозможна», повторный контроль.
	const a, b = "ENT01:F-031", "ENT01:F-032"
	w.put(catalog.OperationRunStarted, "item:"+a, a, map[string]any{"operation_run_id": "RUN-A", "operation_code": "WELD", "step_key": "welding.weld", "operator_id": "O17"})
	w.put(catalog.InspectionResultRecorded, "item:"+a, a, camera("no_defect_indicated", 2500, 9900))
	w.process(a)

	// Изделие Б: прожог с низкой уверенностью (критический вид) и повторные наблюдения.
	burn := map[string]any{"defect_type_code": "W-BURNTHRU", "zone_id": "W-1.U2", "severity": "unknown", "stage_confidence_bp": 1500}
	w.put(catalog.OperationRunStarted, "item:"+b, b, map[string]any{"operation_run_id": "RUN-B", "operation_code": "WELD", "step_key": "welding.weld", "operator_id": "O17"})
	w.put(catalog.InspectionResultRecorded, "item:"+b, b, camera("defect_indicated", 9000, 1500, burn))
	w.process(b)
	if got := w.metric(appquality.MetricDefects); got != 1 {
		t.Fatalf("дефектов: %d", got)
	}
	w.put(catalog.InspectionResultRecorded, "item:"+b, b, map[string]any{"method": "radiography", "phase": "after_operation", "outcome": "defect_indicated",
		"processing_state": "completed", "step_key": "welding.kt3_radiography", "zone_ids": []string{"W-1"}, "defects": []any{burn}})
	w.put(catalog.InspectionResultRecorded, "item:"+b, b, camera("defect_indicated", 9000, 2000, burn))
	w.process(b)

	// Повторные наблюдения не растят показатели (FR-37, AD-45).
	if d, obs, it := w.metric(appquality.MetricDefects), w.metric(appquality.MetricObservations), w.metric(appquality.MetricItemsWithDefect); d != 1 || obs != 4 || it != 1 {
		t.Fatalf("показатели: дефектов %d, наблюдений %d, изделий с дефектом %d", d, obs, it)
	}
	if n := len(w.records(b, catalog.QualityDefectIdentified)); n != 1 {
		t.Fatalf("quality.defect.identified: %d", n)
	}
	if n := len(w.records(b, catalog.QualityObservationLinked)); n != 2 {
		t.Fatalf("quality.observation.linked: %d", n)
	}
	sig := w.records(b, catalog.QualitySignalRaised)
	if len(sig) == 0 || sig[0].BasisSeq == nil {
		t.Fatalf("сигнал в журнале: %d", len(sig))
	}
	if n := len(w.records(a, catalog.QualitySignalRaised)); n != 1 {
		t.Fatalf("сигнал «оценка невозможна»: %d", n)
	}

	// Живые операции quality над проекциями.
	w.project()
	svc := appquality.NewService(appquality.WithStore(w.engine))
	m := platform.Moment{Axis: platform.AxisOccurred}
	cov, err := svc.Coverage(w.ctx, a, m)
	if err != nil || cov.QualityState != "unable_to_assess" {
		t.Fatalf("изделие А: %+v %v", cov, err)
	}
	list, err := svc.Signals(w.ctx, appquality.SignalFilter{}, m, platform.Page{})
	if err != nil || len(list.Items) != 2 {
		t.Fatalf("сигналы: %+v %v", list, err)
	}
	for _, s := range list.Items {
		if s.ItemID == b && (s.ReactionOutcome != "isolate" || s.Containment != "item_hold" || s.TrustLevel == nil || *s.TrustLevel != 3) {
			t.Fatalf("низкая уверенность × критическая тяжесть — блок: %+v", s)
		}
		if s.ItemID == a && (!s.UnableToAssess || s.ReactionOutcome != "manual_review") {
			t.Fatalf("плохой кадр — повторный контроль: %+v", s)
		}
	}
	defs, err := svc.Defects(w.ctx, "", m, platform.Page{})
	if err != nil || defs.DefectCount != 1 || defs.Items[0].Observations != 3 {
		t.Fatalf("дефекты: %+v %v", defs, err)
	}
}
