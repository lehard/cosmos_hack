package access

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	app "ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	"ant/internal/infrastructure/fixtures/loader"
)

// Мир с одной сменой: W21 на посту сварки 2; на пост сварки 1 кандидаты —
// W21, W22 (квалификация действует) и кладовщик без квалификации.
const rosterStep = `format: 1
step: 0
clock: 2026-09-23T05:00:00Z
title: шаг 0
responses:
  - op: reference.shift.list
    body: {"items": [{"shift_id": "SHIFT-1@2026-09-23", "location_id": "WS-WC", "name": "Первая смена", "starts_at": "2026-09-23T05:00:00Z", "ends_at": "2026-09-23T13:30:00Z"},
                     {"shift_id": "SHIFT-2@2026-09-23", "location_id": "WS-WC", "name": "Вторая смена", "starts_at": "2026-09-23T13:30:00Z", "ends_at": "2026-09-23T22:00:00Z"}]}
  - op: access.assignment.list
    body: {"items": [{"workplace_id": "WP-WELD-2", "shift_id": "SHIFT-1@2026-09-23", "person_id": "W21", "assignee_role": "performer", "admitted": true, "qualification_ok": true},
                     {"workplace_id": "WP-WELD-2", "shift_id": "SHIFT-2@2026-09-23", "person_id": "W22", "assignee_role": "performer", "admitted": false, "qualification_ok": true}], "basis_seq": 9}
  - op: access.candidate.list
    params: {workplace_id: WP-WELD-1}
    body: {"workplace_id": "WP-WELD-1", "items": [
      {"person_id": "W21", "display": "Сварщик W21", "role": "performer", "qualification_verdict": "ok", "why": "действует", "allowed": true},
      {"person_id": "W22", "display": "Сварщик W22", "role": "performer", "qualification_verdict": "ok", "why": "действует", "allowed": true},
      {"person_id": "STK-51", "display": "Кладовщик", "role": "performer", "qualification_verdict": "missing", "why": "нет", "allowed": false}], "basis_seq": 9}
`

func setupRoster(t *testing.T) {
	t.Helper()
	fsys := fstest.MapFS{
		"common/responses.yaml": {Data: []byte(common)},
		"sc/scenario.yaml":      {Data: []byte(strings.Replace(manifest, "2026-09-18T05:00:00Z", "2026-09-23T05:00:00Z", 1))},
		"sc/steps/00.yaml":      {Data: []byte(rosterStep)},
	}
	lib, err := loader.LoadFS(fsys, ".")
	if err != nil {
		t.Fatal(err)
	}
	rt := loader.New(lib, loader.NewMemoryCursor())
	old := runtime
	runtime = func() (*loader.Runtime, error) { return rt, nil }
	t.Cleanup(func() { runtime = old })
}

func TestRosterSessionOverlay(t *testing.T) {
	setupRoster(t)
	a := New()
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "FOR-WC", Role: "site_foreman"})
	cur, err := a.Assignments(ctx, "", "", platform.Moment{})
	if err != nil || len(cur.Items) != 1 || cur.Items[0].PersonID != "W21" {
		t.Fatalf("план идущей смены: %+v %v", cur, err)
	}
	if next, _ := a.Assignments(ctx, "SHIFT-2", "", platform.Moment{}); len(next.Items) != 1 || next.Items[0].PersonID != "W22" {
		t.Fatalf("план второй смены по шаблону: %+v", next)
	}
	shift := "SHIFT-1@2026-09-23"
	meta := func(id string) platform.CommandHeader { return platform.CommandHeader{CommandID: id} }
	if _, err := a.SetAssignment(ctx, app.SetAssignment{CommandHeader: meta("c1"), WorkplaceID: "WP-WELD-1", ShiftID: shift, PersonID: "STK-51", AssigneeRole: "performer"}); !isCode(err, errcodes.AccessNotQualified) {
		t.Fatalf("без квалификации: %v", err)
	}
	if _, err := a.SetAssignment(ctx, app.SetAssignment{CommandHeader: meta("c2"), WorkplaceID: "WP-WELD-1", ShiftID: shift, PersonID: "W22", AssigneeRole: "performer"}); err != nil {
		t.Fatalf("назначить: %v", err)
	}
	if cur, _ := a.Assignments(ctx, shift, "", platform.Moment{}); len(cur.Items) != 2 {
		t.Fatalf("назначение в плане: %+v", cur)
	}
	c, err := a.Candidates(ctx, "WP-WELD-1", shift, platform.Moment{})
	if err != nil || c.ShiftID != shift {
		t.Fatalf("кандидаты: %+v %v", c, err)
	}
	for _, x := range c.Items {
		switch x.PersonID {
		case "W22":
			if !x.AssignedHere {
				t.Fatalf("W22 назначен сюда: %+v", x)
			}
		case "W21":
			if x.AssignedElsewhere != "WP-WELD-2" {
				t.Fatalf("W21 на другом посту: %+v", x)
			}
		}
	}
	if c.Items[len(c.Items)-1].PersonID != "STK-51" {
		t.Fatalf("недопустимые — в конце: %+v", c.Items)
	}
	if _, err := a.ClearAssignment(ctx, app.ClearAssignment{CommandHeader: meta("c3"), WorkplaceID: "WP-WELD-1", ShiftID: shift, PersonID: "W22"}); err != nil {
		t.Fatalf("снять: %v", err)
	}
	if _, err := a.ClearAssignment(ctx, app.ClearAssignment{CommandHeader: meta("c4"), WorkplaceID: "WP-WELD-1", ShiftID: shift, PersonID: "W22"}); !isCode(err, errcodes.ApiNotFound) {
		t.Fatalf("снять снятое: %v", err)
	}
	if cur, _ := a.Assignments(ctx, shift, "", platform.Moment{}); len(cur.Items) != 1 {
		t.Fatalf("после снятия: %+v", cur)
	}
}
