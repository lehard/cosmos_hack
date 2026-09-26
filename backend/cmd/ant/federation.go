package main

import (
	"context"

	federationapp "ant/internal/application/federation"
	"ant/internal/infrastructure/security/profiles"
	materialsstore "ant/internal/infrastructure/storage/materials"
)

// Модуль federation (эпик 41; FR-131, FR-132, AD-19): партнёры, выписки
// паспорта, порт межзаводского обмена. Пакеты выписок — в хранилище
// материалов; проверка подписей — профили gost/pq (profiles.Verify);
// исходящая выписка подписывается ключом шлюза предприятия gateway-ingest@1
// из тома ant (нет ключа — пакет без подписи шлюза).

// federationGatewayKey — ключ шлюза предприятия (уровень 0) для исходящих выписок.
const federationGatewayKey = "gateway-ingest@1"

// federationLive — live-реализация операций federation для роли api.
func federationLive(ctx context.Context, env *environment) (*federationapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	cfg := env.cfg
	deps := federationapp.Deps{Journal: c.journal, Codec: c.codec, Crypto: profiles.Verify, Enterprise: cfg.ERP.OneC.Enterprise,
		Now: c.codec.Now, Log: env.log, DomainBuild: domainBuild(), Partitions: cfg.Engine.Partitions}
	if deps.Enterprise == "" {
		deps.Enterprise = "ENT01"
	}
	if mat, err := materialsstore.NewVolume(cfg.Materials.Dir, materialsOptions(cfg)...); err == nil {
		deps.Materials = mat
	} else {
		env.log.Warn("federation: хранилище материалов недоступно — пакеты выписок не сохраняются", "err", err)
	}
	if _, _, signer := genesisIngest(env, c); signer != nil {
		deps.Sign = func(ctx context.Context, payloadType string, payload []byte) ([]byte, error) {
			return signer.Sign(ctx, payloadType, payload, federationGatewayKey)
		}
	}
	return federationapp.NewService(deps), nil
}
