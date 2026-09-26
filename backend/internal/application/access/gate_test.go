package access_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
)

// scopeAC — вычислитель-заглушка: W21 может confirm_step только в области ent01/b1/wc.
type scopeAC struct{}

func (scopeAC) Enforce(_ context.Context, rq access.Request) (access.Decision, error) {
	ok := rq.Principal.PersonID == "W21" && rq.Action.ID == "access.operator.confirm_step" && (rq.Scope == "" || accessdom.ScopeCovers("ent01/b1/wc", rq.Scope))
	return access.Decision{Allowed: ok}, nil
}

func (scopeAC) PolicySeq(context.Context) (int64, error) { return 3, nil }

type places map[string]string

func (p places) ScopeOf(id string) (string, bool) { s, ok := p[id]; return s, ok }

type denials struct{ n int }

func (d *denials) AuthFailed(context.Context, access.AuthFailure) error { return nil }
func (d *denials) AccessDenied(context.Context, access.Denial) error    { d.n++; return nil }

func codeOf(err error) errcodes.Code {
	var pe *platform.Error
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

// Барьер 3: место операции — рабочее место команды, объекта или сеанса; коды отказа.
func TestGatePlaceAndRefusals(t *testing.T) {
	ev := &denials{}
	g := access.NewGate(scopeAC{}, nil, nil)
	g.Places = places{"WP-WELD-1": "ent01/b1/wc/weld/wp1", "WP-CNC-1": "ent01/b1/mc/cnc/wp1"}
	g.Events = ev
	ctx := context.Background()
	w21 := platform.Principal{PersonID: "W21", Role: "performer"}
	act := platform.Action{ID: "access.operator.confirm_step", Class: platform.ClassRecord}
	wp := func(id string) platform.ObjectRef { return platform.ObjectRef{Kind: "workplace", ID: id} }

	if err := g.Authorize(ctx, w21, act, wp("WP-WELD-1"), nil); err != nil {
		t.Fatal("свой пост", err)
	}
	if err := g.Authorize(ctx, w21, act, wp("WP-CNC-1"), nil); codeOf(err) != errcodes.AccessWrongWorkplace {
		t.Fatal("чужой пост", err)
	}
	// Рабочее место команды важнее объекта; неизвестное место не входит ни в одну область.
	if err := g.Authorize(ctx, w21, act, wp("WP-WELD-1"), &platform.CommandMeta{WorkplaceID: "WP-CNC-1"}); codeOf(err) != errcodes.AccessWrongWorkplace {
		t.Fatal("рабочее место команды", err)
	}
	if err := g.Authorize(ctx, w21, act, wp("WP-NOPE"), nil); codeOf(err) != errcodes.AccessWrongWorkplace {
		t.Fatal("неизвестное место", err)
	}
	// Место сеанса (допуск, барьер 2), если у команды и объекта его нет.
	if err := g.Authorize(ctx, platform.Principal{PersonID: "W21", WorkplaceID: "WP-CNC-1"}, act, platform.ObjectRef{}, nil); codeOf(err) != errcodes.AccessWrongWorkplace {
		t.Fatal("место сеанса", err)
	}
	if err := g.Authorize(ctx, platform.Principal{PersonID: "INS-01"}, act, wp("WP-WELD-1"), nil); codeOf(err) != errcodes.AccessForbidden {
		t.Fatal("чужая операция", err)
	}
	if err := g.Authorize(ctx, platform.Principal{}, act, wp("WP-WELD-1"), nil); codeOf(err) != errcodes.AccessUnauthenticated {
		t.Fatal("без сеанса", err)
	}
	if ev.n != 5 {
		t.Fatalf("отказы в шине безопасности (без анонимного): %d", ev.n)
	}
	// Объяснение прав — тем же кодом.
	ex, err := (func() (access.Explanation, error) {
		g.SetCatalog(func() []platform.Action { return []platform.Action{act} })
		return g.Explain(ctx, w21, act.ID, wp("WP-CNC-1"))
	})()
	if err != nil || ex.Allowed || ex.Code != string(errcodes.AccessWrongWorkplace) || ex.Reason == "" {
		t.Fatalf("объяснение: %+v %v", ex, err)
	}
	list, err := g.Permissions(ctx, w21, &platform.ObjectRef{Kind: "all"}, platform.Moment{})
	if err != nil || list.PolicySeq != 3 {
		t.Fatal(list, err)
	}
}

// memLog — журнал политики в памяти.
type memLog struct{ recs []accessdom.Record }

func (m *memLog) Since(_ context.Context, after int64) ([]accessdom.Record, error) {
	var out []accessdom.Record
	for _, r := range m.recs {
		if r.Seq > after {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memLog) Head() int64 { return 0 }

func assigned(t *testing.T, seq int64, person, role string, genesis bool) accessdom.Record {
	b, err := json.Marshal(ev.PolicyRoleAssignedV1{PersonID: ev.PersonRef(person), RoleID: ev.ObjectID(role), Scope: "ent01"})
	if err != nil {
		t.Fatal(err)
	}
	return accessdom.Record{Seq: seq, Type: string(catalog.PolicyRoleAssigned), Data: b, Genesis: genesis}
}

// Проекция: затравка + журнал; журнал с генезисом — без затравки (AD-33); PrincipalOf по политике.
func TestProjection(t *testing.T) {
	ctx := context.Background()
	seed := accessdom.FromSeed(accessdom.Seed{Root: "ent01",
		Roles:   []accessdom.Role{{ID: "staff", Actions: []string{"item.*.read"}}, {ID: "technologist", Inherits: []string{"staff"}}},
		Persons: []accessdom.SeedPerson{{ID: "TEC-01", Name: "Технолог", Roles: []accessdom.SeedGrant{{Role: "technologist", Scope: "ent01"}}}}})
	log := &memLog{recs: []accessdom.Record{assigned(t, 5, "TEC-01", "staff", false)}}
	p := access.NewProjection(seed, log)
	p.Every = 0
	pol, err := p.Policy(ctx)
	if err != nil || pol.Seq != 5 || len(pol.Assignments) != 2 {
		t.Fatalf("затравка + журнал: %+v %v", pol, err)
	}
	log.recs = append(log.recs, assigned(t, 9, "TEC-01", "technologist", false))
	if pol, _ = p.Policy(ctx); pol.Seq != 9 || len(pol.Assignments) != 3 {
		t.Fatalf("догоняет журнал: %+v", pol)
	}
	pr, ok := access.PrincipalOf(pol, "TEC-01", "", time.Now())
	if !ok || pr.Role != "technologist" || !pr.HasRole("staff") || pr.PolicySeq != 9 || pr.Name != "Технолог" {
		t.Fatalf("субъект: %+v", pr)
	}
	if _, ok := access.PrincipalOf(pol, "NOPE", "", time.Now()); ok {
		t.Fatal("неизвестный сотрудник")
	}
	g := access.NewProjection(seed, &memLog{recs: []accessdom.Record{assigned(t, 1, "GEN-1", "staff", true)}})
	if pol, _ := g.Policy(ctx); len(pol.Roles) != 0 || len(pol.Assignments) != 1 {
		t.Fatalf("генезис заменяет затравку: %+v", pol)
	}
}
