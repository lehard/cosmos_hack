package nonconformity_test

import (
	"context"
	"os"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	app "ant/internal/application/nonconformity"
	"ant/internal/application/nonconformity/nctest"
	notifapp "ant/internal/application/notifications"
	"ant/internal/application/platform"
	procapp "ant/internal/application/process"
	qualityapp "ant/internal/application/quality"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
	qualitystore "ant/internal/infrastructure/storage/quality"
)

// Показ SHOW-IS2: на ЗТ-3 изделие встаёт по результатам КТ-3 и рентгена, без
// записи предъявления мастером. Живой движок (воркер с нормативным слоем
// фланца): «Выполнено» → КТ-3/рентген → строка у контролёра и точка в
// presentation.read → «Принять» проходит гард → изделие уходит с точки.
func TestProcessGateWithoutPresentation(t *testing.T) {
	xml, err := os.ReadFile("../../../../normative/process/flange-process.bpmn")
	if err != nil {
		t.Fatal(err)
	}
	store := &procapp.MemVersions{}
	seed, err := procapp.EnsureSeed(context.Background(), store, xml, nctest.T0.Add(-72*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	bundles := notifapp.Bundles{Next: &procapp.Bundles{Store: store, TTL: time.Nanosecond}}
	w := nctest.NewWorld(t)
	reg := engineapp.NewRegistry()
	if err := app.RegisterProjections(reg); err != nil {
		t.Fatal(err)
	}
	w.Worker = engineapp.NewWorker(engineapp.WorkerConfig{Feed: w.Feed, Codec: w.Codec, Fold: engine.Fold, Bundles: bundles, Projections: reg})
	svc := app.NewService(app.WithDeps(app.Deps{Journal: w.J, Codec: w.Codec, Fold: engine.Fold, Bundles: bundles, Routes: app.DemoRoutes{},
		Now: func() time.Time { return w.Now }}), app.WithConfig(app.Config{Partitions: 1}))
	item := "ENT01:show-is2-20260921-1/I-9DE0BA5E"
	at := func(m int) time.Time { return nctest.T0.Add(time.Duration(m) * time.Minute) }
	w.Add(nctest.Record(catalog.ItemItemRegistered, item, at(0), map[string]any{"item_id": item, "item_type_id": "FL-100.00.000", "item_revision": "Б",
		"process_version_hash": seed.Hash, "normative_rev": "flange-1", "lot_ids": []string{"LOT-FL-1"}, "entry_step_key": "welding.edge_prep"}))
	w.Add(nctest.Record(catalog.OperationRunStarted, item, at(10), map[string]any{"operation_run_id": "01a0df29-0000-7000-8000-000000000001",
		"operation_code": "030", "step_key": "welding.weld", "station_id": "WP-WELD-1", "equipment_id": "IS-1", "operator_id": "W21"}))
	w.Add(nctest.Record(catalog.OperationRunFinished, item, at(50), map[string]any{"operation_run_id": "01a0df29-0000-7000-8000-000000000001", "completion": "completed"}))
	kt3 := nctest.Record(catalog.InspectionResultRecorded, item, at(60), map[string]any{"method": "camera", "phase": "after_operation",
		"step_key": "welding.kt3_camera", "inspection_point": "KT-3", "outcome": "no_defect_indicated", "processing_state": "completed",
		"operation_run_id": "01a0df29-0000-7000-8000-000000000001"})
	xray := nctest.Record(catalog.InspectionResultRecorded, item, at(63), map[string]any{"method": "radiography", "phase": "after_operation",
		"step_key": "welding.kt3_radiography", "outcome": "no_defect_indicated", "processing_state": "completed",
		"operation_run_id": "01a0df29-0000-7000-8000-000000000001"})
	w.Add(kt3, xray)
	w.Now = at(70)
	w.Settle()
	ctx := nctest.As("INS-01", "quality_inspector")
	q, err := svc.Queue(ctx, app.QueueFilter{Kind: "presentation"}, platform.Moment{}, platform.Page{})
	if err != nil || len(q.Items) != 1 || q.Items[0].ItemID != item || q.Items[0].StepKey != "welding.zt3_acceptance" {
		t.Fatalf("очередь контролёра: %+v %v", q.Items, err)
	}
	pv, err := svc.Presentation(ctx, item, platform.Moment{})
	if err != nil || pv.Presentation.ClosingPoint != "ZT-3" || pv.Presentation.StepKey != "welding.zt3_acceptance" || pv.Presentation.PresentationNo != 1 {
		t.Fatalf("точка предъявления: %+v %v", pv.Presentation, err)
	}
	if _, err := svc.ResolvePresentation(ctx, item, app.ResolvePresentation{CommandHeader: hdr(pv.BasisSeq), StepKey: pv.Presentation.StepKey,
		ClosingPoint: "ZT-3", Resolution: "accept", PresentationNo: pv.Presentation.PresentationNo,
		MethodEventIDs: []string{kt3.Entry.EventID, xray.Entry.EventID}}); err != nil {
		t.Fatalf("решение на ЗТ-3: %v", err)
	}
	w.Settle()
	q, _ = svc.Queue(ctx, app.QueueFilter{Kind: "presentation"}, platform.Moment{}, platform.Page{})
	if len(q.Items) != 0 {
		t.Fatalf("после решения точка ещё ждёт: %+v", q.Items)
	}
}

// Показ SHOW-IS2, Ф-002: КТ-3 «признаков нет» при кадре хуже порога карты
// (R-02, «оценка невозможна») → доп. проверка по изделию → рентген и повторный
// кадр в норме → Ф-002 на ЗТ-3 у контролёра, «Принять» проходит гард
// (доп. проверка не блокирует, сигнал «оценка невозможна» закрыт повторным кадром).
func TestRecheckThenGate(t *testing.T) {
	xml, err := os.ReadFile("../../../../normative/process/flange-process.bpmn")
	if err != nil {
		t.Fatal(err)
	}
	store := &procapp.MemVersions{}
	seed, err := procapp.EnsureSeed(context.Background(), store, xml, nctest.T0.Add(-72*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	qenv, err := qualitystore.SeedEnv("flange-1")
	if err != nil {
		t.Fatal(err)
	}
	bundles := notifapp.Bundles{Next: qualityapp.Bundles{Next: &procapp.Bundles{Store: store, TTL: time.Nanosecond}, Env: qenv}}
	w := nctest.NewWorld(t)
	reg := engineapp.NewRegistry()
	if err := app.RegisterProjections(reg); err != nil {
		t.Fatal(err)
	}
	w.Worker = engineapp.NewWorker(engineapp.WorkerConfig{Feed: w.Feed, Codec: w.Codec, Fold: engine.Fold, Bundles: bundles, Projections: reg})
	svc := app.NewService(app.WithDeps(app.Deps{Journal: w.J, Codec: w.Codec, Fold: engine.Fold, Bundles: bundles, Routes: app.DemoRoutes{},
		Now: func() time.Time { return w.Now }}), app.WithConfig(app.Config{Partitions: 1}))
	item := "ENT01:show-is2-20260921-1/I-F0020000"
	at := func(m int) time.Time { return nctest.T0.Add(time.Duration(m) * time.Minute) }
	run := "01a0df29-0000-7000-8000-000000000002"
	zones := []string{"W-1.U1", "W-1.U2", "W-1.U3", "W-1.U4", "W-1.U5", "W-1.U6", "W-1.U7", "W-1.U8"}
	versions := map[string]any{"analyzer_version": "vqc-weld 2.3.1", "contract_version": "1.0", "recipe_ref": "kt3-weld@1", "camera_config": "angle-1", "item_revision": "Б"}
	w.Add(nctest.Record(catalog.ItemItemRegistered, item, at(0), map[string]any{"item_id": item, "item_type_id": "FL-100.00.000", "item_revision": "Б",
		"process_version_hash": seed.Hash, "normative_rev": "flange-1", "lot_ids": []string{"LOT-FL-1"}, "entry_step_key": "welding.edge_prep"}))
	w.Add(nctest.Record(catalog.OperationRunStarted, item, at(10), map[string]any{"operation_run_id": run, "operation_code": "030", "step_key": "welding.weld",
		"station_id": "WP-WELD-2", "equipment_id": "IS-2", "operator_id": "W21"}))
	w.Add(nctest.Record(catalog.OperationRunFinished, item, at(50), map[string]any{"operation_run_id": run, "completion": "completed"}))
	w.Add(nctest.Record(catalog.InspectionResultRecorded, item, at(60), map[string]any{"method": "camera", "phase": "after_operation", "step_key": "welding.kt3_camera",
		"inspection_point": "KT-3", "operation_run_id": run, "zone_ids": zones, "processing_state": "completed", "outcome": "no_defect_indicated",
		"analyzer_confidence_bp": 9100, "observation_quality_bp": 3400, "versions": versions}))
	w.Now = at(65)
	w.Settle()
	ctx := nctest.As("INS-01", "quality_inspector")
	if _, err := svc.RequestRecheck(ctx, item, app.RequestRecheck{CommandHeader: hdr(0), Method: "radiography", Reason: reason("режим нарушен, кадр плохой")}); err != nil {
		t.Fatalf("доп. проверка по изделию: %v", err)
	}
	w.Settle()
	xray := nctest.Record(catalog.InspectionResultRecorded, item, at(80), map[string]any{"method": "radiography", "phase": "after_operation",
		"step_key": "welding.kt3_radiography", "operation_run_id": run, "zone_ids": zones, "outcome": "no_defect_indicated", "processing_state": "completed"})
	kt32 := nctest.Record(catalog.InspectionResultRecorded, item, at(82), map[string]any{"method": "camera", "phase": "after_operation", "step_key": "welding.kt3_camera",
		"inspection_point": "KT-3", "operation_run_id": run, "zone_ids": zones, "processing_state": "completed", "outcome": "no_defect_indicated",
		"observation_quality_bp": 9000, "inspector_id": "INS-01"})
	// Повторный кадр — осмотр контролёром (паспортов анализатора в тесте нет:
	// кадр анализатора уровня доверия 0 точку контроля не закрывает).
	w.Add(xray, kt32)
	w.Now = at(90)
	w.Settle()
	q, err := svc.Queue(ctx, app.QueueFilter{Kind: "presentation"}, platform.Moment{}, platform.Page{})
	if err != nil || len(q.Items) != 1 || q.Items[0].ItemID != item {
		t.Fatalf("Ф-002 на ЗТ-3 у контролёра: %+v %v", q.Items, err)
	}
	pv, err := svc.Presentation(ctx, item, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResolvePresentation(ctx, item, app.ResolvePresentation{CommandHeader: hdr(pv.BasisSeq), StepKey: pv.Presentation.StepKey,
		ClosingPoint: "ZT-3", Resolution: "accept", PresentationNo: pv.Presentation.PresentationNo,
		MethodEventIDs: []string{xray.Entry.EventID, kt32.Entry.EventID}}); err != nil {
		t.Fatalf("ЗТ-3 Ф-002 после доп. проверки: %v; точки %+v", err, pointsOf(t, w, bundles, item))
	}
}

func pointsOf(t *testing.T, w *nctest.World, bundles notifapp.Bundles, item string) any {
	in, err := w.Codec.LoadItem(context.Background(), item, 0)
	if err != nil {
		return err
	}
	b, _, _ := bundles.Bundle(context.Background(), item, in.Input)
	s, _ := engine.Fold(b, in.Input)
	return s.Quality.Points
}

// Очередь контролёра: изделия, с которыми работают сейчас (живая партия
// прогона), — выше изделий, чьи события давние (история прогона).
func TestQueueCurrentItemsFirst(t *testing.T) {
	w := nctest.NewWorld(t)
	svc := w.Service(app.DemoRoutes{})
	old, live := "ENT01:run-1/I-F1210000", "ENT01:run-1/I-F0010000"
	w.Add(nctest.Record(catalog.ItemPresentationRecorded, old, nctest.T0.Add(-72*time.Hour), map[string]any{"step_key": "welding.zt3_acceptance",
		"presentation_no": 1, "presented_to": "qc", "presented_by": "master-1"}))
	w.Add(nctest.Record(catalog.ItemPresentationRecorded, live, nctest.T0, map[string]any{"step_key": "welding.zt3_acceptance",
		"presentation_no": 1, "presented_to": "qc", "presented_by": "master-1"}))
	w.Settle()
	q, err := svc.Queue(nctest.As("INS-01", "quality_inspector"), app.QueueFilter{}, platform.Moment{}, platform.Page{})
	if err != nil || len(q.Items) != 2 || q.Items[0].ItemID != live {
		t.Fatalf("живая партия не первой: %+v %v", q.Items, err)
	}
}
