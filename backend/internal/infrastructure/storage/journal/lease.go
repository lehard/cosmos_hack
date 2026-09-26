package journal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	app "ant/internal/application/journal"
)

// Leases — адаптер postgres порта LeaseStore (AD-6, AD-35, ключ lease_store):
// аренды партиций воркеров и ролей-лидеров с эпохой. Время — только
// InfraClock (AD-37): виртуальное время сценария на аренды не влияет.
//
// Эпоха растёт при каждой смене держателя и при повторном взятии после
// истечения: запись с прежней эпохой Append отвергает (ErrFenced), даже если
// копия «проснулась» после паузы и считает аренду своей.
type Leases struct {
	pool  *pgxpool.Pool
	clock app.InfraClock
}

// NewLeases создаёт хранилище аренд.
func NewLeases(pool *pgxpool.Pool, clock app.InfraClock) *Leases {
	return &Leases{pool: pool, clock: clock}
}

var _ app.LeaseStore = (*Leases)(nil)

// Acquire берёт свободную или истёкшую аренду name либо продлевает свою
// действующую (эпоха та же). Занята другой копией — ok = false.
func (l *Leases) Acquire(ctx context.Context, name, holder string, ttl time.Duration) (app.Fence, bool, error) {
	if name == "" || holder == "" || ttl <= 0 {
		return app.Fence{}, false, fmt.Errorf("аренда: имя %q, держатель %q, ttl %v", name, holder, ttl)
	}
	now := l.clock.Now().UTC()
	var epoch int64
	err := l.pool.QueryRow(ctx, `INSERT INTO journal_state.leases AS l (name, holder, epoch, expires_at, acquired_at)
VALUES ($1, $2, 1, $4, $3)
ON CONFLICT (name) DO UPDATE SET
    epoch = CASE WHEN l.holder = EXCLUDED.holder AND l.expires_at > $3 THEN l.epoch ELSE l.epoch + 1 END,
    acquired_at = CASE WHEN l.holder = EXCLUDED.holder AND l.expires_at > $3 THEN l.acquired_at ELSE $3 END,
    holder = EXCLUDED.holder,
    expires_at = EXCLUDED.expires_at
WHERE l.holder = EXCLUDED.holder AND l.expires_at > $3 OR l.expires_at <= $3
RETURNING epoch`, name, holder, now, now.Add(ttl)).Scan(&epoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.Fence{}, false, nil
	}
	if err != nil {
		return app.Fence{}, false, err
	}
	return app.Fence{Lease: name, Epoch: epoch}, true, nil
}

// Release отдаёт аренду (только ту же эпоху): следующий держатель получит
// новую эпоху сразу, не дожидаясь истечения.
func (l *Leases) Release(ctx context.Context, f app.Fence) error {
	_, err := l.pool.Exec(ctx, `UPDATE journal_state.leases SET expires_at = $3
WHERE name = $1 AND epoch = $2 AND expires_at > $3`, f.Lease, f.Epoch, l.clock.Now().UTC())
	return err
}

// Lease — состояние аренды.
type Lease struct {
	Name      string
	Holder    string
	Epoch     int64
	ExpiresAt time.Time
}

// Live — действующие аренды с префиксом имени (распределение партиций
// между копиями, страница ops).
func (l *Leases) Live(ctx context.Context, prefix string) ([]Lease, error) {
	rows, err := l.pool.Query(ctx, `SELECT name, holder, epoch, expires_at FROM journal_state.leases
WHERE starts_with(name, $1) AND expires_at > $2 ORDER BY name`, prefix, l.clock.Now().UTC())
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Lease, error) {
		var x Lease
		err := r.Scan(&x.Name, &x.Holder, &x.Epoch, &x.ExpiresAt)
		return x, err
	})
}
