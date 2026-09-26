package notifications_test

import (
	"testing"

	notifapp "ant/internal/application/notifications"
	qualityapp "ant/internal/application/quality"
	"ant/internal/contracts/catalog"
	notif "ant/internal/domain/notifications"
	qualitystore "ant/internal/infrastructure/storage/quality"
)

// Показ SHOW-IS2, остановка 9: КТ-3 Ф-002 «признаков нет» при кадре хуже
// порога карты (R-02: «оценка невозможна», доп. проверка) — задача
// контролёру «доп. проверка» по изделию: operation_id
// nonconformity.recheck.request; назначенная доп. проверка её снимает.
func TestRecheckTaskOnPoorFrame(t *testing.T) {
	f := newFlow(t)
	qenv, err := qualitystore.SeedEnv("flange-1")
	if err != nil {
		t.Fatal(err)
	}
	f.bundles = notifapp.Bundles{Next: qualityapp.Bundles{Next: f.bundles.Next, Env: qenv}}
	f.add(catalog.ItemItemRegistered, 0, map[string]any{"item_id": f.item, "item_type_id": "FL-100.00.000", "item_revision": "Б",
		"process_version_hash": f.hash, "normative_rev": "flange-1", "lot_ids": []string{"LOT-FL-1"}, "entry_step_key": "welding.edge_prep"})
	f.add(catalog.ItemCarrierApplied, 0.01, map[string]any{"carrier_type": "tag_qr", "value": "show-is2-20260921/TAG:F-002", "is_temporary": true})
	f.add(catalog.OperationRunStarted, 0.5, map[string]any{"operation_run_id": "SV-2", "operation_code": "030", "step_key": "welding.weld", "operator_id": "W21",
		"equipment_id": "IS-2"})
	f.add(catalog.OperationRunFinished, 1.2, map[string]any{"operation_run_id": "SV-2", "completion": "completed"})
	zones := []string{"W-1.U1", "W-1.U2", "W-1.U3", "W-1.U4", "W-1.U5", "W-1.U6", "W-1.U7", "W-1.U8"}
	f.add(catalog.InspectionResultRecorded, 1.3, map[string]any{"observation_id": "KT3-KT3-F-002", "method": "camera", "phase": "after_operation",
		"step_key": "welding.kt3_camera", "inspection_point": "KT-3", "operation_run_id": "SV-2", "zone_ids": zones,
		"processing_state": "completed", "outcome": "no_defect_indicated", "analyzer_confidence_bp": 9100, "observation_quality_bp": 3400,
		"versions": map[string]any{"analyzer_version": "vqc-weld 2.3.1", "contract_version": "1.0", "recipe_ref": "kt3-weld@1", "camera_config": "angle-1", "item_revision": "Б"}})
	var task notif.TaskData
	open, _ := f.tasks()
	for _, d := range open {
		if d.Kind == "recheck" {
			task = d
		}
	}
	if task.TaskID == "" {
		// задачи процесса (process_step) здесь не нужны — ищем среди всех
		t.Fatalf("нет задачи доп. проверки: %+v", open)
	}
	if task.AssigneeRoleID != "quality_inspector" || task.OperationID != notif.OpRecheckRequest || task.ItemID != f.item || task.ItemLabel != "Ф-002" {
		t.Fatalf("задача доп. проверки: %+v", task)
	}
	f.add(catalog.DecisionRecheckRequested, 1.5, map[string]any{"method": "radiography", "reason": map[string]any{"code": "regime_violation", "text": "режим нарушен"}})
	if _, closed := f.tasks(); !closed[task.TaskID] {
		t.Fatal("доп. проверка назначена — задача не снята")
	}
}
