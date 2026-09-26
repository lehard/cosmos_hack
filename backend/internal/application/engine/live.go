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
// ChangeLog), дочитывает изменения после последней позиции и раздаёт их
// подписчикам. Подписка догоняет пропущенное по seq (Last-Event-ID).
// Метрика ant_event_to_sse_seconds — от received_at записи-триггера до
// выдачи сообщения потоку SSE.
//
// Реализует метод Subscribe ведущего порта journal.Queries: live-реализация
// journal (эпик 04) делегирует ему операцию journal.stream.subscribe
// (journal.WithLive).
type LiveUpdates struct {
	log       ChangeLog
	telemetry platform.Telemetry
	now       func() time.Time
	mode      platform.Mode
	logger    *slog.Logger

	mu    sync.Mutex
	ready bool
	// last — позиция последнего раздатого изменения.
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

// liveBatch — сколько изменений читается за шаг.
const liveBatch = 500

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

// initLocked — начать раздачу с текущего конца журнала изменений (прошлое
// подписки берут сами). Вызывается под l.mu.
func (l *LiveUpdates) initLocked(ctx context.Context) error {
	if l.ready {
		return nil
	}
	tail, err := l.log.Tail(ctx)
	if err != nil {
		return err
	}
	l.last, l.ready = tail, true
	return nil
}

// Run — цикл публикатора до отмены ctx: ждёт сигнал, дочитывает и раздаёт.
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

// Pump дочитывает изменения после последней раздатой позиции и раздаёт их.
func (l *LiveUpdates) Pump(ctx context.Context) error {
	for {
		l.mu.Lock()
		err := l.initLocked(ctx)
		after := l.last
		l.mu.Unlock()
		if err != nil {
			return err
		}
		changes, err := l.log.After(ctx, ChangeQuery{AfterPos: after, Limit: liveBatch})
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			return nil
		}
		l.mu.Lock()
		for _, c := range changes {
			if c.Pos <= l.last {
				continue
			}
			l.last = c.Pos
			for s := range l.subs {
				s.offer(c)
			}
		}
		l.mu.Unlock()
		if len(changes) < liveBatch {
			return nil
		}
	}
}

// Subscribe открывает подписку на изменения после afterSeq в пределах прогона
// runID (AD-38; пусто — все). afterSeq > 0 (Last-Event-ID) — сначала
// догоняются изменения с seq > afterSeq из журнала изменений, затем идут
// живые; afterSeq = 0 — только живые (текущее состояние клиент читает сам).
func (l *LiveUpdates) Subscribe(ctx context.Context, afterSeq int64, runID string) (appjournal.Subscription, error) {
	s := &subscription{owner: l, runID: runID, ch: make(chan Change, 256), done: make(chan struct{})}
	// Догнать пропущенное до текущего конца раздачи под замком раздачи:
	// живые сообщения не обгонят догоняемые и не повторят их.
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.initLocked(ctx); err != nil {
		return nil, err
	}
	if afterSeq > 0 && l.last > 0 {
		for pos := int64(0); ; {
			changes, err := l.log.After(ctx, ChangeQuery{AfterPos: pos, UpToPos: l.last, AfterSeq: afterSeq, Limit: liveBatch})
			if err != nil {
				return nil, err
			}
			for _, c := range changes {
				s.offer(c)
				pos = c.Pos
			}
			if len(changes) < liveBatch {
				break
			}
		}
	}
	l.subs[s] = struct{}{}
	return s, nil
}

type subscription struct {
	owner   *LiveUpdates
	runID   string
	mu      sync.Mutex
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
	if s.lagging {
		return
	}
	select {
	case s.ch <- c:
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
