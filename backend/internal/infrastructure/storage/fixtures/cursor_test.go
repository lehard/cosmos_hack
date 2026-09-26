package fixtures

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ant/internal/application/platform"
)

// testPool — своя БД агента (make dev-db: ANT_DB_*); нет — тест пропускается.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	host := os.Getenv("ANT_DB_HOST")
	if host == "" {
		t.Skip("нет ANT_DB_HOST (make dev-db)")
	}
	pw, _ := os.ReadFile(os.Getenv("ANT_DB_PASSWORD_FILE"))
	cfg, err := pgxpool.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Host, cfg.ConnConfig.Database, cfg.ConnConfig.User = host, os.Getenv("ANT_DB_NAME"), os.Getenv("ANT_DB_USER")
	cfg.ConnConfig.Port = 5432
	cfg.ConnConfig.Password = strings.TrimSpace(string(pw))
	cfg.ConnConfig.TLSConfig = nil
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestCursor — курсор общий для копий api: вторая копия видит шаг первой.
func TestCursor(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	a, b := New(pool), New(pool)
	if err := a.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if err := a.Move(ctx, platform.CursorState{}); err != nil {
		t.Fatal(err)
	}
	if st, err := b.Current(ctx); err != nil || st.Scenario != "" {
		t.Fatalf("после сброса: %+v %v", st, err)
	}
	at := time.Date(2026, 9, 23, 8, 6, 0, 0, time.UTC)
	want := platform.CursorState{Scenario: "flange-bad-day", RunID: "fx-1", Step: 8, Paused: true, Speed: 1000, ClockAt: at}
	if err := a.Move(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err := b.Current(ctx)
	if err != nil || got != want {
		t.Fatalf("вторая копия видит %+v, ожидалось %+v (%v)", got, want, err)
	}
	if err := a.Move(ctx, platform.CursorState{}); err != nil {
		t.Fatal(err)
	}
}
