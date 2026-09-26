package access

import (
	"context"
	"net"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

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
	registerAdmin(api, q, c)

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

	type sessionCreateOut struct {
		httpapi.Meta
		SetCookie http.Cookie `header:"Set-Cookie" doc:"Cookie сеанса веба (HttpOnly, SameSite=Strict)."`
		Body      app.Session
	}
	r := httpapi.Post("/auth/session", "Войти",
		"FR-128. Вход по логину и паролю (argon2id, сеанс scs в Postgres, ограничение частоты, блокировка после N неудач; "+
			"неудача — access.login_failed и событие security.auth.failed) или демо-персоной (persona_id, только профили fixtures и demo, без пароля). "+
			"Ответ ставит cookie сеанса.")
	r.Status = http.StatusCreated
	r.NoCommandMeta = true
	httpapi.Register(api, r,
		platform.Action{ID: "access.session.create", Class: platform.ClassRecord, Owner: "access", Anonymous: true},
		func(ctx context.Context, in *sessionCreateIn) (*sessionCreateOut, error) {
			s, token, err := c.OpenSession(app.WithClientIP(ctx, in.clientIP), in.Body)
			if err != nil {
				return nil, err
			}
			out := &sessionCreateOut{SetCookie: http.Cookie{Name: httpapi.SessionCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode}}
			out.Body = s
			return out, nil
		})

	type registrationOut struct {
		httpapi.Meta
		Body app.AccountRequestResult
	}
	reg := httpapi.Post("/auth/registration", "Заявка на регистрацию",
		"FR-128: сотрудник сам подаёт заявку (логин, пароль, имя); учётная запись ждёт активации администратором с назначением роли "+
			"(access.account.activate). Пароль хранится только хешем argon2id; частота заявок ограничена.")
	reg.Status = http.StatusAccepted
	reg.NoCommandMeta = true
	httpapi.Register(api, reg,
		platform.Action{ID: "access.account.request", Class: platform.ClassRecord, Owner: "access", Subject: "policy", Anonymous: true},
		func(ctx context.Context, in *registrationIn) (*registrationOut, error) {
			v, err := c.RequestAccount(app.WithClientIP(ctx, in.clientIP), in.Body)
			if err != nil {
				return nil, err
			}
			return &registrationOut{Body: v}, nil
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

	httpapi.Read(api, httpapi.Get("/workplaces/{workplace_id}", "Пост",
		"UI-16, FR-6, FR-81: карточка окна «Пост» — строка панели «Посты» (кто назначен, на месте ли, текущее изделие) и назначения текущей смены поста. "+
			"Права — в области роли (пост — место операции, барьер 3)."),
		platform.Action{ID: "access.workplace.read", Owner: "access", Subject: "workplace"},
		func(ctx context.Context, in *workplaceReadIn, m platform.Moment) (app.WorkplaceCard, error) {
			return q.WorkplaceCard(ctx, in.WorkplaceID, m)
		})
	httpapi.Read(api, httpapi.Get("/workplaces/{workplace_id}/history", "История поста",
		"UI-16, FR-81, FR-83, FR-84: события поста, новые сверху — назначение и снятие, токен вставлен и извлечён, допуск открыт, завершён, снят, "+
			"отклонение присутствия. Без доступа ко всему журналу (journal.entry.list)."),
		platform.Action{ID: "access.workplace.history", Owner: "access", Subject: "workplace"},
		func(ctx context.Context, in *workplaceHistoryIn, m platform.Moment) (app.WorkplaceHistory, error) {
			return q.WorkplaceHistory(ctx, in.WorkplaceID, m, in.Page())
		})
	httpapi.Read(api, httpapi.Get("/persons/{person_id}/card", "Карточка сотрудника",
		"UI-16, FR-80, FR-81: окно «Сотрудник» — имя, подразделение, роли в областях, квалификации со сроками, текущие посты. "+
			"Без логина и состояния учётной записи (их отдаёт access.person.read администратору)."),
		platform.Action{ID: "access.person.card", Owner: "access", Subject: "policy"},
		func(ctx context.Context, in *struct {
			PersonID string `path:"person_id" maxLength:"64" doc:"Псевдоним сотрудника."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.PersonCard, error) {
			return q.PersonCard(ctx, in.PersonID, m)
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

// workplaceReadIn — чтение карточки поста.
type workplaceReadIn struct {
	WorkplaceID string `path:"workplace_id" maxLength:"128" doc:"Пост (рабочее место)."`
	httpapi.MomentQuery
}

// Object — пост карточки (права по объекту: область роли, барьер 3).
func (w *workplaceReadIn) Object() platform.ObjectRef {
	return platform.ObjectRef{Kind: "workplace", ID: w.WorkplaceID}
}

// workplaceHistoryIn — чтение истории поста страницами.
type workplaceHistoryIn struct {
	WorkplaceID string `path:"workplace_id" maxLength:"128" doc:"Пост (рабочее место)."`
	httpapi.MomentQuery
	httpapi.PageQuery
}

// Object — пост истории (права по объекту: область роли, барьер 3).
func (w *workplaceHistoryIn) Object() platform.ObjectRef {
	return platform.ObjectRef{Kind: "workplace", ID: w.WorkplaceID}
}

// sessionCreateIn — тело входа и адрес клиента (ограничение частоты попыток,
// событие security.auth.failed; AD-15).
type sessionCreateIn struct {
	Body     app.SessionCreate
	clientIP string
}

// Resolve запоминает адрес клиента (huma.Resolver).
func (in *sessionCreateIn) Resolve(ctx huma.Context) []error {
	in.clientIP = clientIP(ctx)
	return nil
}

// registrationIn — заявка на регистрацию и адрес клиента.
type registrationIn struct {
	Body     app.AccountRequest
	clientIP string
}

// Resolve запоминает адрес клиента (huma.Resolver).
func (in *registrationIn) Resolve(ctx huma.Context) []error {
	in.clientIP = clientIP(ctx)
	return nil
}

// clientIP — адрес клиента без порта. Заголовкам прокси не доверяем: ant
// слушает напрямую или за своим обратным прокси (AD-25).
func clientIP(ctx huma.Context) string {
	addr := ctx.RemoteAddr()
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}
