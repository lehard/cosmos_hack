package erp

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	app "ant/internal/application/erp"
	appjournal "ant/internal/application/journal"
	dom "ant/internal/domain/erp"
	journalstore "ant/internal/infrastructure/storage/journal"
)

// Migrations — миграции goose схемы erp: в cmd/ant/migrate.go — строка
// {Module: "erp", FS: Migrations, Dir: MigrationsDir}.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsDir — каталог миграций внутри Migrations.
const MigrationsDir = "migrations"

// Store — хранилище модуля erp в Postgres (схема erp).
type Store struct {
	pool *pgxpool.Pool
}

// NewStore — хранилище на пуле роли приложения (ant_app).
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

var (
	_ app.OutboxStore = (*Store)(nil)
	_ app.GatewaySeen = (*Store)(nil)
)

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *Store) q(ctx context.Context) querier {
	if tx, ok := journalstore.Tx(ctx); ok {
		return tx
	}
	return s.pool
}

// ApplyEffect — применяющий эффекты модуля erp в транзакции Append
// (journalstore.WithEffects): строки очереди исходящих.
func ApplyEffect(ctx context.Context, tx pgx.Tx, e appjournal.Effect) (bool, error) {
	switch x := e.(type) {
	case app.OutboxView:
		view, err := json.Marshal(x.View)
		if err != nil {
			return true, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO erp.outbox (business_key, system, status, token, item_id, first_seq, view, updated)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (business_key) DO UPDATE SET system = EXCLUDED.system, status = EXCLUDED.status, token = EXCLUDED.token,
  item_id = EXCLUDED.item_id, first_seq = EXCLUDED.first_seq, view = EXCLUDED.view, updated = now()`,
			x.View.Key, x.View.System, string(x.View.Status), x.Token, x.View.ItemID, x.View.FirstSeq, view)
		return true, err
	case app.OutboxTransport:
		next := x.NextAt
		if next.IsZero() {
			next = time.Unix(0, 0)
		}
		_, err := tx.Exec(ctx, `UPDATE erp.outbox SET handled = $2, retry_token = $3, tries = $4, next_at = $5, last_error = $6, updated = now()
WHERE business_key = $1`, x.Key, x.Handled, x.RetryToken, x.Tries, next.UTC(), x.LastError)
		return true, err
	}
	return false, nil
}

const rowCols = `view, token, handled, retry_token, tries, next_at, last_error`

func scanRow(r pgx.Row) (app.OutboxRow, error) {
	var (
		row  app.OutboxRow
		view []byte
	)
	if err := r.Scan(&view, &row.Token, &row.Handled, &row.RetryToken, &row.Tries, &row.NextAt, &row.LastError); err != nil {
		return row, err
	}
	if err := json.Unmarshal(view, &row.View); err != nil {
		return row, fmt.Errorf("erp.outbox: %w", err)
	}
	return row, nil
}

// Row — строка очереди по бизнес-ключу.
func (s *Store) Row(ctx context.Context, key string) (app.OutboxRow, bool, error) {
	row, err := scanRow(s.q(ctx).QueryRow(ctx, `SELECT `+rowCols+` FROM erp.outbox WHERE business_key = $1`, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return app.OutboxRow{}, false, nil
	}
	return row, err == nil, err
}

// Due — к отправке: в очереди, поколение не обработано, срок попытки наступил.
func (s *Store) Due(ctx context.Context, system string, now time.Time, limit int) ([]app.OutboxRow, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT `+rowCols+` FROM erp.outbox
WHERE system = $1 AND status = $2 AND handled <> token AND (retry_token <> token OR next_at <= $3)
ORDER BY first_seq LIMIT $4`, system, string(dom.StatusQueued), now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []app.OutboxRow
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Counts — в очереди и в карантине.
func (s *Store) Counts(ctx context.Context, system string) (int, int, error) {
	var q, z int
	err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FILTER (WHERE status = 'queued'), count(*) FILTER (WHERE status IN ('quarantined', 'rejected'))
FROM erp.outbox WHERE system = $1`, system).Scan(&q, &z)
	return q, z, err
}

func scanChannel(r pgx.Row) (app.Channel, error) {
	var c app.Channel
	err := r.Scan(&c.System, &c.State, &c.Endpoint, &c.Stand, &c.ContractVersion, &c.Detail, &c.CheckedAt, &c.LastExchangeAt)
	return c, err
}

const chanCols = `system, state, endpoint, stand, contract_version, detail, checked_at, last_exchange_at`

// Channel — состояние канала.
func (s *Store) Channel(ctx context.Context, system string) (app.Channel, bool, error) {
	c, err := scanChannel(s.q(ctx).QueryRow(ctx, `SELECT `+chanCols+` FROM erp.channels WHERE system = $1`, system))
	if errors.Is(err, pgx.ErrNoRows) {
		return app.Channel{}, false, nil
	}
	return c, err == nil, err
}

// SetChannel — записать состояние канала.
func (s *Store) SetChannel(ctx context.Context, c app.Channel) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO erp.channels (`+chanCols+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (system) DO UPDATE SET state = EXCLUDED.state, endpoint = EXCLUDED.endpoint, stand = EXCLUDED.stand,
  contract_version = EXCLUDED.contract_version, detail = EXCLUDED.detail, checked_at = EXCLUDED.checked_at,
  last_exchange_at = COALESCE(EXCLUDED.last_exchange_at, erp.channels.last_exchange_at)`,
		c.System, c.State, c.Endpoint, c.Stand, c.ContractVersion, c.Detail, c.CheckedAt.UTC(), c.LastExchangeAt)
	return err
}

// Channels — все каналы.
func (s *Store) Channels(ctx context.Context) ([]app.Channel, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT `+chanCols+` FROM erp.channels ORDER BY system`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []app.Channel
	for rows.Next() {
		c, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// NextSourceSeq — n номеров source_seq источника подряд; возвращает первый.
func (s *Store) NextSourceSeq(ctx context.Context, source string, n int) (int64, error) {
	var next int64
	err := s.q(ctx).QueryRow(ctx, `INSERT INTO erp.gateway_seq (source_id, next_seq) VALUES ($1, $2 + 1)
ON CONFLICT (source_id) DO UPDATE SET next_seq = erp.gateway_seq.next_seq + $2
RETURNING next_seq - $2`, source, n).Scan(&next)
	return next, err
}

// Seen — какие факты источник уже подал и с каким source_seq.
func (s *Store) Seen(ctx context.Context, source string, ids []string) (map[string]int64, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT event_id, source_seq FROM erp.gateway_seen WHERE source_id = $1 AND event_id = ANY($2)`, source, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var id string
		var seq int64
		if err := rows.Scan(&id, &seq); err != nil {
			return nil, err
		}
		out[id] = seq
	}
	return out, rows.Err()
}

// MarkSeen — отметить поданные факты.
func (s *Store) MarkSeen(ctx context.Context, source string, seqs map[string]int64) error {
	b := &pgx.Batch{}
	for id, seq := range seqs {
		b.Queue(`INSERT INTO erp.gateway_seen (source_id, event_id, source_seq) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, source, id, seq)
	}
	return s.pool.SendBatch(ctx, b).Close()
}
