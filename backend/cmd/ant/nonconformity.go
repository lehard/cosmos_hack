package main

import (
	"context"

	"ant/cmd/internal/config"
	nonconformityapp "ant/internal/application/nonconformity"
	referenceapp "ant/internal/application/reference"
	"ant/internal/infrastructure/storage/journal/clock"
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
func nonconformityLive(ctx context.Context, env *environment) (*nonconformityapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	var routes nonconformityapp.RouteGate = nonconformityapp.PendingRoutes{}
	if env.cfg.Profile == config.ProfileDemo || env.cfg.Profile == config.ProfileFixtures {
		routes = nonconformityapp.DemoRoutes{}
	}
	return nonconformityapp.NewService(
		nonconformityapp.WithDeps(nonconformityapp.Deps{
			Journal: c.journal, Codec: c.codec, Bundles: c.bundleSource(), // та же версия, что у воркера
			DomainClock: clock.NewJournal(c.journal), Routes: routes, Now: c.codec.Now,
			// Срок решения — по производственному календарю справочника (эпик 19, FR-55).
			Calendar: referenceapp.WorkingCalendar{Source: c.refSource},
		}),
		nonconformityapp.WithConfig(nonconformityapp.Config{
			DomainBuild: c.codec.DomainBuild, Partitions: env.cfg.Engine.Partitions,
		}),
	), nil
}
