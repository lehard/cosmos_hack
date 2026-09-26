package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// DefaultTTL — срок сеанса демо-персоны.
const DefaultTTL = 12 * time.Hour

// Options — настройки Demo.
type Options struct {
	// Personas — вход демо-персоной разрешён (профили fixtures и demo, FR-128);
	// иначе экран входа не показывает персон (api.not_found), вход персоной — отказ.
	Personas bool
	// Key — ключ подписи токена сеанса (HMAC-SHA-256); пусто — случайный на
	// процесс (сеансы не переживают перезапуск и не делятся между копиями api).
	Key []byte
	// TTL — срок сеанса; 0 — DefaultTTL.
	TTL time.Duration
	// Now — инфраструктурные часы (AD-37); nil — системные.
	Now func() time.Time
}

// Demo — IdentityProvider без базы данных (ключ identity_provider = demo;
// выгрузка OpenAPI, тесты, запуск без Postgres): вход демо-персоной
// без пароля; сеанс — подписанный токен в cookie ant_session
// (‹base64url(claims)›.‹base64url(HMAC)›), без хранилища сеансов: токен
// проверяется подписью и сроком, выход заносит сеанс в список отозванных до
// истечения срока. Заголовок Ant-Demo-Persona — вход без cookie (curl,
// проверки), только при Personas.
type Demo struct {
	dir  *access.Directory
	opts Options

	mu      sync.Mutex
	revoked map[string]time.Time // session_id → срок токена
}

// NewDemo создаёт провайдер входа над каталогом политики dir.
func NewDemo(dir *access.Directory, opts Options) (*Demo, error) {
	if dir == nil {
		return nil, errors.New("identity: нет каталога политики")
	}
	if len(opts.Key) == 0 {
		opts.Key = make([]byte, 32)
		if _, err := rand.Read(opts.Key); err != nil {
			return nil, err
		}
	}
	if opts.TTL <= 0 {
		opts.TTL = DefaultTTL
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Demo{dir: dir, opts: opts, revoked: map[string]time.Time{}}, nil
}

var _ access.IdentityProvider = (*Demo)(nil)

// claims — содержимое токена сеанса.
type claims struct {
	SessionID string `json:"sid"`
	PersonID  string `json:"sub"`
	Role      string `json:"role"`
	Scope     string `json:"scope,omitempty"`
	Demo      bool   `json:"demo,omitempty"`
	Expires   int64  `json:"exp"`
}

// Identify — субъект по cookie сеанса или заголовку демо-персоны; нет
// сеанса, подпись не сходится, срок истёк, сеанс закрыт — анонимный без ошибки.
func (d *Demo) Identify(_ context.Context, cred access.Credentials) (platform.Principal, error) {
	if cred.SessionToken != "" {
		if c, ok := d.verify(cred.SessionToken); ok {
			return d.principal(c), nil
		}
	}
	if cred.DemoPersona != "" && d.opts.Personas {
		if p, ok := d.dir.Persona(cred.DemoPersona); ok {
			return d.principal(claims{SessionID: "header:" + p.ID, PersonID: p.ID, Role: p.Role.ID, Scope: p.Scope, Demo: true}), nil
		}
	}
	return platform.Principal{}, nil
}

// Open — вход демо-персоной (persona_id); вход по логину и паролю — эпик 08
// во втором слое (сейчас access.login_failed).
func (d *Demo) Open(_ context.Context, rq access.SessionCreate) (platform.Principal, string, error) {
	if rq.PersonaID == "" || !d.opts.Personas {
		return platform.Principal{}, "", platform.Fail(errcodes.AccessLoginFailed)
	}
	p, ok := d.dir.Persona(rq.PersonaID)
	if !ok {
		return platform.Principal{}, "", platform.Fail(errcodes.AccessLoginFailed)
	}
	sid := make([]byte, 16)
	if _, err := rand.Read(sid); err != nil {
		return platform.Principal{}, "", err
	}
	c := claims{
		SessionID: base64.RawURLEncoding.EncodeToString(sid), PersonID: p.ID, Role: p.Role.ID, Scope: p.Scope, Demo: true,
		Expires: d.opts.Now().Add(d.opts.TTL).Unix(),
	}
	token, err := d.sign(c)
	if err != nil {
		return platform.Principal{}, "", err
	}
	return d.principal(c), token, nil
}

// Close — выход: сеанс в списке отозванных до истечения срока токена.
func (d *Demo) Close(_ context.Context, token string) error {
	c, ok := d.verify(token)
	if !ok {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	now := d.opts.Now()
	for sid, exp := range d.revoked {
		if now.After(exp) {
			delete(d.revoked, sid)
		}
	}
	d.revoked[c.SessionID] = time.Unix(c.Expires, 0)
	return nil
}

// DemoPersonas — демо-персоны экрана входа; вне fixtures и demo — api.not_found.
func (d *Demo) DemoPersonas(context.Context) ([]access.DemoPersona, error) {
	if !d.opts.Personas {
		return nil, platform.Fail(errcodes.ApiNotFound, "object", "демо-персоны", "id", "")
	}
	return append([]access.DemoPersona(nil), d.dir.Personas...), nil
}

func (d *Demo) principal(c claims) platform.Principal {
	p := platform.Principal{PersonID: c.PersonID, Role: c.Role, Scope: c.Scope, Demo: c.Demo, SessionID: c.SessionID}
	if pp, ok := d.dir.Persona(c.PersonID); ok {
		p.Name = pp.Name
	}
	p.Roles = d.dir.Hierarchy().Closure(c.Role)
	return p
}

func (d *Demo) mac(payload string) []byte {
	m := hmac.New(sha256.New, d.opts.Key)
	_, _ = m.Write([]byte("ant-session-v1\x00"))
	_, _ = m.Write([]byte(payload))
	return m.Sum(nil)
}

func (d *Demo) sign(c claims) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(b)
	return payload + "." + base64.RawURLEncoding.EncodeToString(d.mac(payload)), nil
}

// verify — подпись, срок, отзыв и персона из политики.
func (d *Demo) verify(token string) (claims, bool) {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok {
		return claims{}, false
	}
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(got, d.mac(payload)) {
		return claims{}, false
	}
	b, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return claims{}, false
	}
	var c claims
	if err := json.Unmarshal(b, &c); err != nil || c.SessionID == "" {
		return claims{}, false
	}
	if !d.opts.Now().Before(time.Unix(c.Expires, 0)) {
		return claims{}, false
	}
	if _, ok := d.dir.Persona(c.PersonID); !ok {
		return claims{}, false
	}
	d.mu.Lock()
	_, gone := d.revoked[c.SessionID]
	d.mu.Unlock()
	return c, !gone
}
