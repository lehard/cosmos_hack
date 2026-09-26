package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	app "ant/internal/application/ingest"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/ingest"
	journalstore "ant/internal/infrastructure/storage/journal"
)

// Store — хранилище приёма в Postgres (схема ingest): реестр идемпотентности,
// учёт source_seq и карантин. Внутри AppendRequest.Project пишет в транзакции
// журнала (journalstore.Tx), иначе — своим пулом.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore — хранилище на пуле роли приложения (ant_app).
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// q — транзакция журнала из контекста Project или пул.
func (s *Store) q(ctx context.Context) querier {
	if tx, ok := journalstore.Tx(ctx); ok {
		return tx
	}
	return s.pool
}

// Seen — запись реестра по ключу.
func (s *Store) Seen(ctx context.Context, src, id string) (*dom.Seen, error) {
	v := dom.Seen{SourceID: src, EventID: id}
	err := s.q(ctx).QueryRow(ctx, `SELECT fingerprint, seq, source_seq, signature_verified FROM ingest.seen
WHERE source_id = $1 AND event_id = $2`, src, id).Scan(&v.Fingerprint, &v.Seq, &v.SourceSeq, &v.SignatureVerified)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// SourceState — учёт номеров источника.
func (s *Store) SourceState(ctx context.Context, src string) (dom.SourceState, error) {
	var raw []byte
	err := s.q(ctx).QueryRow(ctx, `SELECT state FROM ingest.sources WHERE source_id = $1`, src).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return dom.SourceState{SourceID: src}, nil
	}
	if err != nil {
		return dom.SourceState{}, err
	}
	var st dom.SourceState
	if err := json.Unmarshal(raw, &st); err != nil {
		return dom.SourceState{}, fmt.Errorf("ingest.sources %s: %w", src, err)
	}
	st.SourceID = src
	return st, nil
}

// Commit — принятое сообщение и состояние источника (одной транзакцией).
func (s *Store) Commit(ctx context.Context, v dom.Seen, st dom.SourceState) error {
	if _, err := s.q(ctx).Exec(ctx, `INSERT INTO ingest.seen (source_id, event_id, fingerprint, seq, source_seq, signature_verified)
VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (source_id, event_id) DO NOTHING`,
		v.SourceID, v.EventID, v.Fingerprint, v.Seq, v.SourceSeq, v.SignatureVerified); err != nil {
		return err
	}
	return s.SaveSourceState(ctx, st)
}

// SaveSourceState — состояние источника.
func (s *Store) SaveSourceState(ctx context.Context, st dom.SourceState) error {
	raw, err := json.Marshal(st)
	if err != nil {
		return err
	}
	_, err = s.q(ctx).Exec(ctx, `INSERT INTO ingest.sources (source_id, state) VALUES ($1, $2)
ON CONFLICT (source_id) DO UPDATE SET state = EXCLUDED.state`, st.SourceID, raw)
	return err
}

// Sources — все источники по source_id.
func (s *Store) Sources(ctx context.Context) ([]dom.SourceState, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT source_id, state FROM ingest.sources ORDER BY source_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dom.SourceState
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		var st dom.SourceState
		if err := json.Unmarshal(raw, &st); err != nil {
			return nil, err
		}
		st.SourceID = id
		out = append(out, st)
	}
	return out, rows.Err()
}

const qcols = `id::text, journal_seq, source_id, event_id, source_seq, event_type, fingerprint, material_address, code, field,
detail, status, received_at, raw, resolved_by`

func scanQ(r pgx.Row) (app.QuarantineRecord, error) {
	var v app.QuarantineRecord
	var code, status string
	err := r.Scan(&v.ID, &v.JournalSeq, &v.SourceID, &v.EventID, &v.SourceSeq, &v.EventType, &v.Fingerprint, &v.MaterialAddress,
		&code, &v.Field, &v.Detail, &status, &v.ReceivedAt, &v.Raw, &v.ResolvedBy)
	v.Code, v.Status = errcodes.Code(code), app.QuarantineStatus(status)
	v.ReceivedAt = v.ReceivedAt.UTC()
	return v, err
}

// Find — по отпечатку исходных байтов и коду.
func (s *Store) Find(ctx context.Context, fp string, code errcodes.Code) (*app.QuarantineRecord, error) {
	v, err := scanQ(s.q(ctx).QueryRow(ctx, `SELECT `+qcols+` FROM ingest.quarantine WHERE fingerprint = $1 AND code = $2`, fp, string(code)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// Put — новая запись карантина.
func (s *Store) Put(ctx context.Context, r app.QuarantineRecord) error {
	_, err := s.q(ctx).Exec(ctx, `INSERT INTO ingest.quarantine (id, journal_seq, source_id, event_id, source_seq, event_type,
fingerprint, material_address, code, field, detail, status, received_at, raw, resolved_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		r.ID, r.JournalSeq, r.SourceID, r.EventID, r.SourceSeq, r.EventType, r.Fingerprint, r.MaterialAddress, string(r.Code),
		r.Field, r.Detail, string(r.Status), r.ReceivedAt, r.Raw, r.ResolvedBy)
	return err
}

// Get — по ID.
func (s *Store) Get(ctx context.Context, id string) (app.QuarantineRecord, error) {
	v, err := scanQ(s.q(ctx).QueryRow(ctx, `SELECT `+qcols+` FROM ingest.quarantine WHERE id::text = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return app.QuarantineRecord{}, app.ErrNotFound
	}
	return v, err
}

// List — по фильтру, новые сначала.
func (s *Store) List(ctx context.Context, f app.QuarantineQuery) ([]app.QuarantineRecord, error) {
	var where []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if f.Status != "" {
		add("status = $%d", string(f.Status))
	}
	if f.SourceID != "" {
		add("source_id = $%d", f.SourceID)
	}
	if f.Code != "" {
		add("code = $%d", string(f.Code))
	}
	sql := `SELECT ` + qcols + ` FROM ingest.quarantine`
	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}
	lim := f.Limit
	if lim <= 0 {
		lim = 1000
	}
	args = append(args, lim, max(f.Offset, 0))
	sql += fmt.Sprintf(" ORDER BY created DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := s.q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []app.QuarantineRecord
	for rows.Next() {
		v, err := scanQ(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Count — число записей в состоянии (пусто — все).
func (s *Store) Count(ctx context.Context, st app.QuarantineStatus) (int64, error) {
	var n int64
	err := s.q(ctx).QueryRow(ctx, `SELECT count(*) FROM ingest.quarantine WHERE $1 = '' OR status = $1`, string(st)).Scan(&n)
	return n, err
}

// CountBySource — нерешённые записи по источникам.
func (s *Store) CountBySource(ctx context.Context) (map[string]int64, error) {
	rows, err := s.q(ctx).Query(ctx, `SELECT source_id, count(*) FROM ingest.quarantine
WHERE status IN ('open', 'still_invalid') GROUP BY source_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var id string
		var n int64
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// Resolve — итог переобработки.
func (s *Store) Resolve(ctx context.Context, id string, st app.QuarantineStatus, by string) error {
	tag, err := s.q(ctx).Exec(ctx, `UPDATE ingest.quarantine SET status = $2, resolved_by = $3 WHERE id::text = $1`, id, string(st), by)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return app.ErrNotFound
	}
	return nil
}

var (
	_ app.Registry        = (*Store)(nil)
	_ app.QuarantineStore = (*Store)(nil)
)
