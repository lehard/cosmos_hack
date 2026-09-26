package ops

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dom "ant/internal/domain/ops"
)

// Роли БД (AD-1).
const (
	RoleOwner    = "ant_owner"
	RoleApp      = "ant_app"
	RoleVerifier = "ant_verifier"
)

// journalTable — таблица журнала, права на которую проверяются.
const journalTable = "journal.entries"

// Probe — проверки базы по пулу подключения процесса (в демо — владелец
// кластера; роль приложения видит pg_roles так же).
type Probe struct {
	Pool *pgxpool.Pool
}

// Ping — база отвечает.
func (p Probe) Ping(ctx context.Context) error { return p.Pool.Ping(ctx) }

// want — какие права на журнал должны быть у ролей (AD-1): приложение —
// INSERT и SELECT, без UPDATE, DELETE, TRUNCATE; верификатор — только SELECT.
var want = []dom.Privilege{
	{Role: RoleApp, Privilege: "SELECT", Want: true},
	{Role: RoleApp, Privilege: "INSERT", Want: true},
	{Role: RoleApp, Privilege: "UPDATE", Want: false},
	{Role: RoleApp, Privilege: "DELETE", Want: false},
	{Role: RoleApp, Privilege: "TRUNCATE", Want: false},
	{Role: RoleVerifier, Privilege: "SELECT", Want: true},
	{Role: RoleVerifier, Privilege: "INSERT", Want: false},
	{Role: RoleVerifier, Privilege: "UPDATE", Want: false},
}

// DBRoles — отсутствующие роли и права существующих ролей на журнал.
// Нет таблицы журнала — права не проверяются (это находка миграций).
func (p Probe) DBRoles(ctx context.Context) ([]string, []dom.Privilege, error) {
	rows, err := p.Pool.Query(ctx, `SELECT r FROM unnest($1::text[]) AS r
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = r) ORDER BY r`, []string{RoleOwner, RoleApp, RoleVerifier})
	if err != nil {
		return nil, nil, err
	}
	missing, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, nil, err
	}
	var table bool
	if err := p.Pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", journalTable).Scan(&table); err != nil {
		return nil, nil, err
	}
	if !table {
		return missing, nil, nil
	}
	var privs []dom.Privilege
	for _, w := range want {
		skip := false
		for _, m := range missing {
			skip = skip || m == w.Role
		}
		if skip {
			continue
		}
		x := w
		x.Object = journalTable
		if err := p.Pool.QueryRow(ctx, "SELECT has_table_privilege($1, $2, $3)", x.Role, journalTable, x.Privilege).Scan(&x.Granted); err != nil {
			return nil, nil, err
		}
		privs = append(privs, x)
	}
	return missing, privs, nil
}
