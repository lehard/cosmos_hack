package main

import (
	"context"

	"ant/cmd/internal/config"
	documentsapp "ant/internal/application/documents"
	engineapp "ant/internal/application/engine"
	domdocs "ant/internal/domain/documents"
	storagedocs "ant/internal/infrastructure/storage/documents"
	"ant/internal/infrastructure/storage/journal/clock"
)

// Модуль documents (эпик 28): документы — детерминированные проекции журнала
// (AD-12). Нормативная часть — шаблоны и срез стартовой политики из
// встроенной копии normative/ (storage/documents.Seed); в профилях demo и
// fixtures подписи без агента токена засчитываются с пометкой
// verification = demo (Д-30), иначе — full.

// documentsEnv — шаблоны документов и срез политики для свёртки и api.
func documentsEnv(env *environment) domdocs.Env {
	mode := domdocs.VerificationFull
	if env.cfg.Profile == config.ProfileDemo || env.cfg.Profile == config.ProfileFixtures {
		mode = domdocs.VerificationDemo
	}
	e, err := storagedocs.SeedEnv(mode)
	if err != nil {
		panic(err) // встроенная копия проверена тестом TestSeedMatchesRepo
	}
	// Политика для обязательных подписей (эпик 26, AD-43): стартовая политика
	// нормативного слоя — детерминированный вход свёртки (сфера выдачи прав,
	// кандидаты этапов). Политика на basis_seq из проекции — когда движок
	// получит её входом (AD-45).
	if _, seed, _, err := seedFS(); err == nil {
		e.Policy = seed
	}
	return e
}

// documentsBundles — нормативный слой изделия с частью documents поверх next
// (названия шагов — из описания процесса закреплённой версии).
func (c *core) documentsBundles(next engineapp.BundleSource) engineapp.BundleSource {
	return documentsapp.Bundles{Next: next, Env: c.docsEnv}
}

// documentsLive — live-реализация операций documents для роли api: чтение —
// свёртка изделия или потока документа из журнала на момент (AD-22);
// команды — доменный гард маршрута и запись в журнал ядра с проверкой AD-39.
func documentsLive(ctx context.Context, env *environment) (*documentsapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	return documentsapp.NewService(
		documentsapp.WithDeps(documentsapp.Deps{
			Journal: c.journal, Codec: c.codec, Bundles: c.bundleSource(), Env: c.docsEnv,
			DomainClock: clock.NewJournal(c.journal), Now: c.codec.Now,
		}),
		documentsapp.WithConfig(documentsapp.Config{DomainBuild: c.codec.DomainBuild, Partitions: env.cfg.Engine.Partitions}),
	), nil
}
