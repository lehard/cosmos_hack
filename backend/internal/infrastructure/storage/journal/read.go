package journal

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	app "ant/internal/application/journal"
	"ant/internal/application/platform"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
)

// DefaultReadLimit — предел чтения без явного Limit.
const DefaultReadLimit = 1000

// Read — записи цепочки в порядке seq с фильтрами потока, изделия, партиции, типа,
// прогона и момента (AD-22, AD-37):
//   - ось recorded («что мы знали на T») — префикс журнала по seq: все записи
//     до последней с recorded_at ≤ T (recorded_at не убывает по seq, поэтому
//     префикс и фильтр по времени совпадают, но граница задаётся seq);
//   - ось occurred («как было на T») — записи с occurred_at ≤ T по всему
//     известному журналу.
//
// Записи возвращаются в сгенерированном типе JournalEntry: заголовок — ровно
// те байты, над которыми посчитано звено, поэтому верификатор может проверить
// цепочку по результату Read.
func (s *Store) Read(ctx context.Context, q app.ReadQuery) ([]jc.JournalEntry, error) {
	chain := q.Chain
	if chain == "" {
		chain = string(jc.JournalEntryChainMain)
	}
	limit := q.Limit
	if limit <= 0 {
		limit = DefaultReadLimit
	}
	var (
		where = []string{"chain = $1"}
		args  = []any{chain}
		order = "seq"
	)
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if q.Backward {
		order = "seq DESC"
	} else {
		add("seq > $%d", q.AfterSeq)
	}
	if q.Stream != "" {
		add("stream = $%d", q.Stream)
	}
	if q.ItemID != "" {
		add("item_id = $%d", q.ItemID)
	}
	if q.Partition != nil {
		add("partition = $%d", *q.Partition)
	}
	if q.EventType != "" {
		add("event_type = $%d", q.EventType)
	}
	if run := cmp.Or(q.RunID, q.Moment.RunID); run != "" {
		add("run_id = $%d", run)
	}
	if q.Moment.AsOf != nil {
		switch q.Moment.Axis {
		case platform.AxisRecorded:
			add("seq <= (SELECT COALESCE(MAX(p.seq), 0) FROM journal.entries p WHERE p.chain = $1 AND p.recorded_at <= $%d)", *q.Moment.AsOf)
		default:
			add("occurred_at <= $%d", *q.Moment.AsOf)
		}
	}
	args = append(args, limit)
	sql := fmt.Sprintf("SELECT header, link, salt, envelope FROM journal.entries WHERE %s ORDER BY %s LIMIT $%d",
		strings.Join(where, " AND "), order, len(args))
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []jc.JournalEntry
	for rows.Next() {
		var header string
		var link, salt, envelope []byte
		if err := rows.Scan(&header, &link, &salt, &envelope); err != nil {
			return nil, err
		}
		e, err := decode(header, link, salt, envelope)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// decode собирает запись из заголовка, звена и (демо-трек) открытого блока.
func decode(header string, link, salt, envelope []byte) (jc.JournalEntry, error) {
	var e jc.JournalEntry
	if err := json.Unmarshal([]byte(header), &e); err != nil {
		return e, fmt.Errorf("заголовок записи: %w", err)
	}
	l, err := dj.DigestFromBytes(link)
	if err != nil {
		return e, err
	}
	e.Link = l.String()
	// TODO(29): sealed — AEAD(DEK, JCS(plain_block)); до эпика 29 — открыто.
	if e.Sealed, err = dj.PlainSealed(salt, envelope); err != nil {
		return e, err
	}
	return e, nil
}

// Head — головы обеих цепочек (seq и звено) для хранителя (AD-8).
func (s *Store) Head(ctx context.Context) (app.Heads, error) {
	var h app.Heads
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT ON (chain) chain, seq, link FROM journal.entries
WHERE chain IN ('main', 'ca') ORDER BY chain, seq DESC`)
	if err != nil {
		return h, err
	}
	defer rows.Close()
	h.MainLink, h.CALink = dj.ZeroLink.String(), dj.ZeroLink.String()
	for rows.Next() {
		var chain string
		var seq int64
		var link []byte
		if err := rows.Scan(&chain, &seq, &link); err != nil {
			return h, err
		}
		d, err := dj.DigestFromBytes(link)
		if err != nil {
			return h, err
		}
		if chain == "main" {
			h.MainSeq, h.MainLink = seq, d.String()
		} else {
			h.CASeq, h.CALink = seq, d.String()
		}
	}
	return h, rows.Err()
}

// Open — конверт записи с проверкой commit (AD-23). Демо-трек: блок открыт.
func (s *Store) Open(_ context.Context, e jc.JournalEntry) (app.Envelope, error) {
	salt, env, err := dj.OpenPlain(e)
	if errors.Is(err, dj.ErrSealed) {
		return app.Envelope{}, app.ErrSealed
	}
	if err != nil {
		return app.Envelope{}, err
	}
	return app.Envelope{Raw: env, Salt: salt}, nil
}

// Entry — одна запись по цепочке и seq (для хранителя, верификатора, ссылок).
func (s *Store) Entry(ctx context.Context, chain string, seq int64) (jc.JournalEntry, error) {
	var header string
	var link, salt, envelope []byte
	err := s.pool.QueryRow(ctx, "SELECT header, link, salt, envelope FROM journal.entries WHERE chain = $1 AND seq = $2", chain, seq).
		Scan(&header, &link, &salt, &envelope)
	if errors.Is(err, pgx.ErrNoRows) {
		return jc.JournalEntry{}, fmt.Errorf("запись %s/%d не найдена", chain, seq)
	}
	if err != nil {
		return jc.JournalEntry{}, err
	}
	return decode(header, link, salt, envelope)
}
