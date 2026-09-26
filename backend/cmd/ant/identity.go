package main

import (
	"crypto/sha256"
	"os"

	"ant/cmd/internal/config"
	accessapp "ant/internal/application/access"
	"ant/internal/infrastructure/fixtures/world"
	"ant/internal/infrastructure/security/identity"
)

// demoIdentity — вход демо-трека эпика 08 (ключ identity_provider: demo):
// каталог политики и столы ролей — встроенная копия normative/policy и
// normative/desks (world.Inputs, сверяется с репозиторием тестом генератора
// заготовок), пока стартовая политика не приходит из журнала (эпики 05, 26).
// Демо-персоны без пароля — только в профилях fixtures и demo.
//
// Ключ подписи токена сеанса выводится из секрета БД (файл в томе секретов,
// AD-25): одинаков у копий api и переживает перезапуск; файла нет — случайный
// ключ процесса. Сеансы scs в Postgres и вход по паролю — эпик 08 во втором слое.
func demoIdentity(cfg *config.Config) (accessapp.IdentityProvider, *accessapp.Directory, error) {
	dir, err := identity.LoadDirectory(world.Inputs())
	if err != nil {
		return nil, nil, err
	}
	opts := identity.Options{Personas: cfg.Profile == config.ProfileFixtures || cfg.Profile == config.ProfileDemo}
	if cfg.DB.PasswordFile != "" {
		if secret, err := os.ReadFile(cfg.DB.PasswordFile); err == nil && len(secret) > 0 {
			k := sha256.Sum256(append([]byte("ant-session-key-v1\x00"), secret...))
			opts.Key = k[:]
		}
	}
	idp, err := identity.NewDemo(dir, opts)
	if err != nil {
		return nil, nil, err
	}
	return idp, dir, nil
}
