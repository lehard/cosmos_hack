package main

import (
	"context"

	"ant/cmd/internal/config"
	accessapp "ant/internal/application/access"
	nonconformityapp "ant/internal/application/nonconformity"
)

// nonconformityLive — live-реализация ведущих портов nonconformity для роли
// api (эпик 21): чтение — свёртка изделия из журнала ядра на момент (AD-22);
// команды — доменный гард над той же свёрткой и решение в журнал ядра с
// проверками AD-39 (поток изделия, поток разрешения на отклонение, атомарный
// расход лимита). Доменное «сейчас» — часы журнала (AD-37).
//
// Порт «маршрут подписей закрыт» (AD-43, режим 4): до модуля documents
// (эпик 28) в профилях demo и fixtures — разрешающая заглушка DemoRoutes
// (решение помечено approvals_status = demo_stub), иначе — PendingRoutes
// (решение не исполняется до document.route.closed).
func nonconformityLive(ctx context.Context, env *environment, dir *accessapp.Directory) (*nonconformityapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	var routes nonconformityapp.RouteGate = nonconformityapp.PendingRoutes{}
	if env.cfg.Profile == config.ProfileDemo || env.cfg.Profile == config.ProfileFixtures {
		routes = nonconformityapp.DemoRoutes{}
	}
	var auth nonconformityapp.AuthorityCheck
	if dir != nil {
		auth = dir // полномочия точки предъявления — по политике (FR-19)
	}
	return nonconformityapp.NewService(
		nonconformityapp.WithDeps(nonconformityapp.Deps{
			Journal: c.journal, Codec: c.codec, Bundles: c.bundleSource(), // та же версия, что у воркера
			DomainClock: c.domainClock(), Routes: routes, Now: c.codec.Now, Authorities: auth,
		}),
		nonconformityapp.WithConfig(nonconformityapp.Config{
			DomainBuild: c.codec.DomainBuild, Partitions: env.cfg.Engine.Partitions,
			// Профиль demo: recorded_at решения — доменное «сейчас» сценария (AD-37).
			ScenarioClock: scenarioClock(env.cfg),
		}),
	), nil
}
