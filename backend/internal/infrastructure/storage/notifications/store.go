package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	app "ant/internal/application/notifications"
	dom "ant/internal/domain/notifications"
)

// Store — чтение проекций notifications на Postgres (ведомый порт
// application/notifications.Projections).
//
// Проекции сроков, задач и уведомлений движок хранит в таблице каркаса
// проекций engine.projections с именами notifications.*: их пишет только
// роль projector эффектами в транзакции journal.Append (AD-45), писатель по
// смыслу — notifications. Своей схемы у модуля нет (как у analytics, Д-44):
// чтение своих имён из таблицы каркаса.
type Store struct {
	Pool *pgxpool.Pool
}

// New создаёт хранилище над пулом роли приложения.
func New(pool *pgxpool.Pool) *Store { return &Store{Pool: pool} }

var _ app.Projections = (*Store)(nil)

// Get — значение проекции name по ключу.
func (s *Store) Get(ctx context.Context, name, key string) (json.RawMessage, bool, error) {
	var v []byte
	err := s.Pool.QueryRow(ctx, `SELECT value FROM engine.projections WHERE name = $1 AND key = $2`, name, key).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return v, true, nil
}

// All — все строки проекции name.
func (s *Store) All(ctx context.Context, name string) (map[string]json.RawMessage, error) {
	rows, err := s.Pool.Query(ctx, `SELECT key, value FROM engine.projections WHERE name = $1`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var (
			k string
			v []byte
		)
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// OpenObligations — действующие сроки (планировщик читает только проекцию
// сроков, AD-4): отбор по состоянию — в запросе.
func (s *Store) OpenObligations(ctx context.Context) ([]dom.ObligationRecord, error) {
	rows, err := s.Pool.Query(ctx, `SELECT value FROM engine.projections WHERE name = $1 AND value->>'state' = $2 ORDER BY key`,
		app.ProjectionObligation, dom.ObligationOpen)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dom.ObligationRecord
	for rows.Next() {
		var v []byte
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		var o dom.ObligationRecord
		if err := json.Unmarshal(v, &o); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
