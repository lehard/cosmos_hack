package main

import (
	"context"
	"errors"
	"path/filepath"
	"sync"

	"ant/cmd/internal/config"
	appjournal "ant/internal/application/journal"
	securityapp "ant/internal/application/security"
	"ant/internal/infrastructure/integration/security/keeper"
	"ant/internal/infrastructure/security/atrest"
	"ant/internal/infrastructure/security/mtls"
	"ant/internal/infrastructure/security/permissive"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/feed"
	materialsstore "ant/internal/infrastructure/storage/materials"
)

// Доверие (эпик 29) на ядре процесса: журнал критических действий в
// транзакции каждой записи (CriticalHook), шифрование при хранении (KEK из
// своего тома), шина безопасности вместо моста эпика 06, хранитель
// (головы, отчёты верификатора, тревоги) и живые операции security.*.

// trustOptions — настройки хранилища журнала от модуля security: записи CA
// в той же транзакции (AD-28) и шифрование блока записи (AD-23).
func trustOptions(cfg *config.Config, env *environment) []journalstore.Option {
	opts := []journalstore.Option{journalstore.WithCritical(securityapp.CriticalHook{Enc: securityEncoder(cfg)})}
	k, err := atrest.Load(cfg.Security.KEKFile)
	switch {
	case err == nil:
		opts = append(opts, journalstore.WithCipher(k))
		env.log.Info("журнал: шифрование при хранении включено", "kek_id", k.KEKID(), "aead", k.AEAD())
	case errors.Is(err, atrest.ErrNoKEK) || cfg.Security.KEKFile == "":
		env.log.Warn("журнал: KEK нет — блок записи хранится открыто (AD-23: только для разработки)", "kek_file", cfg.Security.KEKFile)
	default:
		env.log.Error("журнал: KEK не читается — блок записи хранится открыто", "err", err)
	}
	return opts
}

// materialsOptions — шифрование материалов при хранении тем же KEK (AD-23).
func materialsOptions(cfg *config.Config) []materialsstore.Option {
	if k, err := atrest.Load(cfg.Security.KEKFile); err == nil {
		return []materialsstore.Option{materialsstore.WithSealer(k)}
	}
	return nil
}

// securityEncoder — сборка записей модуля security (эмитент security, AD-40).
func securityEncoder(cfg *config.Config) securityapp.Encoder {
	return securityapp.Encoder{Signer: permissive.Signer{}, DomainBuild: domainBuild(), StagePartition: cfg.Engine.Partitions}
}

// keeperClient — mTLS-клиент хранителя; nil — хранитель не настроен.
func keeperClient(cfg *config.Config) (securityapp.Keeper, error) {
	if cfg.Security.KeeperURL == "" {
		return nil, nil
	}
	tlsCfg, err := mtls.Client(mtls.Files{Dir: cfg.Security.PKIDir, Name: "ant"})
	if err != nil {
		return nil, err
	}
	return keeper.New(cfg.Security.KeeperURL, tlsCfg), nil
}

// securityLive — живые операции security.* (эпик 29): журнал CA, шина
// безопасности, индикатор целостности, отчёты верификатора.
func securityLive(ctx context.Context, env *environment) (*securityapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	k, err := keeperClient(env.cfg)
	if err != nil {
		env.log.Warn("хранитель: сертификат mTLS не загружен — отчёты целиком недоступны", "err", err)
	}
	opts := []securityapp.Option{securityapp.WithJournal(c.journal), securityapp.WithInterval(env.cfg.Security.VerifierInterval)}
	if k != nil {
		opts = append(opts, securityapp.WithKeeper(k))
	}
	return securityapp.NewService(opts...), nil
}

// runSecurity — роль security (эпик 29): одна копия-лидер по аренде
// передаёт хранителю головы (AD-8), забирает отчёты верификатора и тревоги
// хранителя (AD-46); подписчики шины безопасности — глобальные потребители
// со своими курсорами (AD-24, AD-45).
func runSecurity(ctx context.Context, env *environment) error {
	c, err := env.readyCore(ctx)
	if err != nil {
		return err
	}
	cfg := env.cfg
	emit := securityapp.Emitter{Journal: c.journal, Enc: securityEncoder(cfg)}
	subs := []securityapp.Subscriber{securityapp.LogSubscriber{Log: env.moduleLog("security")}}
	if cfg.Security.ExportFile != "" {
		// Экспорт во внешний мониторинг ИБ (JSON-строки) — ещё один подписчик
		// без изменения источников.
		subs = append(subs, &securityapp.FileExport{Path: filepath.Clean(cfg.Security.ExportFile)})
	}
	bus := &securityapp.Bus{Consumer: feed.NewConsumer(c.journal, c.leases, c.listener, c.feedOptions(env, "security")),
		Journal: c.journal, Subscribers: subs, Log: env.moduleLog("security")}
	var wg sync.WaitGroup
	wg.Go(func() { _ = bus.Run(ctx) })
	k, err := keeperClient(cfg)
	if err != nil {
		env.log.Error("хранитель: сертификат mTLS не загружен", "err", err)
	}
	if k != nil {
		heads := &securityapp.HeadsSender{Journal: c.journal, Keeper: k, Emit: emit, Log: env.moduleLog("security")}
		poll := &securityapp.IntegrityPoller{Journal: c.journal, Keeper: k, Emit: emit, Log: env.moduleLog("security")}
		wg.Go(func() {
			_ = c.leader(env, "security").Run(ctx, func(ctx context.Context, _ appjournal.Fence) error {
				var lw sync.WaitGroup
				lw.Go(func() { _ = heads.Run(ctx, cfg.Security.Interval) })
				lw.Go(func() { _ = poll.Run(ctx, cfg.Security.Interval) })
				lw.Wait()
				return nil
			})
		})
	} else {
		env.log.Warn("хранитель не настроен (security.keeper_url) — головы не передаются, индикатор целостности «неизвестно»")
	}
	wg.Wait()
	return nil
}
