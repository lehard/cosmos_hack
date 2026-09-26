package identity

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	accessdom "ant/internal/domain/access"
)

// memCreds — учётные данные в памяти (порт CredentialStore).
type memCreds struct {
	mu sync.Mutex
	m  map[string]access.Credential
}

func (s *memCreds) Get(_ context.Context, login string) (access.Credential, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.m[login]
	return c, ok, nil
}

func (s *memCreds) Create(_ context.Context, c access.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[c.Login]; ok {
		return access.ErrLoginTaken
	}
	s.m[c.Login] = c
	return nil
}

func (s *memCreds) Activate(_ context.Context, login, person string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.m[login]
	c.Status, c.PersonID, c.Failures, c.LockedUntil = access.AccountActive, person, 0, time.Time{}
	s.m[login] = c
	return nil
}

func (s *memCreds) Failed(_ context.Context, login string, now time.Time, max int, lockFor time.Duration) (access.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.m[login]
	c.Failures++
	if c.Failures >= max {
		c.Failures, c.LockedUntil = 0, now.Add(lockFor)
	}
	s.m[login] = c
	return c, nil
}

func (s *memCreds) Succeeded(_ context.Context, login string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.m[login]
	c.Failures, c.LockedUntil = 0, time.Time{}
	s.m[login] = c
	return nil
}

func (s *memCreds) List(context.Context) ([]access.Credential, error) { return nil, nil }

// events — события шины безопасности.
type events struct {
	mu   sync.Mutex
	auth []access.AuthFailure
}

func (e *events) AuthFailed(_ context.Context, f access.AuthFailure) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.auth = append(e.auth, f)
	return nil
}

func (e *events) AccessDenied(context.Context, access.Denial) error { return nil }

func (e *events) last() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.auth) == 0 {
		return ""
	}
	return e.auth[len(e.auth)-1].Reason
}

// fastHash — argon2id с малой памятью для тестов.
var fastHash = Argon2id{Memory: 64, Time: 1, Threads: 1, KeyLen: 32, SaltLen: 16}

func code(err error) errcodes.Code {
	var pe *platform.Error
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

func TestArgon2id(t *testing.T) {
	h, err := DefaultArgon2id.Hash("пароль-1")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := DefaultArgon2id.Verify(h, "пароль-1"); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if ok, _ := DefaultArgon2id.Verify(h, "пароль-2"); ok {
		t.Fatal("чужой пароль")
	}
	if _, err := DefaultArgon2id.Verify("$2a$10$bcrypt", "x"); err == nil {
		t.Fatal("не argon2id")
	}
}

// Барьер 1 (AD-15, FR-128): вход по логину и паролю, блокировка после N
// неудач, ограничение частоты, заявка до активации, демо-персоны только в
// демо-профилях, события неудачного входа.
func TestLocalLogin(t *testing.T) {
	ctx := access.WithClientIP(context.Background(), "10.0.0.1")
	dir := directory(t)
	seed, err := LoadSeed(os.DirFS(repo))
	if err != nil {
		t.Fatal(err)
	}
	pol := accessdom.FromSeed(seed)
	pol.Persons = append(pol.Persons, accessdom.Person{ID: "U-ivanov", Name: "Иванов", Login: "ivanov", Active: true})
	pol.Assignments = append(pol.Assignments, accessdom.Assignment{PersonID: "U-ivanov", RoleID: "head_of_qc", Scope: "ent01"})
	pol.Seq = 7
	creds := &memCreds{m: map[string]access.Credential{}}
	hash, _ := fastHash.Hash("верный-пароль")
	_ = creds.Create(ctx, access.Credential{Login: "ivanov", PersonID: "U-ivanov", Hash: hash, Status: access.AccountActive})
	_ = creds.Create(ctx, access.Credential{Login: "petrov", PersonID: "U-petrov", Hash: hash, Status: access.AccountPending})
	ev := &events{}
	now := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	idp, err := NewLocal(dir, access.StaticPolicy(pol), creds, fastHash, ev, nil, LocalOptions{
		Personas: true, Now: func() time.Time { return now }, PerIP: time.Millisecond, PerLogin: time.Millisecond, Burst: 100})
	if err != nil {
		t.Fatal(err)
	}

	p, token, err := idp.Open(ctx, access.SessionCreate{Login: "Ivanov", Password: "верный-пароль"})
	if err != nil || token == "" || p.Role != "head_of_qc" || p.Demo || p.PolicySeq != 7 {
		t.Fatalf("вход: %+v %q %v", p, token, err)
	}
	if !p.HasRole("quality_inspector") || !p.HasRole("staff") {
		t.Fatalf("наследование ролей в субъекте: %v", p.Roles)
	}
	got, _ := idp.Identify(ctx, access.Credentials{SessionToken: token})
	if got.PersonID != "U-ivanov" || got.Name != "Иванов" || got.SessionID == "" || got.SessionID == token {
		t.Fatalf("сеанс: %+v", got)
	}
	if err := idp.Close(ctx, token); err != nil {
		t.Fatal(err)
	}
	if got, _ := idp.Identify(ctx, access.Credentials{SessionToken: token}); !got.Anonymous() {
		t.Fatal("после выхода")
	}

	// Неизвестный логин и неверный пароль — одинаковый отказ и событие.
	if _, _, err := idp.Open(ctx, access.SessionCreate{Login: "nobody", Password: "x"}); code(err) != errcodes.AccessLoginFailed || ev.last() != access.AuthBadCredentials {
		t.Fatal(err, ev.last())
	}
	// Заявка без активации — account_pending (пароль верный).
	if _, _, err := idp.Open(ctx, access.SessionCreate{Login: "petrov", Password: "верный-пароль"}); code(err) != errcodes.AccessAccountPending || ev.last() != access.AuthAccountPending {
		t.Fatal(err)
	}
	// Блокировка после 5 неудач: и верный пароль не пускает до конца срока.
	for i := 0; i < 5; i++ {
		if _, _, err := idp.Open(ctx, access.SessionCreate{Login: "ivanov", Password: "неверный"}); code(err) != errcodes.AccessLoginFailed {
			t.Fatal(err)
		}
	}
	if ev.last() != access.AuthAccountLocked {
		t.Fatalf("пятая неудача блокирует: %s", ev.last())
	}
	if _, _, err := idp.Open(ctx, access.SessionCreate{Login: "ivanov", Password: "верный-пароль"}); code(err) != errcodes.AccessLoginFailed {
		t.Fatal("вход при блокировке")
	}
	now = now.Add(16 * time.Minute)
	if _, _, err := idp.Open(ctx, access.SessionCreate{Login: "ivanov", Password: "верный-пароль"}); err != nil {
		t.Fatal("после блокировки", err)
	}

	// Демо-персона без пароля и заголовок — только в демо-профилях.
	p, _, err = idp.Open(ctx, access.SessionCreate{PersonaID: "INS-01"})
	if err != nil || !p.Demo || p.Role != "quality_inspector" {
		t.Fatal(p, err)
	}
	if got, _ := idp.Identify(ctx, access.Credentials{DemoPersona: "TEC-01"}); got.Role != "technologist" {
		t.Fatal(got)
	}
	prod, _ := NewLocal(dir, access.StaticPolicy(pol), creds, fastHash, ev, nil, LocalOptions{})
	if _, _, err := prod.Open(ctx, access.SessionCreate{PersonaID: "INS-01"}); code(err) != errcodes.AccessLoginFailed {
		t.Fatal("персона вне демо-профиля")
	}
	if got, _ := prod.Identify(ctx, access.Credentials{DemoPersona: "INS-01"}); !got.Anonymous() {
		t.Fatal("заголовок вне демо-профиля")
	}
	if _, err := prod.DemoPersonas(ctx); code(err) != errcodes.ApiNotFound {
		t.Fatal("персоны вне демо-профиля")
	}
}

// Ограничение частоты (golang.org/x/time/rate): после запаса попыток — 429 и событие rate_limited.
func TestLocalRateLimit(t *testing.T) {
	ctx := access.WithClientIP(context.Background(), "10.0.0.2")
	creds := &memCreds{m: map[string]access.Credential{}}
	ev := &events{}
	now := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	idp, err := NewLocal(directory(t), access.StaticPolicy(accessdom.Policy{}), creds, fastHash, ev, nil, LocalOptions{
		Now: func() time.Time { return now }, PerIP: time.Minute, PerLogin: time.Minute, Burst: 3, MaxFailures: 100})
	if err != nil {
		t.Fatal(err)
	}
	var limited bool
	for i := 0; i < 10; i++ {
		if _, _, err := idp.Open(ctx, access.SessionCreate{Login: "x" + string(rune('a'+i)), Password: "p"}); code(err) == errcodes.ApiRateLimited {
			limited = true
			break
		}
	}
	if !limited || ev.last() != access.AuthRateLimited {
		t.Fatal("ограничение частоты по адресу", ev.last())
	}
	now = now.Add(time.Hour)
	if _, _, err := idp.Open(ctx, access.SessionCreate{Login: "y", Password: "p"}); code(err) != errcodes.AccessLoginFailed {
		t.Fatal("после паузы", err)
	}
}
