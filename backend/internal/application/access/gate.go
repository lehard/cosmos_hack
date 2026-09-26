package access

import (
	"context"
	"slices"
	"strings"
	"time"

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
// Волна 1: AccessControl — разрешающая заглушка, гардов нет, сеанс не
// обязателен (RequireSession = false); эпик 08 включает барьеры.
type Gate struct {
	ac      AccessControl
	guards  GuardChecker
	actions func() []platform.Action
	// RequireSession — без сеанса отказ access.unauthenticated (кроме операций входа).
	RequireSession bool
	// Now — доменное «сейчас» для атрибута времени запроса (DomainClock, AD-37).
	Now func() time.Time
}

// NewGate — декоратор над портом прав ac; guards может быть nil; actions —
// каталог операций API (httpapi.API.Actions).
func NewGate(ac AccessControl, guards GuardChecker, actions func() []platform.Action) *Gate {
	return &Gate{ac: ac, guards: guards, actions: actions, Now: func() time.Time { return time.Time{} }}
}

// SetCatalog задаёт каталог операций (после регистрации всех операций API).
func (g *Gate) SetCatalog(actions func() []platform.Action) { g.actions = actions }

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
	d, err := g.ac.Enforce(ctx, Request{Principal: p, Action: act, Object: obj, At: g.Now()})
	if err != nil {
		return err
	}
	if !d.Allowed {
		code := d.Code
		if code == "" {
			code = errcodes.AccessForbidden
		}
		e := platform.Fail(code, "action", act.ID)
		e.Detail = d.Reason
		e.AllowedActions = d.AllowedActions
		return e
	}
	if act.IsCommand() && meta != nil && g.guards != nil {
		if err := g.guards.Check(ctx, act, obj, *meta); err != nil {
			return err
		}
	}
	return nil
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
		d, err := g.ac.Enforce(ctx, Request{Principal: p, Action: act, Object: o, At: g.Now()})
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

// Explanation — «почему вы можете / не можете» (FR-136, FR-146): решение по
// действию над объектом и что можно сделать вместо.
type Explanation struct {
	Action         string   `json:"action" doc:"x-ant-action id."`
	Allowed        bool     `json:"allowed"`
	Code           string   `json:"code,omitempty" doc:"Код отказа из contracts/errors.yaml."`
	Reason         string   `json:"reason,omitempty" doc:"Объяснение по-русски: роль, область, полномочие, клеймо, разделение обязанностей."`
	AllowedActions []string `json:"allowed_actions" doc:"Что можно сделать вместо (например, «Запросить решение»)."`
}

// Explain — объяснение прав по действию над объектом.
func (g *Gate) Explain(ctx context.Context, p platform.Principal, actionID string, obj platform.ObjectRef) (Explanation, error) {
	if g.actions != nil {
		for _, act := range g.actions() {
			if act.ID != actionID {
				continue
			}
			d, err := g.ac.Enforce(ctx, Request{Principal: p, Action: act, Object: obj, At: g.Now()})
			if err != nil {
				return Explanation{}, err
			}
			ex := Explanation{Action: actionID, Allowed: d.Allowed, Code: string(d.Code), Reason: d.Reason, AllowedActions: d.AllowedActions}
			if ex.AllowedActions == nil {
				ex.AllowedActions = []string{}
			}
			return ex, nil
		}
	}
	return Explanation{}, platform.Fail(errcodes.ApiNotFound, "object", "операция", "id", actionID)
}
