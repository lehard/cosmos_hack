package analysis_test

import (
	"context"
	"testing"
	"time"

	"ant/internal/application/analysis/analysistest"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
)

// Шаг 10 показа SHOW-IS2 (Д-85): инцидент по ИС-2 открывает сама система —
// межизделийная стадия analysis по подтверждённому НС Ф-003, без шага
// сценария. Область — сварки на ИС-2 после последней подтверждённо годной
// (Ф-001 принят на ЗТ-3 — отсчёт, в область не входит): Ф-002, Ф-003; у НС
// Ф-003 — гипотеза «оборудование» для стола технолога.
func TestShowIS2IncidentOpensItself(t *testing.T) {
	w, j := memWorld(t)
	ctx := context.Background()
	msk := func(h, m int) time.Time { return time.Date(2026, 9, 21, h, m, 0, 0, analysistest.MSK).UTC() }
	f1, f2, f3 := "ENT01:show-is2-20260921-1/I-F001", "ENT01:show-is2-20260921-1/I-F002", "ENT01:show-is2-20260921-1/I-F003"
	weldOn := func(eq, item, run string, h1, m1, h2, m2 int) {
		if _, err := w.Fact(ctx, catalog.OperationRunStarted, item, "", msk(h1, m1), map[string]any{"operation_run_id": run, "step_key": analysistest.StepWeld,
			"operation_code": "030", "equipment_id": eq, "operator_id": "W21", "station_id": "ST-WELD"}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Fact(ctx, catalog.OperationRunFinished, item, "", msk(h2, m2), map[string]any{"operation_run_id": run, "completion": "completed"}); err != nil {
			t.Fatal(err)
		}
	}
	weld := func(item, run string, h1, m1, h2, m2 int) { weldOn("IS-2", item, run, h1, m1, h2, m2) }
	weld(f1, "SV-001-1", 8, 15, 8, 55)
	if _, err := w.Fact(ctx, catalog.DecisionPresentationResolved, f1, "", msk(9, 12), map[string]any{"step_key": "welding.zt3_acceptance",
		"closing_point": "ZT-3", "resolution": "accept", "presentation_no": 1, "method_event_ids": []string{}}); err != nil {
		t.Fatal(err)
	}
	weld(f2, "SV-002-1", 9, 20, 10, 0)
	weld(f3, "SV-003-1", 10, 5, 10, 45)
	// Сварщик W21 между делом сварил Ф-004 на ИС-1: журнал ИС-1 в уставке.
	f4 := "ENT01:show-is2-20260921-1/I-F004"
	weldOn("IS-1", f4, "SV-004-1", 9, 25, 9, 50)
	if _, err := w.Fact(ctx, catalog.EquipmentCycleSummarized, "", "equipment:IS-1", msk(9, 50), map[string]any{"equipment_id": "IS-1",
		"operation_run_id": "SV-004-1", "window_start": msk(9, 25).Format(time.RFC3339), "window_end": msk(9, 50).Format(time.RFC3339),
		"parameters": []map[string]any{{"parameter": "current", "out_of_setpoint_ms": 0}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Fact(ctx, catalog.DecisionNonconformityConfirmed, f3, "", msk(11, 0), map[string]any{"nc_id": "NC-F003",
		"defect_type_code": "W-BURNTHRU", "severity": "critical", "signal_ids": []string{"SIG-F003"},
		"reason": map[string]any{"text": "Прожог У2 подтверждён по двум ракурсам КТ-3"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	svc := w.Service(j, msk(11, 5))
	m := platform.Moment{Axis: platform.AxisOccurred}
	incs, err := svc.Incidents(ctx, m, platform.Page{})
	if err != nil || len(incs.Items) != 1 {
		t.Fatalf("инциденты: %+v %v", incs.Items, err)
	}
	inc := incs.Items[0]
	if inc.PrimaryNCID == nil || *inc.PrimaryNCID != "NC-F003" || len(inc.NCIDs) != 1 || inc.CommonFactor == nil || inc.CommonFactor.Value != "IS-2" {
		t.Fatalf("инцидент для технолога: %+v", inc)
	}
	rs, err := svc.RiskScope(ctx, inc.IncidentID, m)
	if err != nil {
		t.Fatal(err)
	}
	in := map[string]string{}
	for _, it := range rs.Items {
		in[it.ItemID] = it.Known
	}
	if in[f3] != "confirmed" || in[f2] != "suspect" || (in[f1] != "" && in[f1] != "excluded") {
		t.Fatalf("область: %v", in)
	}
	// Сужение в один щелчок: «исключить выполненные на ИС-1» с журналом ИС-1.
	if in[f4] != "suspect" || len(rs.NarrowOptions) != 1 || len(rs.NarrowOptions[0].ItemIDs) != 1 || rs.NarrowOptions[0].ItemIDs[0] != f4 ||
		len(rs.NarrowOptions[0].Evidence) == 0 || rs.NarrowOptions[0].Evidence[0].Text == nil {
		t.Fatalf("предложения сужения: %+v (область %v)", rs.NarrowOptions, in)
	}
	hs, err := svc.Hypotheses(ctx, "NC-F003", m)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, h := range hs.Hypotheses {
		found = found || h.Category == "equipment"
	}
	if !found {
		t.Fatalf("нет гипотезы «оборудование»: %+v", hs.Hypotheses)
	}
}
