package main

import (
	"context"

	"ant/cmd/internal/config"
	crossitemapp "ant/internal/application/crossitem"
	ingestapp "ant/internal/application/ingest"
	appjournal "ant/internal/application/journal"
	securityapp "ant/internal/application/security"
	ingeststore "ant/internal/infrastructure/storage/ingest"
	"ant/internal/infrastructure/storage/journal/clock"
	materialsstore "ant/internal/infrastructure/storage/materials"
)

// ingestLive — live-приём (эпик 06) на ядре процесса: факты пишутся в журнал
// ядра (эпик 04) и сворачиваются движком (эпик 07); реестр идемпотентности,
// учёт source_seq и карантин — схема ingest на пуле ant_app (строки реестра —
// в транзакции journal.Append, AD-45); содержимое карантина — в томе
// материалов (AD-23). Партиция факта — kernel.PartitionOf с тем же P, что у
// движка (engine.partitions, AD-6).
//
// Подпись источника: в профилях demo и fixtures неподписанное принимается с
// пометкой «подпись не проверялась» (заметка эпика 06) до эпика 05; ключ шлюза
// и реестр ключей — эпик 05; метрики приёма (FR-41) — в телеметрию процесса
// (/metrics, эпик 34) и в счётчики стола администратора.
func ingestLive(ctx context.Context, env *environment) (*ingestapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	cfg := env.cfg
	mat, err := materialsstore.NewVolume(cfg.Materials.Dir, materialsOptions(cfg)...)
	if err != nil {
		return nil, err
	}
	store := ingeststore.NewStore(c.pool)
	// Эпик 05: проверка подписи источников по реестру ключей из генезиса и
	// подпись служебных записей ключом шлюза gateway-ingest@1 (genesisIngest, init.go).
	sigVerifier, sigKeys, sigSigner := genesisIngest(env, c)
	ic := ingestapp.DefaultConfig()
	ic.Profile = cfg.Profile
	if cfg.Profile == config.ProfileFixtures || cfg.Profile == config.ProfileDemo {
		ic.Signature = ingestapp.SignatureDemoUnverified
	}
	ic.Partitions = cfg.Engine.Partitions
	ic.StagePartition = cfg.Engine.Partitions // партиция записей вне изделия — за пределами 0…P-1
	ic.DomainBuild = domainBuild()
	// Профиль demo — журнал в режиме часов scenario (AD-37, эпик 16): время
	// приёма — доменное «сейчас» прогона, recorded_at — оно же.
	var domain appjournal.DomainClock = clock.SystemDomain{}
	if scenarioClock(cfg) {
		ic.ScenarioClock = true
		domain = c.domainClock()
	}
	return ingestapp.NewService(
		ingestapp.WithConfig(ic),
		ingestapp.WithDeps(ingestapp.Deps{
			// Ключи из генезиса (эпик 05).
			Verifier: sigVerifier, Keys: sigKeys, ServerSigner: sigSigner,
			Journal:    c.journal,
			Registry:   store,
			Quarantine: store,
			Materials:  mat,
			// Реестр носителей стадии (эпик 18, AD-41): разрешение до выбора партиции.
			Carriers:    crossitemapp.ProjectedCarriers{Store: c.engine},
			DomainClock: domain,
			InfraClock:  clock.System{},
			Telemetry:   env.telemetry(),
			// Шина безопасности модуля security (эпик 29) вместо моста эпика 06.
			Security: securityapp.IngestBus{Enc: securityEncoder(cfg)},
			// Эпик 48: выключенные источники и интеграции отвергаются (AD-28, AD-47).
			Gate: integrationSwitch(env, c),
		}),
	), nil
}
