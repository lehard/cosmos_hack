package access

import (
	"context"
	"net/http"

	app "ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля access: вход демо-персоной или по логину,
// сеанс, выход, стол роли, права для @casl/vue, объяснение прав (FR-128,
// FR-136, FR-146; AD-15, AD-21). Права и допустимые действия вычисляет общий
// декоратор (Gate) тем же Enforce, что проверяет команды.
func Register(api *httpapi.API, q app.Queries, c app.Commands, gate *app.Gate) {
	httpapi.Enum[app.Density](api)
	httpapi.Enum[app.DeskLayout](api)
	httpapi.Enum[app.DeskArea](api)

	httpapi.Register(api, httpapi.Get("/auth/personas", "Демо-персоны для входа без пароля",
		"Демо-трек (эпик 08): экран входа предлагает выбрать демо-персону — псевдоним из стартовой политики (normative/policy) с ролью и областью. "+
			"Вне профилей fixtures и demo операция отвечает 404 api.not_found, и экран показывает только вход по логину."),
		platform.Action{ID: "access.persona.list", Class: platform.ClassRead, Owner: "access", Anonymous: true},
		func(ctx context.Context, _ *struct{}) (*httpapi.Out[app.DemoPersonaList], error) {
			v, err := q.Personas(ctx)
			return httpapi.OK(v), err
		})

	httpapi.Register(api, httpapi.Get("/auth/session", "Текущий сеанс",
		"Пользователь, активная роль, область, смена, рабочее место, версия политики (FR-128); 401 access.unauthenticated — сеанса нет."),
		platform.Action{ID: "access.session.read", Class: platform.ClassRead, Owner: "access"},
		func(ctx context.Context, _ *struct{}) (*httpapi.Out[app.Session], error) {
			v, err := q.Session(ctx)
			return httpapi.OK(v), err
		})

	type sessionCreateIn struct {
		Body app.SessionCreate
	}
	type sessionCreateOut struct {
		httpapi.Meta
		SetCookie http.Cookie `header:"Set-Cookie" doc:"Cookie сеанса веба (HttpOnly, SameSite=Strict)."`
		Body      app.Session
	}
	r := httpapi.Post("/auth/session", "Войти",
		"FR-128. Вход демо-персоной (persona_id, только профили fixtures и demo) или по логину. Пароль пока необязателен (демо-трек); "+
			"после эпика 08 — обязателен для входа по логину (сеанс scs, argon2id). Ответ ставит cookie сеанса.")
	r.Status = http.StatusCreated
	r.NoCommandMeta = true
	httpapi.Register(api, r,
		platform.Action{ID: "access.session.create", Class: platform.ClassRecord, Owner: "access", Anonymous: true},
		func(ctx context.Context, in *sessionCreateIn) (*sessionCreateOut, error) {
			s, token, err := c.OpenSession(ctx, in.Body)
			if err != nil {
				return nil, err
			}
			out := &sessionCreateOut{SetCookie: http.Cookie{Name: httpapi.SessionCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode}}
			out.Body = s
			return out, nil
		})

	d := httpapi.Delete("/auth/session", "Выйти", "Закрыть сеанс веба (FR-128).")
	d.NoCommandMeta = true
	httpapi.Register(api, d,
		platform.Action{ID: "access.session.delete", Class: platform.ClassRecord, Owner: "access"},
		func(ctx context.Context, in *struct {
			Session string `cookie:"ant_session" doc:"Cookie сеанса."`
		}) (*httpapi.NoBody, error) {
			return &httpapi.NoBody{}, c.CloseSession(ctx, in.Session)
		})

	httpapi.Register(api, httpapi.Get("/desk", "Стол активной роли",
		"AD-21: стол — данные normative/desks/‹роль›.yaml (раскладка → вкладки → слоты → виджеты → срез и плотность), отдаётся через access.Queries.Desks. "+
			"Для роли-наследника без своего файла — стол ближайшей базовой роли (inherits)."),
		platform.Action{ID: "access.desk.read", Class: platform.ClassRead, Owner: "access"},
		func(ctx context.Context, _ *struct{}) (*httpapi.Out[app.Desk], error) {
			v, err := q.Desks(ctx)
			return httpapi.OK(v), err
		})

	httpapi.Read(api, httpapi.Get("/workplaces", "Посты",
		"Панель «Посты» стола руководителя и участок мастера (PRD §3a, FR-6, FR-81): участок — кто назначен — на месте ли (СКУД, ключ) — текущая деталь."),
		platform.Action{ID: "access.workplace.list", Owner: "access", Subject: "workplace"},
		func(ctx context.Context, in *struct {
			Workshop string `query:"workshop" maxLength:"128" doc:"Цех; пусто — вся область роли."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.PostList, error) {
			return q.Workplaces(ctx, in.Workshop, m)
		})

	type permissionsIn struct {
		Subject platform.EntityKind `query:"subject" doc:"Вид объекта."`
		ID      string              `query:"id" maxLength:"128" doc:"Идентификатор объекта."`
		httpapi.MomentQuery
	}
	httpapi.Register(api, httpapi.Get("/permissions", "Разрешённые действия",
		"AD-15 «Для фронтенда»: без subject — плоский список «действие → объект» для @casl/vue; с subject и id — допустимые действия по конкретному объекту. "+
			"Сервер вычисляет его тем же Enforce, что проверяет команды («в списке ⇔ разрешено»). На момент as_of (воспроизведение) команд в списке нет."),
		platform.Action{ID: "access.permission.list", Class: platform.ClassRead, Owner: "access"},
		func(ctx context.Context, in *permissionsIn) (*httpapi.Out[app.PermissionList], error) {
			m, err := in.Moment()
			if err != nil {
				return nil, err
			}
			var obj *platform.ObjectRef
			if in.Subject != "" {
				obj = &platform.ObjectRef{Kind: string(in.Subject), ID: in.ID}
			}
			v, err := gate.Permissions(ctx, platform.PrincipalFrom(ctx), obj, m)
			return httpapi.OK(v), err
		})

	type explainIn struct {
		Action  string              `query:"action" required:"true" maxLength:"128" doc:"x-ant-action id."`
		Subject platform.EntityKind `query:"subject" doc:"Вид объекта."`
		ID      string              `query:"id" maxLength:"128" doc:"Идентификатор объекта."`
	}
	httpapi.Register(api, httpapi.Get("/permissions/explain", "Почему вы можете или не можете",
		"FR-136, FR-146: решение по действию над объектом с объяснением (роль, область, полномочие, клеймо, разделение обязанностей) "+
			"и тем, что можно сделать вместо («Запросить решение»)."),
		platform.Action{ID: "access.permission.explain", Class: platform.ClassRead, Owner: "access"},
		func(ctx context.Context, in *explainIn) (*httpapi.Out[app.Explanation], error) {
			v, err := gate.Explain(ctx, platform.PrincipalFrom(ctx), in.Action, platform.ObjectRef{Kind: string(in.Subject), ID: in.ID})
			return httpapi.OK(v), err
		})
}
