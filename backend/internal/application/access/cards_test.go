package access_test

import (
	"testing"
	"time"

	"ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	accessdom "ant/internal/domain/access"
)

// UI-16: история поста — события потока поста по видам, новые сверху;
// сотрудник завершения допуска — из открывшей его записи; чужие типы пропущены.
func TestWorkplaceEvents(t *testing.T) {
	t0 := time.Date(2026, 9, 24, 5, 0, 0, 0, time.UTC)
	rec := func(seq int64, typ catalog.Type, data string) accessdom.Record {
		return accessdom.Record{Seq: seq, Type: string(typ), Data: []byte(data), OccurredAt: t0.Add(time.Duration(seq) * time.Minute)}
	}
	recs := []accessdom.Record{
		rec(1, catalog.AccessAssignmentSet, `{"person_id":"W21","workplace_id":"WP-WELD-1","shift_id":"SHIFT-1","assignee_role":"performer"}`),
		rec(2, catalog.AccessTokenPresenceChanged, `{"person_id":"W21","workplace_id":"WP-WELD-1","present":true}`),
		rec(3, catalog.AccessWorkplaceAdmitted, `{"person_id":"W21","workplace_id":"WP-WELD-1","workplace_session_id":"s1","shift_id":"SHIFT-1"}`),
		rec(4, catalog.OperatorStepConfirmed, `{"operator_id":"W21","step_key":"welding.weld"}`),
		rec(5, catalog.SecurityPresenceDeviation, `{"person_id":"W21","workplace_id":"WP-WELD-1","deviation":"token_without_presence"}`),
		rec(6, catalog.AccessWorkplaceRevoked, `{"workplace_id":"WP-WELD-1","workplace_session_id":"s1","cause":"shift_ended"}`),
		rec(7, catalog.AccessAssignmentCleared, `{"person_id":"W21","workplace_id":"WP-WELD-1","shift_id":"SHIFT-1","reason":{"text":"конец смены"}}`),
	}
	got := access.WorkplaceEvents(recs, func(id string) string { return "Сварщик " + id })
	want := []string{"cleared", "revoked", "presence_deviation", "admitted", "token_in", "assigned"}
	if len(got) != len(want) {
		t.Fatalf("события: %+v", got)
	}
	for i, k := range want {
		if got[i].Kind != k || got[i].PersonID != "W21" || got[i].PersonDisplay != "Сварщик W21" {
			t.Fatalf("%d: %+v", i, got[i])
		}
	}
	if got[0].Reason != "конец смены" || got[1].Reason != "shift_ended" || got[2].Reason != "token_without_presence" || got[3].ShiftID != "SHIFT-1" {
		t.Fatalf("основания: %+v", got)
	}
	page, next := access.PageOf(got, platform.Page{Limit: 4})
	if len(page) != 4 || next != "4" {
		t.Fatalf("страница: %d %q", len(page), next)
	}
}
