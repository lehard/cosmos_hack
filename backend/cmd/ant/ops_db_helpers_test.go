package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/clock"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

// adminPool — пул пользователя конфигурации (как у роли api): читает pg_roles
// и таблицы версий миграций.
func adminPool(t *testing.T, db *journaltest.DB) *pgxpool.Pool {
	t.Helper()
	p, err := pgxpool.NewWithConfig(context.Background(), db.Admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func journaltestStore(pool *pgxpool.Pool) *journalstore.Store {
	return journalstore.NewStore(pool, clock.System{})
}

func journaltestLeases(pool *pgxpool.Pool) *journalstore.Leases {
	return journalstore.NewLeases(pool, clock.System{})
}
