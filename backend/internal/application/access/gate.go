package access

import (
	"context"
	"slices"
	"strings"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// Gate — общий декоратор приложения над ведущими портами всех модулей (AD-36,
// AD-15, AD-39): права по x-ant-action, место сеанса, доменные гарды,
// допустимые действия и объяснение прав. Одинаков для реализаций live и
// fixtures: transport вызывает его до порта для каждой операции, а плоский
// список прав фронтенда и допустимые действия по объекту вычисляет тем же
// Enforce («в списке ⇔ разрешено»).
//
// Барьер 3 (AD-15, FR-85): решение — у AccessControl (Casbin — единственный
// вычислитель): субъект сеанса (или субъект без сеанса), действие, место
// операции (рабочее место команды, объекта или сеанса — Places) и доменное
// время. Отказ без сеанса — access.unauthenticated; действие разрешено ролью,
// но не в этом месте — access.wrong_workplace; иначе — access.forbidden.
// Каждый отказ операции — событие шины безопасности security.access.denied.
type Gate struct {
	ac      AccessControl
	guards  GuardChecker
	actions func() []platform.Action
	// RequireSession — без сеанса отказ access.unauthenticated до вычислителя
	// (кроме операций входа). Выключен: субъекту без сеанса политика даёт
	// только приём фактов от устройств (Policy.Unauthenticated).
	RequireSession bool
	// Now — доменное «сейчас» для атрибута времени запроса (DomainClock, AD-37).
	Now func() time.Time
	// Places — области рабочих мест (место операции); nil — место не определяется.
	Places Places
	// Events — шина безопасности для отказов (AD-24); nil — не сообщать.
	Events SecurityEvents
	// OnEventError — ошибка записи события безопасности (отказ остаётся отказом).
	OnEventError func(error)
	// Policy — действующая политика (эпик 26): потоки политики субъекта для
	// проверки policy_seq команды (AD-39), редкие подписанты (FR-136),
	// объяснение прав своим кодом. nil — только решение вычислителя.
	Policy PolicySource
	// Cards — карточки решения редких подписантов: объект виден, только если
	// он из адресованной сотруднику карточки (FR-136); nil — не сужает.
	Cards Cards
}

// NewGate — декоратор над портом прав ac; guards может быть nil; actions —
// каталог операций API (httpapi.API.Actions).
func NewGate(ac AccessControl, guards GuardChecker, actions func() []platform.Action) *Gate {
	return &Gate{ac: ac, guards: guards, actions: actions, Now: func() time.Time { return time.Time{} }}
}

// SetCatalog задаёт каталог операций (после регистрации всех операций API).
func (g *Gate) SetCatalog(actions func() []platform.Action) { g.actions = actions }

// unknownPlace — область неизвестного рабочего места: не входит ни в одну
// область политики, кроме всего предприятия без пути («*»).
const unknownPlace = "?"

// Place — место операции (барьер 3): рабочее место команды (workplace_id),
// объекта вида workplace или допуска сеанса (барьер 2); пусто — не определено.
func (g *Gate) Place(p platform.Principal, obj platform.ObjectRef, meta *platform.CommandMeta) string {
	id := ""
	switch {
	case meta != nil && meta.WorkplaceID != "":
		id = meta.WorkplaceID
	case obj.Kind == "workplace" && obj.ID != "":
		id = obj.ID
	case p.WorkplaceID != "":
		id = p.WorkplaceID
	}
	if id == "" || g.Places == nil {
		return ""
	}
	if s, ok := g.Places.ScopeOf(id); ok {
		return s
	}
	return unknownPlace
}

func (g *Gate) now() time.Time {
	if g.Now == nil {
		return time.Time{}
	}
	return g.Now()
}

// Authorize — проверка операции до вызова порта: сеанс (барьер 1), права и
// место (барьеры 2–3), гард владельца операции (AD-39). Отказ — *platform.Error
// с кодом из contracts/errors.yaml.
func (g *Gate) Authorize(ctx context.Context, p platform.Principal, act platform.Action, obj platform.ObjectRef, meta *platform.CommandMeta) error {
	if act.Anonymous {
		return nil
	}
	if p.Anonymous() && g.RequireSession {
		return platform.Fail(errcodes.AccessUnauthenticated)
	}
	rq := Request{Principal: p, Action: act, Object: obj, Scope: g.Place(p, obj, meta), At: g.now()}
	d, err := g.decide(ctx, rq)
	if err != nil {
		return err
	}
	if !d.Allowed {
		e := g.refusal(ctx, rq, d)
		g.report(ctx, p, act, obj, e.Code, rq.At)
		return e
	}
	if act.IsCommand() && meta != nil && g.guards != nil {
		if err := g.guards.Check(ctx, act, obj, *meta); err != nil {
			return err
		}
	}
	return nil
}

// Admit — Authorize и контекст команды с проверками политики субъекта
// (AD-39, AD-15): решение принято по политике версии S (вычислитель); если в
// потоках политики субъекта (корень и области его назначений, полномочий,
// клейм) после min(S, policy_seq клиента) появилась запись — journal.Append
// любой копии api отвергнет запись команды с 409 journal.stale_policy, и
// запись CA не делается. Так отзыв роли доходит и до копии, чья проекция ещё
// не перечитала журнал. Политика только из затравки (S = 0) — не проверяется.
func (g *Gate) Admit(ctx context.Context, p platform.Principal, act platform.Action, obj platform.ObjectRef, meta *platform.CommandMeta) (context.Context, error) {
	if err := g.Authorize(ctx, p, act, obj, meta); err != nil {
		return ctx, err
	}
	if act.Anonymous || !act.IsCommand() || p.Anonymous() || g.Policy == nil {
		return ctx, nil
	}
	seq, err := g.ac.PolicySeq(ctx)
	if err != nil {
		return ctx, err
	}
	if seq <= 0 {
		return ctx, nil
	}
	if meta != nil && meta.PolicySeq > 0 && meta.PolicySeq < seq {
		seq = meta.PolicySeq
	}
	pol, err := g.Policy.Policy(ctx)
	if err != nil {
		return ctx, err
	}
	var checks []appjournal.Check
	for _, s := range pol.SubjectStreams(p.PersonID, g.now()) {
		checks = append(checks, appjournal.Check{PolicyStream: s, PolicySeq: seq})
	}
	return appjournal.WithPolicyChecks(ctx, checks...), nil
}

// decide — решение вычислителя (Casbin, по ролям) и сужение для редких
// подписантов (FR-136): у сотрудника «только карточка» объект операции
// должен быть из адресованной ему карточки решения.
func (g *Gate) decide(ctx context.Context, rq Request) (Decision, error) {
	d, err := g.ac.Enforce(ctx, rq)
	if err != nil || !d.Allowed || g.Cards == nil || g.Policy == nil || rq.Object.ID == "" || rq.Principal.Anonymous() {
		return d, err
	}
	pol, err := g.Policy.Policy(ctx)
	if err != nil {
		return Decision{}, err
	}
	if !pol.CardOnly(rq.Principal.PersonID, rq.At) {
		return d, nil
	}
	ok, err := g.Cards.Visible(ctx, rq.Principal.PersonID, rq.Object)
	if err != nil {
		return Decision{}, err
	}
	if !ok {
		return Decision{Code: errcodes.AccessForbidden, Reason: "Редкий подписант видит только адресованные ему карточки решения (FR-136): " +
			rq.Object.Kind + " " + rq.Object.ID + " не из вашей карточки", AllowedActions: []string{}}, nil
	}
	return d, nil
}

// refusal — отказ с кодом: без сеанса — нужен вход; роль разрешает действие,
// но не в этом месте — не своё рабочее место (FR-78: исполнитель поста А не
// действует на посту Б); иначе — нет полномочий.
func (g *Gate) refusal(ctx context.Context, rq Request, d Decision) *platform.Error {
	code := d.Code
	if code == "" {
		code = errcodes.AccessForbidden
		switch {
		case rq.Principal.Anonymous():
			code = errcodes.AccessUnauthenticated
		case rq.Scope != "":
			anywhere := rq
			anywhere.Scope = ""
			if d2, err := g.ac.Enforce(ctx, anywhere); err == nil && d2.Allowed {
				code = errcodes.AccessWrongWorkplace
			}
		}
	}
	kv := []string{"action_id", rq.Action.ID, "action", rq.Action.ID}
	if obj := strings.TrimSpace(rq.Object.Kind + " " + rq.Object.ID); obj != "" {
		kv = append(kv, "object", obj)
	}
	if rq.Scope != "" {
		kv = append(kv, "workplace", rq.Scope)
	}
	e := platform.Fail(code, kv...)
	e.Detail = d.Reason
	if e.Detail == "" {
		switch code {
		case errcodes.AccessForbidden:
			e.Detail = "Действие " + rq.Action.ID + " не разрешено вашим ролям политикой"
		case errcodes.AccessWrongWorkplace:
			e.Detail = "Действие " + rq.Action.ID + " разрешено вашей роли, но не в этом месте (" + rq.Scope + ")"
		}
	}
	e.AllowedActions = d.AllowedActions
	return e
}

// report — событие шины безопасности об отказе (AD-24, FR-85). Запрос без
// сеанса — не отказ в доступе, а отсутствие входа (барьер 1): его события —
// неудачные входы (security.auth.failed), а не каждый анонимный запрос.
func (g *Gate) report(ctx context.Context, p platform.Principal, act platform.Action, obj platform.ObjectRef, code errcodes.Code, at time.Time) {
	if g.Events == nil || p.Anonymous() {
		return
	}
	err := g.Events.AccessDenied(ctx, Denial{PersonID: p.PersonID, ActionID: act.ID, Object: obj, Code: string(code), At: at})
	if err != nil && g.OnEventError != nil {
		g.OnEventError(err)
	}
}

// Permission — разрешённое действие для @casl/vue (AD-15 «Для фронтенда»).
type Permission struct {
	Action      string         `json:"action" doc:"x-ant-action id ‹модуль›.‹объект›.‹действие›."`
	ActionClass platform.Class `json:"action_class"`
	Subject     string         `json:"subject" doc:"Вид объекта (EntityKind) или all."`
	ObjectID    string         `json:"object_id,omitempty" doc:"Для списка по объекту — его id."`
}

// PermissionList — плоский список «действие → объект» или допустимые действия по объекту.
type PermissionList struct {
	PolicySeq int64        `json:"policy_seq" minimum:"0" doc:"Версия политики, по которой вычислен список (AD-39)."`
	Items     []Permission `json:"items"`
}

// Permissions — разрешённые действия субъекта: без объекта — плоский список по
// каталогу операций; с объектом — допустимые действия по нему. В
// воспроизведении (moment.IsReplay) команд в списке нет (AD-21).
func (g *Gate) Permissions(ctx context.Context, p platform.Principal, obj *platform.ObjectRef, moment platform.Moment) (PermissionList, error) {
	seq, err := g.ac.PolicySeq(ctx)
	if err != nil {
		return PermissionList{}, err
	}
	out := PermissionList{PolicySeq: seq, Items: []Permission{}}
	if g.actions == nil {
		return out, nil
	}
	at := g.now()
	for _, act := range g.actions() {
		if act.Anonymous {
			continue
		}
		if moment.IsReplay() && act.IsCommand() {
			continue
		}
		subject := act.Subject
		if subject == "" {
			subject = "all"
		}
		var o platform.ObjectRef
		if obj != nil {
			if subject != obj.Kind {
				continue
			}
			o = *obj
		}
		d, err := g.decide(ctx, Request{Principal: p, Action: act, Object: o, Scope: g.Place(p, o, nil), At: at})
		if err != nil {
			return PermissionList{}, err
		}
		if d.Allowed {
			out.Items = append(out.Items, Permission{Action: act.ID, ActionClass: act.Class, Subject: subject, ObjectID: o.ID})
		}
	}
	slices.SortFunc(out.Items, func(a, b Permission) int { return strings.Compare(a.Action, b.Action) })
	return out, nil
}
