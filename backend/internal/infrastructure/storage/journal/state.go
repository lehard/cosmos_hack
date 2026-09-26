package journal

import (
	"context"

	"github.com/jackc/pgx/v5"

	app "ant/internal/application/journal"
)

// Чтение состояния потребителей и аренд для модуля ops (эпик 34, FR-127):
// страница «состояние компонентов» — роли по арендам, очереди по курсорам и
// их отставание от головы журнала. Только чтение схемы journal_state и
// индексов журнала; писатели аренд и курсоров — Leases и Append.

// All — все аренды, включая истёкшие (роль остановлена — аренда истекла; не
// запускалась — строки нет), по имени.
func (l *Leases) All(ctx context.Context) ([]Lease, error) {
	rows, err := l.pool.Query(ctx, `SELECT name, holder, epoch, expires_at FROM journal_state.leases ORDER BY name`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Lease, error) {
		var x Lease
		err := r.Scan(&x.Name, &x.Holder, &x.Epoch, &x.ExpiresAt)
		x.ExpiresAt = x.ExpiresAt.UTC()
		return x, err
	})
}

// Backlog — курсор потребителя и его отставание: Head — голова того, что
// потребитель читает (глобальный — основная цепочка, воркер партиции —
// последняя запись-триггер партиции), Pending — записей после курсора.
type Backlog struct {
	Name      string
	Partition int
	Seq       int64
	Head      int64
	Pending   int64
}

// GlobalBacklog — глобальные потребители (partition = -1, AD-45): курсор,
// голова основной цепочки и число записей после курсора.
func (s *Store) GlobalBacklog(ctx context.Context) ([]Backlog, error) {
	rows, err := s.pool.Query(ctx, `SELECT o.name, o.seq,
    COALESCE((SELECT MAX(seq) FROM journal.entries WHERE chain = 'main'), 0),
    (SELECT COUNT(*) FROM journal.entries e WHERE e.chain = 'main' AND e.seq > o.seq)
FROM journal_state.consumer_offsets o
WHERE o.partition = $1
ORDER BY o.name`, app.GlobalPartition)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Backlog, error) {
		b := Backlog{Partition: app.GlobalPartition}
		err := r.Scan(&b.Name, &b.Seq, &b.Head, &b.Pending)
		return b, err
	})
}

// WorkerBacklog — воркер по партициям 0…partitions-1 (AD-5, AD-6): курсор
// engine.worker, последняя запись-триггер партиции и число триггеров после
// курсора (изделия с необработанным входом).
func (s *Store) WorkerBacklog(ctx context.Context, partitions int) ([]Backlog, error) {
	if partitions <= 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT p.n, COALESCE(o.seq, 0), COALESCE(t.last, 0), COALESCE(t.pending, 0)
FROM generate_series(0, $2::int - 1) AS p(n)
LEFT JOIN journal_state.consumer_offsets o ON o.name = $1 AND o.partition = p.n
LEFT JOIN LATERAL (
    SELECT MAX(e.seq) AS last, COUNT(*) AS pending FROM journal.entries e
    WHERE e.chain = 'main' AND e.partition = p.n AND e.is_trigger AND e.seq > COALESCE(o.seq, 0)
) t ON true
ORDER BY p.n`, app.WorkerConsumer, partitions)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Backlog, error) {
		b := Backlog{Name: app.WorkerConsumer}
		err := r.Scan(&b.Partition, &b.Seq, &b.Head, &b.Pending)
		if b.Head < b.Seq {
			b.Head = b.Seq
		}
		return b, err
	})
}
