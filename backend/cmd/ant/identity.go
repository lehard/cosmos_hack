package main

import (
	"context"
	"crypto/sha256"
	"os"
	"sync"
	"time"

	"ant/cmd/internal/config"
	accessapp "ant/internal/application/access"
	accessdom "ant/internal/domain/access"
	"ant/internal/infrastructure/fixtures/world"
	"ant/internal/infrastructure/security/casbin"
	"ant/internal/infrastructure/security/identity"
	accessstore "ant/internal/infrastructure/storage/access"
	"ant/internal/infrastructure/storage/journal/clock"
)

// accessBundle — вход и права (эпик 08): каталог затравки (демо-персоны,
// столы ролей), проекция политики, вычислитель Casbin, порт входа, места,
// шина безопасности и зависимости живых операций access.
type accessBundle struct {
	directory *accessapp.Directory
	identity  accessapp.IdentityProvider
	control   accessapp.AccessControl
	policy    accessapp.PolicySource
	places    accessapp.Places
	events    accessapp.SecurityEvents
	creds     accessapp.CredentialStore
	hasher    accessapp.PasswordHasher
	decisions accessapp.DecisionWriter
	// now — доменное «сейчас» (AD-37) для сроков полномочий и записей решений.
	now func(ctx context.Context) (time.Time, error)
}

// seedFS — встроенная копия нормативного слоя (world.Inputs: normative/policy,
// normative/desks, справочник мест; совпадение с репозиторием проверяет тест
// генератора заготовок), пока стартовая политика не приходит из журнала
// генезисом (эпик 05).
func seedFS() (*accessapp.Directory, accessdom.Policy, identity.Places, error) {
	fsys := world.Inputs()
	dir, err := identity.LoadDirectory(fsys)
	if err != nil {
		return nil, accessdom.Policy{}, nil, err
	}
	seed, err := identity.LoadSeed(fsys)
	if err != nil {
		return nil, accessdom.Policy{}, nil, err
	}
	places, err := identity.LoadPlaces(fsys)
	if err != nil {
		return nil, accessdom.Policy{}, nil, err
	}
	return dir, accessdom.FromSeed(seed), places, nil
}

// personasAllowed — вход демо-персоной без пароля: только профили fixtures и demo (FR-128).
func personasAllowed(cfg *config.Config) bool {
	return cfg.Profile == config.ProfileFixtures || cfg.Profile == config.ProfileDemo
}

// accessLive — вход и права роли api по ключам identity_provider и
// access_control (AD-35):
//   - identity_provider = local — логин и пароль (argon2id), сеансы scs в
//     Postgres, блокировка и ограничение частоты; в fixtures и demo — ещё
//     и демо-персоны; demo — только демо-персоны, токен HMAC без базы;
//   - access_control = casbin — Casbin над проекцией политики из журнала;
//     permissive — разрешающая заглушка волны 1.
func accessLive(ctx context.Context, env *environment) (*accessBundle, error) {
	cfg := env.cfg
	dir, seed, places, err := seedFS()
	if err != nil {
		return nil, err
	}
	b := &accessBundle{directory: dir, places: places, hasher: identity.DefaultArgon2id}
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	dc := &cachedClock{src: clock.NewJournal(c.journal), every: time.Second}
	b.now = dc.Now
	proj := accessapp.NewProjection(seed, accessstore.PolicyLog{Journal: c.journal, Codec: c.codec, Signal: c.listener})
	b.policy = proj
	b.events = accessstore.NewSecurityBus(c.journal, c.codec)
	b.creds = accessstore.NewCredentials(c.pool)
	b.decisions = accessapp.JournalDecisions{Journal: c.journal, DomainBuild: c.codec.DomainBuild, Now: c.codec.Now}

	switch cfg.Ports.Adapters["identity_provider"] {
	case "demo":
		b.identity, err = demoIdentity(cfg, dir)
	default: // local
		b.identity, err = identity.NewLocal(dir, proj, b.creds, b.hasher, b.events, identity.SessionStore(c.pool), identity.LocalOptions{
			Personas: personasAllowed(cfg),
			At:       func(ctx context.Context) time.Time { t, _ := dc.Now(ctx); return t },
		})
	}
	if err != nil {
		return nil, err
	}
	if cfg.Ports.Adapters["access_control"] == "casbin" {
		b.control = casbin.New(proj)
	}
	return b, nil
}

// demoIdentity — вход без базы (ключ identity_provider: demo): демо-персоны,
// токен сеанса HMAC в cookie. Ключ подписи выводится из секрета БД (файл в
// томе секретов, AD-25): одинаков у копий api и переживает перезапуск; файла
// нет — случайный ключ процесса.
func demoIdentity(cfg *config.Config, dir *accessapp.Directory) (accessapp.IdentityProvider, error) {
	opts := identity.Options{Personas: personasAllowed(cfg)}
	if cfg.DB.PasswordFile != "" {
		if secret, err := os.ReadFile(cfg.DB.PasswordFile); err == nil && len(secret) > 0 {
			k := sha256.Sum256(append([]byte("ant-session-key-v1\x00"), secret...))
			opts.Key = k[:]
		}
	}
	return identity.NewDemo(dir, opts)
}

// cachedClock — доменные часы журнала с кэшем на every (права проверяются на
// каждый запрос, а «сейчас» сценария меняется тиками). Нет тика — системное время.
type cachedClock struct {
	src interface {
		Now(context.Context) (time.Time, error)
	}
	every time.Duration

	mu   sync.Mutex
	at   time.Time
	last time.Time
}

// Now — доменное «сейчас».
func (c *cachedClock) Now(ctx context.Context) (time.Time, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if !c.last.IsZero() && now.Sub(c.last) < c.every {
		return c.at.Add(now.Sub(c.last)), nil
	}
	t, err := c.src.Now(ctx)
	if err != nil {
		t = now.UTC()
	}
	c.at, c.last = t, now
	return t, nil
}
