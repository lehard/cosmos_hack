package access

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
)

func TestClosure(t *testing.T) {
	r := Roles{"head_of_qc": {"quality_inspector"}, "quality_inspector": {"staff"}, "staff": {"employee"}, "loop": {"loop2"}, "loop2": {"loop"}}
	if got := r.Closure("head_of_qc"); !slices.Equal(got, []string{"head_of_qc", "quality_inspector", "staff", "employee"}) {
		t.Fatalf("наследование: %v", got)
	}
	if got := r.Closure("loop"); !slices.Equal(got, []string{"loop", "loop2"}) {
		t.Fatalf("цикл: %v", got)
	}
	if !r.Covers("head_of_qc", "staff") || r.Covers("quality_inspector", "head_of_qc") || r.Closure("") != nil {
		t.Fatal("Covers")
	}
}

func TestScopeCovers(t *testing.T) {
	for _, c := range []struct {
		g, t string
		ok   bool
	}{
		{"ent01/b1/wc", "ent01/b1/wc/weld/wp1", true},
		{"ent01/b1/wc", "ent01/b1/wc", true},
		{"ent01/b1/wc", "ent01/b1/wcx", false},
		{"ent01/b1/wc", "ent01/b1/mc/cnc/wp1", false},
		{"ent01/b1/wc/weld/wp1", "ent01/b1/wc/weld/wp2", false},
		{"ent01", "ent01/b1/qa/final/wp1", true},
		{"", "ent01/b1", true},
		{"*", "x", true},
		{"ent01", "?", false},
	} {
		if got := ScopeCovers(c.g, c.t); got != c.ok {
			t.Errorf("ScopeCovers(%q, %q) = %v", c.g, c.t, got)
		}
	}
}

func TestActionMatches(t *testing.T) {
	if !ActionMatches("quality.signal.read", "quality.*.read") || !ActionMatches("simulation.run.start", "simulation.*.*") ||
		ActionMatches("quality.signal.list", "quality.*.read") || ActionMatches("quality.signal", "quality.*") || ActionMatches("", "*.*.*") {
		t.Fatal("шаблоны действий")
	}
	if ReadAlias("quality.signal.list", "read") != "quality.*.read" || ReadAlias("quality.signal.reject", "permissive") != "" {
		t.Fatal("второе имя чтения")
	}
	// Второе имя чтения подходит под шаблон роли, но не под чужой модуль.
	if !ActionMatches(ReadAlias("mes.order.list", "read"), "mes.*.read") || ActionMatches(ReadAlias("mes.order.list", "read"), "cad.*.read") {
		t.Fatal("второе имя под шаблоном")
	}
}

func TestPlaceMatch(t *testing.T) {
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	g := GrantPlace("ent01/b1/wc", from, until)
	in := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name string
		rq   string
		ok   bool
	}{
		{"в области и в сроке", RequestPlace("ent01/b1/wc/weld/wp1", in), true},
		{"место не определено", RequestPlace("", in), true},
		{"чужой цех", RequestPlace("ent01/b1/mc", in), false},
		{"до срока", RequestPlace("ent01/b1/wc", from.Add(-time.Second)), false},
		{"срок истёк (граница)", RequestPlace("ent01/b1/wc", until), false},
		{"нулевой момент", RequestPlace("ent01/b1/wc", time.Time{}), true},
	} {
		if got := PlaceMatch(c.rq, g); got != c.ok {
			t.Errorf("%s: PlaceMatch(%q, %q) = %v", c.name, c.rq, g, got)
		}
	}
	if !PlaceMatch(RequestPlace("x", in), AnyPlace) || PlaceMatch("bad", g) || PlaceMatch(RequestPlace("", in), "a|b") {
		t.Fatal("особые домены")
	}
	if !PlaceMatch(RequestPlace("ent01/b1", in), GrantPlace("ent01", time.Time{}, time.Time{})) {
		t.Fatal("бессрочно")
	}
}

func rec(t *testing.T, seq int64, typ catalog.Type, data any) Record {
	t.Helper()
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return Record{Seq: seq, Type: string(typ), Data: b}
}

func TestApply(t *testing.T) {
	p := FromSeed(Seed{Root: "ent01", Unauthenticated: "device_source",
		Roles:   []Role{{ID: "performer", Actions: []string{"process.operation.start"}}, {ID: "device_source", Actions: []string{"ingest.batch.submit"}}},
		Persons: []SeedPerson{{ID: "W21", Name: "Сварщик", Roles: []SeedGrant{{Role: "performer", Scope: "ent01/b1/wc"}}}}})
	t0 := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	until := ev.Timestamp(t0.Add(time.Hour))
	for _, r := range []Record{
		rec(t, 10, catalog.AccessPersonRegistered, ev.AccessPersonRegisteredV1{PersonID: "U-ivanov", DisplayName: "Иванов"}),
		rec(t, 11, catalog.AccessAccountActivated, ev.AccessAccountActivatedV1{PersonID: "U-ivanov", Login: "ivanov"}),
		rec(t, 12, catalog.PolicyRoleAssigned, ev.PolicyRoleAssignedV1{PersonID: "U-ivanov", RoleID: "performer", Scope: "ent01/b1/wc/weld/wp1", ValidFrom: ev.Timestamp(t0), ValidUntil: &until}),
		rec(t, 13, catalog.PolicyRoleDefined, ev.PolicyRoleDefinedV1{RoleID: "welder", Title: "Сварщик", Inherits: []ev.ObjectID{"performer"}, Actions: []string{"item.assembly.record"}}),
		rec(t, 14, catalog.PolicyRoleUnassigned, ev.PolicyRoleUnassignedV1{PersonID: "W21", RoleID: "performer", Scope: "ent01/b1/wc", EffectiveFrom: ev.Timestamp(t0)}),
		rec(t, 15, "item.item.registered", map[string]any{}),
	} {
		if err := p.Apply(r); err != nil {
			t.Fatal(err)
		}
	}
	if p.Seq != 15 {
		t.Fatalf("seq %d", p.Seq)
	}
	x, ok := p.PersonByLogin("ivanov")
	if !ok || x.ID != "U-ivanov" || !x.Active || x.Name != "Иванов" {
		t.Fatalf("учётная запись: %+v", x)
	}
	if as := p.AssignmentsOf("U-ivanov", t0.Add(time.Minute)); len(as) != 1 || as[0].Scope != "ent01/b1/wc/weld/wp1" {
		t.Fatalf("назначение: %+v", as)
	}
	if as := p.AssignmentsOf("U-ivanov", t0.Add(2*time.Hour)); len(as) != 0 {
		t.Fatalf("срок истёк: %+v", as)
	}
	if as := p.AssignmentsOf("W21", t0.Add(-time.Minute)); len(as) != 1 {
		t.Fatalf("до снятия роль действует: %+v", as)
	}
	if as := p.AssignmentsOf("W21", t0); len(as) != 0 {
		t.Fatalf("роль снята: %+v", as)
	}
	if h := p.Hierarchy(); !h.Covers("welder", "performer") {
		t.Fatal("новая роль записью политики — без кода (FR-78)")
	}
	rules := p.Rules()
	var anon bool
	for _, r := range rules {
		if r.PType == "g" && r.V[0] == AnonymousSubject && r.V[1] == "device_source" {
			anon = true
		}
	}
	if !anon {
		t.Fatalf("роль субъекта без сеанса: %v", rules)
	}
	if err := p.Apply(Record{Seq: 16, Type: string(catalog.PolicyRoleAssigned), Data: []byte("{")}); err == nil {
		t.Fatal("битая запись")
	}
}
