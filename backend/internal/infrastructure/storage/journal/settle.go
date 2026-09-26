package journal

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
// курсора потребителем этого процесса, Store.OnProgress) и перепроверка раз в
// settleRecheck для потребителей других процессов; срок — Timeout
// (обязателен: потерянный воркер не должен вешать прогон навсегда).
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

	// hook — подписка на сдвиги курсоров этого процесса (Store.OnProgress).
	hook sync.Once
}

// settleRecheck — как часто Settler перепроверяет условие без сигнала:
// курсоры потребителей другого процесса (отдельный worker, k8s) сигнала в
// этот процесс не дают (эпик 35: pg_notify на каждый сдвиг курсора убран).
const settleRecheck = 100 * time.Millisecond

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
	// Сдвиг курсора потребителем этого процесса будит ожидание сразу.
	s.hook.Do(func() { s.Store.OnProgress(s.Signal.bump) })
	recheck := time.NewTimer(settleRecheck)
	defer recheck.Stop()
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
		recheck.Reset(settleRecheck)
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w за %s", ErrNotSettled, timeout)
		case <-ch:
		case <-recheck.C:
		}
	}
}

// Settled — потребители consumers дочитали основную цепочку до головы, и ни в
// одной из partitions партиций нет записей-триггеров после курсора воркера
// (AD-5: изделие с необработанным входом).
func (s *Store) Settled(ctx context.Context, consumers []string, partitions int) (bool, error) {
	// Эпик 35 (узкое место MS-1): одна выборка вместо запроса на каждого
	// потребителя — Settler перепроверяет условие на каждый сдвиг любого
	// курсора, а потребителей у проектора десятки.
	if consumers == nil {
		consumers = []string{}
	}
	var settled bool
	err := s.pool.QueryRow(ctx, `WITH h AS (SELECT COALESCE(MAX(seq), 0) AS head FROM journal.entries WHERE chain = 'main')
SELECT NOT EXISTS (
    SELECT 1 FROM unnest($1::text[]) AS c(name)
    LEFT JOIN journal_state.consumer_offsets o ON o.name = c.name AND o.partition = $2
    WHERE COALESCE(o.seq, 0) < (SELECT head FROM h)
) AND NOT EXISTS (
    SELECT 1 FROM generate_series(0, $4::int - 1) AS p(n)
    CROSS JOIN LATERAL (
        SELECT 1 FROM journal.entries e
        WHERE e.chain = 'main' AND e.partition = p.n AND e.is_trigger
          AND e.seq > COALESCE((SELECT o.seq FROM journal_state.consumer_offsets o WHERE o.name = $3 AND o.partition = p.n), 0)
        LIMIT 1
    ) x
)`, consumers, app.GlobalPartition, app.WorkerConsumer, max(partitions, 0)).Scan(&settled)
	if err != nil {
		return false, err
	}
	return settled, nil
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
