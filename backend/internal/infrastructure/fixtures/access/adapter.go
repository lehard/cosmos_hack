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

// Workplaces — панель «Посты» (access.workplace.list, FR-6); назначения
// мастера в сессии (access.assignment.set / clear) видны на панели.
func (a Adapter) Workplaces(ctx context.Context, workshop string, m platform.Moment) (app.PostList, error) {
	v, err := respond[app.PostList](ctx, "access.workplace.list", map[string]string{"workshop": workshop}, &m)
	if err != nil {
		return v, err
	}
	if cur, ok := a.sessionPlan(ctx, m); ok {
		for i := range v.Items {
			overlayRow(&v.Items[i], cur[v.Items[i].WorkplaceID])
		}
	}
	if workshop == "" {
		return v, nil
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

// WorkplaceCard — карточка поста (access.workplace.read, UI-16).
func (a Adapter) WorkplaceCard(ctx context.Context, workplaceID string, m platform.Moment) (app.WorkplaceCard, error) {
	v, err := respond[app.WorkplaceCard](ctx, "access.workplace.read", map[string]string{"workplace_id": workplaceID}, &m)
	if err != nil {
		return v, err
	}
	if cur, ok := a.sessionPlan(ctx, m); ok {
		list := cur[workplaceID]
		overlayRow(&v.PostRow, list)
		v.Assignments = []app.WorkplaceAssignee{}
		for _, x := range list {
			v.ShiftID = x.ShiftID
			v.Assignments = append(v.Assignments, app.WorkplaceAssignee{PersonID: x.PersonID, PersonDisplay: x.display, ShiftID: x.ShiftID,
				AssigneeRole: x.AssigneeRole, QualificationOK: x.QualificationOK})
		}
	}
	return v, nil
}

// WorkplaceHistory — история поста (access.workplace.history, UI-16): весь
// список шага из мира заготовок, страница — здесь.
func (Adapter) WorkplaceHistory(ctx context.Context, workplaceID string, m platform.Moment, p platform.Page) (app.WorkplaceHistory, error) {
	v, err := respond[app.WorkplaceHistory](ctx, "access.workplace.history", map[string]string{"workplace_id": workplaceID}, &m)
	if err != nil {
		return v, err
	}
	v.Items, v.NextCursor = app.PageOf(v.Items, p)
	return v, nil
}

// PersonCard — карточка сотрудника (access.person.card, UI-16).
func (Adapter) PersonCard(ctx context.Context, personID string, m platform.Moment) (app.PersonCard, error) {
	return respond[app.PersonCard](ctx, "access.person.card", map[string]string{"person_id": personID}, &m)
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

// AdmitWorkplace — допуск к рабочему месту (access.workplace.admit, эпик 37):
// на заготовках — квитанция шага курсора; присутствие поста — у мира заготовок.
func (Adapter) AdmitWorkplace(ctx context.Context, workplaceID string, in app.AdmitWorkplace) (platform.Receipt, error) {
	return decide(ctx, "access.workplace.admit", "workplace", workplaceID, in.CommandMeta())
}

// ReleaseWorkplace — снять допуск (access.workplace.release, эпик 37).
func (Adapter) ReleaseWorkplace(ctx context.Context, workplaceID string, in app.ReleaseWorkplace) (platform.Receipt, error) {
	return decide(ctx, "access.workplace.release", "workplace", workplaceID, in.CommandMeta())
}

// GrantQualification — выдать квалификацию (access.qualification.grant, эпик 37).
func (Adapter) GrantQualification(ctx context.Context, personID string, in app.GrantQualification) (platform.Receipt, error) {
	return decide(ctx, "access.qualification.grant", "person", personID, in.CommandMeta())
}

// RevokeQualification — отозвать квалификацию (access.qualification.revoke, эпик 37).
func (Adapter) RevokeQualification(ctx context.Context, personID string, in app.RevokeQualification) (platform.Receipt, error) {
	return decide(ctx, "access.qualification.revoke", "person", personID, in.CommandMeta())
}

// decide — команда на заготовках: квитанция шага курсора (loader.Runtime.Decide).
func decide(ctx context.Context, op, kind, id string, meta platform.CommandMeta) (platform.Receipt, error) {
	rt, err := runtime()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Decide(ctx, op, loader.ObjectRef{Kind: kind, ID: id}, meta)
}

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
