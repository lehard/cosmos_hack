package access

import (
	"context"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// AccessControl — ведомый порт решений о доступе (AD-15, AD-35; ключ
// конфигурации access_control): Casbin — единственный вычислитель; в волне 1 —
// разрешающая заглушка (infrastructure/security/permissive), с эпика 08 —
// infrastructure/security/casbin над проекцией политики (PolicySource).
type AccessControl interface {
	// Enforce — решение по запросу «кто — что — над чем — где — когда».
	Enforce(ctx context.Context, rq Request) (Decision, error)
	// PolicySeq — версия политики, загруженной в вычислитель (AD-39).
	PolicySeq(ctx context.Context) (int64, error)
}

// Request — запрос решения о доступе «кто — что — над чем — где — когда».
type Request struct {
	Principal platform.Principal
	Action    platform.Action
	Object    platform.ObjectRef
	// Scope — место операции: путь области рабочего места команды или сеанса
	// (барьер 3, AD-15); пусто — место не определено, область не сужает.
	Scope string
	// At — доменное время запроса (сроки полномочий и клейм — по нему, AD-15, AD-37).
	At time.Time
}

// Decision — решение: разрешено или отказ с кодом (access.forbidden,
// access.no_stamp, access.separation_of_duties …) и тем, что можно сделать вместо (FR-146).
type Decision struct {
	Allowed        bool
	Code           errcodes.Code
	Reason         string
	AllowedActions []string
}

// IdentityProvider — ведомый порт входа (барьер 1, AD-15, AD-35; ключ
// identity_provider): локальные пользователи и демо-персоны в MVP;
// LDAP / ALD Pro / FreeIPA — следующий адаптер.
type IdentityProvider interface {
	// Identify — субъект по данным запроса; сеанса нет — анонимный без ошибки.
	Identify(ctx context.Context, cred Credentials) (platform.Principal, error)
	// Open — открыть сеанс: демо-персоной (без пароля, только профили fixtures и
	// demo) или по логину и паролю. Возвращает субъекта и токен сеанса для cookie.
	Open(ctx context.Context, rq SessionCreate) (platform.Principal, string, error)
	// Close — закрыть сеанс.
	Close(ctx context.Context, token string) error
	// DemoPersonas — демо-персоны для экрана входа; вне демо-профиля — api.not_found.
	DemoPersonas(ctx context.Context) ([]DemoPersona, error)
}

// Credentials — данные запроса для определения субъекта.
type Credentials struct {
	// SessionToken — cookie сеанса.
	SessionToken string
	// DemoPersona — заголовок демо-персоны (только fixtures и demo).
	DemoPersona string
}

// GuardChecker — порт доменных гардов для общего декоратора (AD-39): гард
// операции над состоянием на basis_seq. Вживую его реализует модуль-владелец
// через свёртку; на заготовках — адаптер заготовок; в волне 1 — нет гардов.
type GuardChecker interface {
	Check(ctx context.Context, act platform.Action, obj platform.ObjectRef, meta platform.CommandMeta) error
}
