package main

import (
	"context"
	"time"

	"ant/cmd/internal/db"
)

// runMigrate — разовая роль migrate (AD-1, AD-25): ждёт БД и применяет
// миграции модулей. Схемы модулей, роли БД ant_owner / ant_app / ant_verifier
// и миграции goose по схемам добавляет эпик 04 «Журнал и хранение»; пока роль
// только проверяет, что БД доступна, чтобы compose уже сейчас строил правильный
// порядок запуска (migrate → ant).
func runMigrate(ctx context.Context, env *environment) error {
	pool, err := db.Open(ctx, env.cfg.DB, "ant-migrate")
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.WaitReady(ctx, pool, time.Minute); err != nil {
		return err
	}
	var serverVersion string
	if err := pool.QueryRow(ctx, "SHOW server_version").Scan(&serverVersion); err != nil {
		return err
	}
	env.log.Info("миграции: БД доступна, миграций модулей пока нет", "postgres", serverVersion)
	return nil
}
