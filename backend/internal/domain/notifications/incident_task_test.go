package notifications_test

import (
	"encoding/json"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
	notif "ant/internal/domain/notifications"
)

// Инцидент открыт системой — технологу «Разобрать инцидент»; вывод о причине
// снимает ту же задачу (тот же task_id).
func TestIncidentInvestigationTask(t *testing.T) {
	at := time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC)
	rec := func(id string, tp catalog.Type, data any) kernel.Record {
		b, _ := json.Marshal(data)
		info, _ := catalog.Lookup(tp)
		return kernel.Record{EventID: id, Type: tp, Kind: info.Kind, Stream: "incident:INC-1", OccurredAt: at, Data: b}
	}
	open := notif.ObjectReact(rec("00000000-0000-7000-8000-000000000001", catalog.IncidentIncidentOpened,
		map[string]any{"incident_id": "INC-1", "common_factor": "equipment", "factor_ref": "IS-2", "trigger_event_ids": []string{"x"}}))
	if len(open) != 1 || open[0].Type != catalog.TaskTaskCreated {
		t.Fatalf("задача: %+v", open)
	}
	var d notif.TaskData
	raw, _ := json.Marshal(open[0].Data)
	_ = json.Unmarshal(raw, &d)
	if d.AssigneeRoleID != "technologist" || d.Kind != "decision_required" || d.SubjectRef != "incident:INC-1" || d.Title == "" {
		t.Fatalf("задача технолога: %+v", d)
	}
	done := notif.ObjectReact(rec("00000000-0000-7000-8000-000000000002", catalog.IncidentCauseConcluded, map[string]any{"incident_id": "INC-1"}))
	if len(done) != 1 || done[0].Type != catalog.TaskTaskWithdrawn {
		t.Fatalf("снятие: %+v", done)
	}
	var w notif.TaskWithdrawnData
	raw, _ = json.Marshal(done[0].Data)
	_ = json.Unmarshal(raw, &w)
	if w.TaskID != d.TaskID {
		t.Fatalf("снята другая задача: %s ≠ %s", w.TaskID, d.TaskID)
	}
}
