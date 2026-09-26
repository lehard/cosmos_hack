package analytics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	app "ant/internal/application/analytics"
	domain "ant/internal/domain/analytics"
)

// Store — чтение показателей на Postgres (ведомый порт application/analytics.Store).
//
// Строки вклада изделий и глобальные проекции analytics движок хранит в
// таблицах каркаса проекций схемы engine (engine.contributions,
// engine.projections с именами analytics.*): их пишет только движок
// эффектами в транзакции journal.Append (AD-45), писатель по смыслу —
// analytics. Своей схемы у analytics нет: агрегаты — запросы над строками
// вклада, материализаций нет (см. отчёт эпика 25 о чтении таблиц движка).
type Store struct {
	Pool *pgxpool.Pool
}

// New создаёт хранилище над пулом роли приложения.
func New(pool *pgxpool.Pool) *Store { return &Store{Pool: pool} }

var _ app.Store = (*Store)(nil)

// Rows — строки вклада показателей metrics (пусто — все).
func (s *Store) Rows(ctx context.Context, metrics ...string) ([]domain.Row, error) {
	q := `SELECT item_id, metric, slice, value, sources FROM engine.contributions`
	args := []any{}
	if len(metrics) > 0 {
		q += ` WHERE metric = ANY($1)`
		args = append(args, metrics)
	}
	rows, err := s.Pool.Query(ctx, q+` ORDER BY item_id, metric, slice`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Row
	for rows.Next() {
		var (
			item, metric, slice string
			value               int64
			sources             []string
		)
		if err := rows.Scan(&item, &metric, &slice, &value, &sources); err != nil {
			return nil, err
		}
		r, err := app.DecodeRow(item, metric, slice, value, sources)
		if err != nil {
			// Строка чужого формата (не analytics) — пропускается, а не роняет показатели.
			continue
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Equipment — проекции оборудования и остановок точек процесса.
func (s *Store) Equipment(ctx context.Context) ([]domain.Equipment, error) {
	return projections[domain.Equipment](ctx, s.Pool, app.ProjectionEquipment)
}

// Incidents — проекции инцидентов.
func (s *Store) Incidents(ctx context.Context) ([]domain.Incident, error) {
	return projections[domain.Incident](ctx, s.Pool, app.ProjectionIncident)
}

func projections[T any](ctx context.Context, pool *pgxpool.Pool, name string) ([]T, error) {
	rows, err := pool.Query(ctx, `SELECT key, value FROM engine.projections WHERE name = $1 ORDER BY key`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []T
	for rows.Next() {
		var (
			key string
			raw []byte
		)
		if err := rows.Scan(&key, &raw); err != nil {
			return nil, err
		}
		var v T
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("проекция %s/%s: %w", name, key, err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
