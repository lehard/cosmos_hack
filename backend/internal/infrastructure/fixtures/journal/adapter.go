package journal

import (
	"context"
	"encoding/json"
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
// шагов между прежним и новым положением (переход назад — тоже), а также
// изменения объектов сессионных фактов — команд людей поверх мира
// (loader.Runtime.Record): открытые экраны перечитывают их сами.
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
	return &subscription{rt: rt, step: st.Step, run: st.RunID, scenario: st.Scenario, sver: rt.SessionVersion(), ticker: time.NewTicker(poll), done: make(chan struct{})}, nil
}

// subscription — подписка на смену шага курсора.
type subscription struct {
	rt       *loader.Runtime
	step     int
	run      string
	scenario string
	// sver — версия сессионных изменений, уже отданных подписке.
	sver   int64
	queue  []loader.SeqChange
	ticker *time.Ticker
	once   sync.Once
	done   chan struct{}
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
			s.queue, s.sver = s.rt.SessionChanges(s.sver)
			continue
		}
		from := s.step
		if st.Scenario != s.scenario || st.RunID != s.run {
			from = -1 // другой прогон или сценарий — изменилось всё, что есть к шагу
		}
		s.queue = s.rt.ChangesBetween(sc, from, st.Step, st.RunID)
		var sess []loader.SeqChange
		sess, s.sver = s.rt.SessionChanges(s.sver)
		s.queue = append(s.queue, sess...)
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

// Entries — журнал событий (journal.entry.list): все записи, известные к
// шагу курсора (ответы journal.entry.read мира заготовок), с тем же отбором,
// порядком (order=desc — новые сверху) и курсором страниц, что у live.
func (a *Adapter) Entries(ctx context.Context, f app.EntryFilter, m platform.Moment, p platform.Page) (app.JournalEntryList, error) {
	rt, err := a.rt()
	if err != nil {
		return app.JournalEntryList{}, err
	}
	var all []app.JournalEntryView
	err = rt.RespondAll(ctx, "journal.entry.read", &m, func(body json.RawMessage) error {
		var v app.JournalEntryView
		if err := json.Unmarshal(body, &v); err != nil {
			return err
		}
		all = append(all, v)
		return nil
	})
	if err != nil {
		return app.JournalEntryList{}, err
	}
	if len(all) == 0 {
		// Мир без записей по seq — прежний ответ списка шага.
		return respond[app.JournalEntryList](ctx, "journal.entry.list", nil, &m)
	}
	// Идентификаторы тел и фильтров — с префиксом прогона одинаково (applyRun).
	return app.PageEntries(all, f, p), nil
}

// Entry — запись журнала по seq (journal.entry.read).
func (a *Adapter) Entry(ctx context.Context, seq int64) (app.JournalEntryView, error) {
	return respond[app.JournalEntryView](ctx, "journal.entry.read", map[string]string{"seq": strconv.FormatInt(seq, 10)}, nil)
}

// Event — запись по event_id для окна записи (journal.event.read, интерфейс 6).
func (a *Adapter) Event(ctx context.Context, eventID string, m platform.Moment) (app.JournalEventView, error) {
	return respond[app.JournalEventView](ctx, "journal.event.read", map[string]string{"event_id": eventID}, &m)
}

// Head — голова журнала (journal.head.read).
func (a *Adapter) Head(ctx context.Context, m platform.Moment) (app.JournalHead, error) {
	return respond[app.JournalHead](ctx, "journal.head.read", nil, &m)
}

// Timeline — таймлайн под живой картой (journal.timeline.read, FR-4).
func (a *Adapter) Timeline(ctx context.Context, m platform.Moment) (app.TimelineData, error) {
	return respond[app.TimelineData](ctx, "journal.timeline.read", nil, &m)
}
