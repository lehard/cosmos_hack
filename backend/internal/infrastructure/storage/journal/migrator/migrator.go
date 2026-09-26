// Пакет migrator — роли БД и миграции goose модулей (AD-1, AD-25): тело
// разовой роли ant migrate.
//
// Слой: infrastructure/storage (модуль journal — владелец ролей БД).
// Связи: вызывается из cmd/ant (роль migrate) и из тестов хранилищ; миграции
// модулей — embed.FS их пакетов storage (например, storage/journal.Migrations).
//
// Порядок: EnsureRoles (суперпользователь или владелец БД) → Up: миграции
// каждого модуля ролью ant_owner (SET ROLE), своя таблица версий goose на
// модуль в схеме ant_migrations; одновременный запуск двух migrate
// сериализуется блокировкой сеанса goose.
//
// Роли (AD-1): ant_owner — DDL, без LOGIN, владелец схем; ant_app — INSERT и
// SELECT в journal, полный доступ к схемам модулей; ant_verifier — только SELECT.
// Роли кластерные: создаются, если их нет; права на базу выдаются каждой базе.
package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Имена ролей БД (AD-1).
const (
	RoleOwner    = "ant_owner"
	RoleApp      = "ant_app"
	RoleVerifier = "ant_verifier"
)

// VersionSchema — схема таблиц версий goose (по таблице на модуль).
const VersionSchema = "ant_migrations"

// Set — миграции одного модуля: схема Postgres модуля, свои версии.
type Set struct {
	// Module — имя модуля (AD-1); таблица версий — ant_migrations.goose_‹module›.
	Module string
	// FS, Dir — каталог *.sql внутри embed.FS пакета storage модуля.
	FS  fs.FS
	Dir string
}

var reModule = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// EnsureRoles создаёт роли ant_owner, ant_app, ant_verifier (NOLOGIN), даёт
// ant_owner право создавать схемы в текущей базе, всем трём — CONNECT, и
// создаёт схему таблиц версий goose. Выполняется подключением
// суперпользователя (или владельца базы с CREATEROLE). Повтор безопасен.
func EnsureRoles(ctx context.Context, conn *pgx.Conn) error {
	var db string
	if err := conn.QueryRow(ctx, "SELECT current_database()").Scan(&db); err != nil {
		return err
	}
	stmts := []string{
		// Роли кластерные: две базы на одном сервере (агенты, прогоны тестов)
		// создают их одновременно — сериализуем блокировкой транзакции.
		"SELECT pg_advisory_xact_lock(7307618427063956596)",
	}
	for _, r := range []string{RoleOwner, RoleApp, RoleVerifier} {
		stmts = append(stmts, fmt.Sprintf(`DO $$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '%[1]s') THEN
    CREATE ROLE %[1]s NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
END $$`, r))
	}
	q := pgx.Identifier{db}.Sanitize()
	stmts = append(stmts,
		fmt.Sprintf("REVOKE CREATE ON DATABASE %s FROM PUBLIC", q),
		fmt.Sprintf("GRANT CONNECT, CREATE ON DATABASE %s TO %s", q, RoleOwner),
		fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s, %s", q, RoleApp, RoleVerifier),
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s AUTHORIZATION %s", VersionSchema, RoleOwner),
		fmt.Sprintf("REVOKE ALL ON SCHEMA %s FROM PUBLIC", VersionSchema),
	)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, s := range stmts {
		if _, err := tx.Exec(ctx, s); err != nil {
			return fmt.Errorf("роли БД: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// Applied — применённая миграция.
type Applied struct {
	Module  string
	Version int64
	Source  string
}

// Up применяет миграции наборов по порядку ролью ant_owner (объекты
// принадлежат ей, AD-1). cfg — подключение суперпользователя или члена ant_owner.
func Up(ctx context.Context, cfg *pgx.ConnConfig, log *slog.Logger, sets ...Set) ([]Applied, error) {
	db := stdlib.OpenDB(*cfg, stdlib.OptionAfterConnect(func(ctx context.Context, c *pgx.Conn) error {
		_, err := c.Exec(ctx, "SET ROLE "+RoleOwner)
		return err
	}))
	defer db.Close()
	var out []Applied
	for _, s := range sets {
		res, err := up(ctx, db, log, s)
		out = append(out, res...)
		if err != nil {
			return out, fmt.Errorf("миграции %s: %w", s.Module, err)
		}
	}
	return out, nil
}

func up(ctx context.Context, db *sql.DB, log *slog.Logger, s Set) ([]Applied, error) {
	if !reModule.MatchString(s.Module) {
		return nil, fmt.Errorf("имя модуля %q", s.Module)
	}
	fsys, err := fs.Sub(s.FS, s.Dir)
	if err != nil {
		return nil, err
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, err
	}
	opts := []goose.ProviderOption{
		goose.WithTableName(VersionSchema + ".goose_" + s.Module),
		goose.WithSessionLocker(locker),
		goose.WithDisableGlobalRegistry(true),
	}
	if log != nil {
		opts = append(opts, goose.WithSlog(log))
	}
	p, err := goose.NewProvider(goose.DialectPostgres, db, fsys, opts...)
	if err != nil {
		return nil, err
	}
	results, err := p.Up(ctx)
	var out []Applied
	for _, r := range results {
		if r != nil && r.Source != nil {
			out = append(out, Applied{Module: s.Module, Version: r.Source.Version, Source: r.Source.Path})
		}
	}
	return out, err
}
