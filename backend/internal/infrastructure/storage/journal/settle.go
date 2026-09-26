package journal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	app "ant/internal/application/journal"
)

// DefaultSettleTimeout — сколько Settler ждёт, пока потребители догонят журнал.
const DefaultSettleTimeout = 30 * time.Second

// Settler — «воркер, межизделийная стадия и проектор обработали всё, что есть
// в журнале» (порт application/simulation.Settler, эпик 16): перед проверкой
// табло автосверки и решением демо-подписанта прогон ждёт, пока курсоры
// догонят голову основной цепочки.
//
// Ожидание — по сигналу журнала (Listener.Progress: новая голова или сдвиг
// курсора потребителя), без опроса по таймеру; срок — Timeout (обязателен:
// потерянный воркер не должен вешать прогон навсегда).
type Settler struct {
	Store  *Store
	Signal *Listener
	// Consumers — глобальные потребители (стадия crossitem, проекции
	// projector), которые должны дочитать журнал до головы.
	Consumers func() []string
	// Partitions — P воркера (engine.partitions): у каждой партиции не должно
	// остаться записей-триггеров после курсора воркера.
	Partitions int
	// Timeout — предел ожидания (0 — DefaultSettleTimeout).
	Timeout time.Duration
}

// ErrNotSettled — потребители не догнали журнал за отведённое время.
var ErrNotSettled = errors.New("журнал: потребители не догнали голову")

// Settle ждёт, пока все потребители обработают журнал до головы; runID не
// сужает ожидание — прогоны идут по очереди (AD-38).
func (s *Settler) Settle(ctx context.Context, _ string) error {
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = DefaultSettleTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var consumers []string
	if s.Consumers != nil {
		consumers = s.Consumers()
	}
	for {
		// Канал берётся до проверки: движение между проверкой и ожиданием не теряется.
		ch := s.Signal.Progress()
		ok, err := s.Store.Settled(ctx, consumers, s.Partitions)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("%w за %s", ErrNotSettled, timeout)
			}
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w за %s", ErrNotSettled, timeout)
		case <-ch:
		}
	}
}

// Settled — потребители consumers дочитали основную цепочку до головы, и ни в
// одной из partitions партиций нет записей-триггеров после курсора воркера
// (AD-5: изделие с необработанным входом).
func (s *Store) Settled(ctx context.Context, consumers []string, partitions int) (bool, error) {
	var head int64
	if err := s.pool.QueryRow(ctx, "SELECT COALESCE(MAX(seq), 0) FROM journal.entries WHERE chain = 'main'").Scan(&head); err != nil {
		return false, err
	}
	for _, name := range consumers {
		cur, err := s.Cursor(ctx, name, app.GlobalPartition)
		if err != nil {
			return false, err
		}
		if cur < head {
			return false, nil
		}
	}
	if partitions <= 0 {
		return true, nil
	}
	var pending bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (
    SELECT 1 FROM generate_series(0, $2::int - 1) AS p(n)
    CROSS JOIN LATERAL (
        SELECT 1 FROM journal.entries e
        WHERE e.chain = 'main' AND e.partition = p.n AND e.is_trigger
          AND e.seq > COALESCE((SELECT o.seq FROM journal_state.consumer_offsets o WHERE o.name = $1 AND o.partition = p.n), 0)
        LIMIT 1
    ) x
)`, app.WorkerConsumer, partitions).Scan(&pending)
	if err != nil {
		return false, err
	}
	return !pending, nil
}

// RecordedHead — recorded_at головы основной цепочки (доменное «сейчас»
// журнала по последней записи, AD-37); журнал пуст — ok = false.
func (s *Store) RecordedHead(ctx context.Context) (time.Time, bool, error) {
	var t time.Time
	err := s.pool.QueryRow(ctx, "SELECT recorded_at FROM journal.entries WHERE chain = 'main' ORDER BY seq DESC LIMIT 1").Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return t.UTC(), true, nil
}
