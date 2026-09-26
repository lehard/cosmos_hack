package main

import (
	"context"
	"io"

	accessapp "ant/internal/application/access"
	signingapp "ant/internal/application/signing"
	"ant/internal/infrastructure/integration/signing/paperscan"
	"ant/internal/infrastructure/security/profiles"
	materialsstore "ant/internal/infrastructure/storage/materials"
)

// Модуль signing в роли api (пачка стыков Д-59): реестр ключей и профилей —
// свёртка журнала ядра (генезис — первые записи), проверка подписей
// профилями gost/pq/hybrid, полномочия второй подписи и заверения бумаги —
// по проекции политики (эпик 26, accessBundle.authorities), скан бумаги —
// из хранилища материалов, QR — paperscan. Тот же сервис — порт проверки
// подписи команд уровня ≥ 1 общего декоратора (platform.SignatureChecker):
// модули решений получают принятую подпись в контексте команды.

// signingLive — живой модуль signing на ядре процесса.
func signingLive(ctx context.Context, env *environment, access *accessBundle) (*signingapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	cfg := env.cfg
	deps := signingapp.Deps{Journal: c.journal, QR: paperscan.Reader{}, QRWriter: paperscan.Writer{}}
	// Полномочия — по проекции политики (эпик 26); без неё — стартовая таблица эпика 27.
	deps.Authorities = signingapp.DefaultAuthorities()
	if access != nil {
		if a, ok := access.authorities(); ok {
			deps.Authorities = a
		}
	}
	if mat, err := materialsstore.NewVolume(cfg.Materials.Dir, materialsOptions(cfg)...); err == nil {
		deps.Scans = scans{mat}
	} else {
		env.log.Warn("signing: хранилище материалов недоступно — заверение бумаги без скана", "err", err)
	}
	svc := signingapp.NewService(
		signingapp.WithConfig(signingapp.Config{Profile: string(cfg.Profile), AllowUnsigned: signingapp.SignatureModeFor(string(cfg.Profile)),
			Partitions: cfg.Engine.Partitions, DomainBuild: domainBuild()}),
		signingapp.WithDeps(deps))
	svc.UseCrypto(profiles.Verifier{Keys: svc.Registry()})
	svc.UseAlerts(signingapp.JournalKeyAlerts{Service: svc})
	return svc, nil
}

// scans — порт Scans над хранилищем материалов (AD-23): байты скана по адресу.
type scans struct{ v *materialsstore.Volume }

func (s scans) Scan(ctx context.Context, address string) ([]byte, error) {
	r, _, err := s.v.Get(ctx, address)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

var _ signingapp.Authorities = accessapp.PolicyAuthorities{}
