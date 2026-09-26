package fixtures

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ant/internal/application/platform"
)

// Migrations — миграции схемы fixtures (goose).
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Cursor — курсор заготовок в Postgres (platform.FixtureCursor).
type Cursor struct {
	pool *pgxpool.Pool
	name string
}

// New — курсор с именем "default" над пулом.
func New(pool *pgxpool.Pool) *Cursor { return &Cursor{pool: pool, name: "default"} }

// EnsureSchema создаёт схему и таблицу курсора, если их нет (DDL миграции Up).
func (c *Cursor) EnsureSchema(ctx context.Context) error {
	b, err := Migrations.ReadFile("migrations/20260926120000_fixtures_cursor.sql")
	if err != nil {
		return err
	}
	up := string(b)
	up = up[strings.Index(up, "-- +goose Up")+len("-- +goose Up") : strings.Index(up, "-- +goose Down")]
	_, err = c.pool.Exec(ctx, up)
	return err
}

// Current — положение курсора; строки нет — нулевое (мир встаёт на сценарий по умолчанию).
func (c *Cursor) Current(ctx context.Context) (platform.CursorState, error) {
	var st platform.CursorState
	err := c.pool.QueryRow(ctx, `SELECT scenario, run_id, step, paused, speed, clock_at FROM fixtures.cursor WHERE name = $1`, c.name).
		Scan(&st.Scenario, &st.RunID, &st.Step, &st.Paused, &st.Speed, &st.ClockAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.CursorState{}, nil
	}
	if err != nil {
		return st, fmt.Errorf("курсор заготовок: %w", err)
	}
	st.ClockAt = st.ClockAt.UTC()
	return st, nil
}

// Move ставит курсор; нулевое положение (нет сценария) — сброс к сценарию по умолчанию.
func (c *Cursor) Move(ctx context.Context, to platform.CursorState) error {
	if to.Scenario == "" {
		_, err := c.pool.Exec(ctx, `DELETE FROM fixtures.cursor WHERE name = $1`, c.name)
		return err
	}
	speed := to.Speed
	if speed < 1 {
		speed = 1
	}
	_, err := c.pool.Exec(ctx, `
		INSERT INTO fixtures.cursor (name, scenario, run_id, step, paused, speed, clock_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		ON CONFLICT (name) DO UPDATE SET scenario = EXCLUDED.scenario, run_id = EXCLUDED.run_id, step = EXCLUDED.step,
			paused = EXCLUDED.paused, speed = EXCLUDED.speed, clock_at = EXCLUDED.clock_at, updated_at = now()`,
		c.name, to.Scenario, to.RunID, to.Step, to.Paused, speed, to.ClockAt.UTC())
	if err != nil {
		return fmt.Errorf("курсор заготовок: %w", err)
	}
	return nil
}

var _ platform.FixtureCursor = (*Cursor)(nil)
