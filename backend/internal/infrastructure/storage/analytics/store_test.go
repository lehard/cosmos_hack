package analytics_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	analyticsapp "ant/internal/application/analytics"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	storage "ant/internal/infrastructure/storage/analytics"
	enginestore "ant/internal/infrastructure/storage/engine"
	store "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/feed"
	jt "ant/internal/infrastructure/storage/journal/journaltest"
)

// Показатели на своей БД (make dev-db): журнал на Postgres, настоящий
// воркер над WorkFeed заменяет вклады изделия в engine.contributions в
// транзакции Append (AD-45), проектор пишет глобальные проекции, live-сервис
// читает их через storage/analytics. Без БД тест пропускается.

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

type clock struct{ t time.Time }

func (c clock) Now(context.Context) (time.Time, error) { return c.t, nil }

type env struct {
	t      *testing.T
	ctx    context.Context
	j      *store.Store
	wf     *feed.WorkFeed
	part   engineapp.Partition
	codec  *engineapp.Codec
	worker *engineapp.WorkerService
	proj   *engineapp.Projector
	reg    *engineapp.Registry
	seq    int64
	svc    *analyticsapp.Service
}

func newEnv(t *testing.T) *env {
	t.Helper()
	d := jt.NewDB(t)
	pool := d.AppPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	j := store.NewStore(pool, jt.SysClock{})
	leases := store.NewLeases(pool, jt.SysClock{})
	sig := store.NewListener(pool, nil)
	go func() { _ = sig.Run(ctx) }()
	wf := feed.NewWorkFeed(j, leases, sig, 1, feed.Options{Holder: "t", TTL: 30 * time.Second})
	parts, err := wf.Partitions(ctx)
	if err != nil || len(parts) != 1 {
		t.Fatalf("партиции: %v %v", parts, err)
	}
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: dj.ZeroLink.String(), Partitions: 1}
	reg := engineapp.NewRegistry()
	if err := analyticsapp.Register(reg); err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, ctx: ctx, j: j, wf: wf, part: parts[0], codec: codec, reg: reg,
		worker: engineapp.NewWorker(engineapp.WorkerConfig{Feed: wf, Codec: codec, Projections: reg}),
		proj:   &engineapp.Projector{Codec: codec, Store: &enginestore.Store{Pool: pool}, Registry: reg}}
	e.svc = analyticsapp.NewService(analyticsapp.WithStore(storage.New(pool)), analyticsapp.WithClock(clock{t0.Add(4 * time.Hour)}))
	return e
}

func (e *env) fact(itemID, id string, typ catalog.Type, at time.Duration, data string) {
	e.t.Helper()
	info, _ := catalog.Lookup(typ)
	stream := "item:" + itemID
	if itemID == "" {
		stream = "global"
	}
	p, err := e.codec.Encode(e.ctx, engineapp.Out{EventID: uid(id), Type: typ, Kind: info.Kind, Stream: stream, ItemID: itemID, OccurredAt: t0.Add(at), Data: json.RawMessage(data)})
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := e.j.Append(e.ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
		e.t.Fatal(err)
	}
}

// settle — один проход воркера по новой работе и проектора по новым записям.
func (e *env) settle() {
	e.t.Helper()
	for {
		ctx, cancel := context.WithTimeout(e.ctx, 500*time.Millisecond)
		works, err := e.wf.Next(ctx, e.part)
		cancel()
		if err != nil || len(works) == 0 {
			break
		}
		if err := e.worker.Process(e.ctx, e.part, works); err != nil {
			e.t.Fatal(err)
		}
	}
	entries, err := e.j.Read(e.ctx, appjournal.ReadQuery{AfterSeq: e.seq, Limit: 1000})
	if err != nil {
		e.t.Fatal(err)
	}
	if len(entries) == 0 {
		return
	}
	e.seq = int64(entries[len(entries)-1].Seq)
	for _, g := range e.reg.Globals() {
		var batch []jc.JournalEntry
		batch = append(batch, entries...)
		rq, err := e.proj.Apply(e.ctx, g, batch)
		if err != nil {
			e.t.Fatal(err)
		}
		if len(rq.Effects) > 0 {
			if _, err := e.j.Append(e.ctx, rq); err != nil {
				e.t.Fatal(err)
			}
		}
	}
}

func (e *env) weld(n int, at time.Duration, resolve bool) {
	id := fmt.Sprintf("ENT01:FL-%03d", n)
	ev := func(s string) string { return fmt.Sprintf("%s-%d", s, n) }
	e.fact(id, ev("reg"), catalog.ItemItemRegistered, at, fmt.Sprintf(`{"item_id":%q,"item_type_id":"FL-100","item_revision":"A","process_version_hash":"x","normative_rev":"r1"}`, id))
	e.fact(id, ev("start"), catalog.OperationRunStarted, at+5*time.Minute, fmt.Sprintf(`{"operation_run_id":"RUN-%d","operation_code":"welding","step_key":"welding.weld","station_id":"ST-2","equipment_id":"IS-2","operator_id":"W21"}`, n))
	e.fact(id, ev("fin"), catalog.OperationRunFinished, at+35*time.Minute, fmt.Sprintf(`{"operation_run_id":"RUN-%d","completion":"completed"}`, n))
	e.fact(id, ev("kt3"), catalog.InspectionResultRecorded, at+40*time.Minute, fmt.Sprintf(`{"observation_id":"OBS-%d","method":"visual","phase":"after_operation","step_key":"welding.kt3_camera","operation_run_id":"RUN-%d","outcome":"no_defect_indicated","processing_state":"completed"}`, n, n))
	e.fact(id, ev("pres"), catalog.ItemPresentationRecorded, at+45*time.Minute, `{"step_key":"welding.zt3_acceptance","presentation_no":1,"presented_to":"qc","presented_by":"M01"}`)
	if resolve {
		e.fact(id, ev("zt3"), catalog.DecisionPresentationResolved, at+55*time.Minute, `{"step_key":"welding.zt3_acceptance","closing_point":"ZT-3","resolution":"accept","presentation_no":1,"method_event_ids":[]}`)
	}
}

func (e *env) overview() analyticsapp.AnalyticsOverview {
	e.t.Helper()
	ov, err := e.svc.Overview(e.ctx, analyticsapp.PeriodQuery{Kind: "shift"}, platform.Moment{})
	if err != nil {
		e.t.Fatal(err)
	}
	return ov
}

func val(ov analyticsapp.AnalyticsOverview, id string) int64 {
	for _, r := range ov.Items {
		if r.MetricID == id {
			return r.Total.Value
		}
	}
	return -1
}

func TestAnalyticsOnPostgres(t *testing.T) {
	e := newEnv(t)
	for i := range 4 {
		e.weld(i, time.Duration(i)*10*time.Minute, true)
	}
	e.fact("ENT01:FL-001", "rt-1", catalog.InspectionResultRecorded, 60*time.Minute, `{"observation_id":"RT-1","method":"radiography","phase":"after_operation","step_key":"welding.kt3_radiography","operation_run_id":"RUN-1","outcome":"defect_indicated","processing_state":"completed","defects":[{"defect_type_code":"POR","zone_id":"W1","location":"12h","severity":"major"}]}`)
	e.fact("ENT01:FL-001", "nc-1", catalog.DecisionNonconformityConfirmed, 70*time.Minute, `{"nc_id":"NC-1","signal_ids":["S-1"],"severity":"major","reason":{"text":"пора"}}`)
	e.fact("", "eq-1", catalog.EquipmentStateChanged, 80*time.Minute, `{"equipment_id":"IS-2","execution":"stopped","controller_mode":"automatic","condition":"fault"}`)
	e.fact("", "eq-2", catalog.EquipmentStateChanged, 95*time.Minute, `{"equipment_id":"IS-2","execution":"running","controller_mode":"automatic","condition":"normal"}`)
	e.settle()
	ov := e.overview()
	for id, want := range map[string]int64{"inspected_items": 4, "items_with_confirmed_nc": 1, "confirmed_defects": 1, "equipment_downtime": 15, "comparable_runs": 4, "first_pass_yield": 7500} {
		if got := val(ov, id); got != want {
			t.Errorf("%s = %d, ожидалось %d", id, got, want)
		}
	}
	// Каждое число раскрывается до записей журнала.
	for _, r := range ov.Items {
		dd, err := e.svc.Drilldown(e.ctx, r.MetricID, "", analyticsapp.PeriodQuery{Kind: "shift"}, platform.Moment{}, platform.Page{Limit: 500})
		if err != nil || dd.Total.Value != r.Total.Value || dd.Total.Unit != r.Total.Unit {
			t.Fatalf("%s: раскрытие %+v ≠ %+v (%v)", r.MetricID, dd.Total, r.Total, err)
		}
		for _, c := range dd.Items {
			for _, id := range c.SourceEventIDs {
				if got, err := e.j.Read(e.ctx, appjournal.ReadQuery{Limit: 1000}); err != nil || !hasEvent(got, id) {
					t.Fatalf("%s: запись %s не найдена в журнале", r.MetricID, id)
				}
			}
		}
	}

	// Повтор: те же факты ещё раз от источника (новые event_id) — показатели те же.
	before, _ := json.Marshal(ov)
	e.fact("ENT01:FL-002", "start-2-bis", catalog.OperationRunStarted, 25*time.Minute, `{"operation_run_id":"RUN-2","operation_code":"welding","step_key":"welding.weld","station_id":"ST-2","equipment_id":"IS-2","operator_id":"W21"}`)
	e.fact("ENT01:FL-002", "kt3-2-bis", catalog.InspectionResultRecorded, 60*time.Minute, `{"observation_id":"OBS-2","method":"visual","phase":"after_operation","step_key":"welding.kt3_camera","operation_run_id":"RUN-2","outcome":"no_defect_indicated","processing_state":"completed"}`)
	e.fact("ENT01:FL-001", "rt-1-bis", catalog.InspectionResultRecorded, 61*time.Minute, `{"observation_id":"RT-1b","method":"radiography","phase":"after_operation","step_key":"welding.kt3_radiography","operation_run_id":"RUN-1","outcome":"defect_indicated","processing_state":"completed","defects":[{"defect_type_code":"POR","zone_id":"W1","location":"12h","severity":"major"}]}`)
	e.settle()
	after := e.overview()
	for _, id := range []string{"inspected_items", "items_with_confirmed_nc", "confirmed_defects", "comparable_runs", "equipment_downtime", "first_pass_yield"} {
		if val(after, id) != val(ov, id) {
			t.Errorf("повтор изменил %s: %d → %d", id, val(ov, id), val(after, id))
		}
	}
	_ = before

	// Позднее событие: дефект изделия 3 случился в 50 мин, пришёл сейчас.
	e.fact("ENT01:FL-003", "rt-3", catalog.InspectionResultRecorded, 72*time.Minute, `{"observation_id":"RT-3","method":"radiography","phase":"after_operation","step_key":"welding.kt3_radiography","operation_run_id":"RUN-3","outcome":"defect_indicated","processing_state":"completed","defects":[{"defect_type_code":"POR","zone_id":"W2","location":"3h","severity":"major"}]}`)
	e.fact("ENT01:FL-003", "nc-3", catalog.DecisionNonconformityConfirmed, 150*time.Minute, `{"nc_id":"NC-3","signal_ids":["S-3"],"severity":"major","reason":{"text":"пора"}}`)
	e.settle()
	late := e.overview()
	if val(late, "confirmed_defects") != 2 || val(late, "items_with_confirmed_nc") != 2 || val(late, "inspected_items") != 4 || val(late, "first_pass_yield") != 5000 {
		t.Errorf("позднее событие: дефекты %d, изделия с НС %d, проверено %d, с первого раза %d",
			val(late, "confirmed_defects"), val(late, "items_with_confirmed_nc"), val(late, "inspected_items"), val(late, "first_pass_yield"))
	}
}

// uid — UUID записи теста по имени (схема записи требует UUID).
func uid(name string) string { return kernel.UUIDv5(constants.NsAnt, "analytics-test/"+name) }

func hasEvent(es []jc.JournalEntry, id string) bool {
	for _, e := range es {
		if e.EventID == id {
			return true
		}
	}
	return false
}

// «Специалист заснул» на своей БД: узел точки предъявления — ограничение
// линии и аномалия (FR-5).
func TestSpecialistAsleepOnPostgres(t *testing.T) {
	e := newEnv(t)
	for i := range 7 {
		e.weld(i, time.Duration(i)*10*time.Minute, i == 0)
	}
	e.settle()
	nc, err := e.svc.NodeCounters(e.ctx, "", analyticsapp.PeriodQuery{Kind: "shift"}, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if nc.Bottleneck == nil || nc.Bottleneck.StepKey != "welding.zt3_acceptance" {
		t.Fatalf("ограничение линии: %+v", nc.Bottleneck)
	}
	found := map[string]bool{}
	for _, a := range nc.Anomalies {
		if a.StepKey == "welding.zt3_acceptance" {
			found[a.Kind] = true
		}
	}
	if !found["queue_above_norm"] || !found["wait_above_norm"] {
		t.Fatalf("аномалии: %+v", nc.Anomalies)
	}
}
