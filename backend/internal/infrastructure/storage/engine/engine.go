package engine

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
)

// Migrations — миграции схемы engine для `ant migrate` (эпик 04 подключает
// их к goose вместе с миграциями остальных модулей).
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Channel — канал NOTIFY «есть новое»: полезная нагрузка — только seq (AD-6).
const Channel = "ant_changes"

// Execer — то, что нужно применяющему эффекты внутри транзакции Append
// (pgx.Tx, pgx.Conn, pgxpool.Pool).
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// ApplyEffect применяет эффект движка внутри транзакции Append (AD-44,
// AD-45). handled=false — эффект не движка (его применяет другой модуль).
// Адаптер журнала (эпик 04) вызывает его для каждого rq.Effects в той же
// транзакции, что записи и курсор.
func ApplyEffect(ctx context.Context, tx Execer, e appjournal.Effect) (handled bool, err error) {
	switch x := e.(type) {
	case engineapp.ProjectionPut:
		_, err = tx.Exec(ctx, `INSERT INTO engine.projections (name, key, item_id, value) VALUES ($1, $2, $3, $4)
			ON CONFLICT (name, key) DO UPDATE SET item_id = EXCLUDED.item_id, value = EXCLUDED.value`,
			x.Name, x.Key, x.ItemID, []byte(x.Value))
	case engineapp.ProjectionReset:
		if x.ItemID == "" {
			_, err = tx.Exec(ctx, `DELETE FROM engine.projections WHERE name = $1`, x.Name)
		} else {
			_, err = tx.Exec(ctx, `DELETE FROM engine.projections WHERE name = $1 AND item_id = $2`, x.Name, x.ItemID)
		}
	case engineapp.ContributionsReplace:
		if _, err = tx.Exec(ctx, `DELETE FROM engine.contributions WHERE item_id = $1`, x.ItemID); err != nil {
			break
		}
		for _, r := range x.Rows {
			sources := r.Sources
			if sources == nil {
				sources = []string{}
			}
			if _, err = tx.Exec(ctx, `INSERT INTO engine.contributions (item_id, metric, slice, value, scale, sources)
				VALUES ($1, $2, $3, $4, $5, $6)`, x.ItemID, r.Metric, r.Slice, r.Value, r.Scale, sources); err != nil {
				break
			}
		}
	case engineapp.Notify:
		var maxSeq int64
		for _, c := range x.Changes {
			var received *time.Time
			if !c.ReceivedAt.IsZero() {
				t := c.ReceivedAt
				received = &t
			}
			if _, err = tx.Exec(ctx, `INSERT INTO engine.changes (seq, entity, id, run_id, received_at) VALUES ($1, $2, $3, $4, $5)`,
				c.Seq, string(c.Entity), c.ID, c.RunID, received); err != nil {
				break
			}
			maxSeq = max(maxSeq, c.Seq)
		}
		if err == nil && len(x.Changes) > 0 {
			// NOTIFY транзакционен: сигнал уйдёт только после фиксации.
			_, err = tx.Exec(ctx, `SELECT pg_notify($1, $2)`, Channel, strconv.FormatInt(maxSeq, 10))
		}
	default:
		return false, nil
	}
	if err != nil {
		return true, fmt.Errorf("эффект %s: %w", e.EffectKind(), err)
	}
	return true, nil
}

// Store — адаптер ProjectionStore и ChangeLog над Postgres.
type Store struct {
	Pool *pgxpool.Pool

	mu     sync.Mutex
	listen *pgxpool.Conn
}

var (
	_ engineapp.ProjectionStore = (*Store)(nil)
	_ engineapp.ChangeLog       = (*Store)(nil)
)

// Get — значение проекции.
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

// After — изменения после seq в порядке seq.
func (s *Store) After(ctx context.Context, afterSeq int64, limit int) ([]engineapp.Change, error) {
	rows, err := s.Pool.Query(ctx, `SELECT seq, entity, id, run_id, received_at FROM engine.changes
		WHERE seq > $1 ORDER BY seq, n LIMIT $2`, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []engineapp.Change
	for rows.Next() {
		var (
			c        engineapp.Change
			entity   string
			received *time.Time
		)
		if err := rows.Scan(&c.Seq, &entity, &c.ID, &c.RunID, &received); err != nil {
			return nil, err
		}
		c.Entity = platform.EntityKind(entity)
		if received != nil {
			c.ReceivedAt = *received
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Wait блокирует до NOTIFY ant_changes (LISTEN на выделенном соединении) или
// отмены ctx. После разрыва соединения возвращается сразу: вызывающий
// дочитает по seq (копия догоняет после переподключения, AD-6).
func (s *Store) Wait(ctx context.Context) error {
	s.mu.Lock()
	conn := s.listen
	if conn == nil {
		c, err := s.Pool.Acquire(ctx)
		if err != nil {
			s.mu.Unlock()
			return err
		}
		if _, err := c.Exec(ctx, "LISTEN "+Channel); err != nil {
			c.Release()
			s.mu.Unlock()
			return err
		}
		s.listen, conn = c, c
	}
	s.mu.Unlock()
	_, err := conn.Conn().WaitForNotification(ctx)
	if err != nil && ctx.Err() == nil {
		s.mu.Lock()
		if s.listen == conn {
			s.listen = nil
			_ = conn.Hijack().Close(context.WithoutCancel(ctx))
		}
		s.mu.Unlock()
		return nil
	}
	return err
}

// Close освобождает соединение LISTEN.
func (s *Store) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listen != nil {
		s.listen.Release()
		s.listen = nil
	}
}
