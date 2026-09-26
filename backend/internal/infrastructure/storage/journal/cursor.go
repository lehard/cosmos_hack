package journal

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	app "ant/internal/application/journal"
	jc "ant/internal/contracts/journal"
)

// Cursor — курсор потребителя name в партиции (GlobalPartition — глобальный);
// нет строки — 0.
func (s *Store) Cursor(ctx context.Context, name string, partition int) (int64, error) {
	var seq int64
	err := s.pool.QueryRow(ctx, "SELECT seq FROM journal_state.consumer_offsets WHERE name = $1 AND partition = $2", name, partition).Scan(&seq)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return seq, err
}

// PendingItem — изделие партиции с необработанным входом после курсора:
// последняя запись-триггер (AD-5).
type PendingItem struct {
	ItemID  string
	UpToSeq int64
	Trigger jc.JournalEntry
}

// PendingItems — изделия партиции с записями-триггерами после afterSeq в
// порядке возрастания их последнего триггера, не больше limit. Воркер,
// обработавший изделия по порядку, может сдвигать курсор до UpToSeq каждого:
// у всех изделий, не попавших в ответ, последний триггер позже (AD-5, AD-45).
func (s *Store) PendingItems(ctx context.Context, partition int, afterSeq int64, limit int) ([]PendingItem, error) {
	if limit <= 0 {
		limit = DefaultReadLimit
	}
	rows, err := s.pool.Query(ctx, `SELECT item_id, seq, header, link, salt, envelope FROM (
    SELECT DISTINCT ON (item_id) item_id, seq, header, link, salt, envelope
    FROM journal.entries
    WHERE chain = 'main' AND partition = $1 AND seq > $2 AND is_trigger
    ORDER BY item_id, seq DESC
) t ORDER BY seq LIMIT $3`, partition, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PendingItem
	for rows.Next() {
		var p PendingItem
		var header string
		var link, salt, envelope []byte
		if err := rows.Scan(&p.ItemID, &p.UpToSeq, &header, &link, &salt, &envelope); err != nil {
			return nil, err
		}
		if p.Trigger, err = decode(header, link, salt, envelope); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// PartitionLeaseName — имя аренды партиции воркера (application/journal.PartitionLease).
func PartitionLeaseName(p int) string { return app.PartitionLease(p) }
