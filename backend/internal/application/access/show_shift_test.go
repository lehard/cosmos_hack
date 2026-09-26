package access_test

import (
	"context"
	"testing"

	"ant/internal/application/access"
	"ant/internal/application/platform"
)

// Показ SHOW-IS2 (Д-85), блокер «W21 не назначен ни на один пост»: план смены
// — история до начала показа: мастер FOR-WC назначает W21 на посты ИС-1 и
// ИС-2 смены SHIFT-1 (как шаги SHOW/plan-W21-* карточки). Живой access:
// W21 видит свои посты на панели «Посты», допуск с терминала (без смены в
// теле — как фронт) открывается, рабочее место сеанса — пост допуска;
// перевод на ИС-2 — снять допуск с ИС-1 и открыть на ИС-2.
func TestShowShiftPlanAdmitsW21(t *testing.T) {
	ctx := context.Background()
	pol := rosterPolicy()
	w := &applyWriter{pol: &pol, n: 10}
	m := newMemPresence()
	dir := dir37()
	dir.Workplaces = append(dir.Workplaces, access.WorkplaceRef{ID: "WP-WELD-2", Name: "Пост сварки 2", Scope: "ent01/b1/wc/weld/wp2", Workshop: "WS-WC", Zone: "Z-WC"})
	s := access.NewService(access.WithPolicy(ptrPolicy{&pol}), access.WithDecisions(w, clock26), access.WithDirectory(dir),
		access.WithLiveRoster(m), access.WithPresence(m, m, shifts37{}), access.WithWorkplaceLog(m))
	for _, wp := range []string{"WP-WELD-1", "WP-WELD-2"} {
		if _, err := s.SetAssignment(as("FOR-WC"), access.SetAssignment{PersonID: "W21", WorkplaceID: wp, ShiftID: "SHIFT-1", AssigneeRole: "performer"}); err != nil {
			t.Fatalf("план смены %s: %v", wp, err)
		}
	}
	// Смена пришла (СКУД stand, Arrive).
	if _, err := s.IngestZonePasses(ctx, []access.ZonePassIn{{SourceEventID: "p1", PersonID: "W21", ZoneID: "Z-WC", Enter: true, At: now26}}); err != nil {
		t.Fatal(err)
	}
	posts, err := s.Workplaces(ctx, "WS-WC", platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	mine := map[string]bool{}
	for _, r := range posts.Items {
		if r.Assigned != nil && r.Assigned.PersonID == "W21" {
			mine[r.WorkplaceID] = true
		}
	}
	if !mine["WP-WELD-1"] || !mine["WP-WELD-2"] {
		t.Fatalf("посты W21 на панели: %+v", posts.Items)
	}
	admit := access.AdmitWorkplace{KeyRef: "w21@1", PinVerified: true}
	if _, err := s.AdmitWorkplace(as("W21"), "WP-WELD-1", admit); err != nil {
		t.Fatalf("допуск к ИС-1: %v", err)
	}
	sess, err := s.Session(as("W21"))
	if err != nil || sess.Workplace == nil || sess.Workplace.ID != "WP-WELD-1" {
		t.Fatalf("рабочее место сеанса: %+v %v", sess.Workplace, err)
	}
	// Перевод на ИС-2 (шаг карточки «допуск к ИС-2» — без снятия допуска с ИС-1).
	if _, err := s.AdmitWorkplace(as("W21"), "WP-WELD-2", admit); err != nil {
		t.Fatalf("допуск к ИС-2: %v", err)
	}
	if sess, _ = s.Session(as("W21")); sess.Workplace == nil || sess.Workplace.ID != "WP-WELD-2" {
		t.Fatalf("рабочее место после перевода: %+v", sess.Workplace)
	}
}
