package journal

import "embed"

// Migrations — миграции goose схем journal и journal_state (AD-1: свои
// миграции модуля, версии — метки времени). Применяет роль migrate ролью
// ant_owner (infrastructure/storage/journal/migrator).
//
//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsDir — каталог миграций внутри Migrations.
const MigrationsDir = "migrations"
