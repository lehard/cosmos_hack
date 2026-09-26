package notifications_test

import (
	"encoding/json"
	"testing"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
	notif "ant/internal/domain/notifications"
)

// Эпик 42 (FR-63, UJ-1): предложение, переданное руководителем, — задача
// ответственному в потоке предложения.
func TestSuggestionForwardedTask(t *testing.T) {
	b, _ := json.Marshal(map[string]any{"suggestion_id": "SUG-1", "responsible_id": "TEC-01", "responsible_role": "technologist", "title": "Сузить область RS-1"})
	r := kernel.Record{Seq: 7, EventID: "00000000-0000-7000-8000-000000000007", Type: catalog.IncidentSuggestionForwarded,
		Kind: catalog.KindDecision, Stream: "suggestion:SUG-1", OccurredAt: t0, Data: b}
	rs := notif.ObjectReact(r)
	if len(rs) != 1 {
		t.Fatalf("задач %d", len(rs))
	}
	d := data[notif.TaskData](t, rs[0])
	if d.AssigneePersonID != "TEC-01" || d.AssigneeRoleID != "technologist" || d.SubjectRef != "suggestion:SUG-1" {
		t.Fatalf("задача: %+v", d)
	}
}
