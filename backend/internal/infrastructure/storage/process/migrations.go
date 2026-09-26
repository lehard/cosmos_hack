package process

import "embed"

// Migrations — миграции goose схемы process (AD-1: свои миграции модуля,
// версии — метки времени). Применяет роль migrate: в cmd/ant/migrate.go —
// строка {Module: "process", FS: Migrations, Dir: MigrationsDir}.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsDir — каталог миграций внутри Migrations.
const MigrationsDir = "migrations"
