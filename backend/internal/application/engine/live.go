package engine

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
)

// ErrLagging — подписчик не успевал забирать изменения: подписка закрыта,
// клиент переподключается с Last-Event-ID и догоняет по seq (AD-6, AD-21).
var ErrLagging = errors.New("живые обновления: подписчик отстал — переподключитесь с Last-Event-ID")

// LiveUpdates — публикатор живых обновлений SSE (AD-6, AD-21, FR-2): одна
// горутина на копию api ждёт сигнала «есть новое» (LISTEN/NOTIFY через порт
// ChangeLog), дочитывает изменения после последнего seq и раздаёт их
// подписчикам. Подписка догоняет пропущенное по seq (Last-Event-ID).
// Метрика ant_event_to_sse_seconds — от received_at записи-триггера до
// выдачи сообщения потоку SSE.
//
// Реализует метод Subscribe ведущего порта journal.Queries: live-реализация
// journal (эпик 04) делегирует ему операцию journal.stream.subscribe.
type LiveUpdates struct {
	log       ChangeLog
	telemetry platform.Telemetry
	now       func() time.Time
	mode      platform.Mode
	logger    *slog.Logger

	mu   sync.Mutex
	last int64
	subs map[*subscription]struct{}
}

// LiveConfig — зависимости публикатора.
type LiveConfig struct {
	Log       ChangeLog
	Telemetry platform.Telemetry
	// Now — InfraClock; nil — time.Now.
	Now    func() time.Time
	Logger *slog.Logger
	// Mode — режим ведущих портов в сообщениях (live).
	Mode platform.Mode
}

// NewLiveUpdates создаёт публикатор.
func NewLiveUpdates(c LiveConfig) *LiveUpdates {
	if c.Telemetry == nil {
		c.Telemetry = NopTelemetry{}
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.Logger == nil {
		c.Logger = slog.New(slog.DiscardHandler)
	}
	if c.Mode == "" {
		c.Mode = platform.ModeLive
	}
	return &LiveUpdates{log: c.Log, telemetry: c.Telemetry, now: c.Now, mode: c.Mode, logger: c.Logger, subs: map[*subscription]struct{}{}}
}

// Run — цикл публикатора до отмены ctx: ждёт сигнал, дочитывает и раздаёт.
// Начинает с текущего конца журнала изменений (прошлое подписки берут сами).
func (l *LiveUpdates) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		if err := l.Pump(ctx); err != nil && ctx.Err() == nil {
			l.logger.Warn("живые обновления: чтение изменений", "err", err)
			pause(ctx, time.Second)
			continue
		}
		if err := l.log.Wait(ctx); err != nil && ctx.Err() == nil {
			l.logger.Warn("живые обновления: ожидание сигнала", "err", err)
			pause(ctx, time.Second)
		}
	}
	return nil
}

// Pump дочитывает изменения после последнего раздатого seq и раздаёт их.
func (l *LiveUpdates) Pump(ctx context.Context) error {
	for {
		l.mu.Lock()
		after := l.last
		l.mu.Unlock()
		changes, err := l.log.After(ctx, after, 500)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			return nil
		}
		l.mu.Lock()
		for _, c := range changes {
			if c.Seq > l.last {
				l.last = c.Seq
			}
			for s := range l.subs {
				s.offer(c)
			}
		}
		l.mu.Unlock()
		if len(changes) < 500 {
			return nil
		}
	}
}

// Subscribe открывает подписку на изменения после afterSeq в пределах прогона
// runID (AD-38; пусто — все). Прошлое после afterSeq догоняется из журнала
// изменений, затем идут живые сообщения.
func (l *LiveUpdates) Subscribe(ctx context.Context, afterSeq int64, runID string) (appjournal.Subscription, error) {
	s := &subscription{owner: l, runID: runID, lastSeq: afterSeq, ch: make(chan Change, 256), done: make(chan struct{})}
	// Догнать пропущенное до текущего конца раздачи под замком раздачи:
	// живые сообщения не обгонят догоняемые.
	l.mu.Lock()
	defer l.mu.Unlock()
	for after := afterSeq; after < l.last; {
		changes, err := l.log.After(ctx, after, 500)
		if err != nil {
			return nil, err
		}
		if len(changes) == 0 {
			break
		}
		for _, c := range changes {
			if c.Seq <= l.last {
				s.offer(c)
			}
			after = c.Seq
		}
	}
	l.subs[s] = struct{}{}
	return s, nil
}

type subscription struct {
	owner   *LiveUpdates
	runID   string
	mu      sync.Mutex
	lastSeq int64
	lagging bool
	ch      chan Change
	done    chan struct{}
	once    sync.Once
}

// offer ставит изменение в очередь подписчика; переполнение — отставание.
func (s *subscription) offer(c Change) {
	if s.runID != "" && c.RunID != s.runID {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lagging || c.Seq < s.lastSeq {
		return
	}
	select {
	case s.ch <- c:
		s.lastSeq = c.Seq
	default:
		s.lagging = true
		close(s.ch)
	}
}

// Next — следующее изменение; блокирует до него или отмены ctx.
func (s *subscription) Next(ctx context.Context) (appjournal.Change, error) {
	select {
	case <-ctx.Done():
		return appjournal.Change{}, ctx.Err()
	case <-s.done:
		return appjournal.Change{}, context.Canceled
	case c, ok := <-s.ch:
		if !ok {
			return appjournal.Change{}, ErrLagging
		}
		if !c.ReceivedAt.IsZero() {
			// FR-2, AD-6: бюджет «событие → экран» ≤ 2 с.
			s.owner.telemetry.Observe(MetricEventToSSE, s.owner.now().Sub(c.ReceivedAt), "entity", string(c.Entity))
		}
		return appjournal.Change{Entity: c.Entity, ID: c.ID, Seq: c.Seq, RunID: c.RunID, Mode: s.owner.mode}, nil
	}
}

// Close освобождает подписку.
func (s *subscription) Close() {
	s.once.Do(func() {
		s.owner.mu.Lock()
		delete(s.owner.subs, s)
		s.owner.mu.Unlock()
		close(s.done)
	})
}
