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

// Результат измерения снимает задачу «измерить» того же запроса.
func TestMeasurementTaskWithdrawn(t *testing.T) {
	at := time.Date(2026, 9, 21, 8, 20, 0, 0, time.UTC)
	rec := func(id string, tp catalog.Type, data any) kernel.Record {
		b, _ := json.Marshal(data)
		info, _ := catalog.Lookup(tp)
		return kernel.Record{EventID: id, Type: tp, Kind: info.Kind, Stream: "incident:INC-1", OccurredAt: at, Data: b}
	}
	req := "00000000-0000-7000-8000-000000000021"
	open := notif.ObjectReact(rec(req, catalog.IncidentMeasurementRequested, map[string]any{"incident_id": "INC-1", "what": "Контрольный образец"}))
	done := notif.ObjectReact(rec("00000000-0000-7000-8000-000000000022", catalog.IncidentMeasurementRecorded,
		map[string]any{"incident_id": "INC-1", "request_event_id": req, "outcome": "supports", "result": "прожог повторён"}))
	if len(open) != 1 || len(done) != 1 || done[0].Type != catalog.TaskTaskWithdrawn {
		t.Fatalf("задача %+v, снятие %+v", open, done)
	}
	var d notif.TaskData
	var w notif.TaskWithdrawnData
	raw, _ := json.Marshal(open[0].Data)
	_ = json.Unmarshal(raw, &d)
	raw, _ = json.Marshal(done[0].Data)
	_ = json.Unmarshal(raw, &w)
	if d.TaskID == "" || w.TaskID != d.TaskID {
		t.Fatalf("снята другая задача: %s ≠ %s", w.TaskID, d.TaskID)
	}
}
