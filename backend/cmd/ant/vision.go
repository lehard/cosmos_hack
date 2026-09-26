package main

import (
	"context"

	"ant/cmd/internal/config"
	appvision "ant/internal/application/vision"
	"ant/internal/infrastructure/storage/journal/feed"
)

// Сборка модуля vision (эпик 33): живые операции паспортов допуска над
// журналом ядра; стартовые паспорта демо — записи блока генезиса (ant init, эпик 05).

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

// visionRollback — правило автоотката версии анализатора (эпик 40, FR-101):
// глобальный потребитель роли projector с курсором vision.rollback (AD-45).
func visionRollback(env *environment, c *core) *appvision.Rollback {
	return &appvision.Rollback{Consumer: feed.NewConsumer(c.journal, c.leases, c.listener, c.feedOptions(env, "projector")),
		Codec: c.codec, Store: c.engine, Log: env.log.With("module", "vision")}
}
