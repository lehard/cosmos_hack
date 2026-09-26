package main

import (
	"context"
	"sync"

	crossitemapp "ant/internal/application/crossitem"
	engineapp "ant/internal/application/engine"
	itemapp "ant/internal/application/item"
	dom "ant/internal/domain/item"
	itemstore "ant/internal/infrastructure/storage/item"
)

// Сборка модулей item и crossitem (эпик 18): нормативная часть item —
// встроенная номенклатура normative/ (зоны и связи по КД), пока источник
// версии процесса (эпик 17) не собирает её сам; чтение — свёртка изделия на
// момент и состояние межизделийной стадии; команды — гард, затем журнал ядра.

var itemEnvOnce = sync.OnceValues(func() (dom.Env, error) { return itemstore.SeedEnv() })

// itemBundles — BundleSource с частью нормативного слоя item поверх next.
func (c *core) itemBundles(next engineapp.BundleSource) engineapp.BundleSource {
	env, err := itemEnvOnce()
	if err != nil {
		// Встроенная копия проверена тестом; сбой разбора — ошибка сборки.
		panic(err)
	}
	return itemapp.Bundles{Next: next, Env: env}
}

// itemWriter — запись команд item и crossitem в журнал ядра.
func (c *core) itemWriter(env *environment) itemapp.JournalWriter {
	return itemapp.JournalWriter{Journal: c.journal, DomainBuild: c.codec.DomainBuild, Partitions: env.cfg.Engine.Partitions, Now: c.codec.Now}
}

// itemLive — живые операции item (AD-36).
func itemLive(ctx context.Context, env *environment) (*itemapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	ienv, err := itemEnvOnce()
	if err != nil {
		return nil, err
	}
	pv, err := itemstore.SeedProcessVersion()
	if err != nil {
		return nil, err
	}
	return itemapp.NewLive(itemapp.Config{Codec: c.codec, Projections: c.engine, Bundles: c.bundleSource(), Writer: c.itemWriter(env),
		Clock: c.domainClock(), Env: ienv, NormativeRev: qualityRev,
		// Новые изделия — по действующей версии процесса (эпик 39, FR-22);
		// стартовая версия — запасная, пока версий в хранилище нет.
		ProcessVersion: pv, ActiveProcess: &activeProcess{store: c.versions}}), nil
}

// crossitemLive — живые операции crossitem (AD-36).
func crossitemLive(ctx context.Context, env *environment) (*crossitemapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	return crossitemapp.NewLive(crossitemapp.Config{Projections: c.engine, Writer: c.itemWriter(env), Clock: c.domainClock()}), nil
}
