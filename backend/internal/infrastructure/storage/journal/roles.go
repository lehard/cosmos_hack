package journal

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AppRole — роль приложения (AD-1): INSERT и SELECT в journal.
const AppRole = "ant_app"

// AfterConnectRole — настройка пула pgx: каждое соединение работает ролью
// role (SET ROLE). Так ant, входящий в БД общим пользователем демо-стенда,
// пишет в журнал правами ant_app: UPDATE, DELETE и TRUNCATE журнала ему
// недоступны (AD-2). В промышленной эксплуатации ant входит отдельным
// пользователем — членом ant_app (см. отчёт эпика 04).
func AfterConnectRole(role string) func(context.Context, *pgx.Conn) error {
	return func(ctx context.Context, c *pgx.Conn) error {
		_, err := c.Exec(ctx, "SET ROLE "+pgx.Identifier{role}.Sanitize())
		return err
	}
}

// NewAppPool — пул pgx, каждое соединение которого работает ролью ant_app
// (AfterConnectRole). Так собирают JournalStore, Leases и Listener роли ant.
func NewAppPool(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
	cfg = cfg.Copy()
	prev := cfg.AfterConnect
	setRole := AfterConnectRole(AppRole)
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		if prev != nil {
			if err := prev(ctx, c); err != nil {
				return err
			}
		}
		return setRole(ctx, c)
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}
