package main

import (
	"context"
	"sync"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	qualityapp "ant/internal/application/quality"
	"ant/internal/domain/engine"
	"ant/internal/domain/quality"
	qualitystore "ant/internal/infrastructure/storage/quality"
)

// Сборка модуля quality (эпик 20): нормативный слой модуля — встроенная
// стартовая версия normative/ (классификатор, карта реакций, точки контроля
// BPMN, зоны), пока источник версии процесса (эпик 17) не собирает его сам;
// паспорта анализатора — из журнала; живые операции — над проекциями движка.

// qualityRev — ревизия встроенной стартовой версии нормативного слоя quality.
const qualityRev = "normative-seed-v1"

var qualityEnvOnce = sync.OnceValues(func() (quality.Env, error) { return qualitystore.SeedEnv(qualityRev) })

// qualityBundles — BundleSource с частью нормативного слоя quality поверх next
// (nil — пустой слой остальных модулей).
func (c *core) qualityBundles(next engineapp.BundleSource) engineapp.BundleSource {
	env, err := qualityEnvOnce()
	if err != nil {
		// Встроенная копия проверена тестом; сбой разбора — ошибка сборки.
		panic(err)
	}
	return qualityapp.Bundles{Next: next, Env: env, Passports: qualitystore.JournalPassports{Journal: c.journal, Codec: c.codec}}
}

// qualityLive — живая реализация операций quality (AD-36): проекции
// quality.item и quality.index, на момент в прошлом — свёртка на момент (AD-22).
func qualityLive(ctx context.Context, env *environment) (*qualityapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	qenv, err := qualityEnvOnce()
	if err != nil {
		return nil, err
	}
	sq := c.states()
	at := func(ctx context.Context, itemID string, m platform.Moment) (engine.Snapshot, error) {
		st, err := sq.Item(ctx, itemID, m)
		return st.Snapshot, err
	}
	return qualityapp.NewService(qualityapp.WithStore(c.engine), qualityapp.WithStateAt(at), qualityapp.WithEnv(qenv)), nil
}
