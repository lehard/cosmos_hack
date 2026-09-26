package process

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
	dp "ant/internal/domain/process"
)

// Bundles — реализация ведомого порта движка BundleSource (AD-17): версия
// процесса, закреплённая при запуске изделия (хеш XML в item.item.registered),
// разложенная по модулям движка. Перед исполнением байты версии проверяются:
// H(байты) = закреплённый хеш = утверждённый подписями, кворум полон (FR-23);
// иначе process.Env.Refusal — изделие не исполняется, гарды отказывают.
// Модулю machinelogs передаётся признак «специальный процесс» (FR-151).
type Bundles struct {
	Store  VersionStore
	Quorum QuorumVerifier
	// Reference — срез справочников для предусловий (эпики 19, 26); nil — нет.
	Reference func(ctx context.Context, itemID string) (dp.Reference, error)
	// Plan — план контроля по типу изделия (plan.* условий); nil — умолчания.
	Plan map[string]bool
	// Now, TTL — InfraClock и срок кэша проверенной версии: байты версии
	// перечитываются и перепроверяются не реже TTL (изменение в обход
	// системы обнаруживается при запуске и в работе).
	Now func() time.Time
	TTL time.Duration

	mu    sync.Mutex
	cache map[string]cachedEnv
}

type cachedEnv struct {
	env dp.Env
	at  time.Time
}

var _ engineapp.BundleSource = (*Bundles)(nil)

// DefaultBundleTTL — срок кэша проверенной версии.
const DefaultBundleTTL = 5 * time.Second

// Registration — запуск изделия во входе: хеш версии и ревизия нормативного слоя.
type Registration struct {
	Hash         string `json:"process_version_hash"`
	NormativeRev string `json:"normative_rev"`
	ItemTypeID   string `json:"item_type_id"`
}

// FindRegistration — первая item.item.registered входа изделия.
func FindRegistration(input []kernel.Record) (Registration, bool) {
	for _, r := range input {
		if r.Type != catalog.ItemItemRegistered {
			continue
		}
		var reg Registration
		if json.Unmarshal(r.Data, &reg) == nil {
			return reg, true
		}
	}
	return Registration{}, false
}

// Bundle — нормативный слой изделия и его ревизия (normative_rev записей).
func (b *Bundles) Bundle(ctx context.Context, itemID string, input []kernel.Record) (engine.Bundle, string, error) {
	reg, ok := FindRegistration(input)
	if !ok {
		return engine.Bundle{}, "", nil
	}
	env, err := b.Env(ctx, reg.Hash)
	if err != nil {
		return engine.Bundle{}, "", err
	}
	if b.Reference != nil {
		if env.Reference, err = b.Reference(ctx, itemID); err != nil {
			return engine.Bundle{}, "", err
		}
	}
	bundle := engine.Bundle{Process: env}
	if env.Def != nil {
		bundle.Machinelogs = machinelogs.Env{SpecialSteps: env.Def.SpecialSteps()}
	}
	rev := reg.NormativeRev
	if rev == "" {
		rev = env.VersionID
	}
	return bundle, rev, nil
}

// Env — нормативный слой процесса по закреплённому хешу версии.
func (b *Bundles) Env(ctx context.Context, hash string) (dp.Env, error) {
	now := time.Now
	if b.Now != nil {
		now = b.Now
	}
	ttl := b.TTL
	if ttl <= 0 {
		ttl = DefaultBundleTTL
	}
	b.mu.Lock()
	c, hit := b.cache[hash]
	b.mu.Unlock()
	if hit && now().Sub(c.at) < ttl {
		return c.env, nil
	}
	env, err := b.load(ctx, hash)
	if err != nil {
		return dp.Env{}, err
	}
	b.mu.Lock()
	if b.cache == nil {
		b.cache = map[string]cachedEnv{}
	}
	if old, ok := b.cache[hash]; ok && old.env.Def != nil && env.Def != nil && old.env.VersionHash == env.VersionHash && env.Refusal == nil {
		env.Def = old.env.Def // одна модель на версию
	}
	b.cache[hash] = cachedEnv{env: env, at: now()}
	b.mu.Unlock()
	return env, nil
}

func (b *Bundles) load(ctx context.Context, hash string) (dp.Env, error) {
	env := dp.Env{VersionHash: hash, Plan: b.Plan}
	rec, ok, err := b.Store.ByHash(ctx, hash)
	if err != nil {
		return env, err
	}
	if !ok {
		r := kernel.Refuse(errcodes.ProcessVersionUnknown, "hash", hash)
		env.Refusal = r
		return env, nil
	}
	env.VersionID, env.Label = rec.ID, rec.Label
	q := b.Quorum
	if q == nil {
		q = RecordedQuorum{}
	}
	appr, err := q.Approval(ctx, rec)
	if err != nil {
		return env, err
	}
	if r := dp.CheckExecutable(rec.ID, hash, rec.XML, appr); r != nil {
		env.Refusal = r
		return env, nil
	}
	def, vs := dp.Load(rec.XML, dp.LoadOptions{})
	if errs := dp.Errors(vs); len(errs) > 0 {
		parts := make([]string, 0, len(errs))
		for _, v := range errs {
			parts = append(parts, v.String())
		}
		r := kernel.Refuse(errs[0].Code, "element", errs[0].Element, "detail", errs[0].Detail)
		r.Detail = "версия " + rec.ID + " не проходит проверку: " + strings.Join(parts, "; ")
		env.Refusal = r
		return env, nil
	}
	env.Def = def
	return env, nil
}
