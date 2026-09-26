package journal

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	app "ant/internal/application/journal"
)

// Listener — адаптер порта Signal на LISTEN/NOTIFY (AD-6): держит одно
// соединение LISTEN ant_journal и раздаёт ожидающим seq головы основной
// цепочки. Данных в сигнале нет; после переподключения голова читается из
// журнала — копия догоняет по seq и ничего не теряет.
type Listener struct {
	pool *pgxpool.Pool
	log  *slog.Logger

	mu      sync.Mutex
	head    int64
	changed chan struct{}
}

// NewListener создаёт слушателя; запускается Run.
func NewListener(pool *pgxpool.Pool, log *slog.Logger) *Listener {
	if log == nil {
		log = slog.Default()
	}
	return &Listener{pool: pool, log: log, changed: make(chan struct{})}
}

var _ app.Signal = (*Listener)(nil)

// Head — последний известный seq основной цепочки.
func (l *Listener) Head() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.head
}

// Wait блокирует, пока голова не станет больше afterSeq, или до отмены ctx.
func (l *Listener) Wait(ctx context.Context, afterSeq int64) (int64, error) {
	for {
		l.mu.Lock()
		h, ch := l.head, l.changed
		l.mu.Unlock()
		if h > afterSeq {
			return h, nil
		}
		select {
		case <-ctx.Done():
			return h, ctx.Err()
		case <-ch:
		}
	}
}

func (l *Listener) advance(seq int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if seq <= l.head {
		return
	}
	l.head = seq
	close(l.changed)
	l.changed = make(chan struct{})
}

// Run слушает канал до отмены ctx, переподключаясь после обрыва (пауза —
// по нарастающей до 5 с).
func (l *Listener) Run(ctx context.Context) error {
	backoff := 100 * time.Millisecond
	for {
		err := l.listen(ctx)
		if ctx.Err() != nil {
			return nil
		}
		l.log.Warn("журнал: LISTEN прерван, переподключение", "err", err, "after", backoff)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, 5*time.Second)
	}
}

func (l *Listener) listen(ctx context.Context) error {
	pc, err := l.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	// Соединение после LISTEN в пул не возвращается.
	conn := pc.Hijack()
	defer func() { _ = conn.Close(context.Background()) }()
	if _, err := conn.Exec(ctx, "LISTEN "+NotifyChannel); err != nil {
		return err
	}
	// Сначала LISTEN, потом голова: сигнал между ними не теряется.
	var seq int64
	if err := conn.QueryRow(ctx, "SELECT COALESCE(MAX(seq), 0) FROM journal.entries WHERE chain = 'main'").Scan(&seq); err != nil {
		return err
	}
	l.advance(seq)
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		seq, err := strconv.ParseInt(n.Payload, 10, 64)
		if err != nil {
			l.log.Warn("журнал: сигнал без seq", "payload", n.Payload)
			continue
		}
		l.advance(seq)
	}
}
