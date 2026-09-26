package engine

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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

// Migrations — миграции схемы engine для `ant migrate` (goose, модуль
// engine в cmd/ant migrationSets; применяет роль ant_owner).
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Channel — канал NOTIFY «есть новое»: полезная нагрузка — только seq (AD-6).
const Channel = "ant_changes"

// MigrationsDir — каталог миграций внутри Migrations (ant migrate, модуль engine).
const MigrationsDir = "migrations"

// lockChanges — ключ pg_advisory_xact_lock журнала изменений («antchg» в
// ASCII): вставка изменений сериализуется до фиксации, поэтому позиция n
// видима строго в порядке фиксации и публикатор, дочитывающий по n, ничего
// не пропускает. Порядок блокировок — после голов цепочек журнала (Append
// применяет эффекты последними), взаимной блокировки нет.
const lockChanges int64 = 0x616e74636867

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
		if len(x.Changes) == 0 {
			break
		}
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, lockChanges); err != nil {
			break
		}
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

	mu      sync.Mutex
	listen  *pgxpool.Conn
	waiting bool
	closed  bool
}

// ErrClosed — Wait после Close.
var ErrClosed = errors.New("engine: журнал изменений закрыт")

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

// After — изменения по запросу в порядке позиции n (порядок фиксации).
func (s *Store) After(ctx context.Context, q engineapp.ChangeQuery) ([]engineapp.Change, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 500
	}
	upTo := q.UpToPos
	if upTo <= 0 {
		upTo = math.MaxInt64
	}
	rows, err := s.Pool.Query(ctx, `SELECT n, seq, entity, id, run_id, received_at FROM engine.changes
		WHERE n > $1 AND n <= $2 AND seq > $3 ORDER BY n LIMIT $4`, q.AfterPos, upTo, q.AfterSeq, limit)
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
		if err := rows.Scan(&c.Pos, &c.Seq, &entity, &c.ID, &c.RunID, &received); err != nil {
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

// Tail — позиция последнего зафиксированного изменения.
func (s *Store) Tail(ctx context.Context) (int64, error) {
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT COALESCE(MAX(n), 0) FROM engine.changes`).Scan(&n)
	return n, err
}

// Wait блокирует до NOTIFY ant_changes (LISTEN на выделенном соединении) или
// отмены ctx. После разрыва соединения возвращается сразу: вызывающий
// дочитает по seq (копия догоняет после переподключения, AD-6). Ждущий —
// один (публикатор копии api).
func (s *Store) Wait(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrClosed
	}
	if s.waiting {
		s.mu.Unlock()
		return errors.New("engine: Wait уже ждёт — ждущий один")
	}
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
	s.waiting = true
	s.mu.Unlock()

	_, err := conn.Conn().WaitForNotification(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.waiting = false
	switch {
	case s.closed:
		s.dropListen()
		return ErrClosed
	case err != nil && ctx.Err() == nil:
		// Обрыв: следующее Wait откроет LISTEN заново.
		s.dropListen()
		return nil
	}
	return err
}

// dropListen закрывает соединение LISTEN (в пул оно не возвращается:
// подписка на канал осталась бы на чужом соединении). Под s.mu.
func (s *Store) dropListen() {
	if s.listen != nil {
		_ = s.listen.Hijack().Close(context.Background())
		s.listen = nil
	}
}

// Close закрывает соединение LISTEN; если Wait ещё ждёт — его закроет Wait.
func (s *Store) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if !s.waiting {
		s.dropListen()
	}
}
