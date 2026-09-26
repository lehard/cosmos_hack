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

// Session — сеанс субъекта запроса (access.session.read): персона мира
// заготовок — пост и смена по назначениям мира; поверх — допуск и снятие
// допуска этой сессии (access.workplace.admit / release, эпик 37).
func (Adapter) Session(ctx context.Context) (app.Session, error) {
	p := platform.PrincipalFrom(ctx)
	if p.Anonymous() {
		return app.Session{}, platform.Fail(errcodes.AccessUnauthenticated)
	}
	s, err := respond[app.Session](ctx, "access.session.read", map[string]string{"persona": p.PersonID}, nil)
	if err != nil {
		return s, err
	}
	return withAdmission(ctx, s), nil
}

// withAdmission — пост сеанса по допускам сессии: последний допуск персоны
// ставит пост (название — из панели «Посты» мира, смена — из команды или
// назначения мира), снятие допуска с этого поста — убирает.
func withAdmission(ctx context.Context, s app.Session) app.Session {
	rt, err := runtime()
	if err != nil {
		return s
	}
	for _, f := range rt.Facts(ctx, nil, "workplace") {
		if f.Actor != s.User.ID {
			continue
		}
		switch b := f.Body.(type) {
		case app.AdmitWorkplace:
			s.Workplace = &app.SessionWorkplace{ID: f.ID, Title: postTitle(ctx, f.ID)}
			if b.ShiftID != "" && (s.Shift == nil || s.Shift.ID != b.ShiftID) {
				s.Shift = &app.SessionShift{ID: b.ShiftID, Title: shiftTitle[b.ShiftID]}
			}
			if s.Shift == nil {
				s.Shift = &app.SessionShift{ID: "SHIFT-1", Title: shiftTitle["SHIFT-1"]}
			}
		case app.ReleaseWorkplace:
			if s.Workplace != nil && s.Workplace.ID == f.ID {
				s.Workplace = nil
			}
		}
	}
	return s
}

// shiftTitle — смены графика мира заготовок (normative/reference/shifts).
var shiftTitle = map[string]string{"SHIFT-1": "Первая смена 08:00–16:30", "SHIFT-2": "Вторая смена 16:30–01:00"}

// postTitle — название поста из панели «Посты» мира; нет — id.
func postTitle(ctx context.Context, id string) string {
	l, err := respond[app.PostList](ctx, "access.workplace.list", nil, nil)
	if err == nil {
		for _, r := range l.Items {
			if r.WorkplaceID == id && r.Station != "" {
				return r.Station
			}
		}
	}
	return id
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
// на заготовках — квитанция и факт сессии: пост и смена появляются в сеансе
// сотрудника (withAdmission) до сброса прогона.
func (Adapter) AdmitWorkplace(ctx context.Context, workplaceID string, in app.AdmitWorkplace) (platform.Receipt, error) {
	return record(ctx, "access.workplace.admit", workplaceID, in.CommandMeta(), in)
}

// ReleaseWorkplace — снять допуск (access.workplace.release, эпик 37): пост уходит из сеанса.
func (Adapter) ReleaseWorkplace(ctx context.Context, workplaceID string, in app.ReleaseWorkplace) (platform.Receipt, error) {
	return record(ctx, "access.workplace.release", workplaceID, in.CommandMeta(), in)
}

// record — команда над постом: квитанция и факт сессии.
func record(ctx context.Context, op, workplaceID string, meta platform.CommandMeta, body any) (platform.Receipt, error) {
	rt, err := runtime()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Record(ctx, op, loader.ObjectRef{Kind: "workplace", ID: workplaceID}, meta, body)
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
