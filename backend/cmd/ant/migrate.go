package main

import (
	"context"
	"fmt"
	"time"

	"ant/cmd/internal/db"
	enginestore "ant/internal/infrastructure/storage/engine"
	storagefx "ant/internal/infrastructure/storage/fixtures"
	ingeststore "ant/internal/infrastructure/storage/ingest"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/migrator"
)

// migrationSets — миграции модулей в порядке применения (AD-1: у каждого
// модуля своя схема и свои миграции goose в infrastructure/storage/‹модуль›/
// migrations). Модуль со своей схемой добавляет сюда строку.
var migrationSets = []migrator.Set{
	{Module: "journal", FS: journalstore.Migrations, Dir: journalstore.MigrationsDir},
	{Module: "engine", FS: enginestore.Migrations, Dir: enginestore.MigrationsDir},
	{Module: "ingest", FS: ingeststore.Migrations, Dir: ingeststore.MigrationsDir},
	{Module: "fixtures", FS: storagefx.Migrations, Dir: "migrations"},
}

// runMigrate — разовая роль migrate (AD-1, AD-25): ждёт БД, создаёт роли БД
// ant_owner / ant_app / ant_verifier и применяет миграции goose модулей ролью
// ant_owner. Подключение — пользователем с правом создавать роли (в демо —
// ant_admin, владелец кластера compose). Повторный запуск ничего не меняет.
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
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	err = migrator.EnsureRoles(ctx, conn.Conn())
	conn.Release()
	if err != nil {
		return err
	}
	applied, err := migrator.Up(ctx, pool.Config().ConnConfig, env.log, migrationSets...)
	for _, a := range applied {
		env.log.Info("миграции: применена", "module", a.Module, "version", a.Version, "source", a.Source)
	}
	if err != nil {
		return fmt.Errorf("миграции: %w", err)
	}
	env.log.Info("миграции: схема актуальна", "postgres", serverVersion, "applied", len(applied), "modules", len(migrationSets))
	// Эпик 33: стартовые паспорта допуска анализаторов демо (профили demo, fixtures).
	return seedVisionPassports(ctx, env)
}
