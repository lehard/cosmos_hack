package access_test

import (
	"context"
	"errors"
	"testing"

	"ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// fakeIDP — вход одной персоной с токеном «t».
type fakeIDP struct{ closed string }

func (fakeIDP) Identify(context.Context, access.Credentials) (platform.Principal, error) {
	return platform.Principal{}, nil
}

func (fakeIDP) Open(_ context.Context, rq access.SessionCreate) (platform.Principal, string, error) {
	return platform.Principal{PersonID: rq.PersonaID, Name: "Начальник ОТК", Role: "head_of_qc", Scope: "ent01", Demo: true}, "t", nil
}

func (f *fakeIDP) Close(_ context.Context, token string) error { f.closed = token; return nil }

func (fakeIDP) DemoPersonas(context.Context) ([]access.DemoPersona, error) { return nil, nil }

func directory() *access.Directory {
	return &access.Directory{
		Roles: []access.RoleRef{
			{ID: "quality_inspector", Title: "Контролёр качества"},
			{ID: "head_of_qc", Title: "Начальник ОТК", Inherits: []string{"quality_inspector"}},
		},
		Desks: map[string]access.Desk{"quality_inspector": {Version: 1, Role: "quality_inspector", TitleKey: "desks.decisionQueue",
			Tabs: []access.DeskTab{{ID: "queue", Layout: "queue-main-side"}}}},
	}
}

// Демо-трек эпика 08: вход, сеанс и стол наследника — живые; прочее — у запасной реализации.
func TestServiceSessionAndDesk(t *testing.T) {
	ctx := context.Background()
	idp := &fakeIDP{}
	s := access.NewService(access.WithIdentity(idp), access.WithDirectory(directory()))

	sess, token, err := s.OpenSession(ctx, access.SessionCreate{PersonaID: "HQC-01"})
	if err != nil || token != "t" || sess.Role.Title != "Начальник ОТК" || len(sess.Role.Inherits) != 1 || !sess.Demo {
		t.Fatalf("%+v %q %v", sess, token, err)
	}
	var pe *platform.Error
	if _, err := s.Session(ctx); !errors.As(err, &pe) || pe.Code != errcodes.AccessUnauthenticated {
		t.Fatalf("без сеанса: %v", err)
	}
	pctx := platform.WithPrincipal(ctx, platform.Principal{PersonID: "HQC-01", Role: "head_of_qc"})
	desk, err := s.Desks(pctx)
	if err != nil || desk.Role != "head_of_qc" || desk.TitleKey != "desks.decisionQueue" {
		t.Fatalf("стол наследника: %+v %v", desk, err)
	}
	if err := s.CloseSession(ctx, "t"); err != nil || idp.closed != "t" {
		t.Fatal(err)
	}
	// Посты — у запасной реализации (по умолчанию 501).
	if _, err := s.Workplaces(pctx, "", platform.Moment{}); !errors.As(err, &pe) || pe.Code != errcodes.ApiNotImplemented {
		t.Fatalf("посты: %v", err)
	}
}
