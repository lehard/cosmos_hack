package main

import (
	"context"

	"ant/cmd/internal/config"
	crossitemapp "ant/internal/application/crossitem"
	ingestapp "ant/internal/application/ingest"
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
// и реестр ключей — эпик 05, Prometheus — эпик 34 (Telemetry nil — счётчики
// только для стола администратора).
func ingestLive(ctx context.Context, env *environment) (*ingestapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	cfg := env.cfg
	mat, err := materialsstore.NewVolume(cfg.Materials.Dir)
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
			DomainClock: clock.SystemDomain{},
			InfraClock:  clock.System{},
		}),
	), nil
}
