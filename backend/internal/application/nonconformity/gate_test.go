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
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
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
