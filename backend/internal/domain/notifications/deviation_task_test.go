package notifications_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
	notif "ant/internal/domain/notifications"
)

// Ток ИС-2 вне уставки (показ SHOW-IS2, шаг 5): действующее отклонение —
// мастеру «решить по посту», руководителю — тревога; закончившееся (опоздавший
// журнал) задач не ставит.
func TestDeviationAlarmTasks(t *testing.T) {
	at := time.Date(2026, 9, 21, 6, 21, 0, 0, time.UTC)
	m := func(v int) map[string]any { return map[string]any{"value": v, "scale": 0, "unit": "A"} }
	rec := func(data map[string]any) kernel.Record {
		b, _ := json.Marshal(data)
		return kernel.Record{EventID: "00000000-0000-7000-8000-000000000011", Type: catalog.EquipmentDeviationDetected, Kind: catalog.KindFact,
			Stream: "equipment:IS-2", OccurredAt: at, Data: b}
	}
	data := map[string]any{"equipment_id": "IS-2", "deviation_kind": "out_of_setpoint", "parameter": "current", "value": m(182),
		"started_at": at.Add(-time.Minute).Format(time.RFC3339), "setpoint": map[string]any{"nominal": m(160), "lower": m(150), "upper": m(170)}}
	rs := notif.ObjectReact(rec(data))
	if len(rs) != 2 {
		t.Fatalf("задачи: %+v", rs)
	}
	roles := map[string]notif.TaskData{}
	for _, r := range rs {
		var d notif.TaskData
		raw, _ := json.Marshal(r.Data)
		_ = json.Unmarshal(raw, &d)
		roles[d.AssigneeRoleID] = d
	}
	f, pm := roles["site_foreman"], roles["production_manager"]
	if f.Kind != "decision_required" || !strings.Contains(f.Title, "ток 182 A при уставке 150 A…170 A") || pm.Title == "" || f.TaskID == pm.TaskID {
		t.Fatalf("мастер %+v, руководитель %+v", f, pm)
	}
	data["ended_at"] = at.Add(30 * time.Minute).Format(time.RFC3339)
	if rs := notif.ObjectReact(rec(data)); len(rs) != 0 {
		t.Fatalf("закончившееся отклонение: %+v", rs)
	}
	data["deviation_kind"] = "tool_life_warning"
	delete(data, "ended_at")
	if rs := notif.ObjectReact(rec(data)); len(rs) != 0 {
		t.Fatalf("ресурс инструмента — не тревога режима: %+v", rs)
	}
}
