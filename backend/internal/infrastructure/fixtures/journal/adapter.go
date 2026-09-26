package journal

import (
	"context"
	"strconv"
	"sync"
	"time"

	app "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
)

// Adapter — реализация fixtures ведущих портов модуля journal (AD-36): журнал
// событий, голова, таймлайн — из мира заготовок; живые обновления (SSE) —
// смена шага курсора: изменения сущностей шагов, через которые прошёл курсор.
type Adapter struct {
	// Poll — период опроса курсора подпиской (по умолчанию 500 мс).
	Poll time.Duration
	// Runtime — мир заготовок; nil — loader.Default() (тесты подставляют свой).
	Runtime *loader.Runtime
}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{Poll: 500 * time.Millisecond} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

func (a *Adapter) rt() (*loader.Runtime, error) {
	if a.Runtime != nil {
		return a.Runtime, nil
	}
	return loader.Default()
}

// Subscribe — живые обновления (journal.stream.subscribe, AD-21): подписка
// опрашивает курсор и при смене шага или прогона отдаёт изменения сущностей
// шагов между прежним и новым положением (переход назад — тоже).
func (a *Adapter) Subscribe(ctx context.Context, _ int64, _ string) (app.Subscription, error) {
	rt, err := a.rt()
	if err != nil {
		return nil, err
	}
	st, _, err := rt.State(ctx)
	if err != nil {
		return nil, err
	}
	poll := a.Poll
	if poll <= 0 {
		poll = 500 * time.Millisecond
	}
	return &subscription{rt: rt, step: st.Step, run: st.RunID, scenario: st.Scenario, ticker: time.NewTicker(poll), done: make(chan struct{})}, nil
}

// subscription — подписка на смену шага курсора.
type subscription struct {
	rt       *loader.Runtime
	step     int
	run      string
	scenario string
	queue    []loader.SeqChange
	ticker   *time.Ticker
	once     sync.Once
	done     chan struct{}
}

// Next блокирует до следующего изменения, отмены ctx или Close.
func (s *subscription) Next(ctx context.Context) (app.Change, error) {
	for len(s.queue) == 0 {
		select {
		case <-ctx.Done():
			return app.Change{}, ctx.Err()
		case <-s.done:
			return app.Change{}, context.Canceled
		case <-s.ticker.C:
		}
		st, sc, err := s.rt.State(ctx)
		if err != nil {
			return app.Change{}, err
		}
		if st.Step == s.step && st.RunID == s.run && st.Scenario == s.scenario {
			continue
		}
		from := s.step
		if st.Scenario != s.scenario || st.RunID != s.run {
			from = -1 // другой прогон или сценарий — изменилось всё, что есть к шагу
		}
		s.queue = s.rt.ChangesBetween(sc, from, st.Step, st.RunID)
		s.step, s.run, s.scenario = st.Step, st.RunID, st.Scenario
	}
	c := s.queue[0]
	s.queue = s.queue[1:]
	return app.Change{Entity: platform.EntityKind(c.Entity), ID: c.ID, Seq: c.Seq, RunID: c.RunID, Mode: platform.ModeFixtures}, nil
}

// Close освобождает подписку.
func (s *subscription) Close() {
	s.once.Do(func() {
		s.ticker.Stop()
		close(s.done)
	})
}

// Entries — журнал событий (journal.entry.list) с фильтром.
func (a *Adapter) Entries(ctx context.Context, f app.EntryFilter, m platform.Moment, _ platform.Page) (app.JournalEntryList, error) {
	params := map[string]string{"item_id": f.ItemID, "stream": f.Stream, "event_type": f.EventType, "entry_kind": f.EntryKind}
	if f.AfterSeq > 0 {
		params["after_seq"] = strconv.FormatInt(f.AfterSeq, 10)
	}
	v, err := respond[app.JournalEntryList](ctx, "journal.entry.list", params, &m)
	if err != nil {
		return v, err
	}
	out := v.Items[:0]
	for _, x := range v.Items {
		if f.ItemID != "" && (x.ItemID == nil || *x.ItemID != f.ItemID) {
			continue
		}
		if (f.Stream != "" && x.Stream != f.Stream) || (f.EventType != "" && x.EventType != f.EventType) ||
			(f.EntryKind != "" && x.EntryKind != f.EntryKind) || x.Seq <= f.AfterSeq {
			continue
		}
		out = append(out, x)
	}
	v.Items = out
	return v, nil
}

// Entry — запись журнала по seq (journal.entry.read).
func (a *Adapter) Entry(ctx context.Context, seq int64) (app.JournalEntryView, error) {
	return respond[app.JournalEntryView](ctx, "journal.entry.read", map[string]string{"seq": strconv.FormatInt(seq, 10)}, nil)
}

// Head — голова журнала (journal.head.read).
func (a *Adapter) Head(ctx context.Context, m platform.Moment) (app.JournalHead, error) {
	return respond[app.JournalHead](ctx, "journal.head.read", nil, &m)
}

// Timeline — таймлайн под живой картой (journal.timeline.read, FR-4).
func (a *Adapter) Timeline(ctx context.Context, m platform.Moment) (app.TimelineData, error) {
	return respond[app.TimelineData](ctx, "journal.timeline.read", nil, &m)
}
