package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// LocalOptions — настройки входа по локальным учётным записям.
type LocalOptions struct {
	// Personas — вход демо-персоной без пароля и заголовок Ant-Demo-Persona
	// (только профили fixtures и demo, FR-128).
	Personas bool
	// TTL — срок сеанса; 0 — DefaultTTL.
	TTL time.Duration
	// MaxFailures — неудачных попыток подряд до блокировки; 0 — 5.
	MaxFailures int
	// LockFor — срок блокировки после MaxFailures неудач; 0 — 15 минут.
	LockFor time.Duration
	// PerIP, PerLogin — не чаще одной попытки в интервал с запасом Burst:
	// по адресу клиента и по логину (golang.org/x/time/rate). 0 — 6 с и 3 с.
	PerIP, PerLogin time.Duration
	Burst           int
	// Now — инфраструктурные часы (AD-37): сеансы, блокировки, частота; nil — системные.
	Now func() time.Time
	// At — доменное «сейчас» для сроков назначений ролей (AD-37); nil — Now.
	At func(ctx context.Context) time.Time
}

// Local — IdentityProvider с локальными пользователями (ключ
// identity_provider = local; барьер 1, AD-15, FR-128): логин и пароль
// (argon2id), сеанс scs в Postgres (cookie ant_session), ограничение частоты
// попыток, блокировка после N неудач, события неудачного входа в шину
// безопасности (security.auth.failed). В профилях fixtures и demo — ещё и вход
// демо-персоной без пароля (тот же сеанс scs). Субъект и его роли — по
// действующей политике на каждый запрос: снятая роль перестаёт действовать
// сразу, без перевыпуска токена. LDAP / ALD Pro / FreeIPA — следующий
// адаптер того же порта.
type Local struct {
	dir    *access.Directory
	policy access.PolicySource
	creds  access.CredentialStore
	hasher access.PasswordHasher
	events access.SecurityEvents
	sm     *scs.SessionManager
	opts   LocalOptions
	byIP   *limiter
	byUser *limiter
	// dummy — хеш для выравнивания времени ответа на неизвестный логин.
	dummy string
}

// NewLocal — вход по учётным данным creds и политике policy; store — хранилище
// сеансов scs (SessionStore над Postgres; nil — в памяти, для тестов);
// events может быть nil.
func NewLocal(dir *access.Directory, policy access.PolicySource, creds access.CredentialStore, hasher access.PasswordHasher,
	events access.SecurityEvents, store scs.Store, opts LocalOptions) (*Local, error) {
	if dir == nil || policy == nil || creds == nil || hasher == nil {
		return nil, errors.New("identity: нет каталога, политики, учётных данных или хеша паролей")
	}
	if opts.TTL <= 0 {
		opts.TTL = DefaultTTL
	}
	if opts.MaxFailures <= 0 {
		opts.MaxFailures = 5
	}
	if opts.LockFor <= 0 {
		opts.LockFor = 15 * time.Minute
	}
	if opts.PerIP <= 0 {
		opts.PerIP = 6 * time.Second
	}
	if opts.PerLogin <= 0 {
		opts.PerLogin = 3 * time.Second
	}
	if opts.Burst <= 0 {
		opts.Burst = 10
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	if store == nil {
		store = memstore.New()
	}
	sm := scs.New()
	sm.Store = store
	sm.Lifetime = opts.TTL
	sm.Cookie.Name = "ant_session"
	dummy, err := hasher.Hash("ant-dummy-password")
	if err != nil {
		return nil, err
	}
	return &Local{dir: dir, policy: policy, creds: creds, hasher: hasher, events: events, sm: sm, opts: opts,
		byIP: newLimiter(opts.PerIP, opts.Burst), byUser: newLimiter(opts.PerLogin, opts.Burst/2+1), dummy: dummy}, nil
}

var _ access.IdentityProvider = (*Local)(nil)

// Ключи данных сеанса scs.
const (
	keyPerson = "person_id"
	keyRole   = "role"
	keyDemo   = "demo"
)

func (l *Local) at(ctx context.Context) time.Time {
	if l.opts.At != nil {
		return l.opts.At(ctx)
	}
	return l.opts.Now()
}

// Identify — субъект по cookie сеанса scs или заголовку демо-персоны; нет
// сеанса, он истёк или закрыт — анонимный без ошибки.
func (l *Local) Identify(ctx context.Context, cred access.Credentials) (platform.Principal, error) {
	if cred.SessionToken != "" {
		sctx, err := l.sm.Load(ctx, cred.SessionToken)
		if err == nil {
			if person := l.sm.GetString(sctx, keyPerson); person != "" {
				p, ok := l.principal(ctx, person, l.sm.GetString(sctx, keyRole))
				if ok {
					p.Demo = l.sm.GetBool(sctx, keyDemo)
					p.SessionID = sessionID(cred.SessionToken)
					return p, nil
				}
			}
		}
	}
	if cred.DemoPersona != "" && l.opts.Personas {
		if dp, ok := l.dir.Persona(cred.DemoPersona); ok {
			if p, ok := l.principal(ctx, dp.ID, dp.Role.ID); ok {
				p.Demo, p.SessionID = true, "header:"+dp.ID
				return p, nil
			}
		}
	}
	return platform.Principal{}, nil
}

// principal — субъект по действующей политике (access.PrincipalOf).
func (l *Local) principal(ctx context.Context, personID, role string) (platform.Principal, bool) {
	pol, err := l.policy.Policy(ctx)
	if err != nil {
		return platform.Principal{}, false
	}
	return access.PrincipalOf(pol, personID, role, l.at(ctx))
}

// Open — вход (FR-128): демо-персоной (только fixtures и demo) или по логину и
// паролю. Неудача — access.login_failed без подробностей (не раскрывает,
// существует ли логин); превышение частоты — api.rate_limited; заявка не
// активирована (пароль верный) — access.account_pending. Каждая неудача —
// событие security.auth.failed (AD-24).
func (l *Local) Open(ctx context.Context, rq access.SessionCreate) (platform.Principal, string, error) {
	if rq.PersonaID != "" {
		return l.openPersona(ctx, rq.PersonaID)
	}
	login := strings.ToLower(strings.TrimSpace(rq.Login))
	if login == "" {
		return platform.Principal{}, "", platform.Fail(errcodes.AccessLoginFailed)
	}
	now := l.opts.Now()
	ip := access.ClientIPFrom(ctx)
	if !l.byIP.allow(ip, now) || !l.byUser.allow(login, now) {
		l.report(ctx, login, ip, access.AuthRateLimited)
		e := platform.Fail(errcodes.ApiRateLimited, "retry_after", "10")
		e.Detail = "Слишком много попыток входа — повторите позже"
		return platform.Principal{}, "", e
	}
	c, found, err := l.creds.Get(ctx, login)
	if err != nil {
		return platform.Principal{}, "", err
	}
	if !found {
		_, _ = l.hasher.Verify(l.dummy, rq.Password)
		l.report(ctx, login, ip, access.AuthBadCredentials)
		return platform.Principal{}, "", platform.Fail(errcodes.AccessLoginFailed)
	}
	if c.LockedUntil.After(now) {
		l.report(ctx, login, ip, access.AuthAccountLocked)
		return platform.Principal{}, "", platform.Fail(errcodes.AccessLoginFailed)
	}
	ok, err := l.hasher.Verify(c.Hash, rq.Password)
	if err != nil || !ok || rq.Password == "" {
		after, ferr := l.creds.Failed(ctx, login, now, l.opts.MaxFailures, l.opts.LockFor)
		reason := access.AuthBadCredentials
		if ferr == nil && after.LockedUntil.After(now) {
			reason = access.AuthAccountLocked
		}
		l.report(ctx, login, ip, reason)
		return platform.Principal{}, "", platform.Fail(errcodes.AccessLoginFailed)
	}
	switch c.Status {
	case access.AccountPending:
		l.report(ctx, login, ip, access.AuthAccountPending)
		return platform.Principal{}, "", platform.Fail(errcodes.AccessAccountPending)
	case access.AccountActive:
	default:
		l.report(ctx, login, ip, access.AuthAccountLocked)
		return platform.Principal{}, "", platform.Fail(errcodes.AccessLoginFailed)
	}
	p, ok := l.principal(ctx, c.PersonID, "")
	if !ok {
		// Активация ещё не дошла до проекции политики — как заявка.
		l.report(ctx, login, ip, access.AuthAccountPending)
		return platform.Principal{}, "", platform.Fail(errcodes.AccessAccountPending)
	}
	if err := l.creds.Succeeded(ctx, login); err != nil {
		return platform.Principal{}, "", err
	}
	token, err := l.newSession(ctx, p.PersonID, p.Role, false)
	if err != nil {
		return platform.Principal{}, "", err
	}
	p.SessionID = sessionID(token)
	return p, token, nil
}

// openPersona — вход демо-персоной без пароля (профили fixtures и demo).
func (l *Local) openPersona(ctx context.Context, personaID string) (platform.Principal, string, error) {
	if !l.opts.Personas {
		return platform.Principal{}, "", platform.Fail(errcodes.AccessLoginFailed)
	}
	dp, ok := l.dir.Persona(personaID)
	if !ok {
		return platform.Principal{}, "", platform.Fail(errcodes.AccessLoginFailed)
	}
	p, ok := l.principal(ctx, dp.ID, dp.Role.ID)
	if !ok {
		return platform.Principal{}, "", platform.Fail(errcodes.AccessLoginFailed)
	}
	token, err := l.newSession(ctx, p.PersonID, p.Role, true)
	if err != nil {
		return platform.Principal{}, "", err
	}
	p.Demo, p.SessionID = true, sessionID(token)
	return p, token, nil
}

// newSession — новый сеанс scs (новый токен, сохранён в хранилище).
func (l *Local) newSession(ctx context.Context, personID, role string, demo bool) (string, error) {
	sctx, err := l.sm.Load(ctx, "")
	if err != nil {
		return "", err
	}
	l.sm.Put(sctx, keyPerson, personID)
	l.sm.Put(sctx, keyRole, role)
	l.sm.Put(sctx, keyDemo, demo)
	token, _, err := l.sm.Commit(sctx)
	return token, err
}

// Close — выход: сеанс удалён из хранилища.
func (l *Local) Close(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	sctx, err := l.sm.Load(ctx, token)
	if err != nil {
		return nil
	}
	return l.sm.Destroy(sctx)
}

// DemoPersonas — демо-персоны экрана входа; вне fixtures и demo — api.not_found.
func (l *Local) DemoPersonas(context.Context) ([]access.DemoPersona, error) {
	if !l.opts.Personas {
		return nil, platform.Fail(errcodes.ApiNotFound, "object", "демо-персоны", "id", "")
	}
	return append([]access.DemoPersona(nil), l.dir.Personas...), nil
}

// report — событие шины безопасности о неудачном входе (ошибка записи не
// меняет ответа входа).
func (l *Local) report(ctx context.Context, login, ip, reason string) {
	if l.events == nil {
		return
	}
	_ = l.events.AuthFailed(ctx, access.AuthFailure{Login: login, ClientIP: ip, Reason: reason, At: l.opts.Now()})
}

// sessionID — идентификатор сеанса для журнала и логов: не сам токен.
func sessionID(token string) string {
	h := sha256.Sum256([]byte("ant-session-id\x00" + token))
	return hex.EncodeToString(h[:8])
}
