package main

import (
	"context"
	"fmt"
	"slices"
	"time"

	"ant/cmd/internal/config"
	"ant/cmd/internal/db"
	appvision "ant/internal/application/vision"
	"ant/internal/contracts/normative"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/clock"
	"ant/internal/infrastructure/storage/journal/feed"
	storevision "ant/internal/infrastructure/storage/vision"
)

// Сборка модуля vision (эпик 33): живые операции паспортов допуска над
// журналом ядра и стартовые паспорта демо генезисом при migrate.

// visionLive — live-реализация операций vision для роли api (AD-36): реестр
// паспортов — свёртка записей analyzer.* журнала на момент (AD-22); команды —
// гард и запись-решение с проверкой AD-39. Порт «протокол допуска подписан»
// (AD-43): до documents (эпик 28) и полного допуска (эпик 40) в профилях
// demo и fixtures — заглушка DemoRoutes, иначе — PendingRoutes (допуск ждёт маршрута).
func visionLive(ctx context.Context, env *environment) (*appvision.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	var routes appvision.RouteGate = appvision.PendingRoutes{}
	if env.cfg.Profile == config.ProfileDemo || env.cfg.Profile == config.ProfileFixtures {
		routes = appvision.DemoRoutes{}
	}
	return appvision.NewService(
		appvision.WithDeps(appvision.Deps{Journal: c.journal, Codec: c.codec, DomainClock: c.domainClock(), Routes: routes, Now: c.codec.Now, Watch: c.engine}),
		appvision.WithConfig(appvision.Config{DomainBuild: c.codec.DomainBuild, Partitions: env.cfg.Engine.Partitions}),
	), nil
}

// seedVisionPassports — стартовые паспорта допуска анализаторов демо
// (normative/vision/analyzer-passports.v1.yaml; AD-29, AD-33): записи
// analyzer.passport.admitted с происхождением genesis в профилях затравки
// (demo, fixtures), в prod — никогда. Иначе камеры демо работают на уровне
// доверия 0. Повторный migrate — дубль, новых записей нет. Генезис целиком
// (ключи, политика, нормативный слой) — роль init эпика 05; полный маршрут
// допуска — эпик 40.
func seedVisionPassports(ctx context.Context, env *environment) error {
	seed, err := storevision.PassportsSeed()
	if err != nil {
		return fmt.Errorf("затравка паспортов: %w", err)
	}
	if !slices.Contains(seed.Profiles, normative.AnalyzerPassportsSeedProfilesElem(env.cfg.Profile)) {
		return nil
	}
	pc, err := db.Config(env.cfg.DB, "ant-migrate-seed")
	if err != nil {
		return err
	}
	pool, err := journalstore.NewAppPool(ctx, pc)
	if err != nil {
		return err
	}
	defer pool.Close()
	// Доверие (эпик 29): записи CA и шифрование при хранении — как у ядра.
	j := journalstore.NewStore(pool, clock.System{}, trustOptions(env.cfg, env)...)
	n, err := appvision.SeedPassports(ctx, j, seed, appvision.SeedConfig{Profile: env.cfg.Profile, DomainBuild: domainBuild(),
		Partitions: env.cfg.Engine.Partitions, Now: time.Now})
	if err != nil {
		return err
	}
	env.log.Info("затравка: паспорта допуска анализаторов", "profile", env.cfg.Profile, "written", n, "in_seed", len(seed.Passports))
	return nil
}

// visionRollback — правило автоотката версии анализатора (эпик 40, FR-101):
// глобальный потребитель роли projector с курсором vision.rollback (AD-45).
func visionRollback(env *environment, c *core) *appvision.Rollback {
	return &appvision.Rollback{Consumer: feed.NewConsumer(c.journal, c.leases, c.listener, c.feedOptions(env, "projector")),
		Codec: c.codec, Store: c.engine, Log: env.log.With("module", "vision")}
}
