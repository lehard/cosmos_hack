package access

import (
	"context"

	app "ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	"ant/internal/infrastructure/fixtures/loader"
)

// Adapter — реализация fixtures ведущих портов модуля access (AD-36):
// демо-персоны, сеанс и стол роли — из мира заготовок по субъекту запроса.
// Субъекта определяет IdentityProvider (барьер 1, эпик 08); здесь сеанса нет.
type Adapter struct {
	// Unimplemented — операции, которых нет в мире заготовок, отвечают 501.
	app.Unimplemented
}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Personas — демо-персоны экрана входа (access.persona.list).
func (Adapter) Personas(ctx context.Context) (app.DemoPersonaList, error) {
	return respond[app.DemoPersonaList](ctx, "access.persona.list", nil, nil)
}

// Session — сеанс субъекта запроса (access.session.read): персона мира заготовок.
func (Adapter) Session(ctx context.Context) (app.Session, error) {
	p := platform.PrincipalFrom(ctx)
	if p.Anonymous() {
		return app.Session{}, platform.Fail(errcodes.AccessUnauthenticated)
	}
	return respond[app.Session](ctx, "access.session.read", map[string]string{"persona": p.PersonID}, nil)
}

// Desks — стол активной роли субъекта (access.desk.read, AD-21).
func (Adapter) Desks(ctx context.Context) (app.Desk, error) {
	p := platform.PrincipalFrom(ctx)
	if p.Anonymous() {
		return app.Desk{}, platform.Fail(errcodes.AccessUnauthenticated)
	}
	return respond[app.Desk](ctx, "access.desk.read", map[string]string{"role": p.Role}, nil)
}

// Workplaces — панель «Посты» (access.workplace.list, FR-6).
func (Adapter) Workplaces(ctx context.Context, workshop string, m platform.Moment) (app.PostList, error) {
	v, err := respond[app.PostList](ctx, "access.workplace.list", map[string]string{"workshop": workshop}, &m)
	if err != nil || workshop == "" {
		return v, err
	}
	out := v.Items[:0]
	for _, r := range v.Items {
		if r.Workshop == workshop {
			out = append(out, r)
		}
	}
	v.Items = out
	return v, nil
}

// OpenSession — вход демо-персоной (access.session.create): сеанс персоны из
// мира заготовок и токен «demo.‹персона›». Вход по логину на заготовках не
// поддержан — только демо-персоны.
func (Adapter) OpenSession(ctx context.Context, rq app.SessionCreate) (app.Session, string, error) {
	if rq.PersonaID == "" {
		return app.Session{}, "", platform.Fail(errcodes.AccessLoginFailed)
	}
	s, err := respond[app.Session](ctx, "access.session.read", map[string]string{"persona": rq.PersonaID}, nil)
	if err != nil {
		return app.Session{}, "", err
	}
	return s, "demo." + rq.PersonaID, nil
}

// CloseSession — выход (access.session.delete): на заготовках сеансов не хранится.
func (Adapter) CloseSession(context.Context, string) error { return nil }

// runtime — мир заготовок (тесты подставляют свой).
var runtime = loader.Default

// respond — ответ операции из мира заготовок на шаге курсора.
func respond[T any](ctx context.Context, op string, params map[string]string, m *platform.Moment) (T, error) {
	var out T
	rt, err := runtime()
	if err != nil {
		return out, err
	}
	err = rt.Respond(ctx, op, params, m, &out)
	return out, err
}
