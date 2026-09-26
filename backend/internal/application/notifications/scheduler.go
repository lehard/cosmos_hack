package notifications

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	dj "ant/internal/domain/journal"
	notif "ant/internal/domain/notifications"
)

// Роль scheduler (AD-4, AD-6, AD-25): одна копия-лидер по аренде
// `scheduler`. Планировщик читает только проекцию сроков и пишет «наступил
// срок» (obligation.due.reached, служебная запись в поток изделия) с id =
// UUIDv5(obligation_id, due_at) — повторный запуск не дублирует запись.
// Время приходит в домен только записью журнала: эскалацию и цену задержки
// вычисляет свёртка изделия по этой записи (occurred_at = срок).
//
// Там же — периодические проверки (FR-57): целостность основной цепочки
// журнала (звенья, AD-8), полнота приёма (разрывы source_seq — CheckLosses
// модуля ingest, AD-7) и задачи по решениям в потоках объектов (ObjectTasks).

// Scheduler — планировщик сроков.
type Scheduler struct {
	// Projections — проекция сроков (только чтение).
	Projections Projections
	// Codec — сборка записей движка (конверт, подпись ключом «движок», AD-10).
	Codec *engineapp.Codec
	// Clock — доменное «сейчас» (AD-37); по прогону — WithRun.
	Clock appjournal.DomainClock
	Log   *slog.Logger
	// BatchMax — записей в одной пачке Append (journal.batch_max); 0 — 64.
	BatchMax int
}

// TickReport — итог прохода планировщика.
type TickReport struct {
	// Due — сроков наступило к «сейчас».
	Due int
	// Written — записей «наступил срок» записано (повтор — 0).
	Written int
	// EventIDs — id записанных записей.
	EventIDs []string
}

// Tick — один проход (AD-4): действующие сроки, наступившие к доменному
// «сейчас» своего прогона и ещё не отмеченные, получают «наступил срок».
// Запись, уже сделанная прежним проходом (проекция ещё не догнала), не
// повторяется: Append отвергает её id как дубль.
func (s *Scheduler) Tick(ctx context.Context, fence *appjournal.Fence) (TickReport, error) {
	var rep TickReport
	rows, err := s.Projections.OpenObligations(ctx)
	if err != nil {
		return rep, err
	}
	slices.SortFunc(rows, func(a, b notif.ObligationRecord) int { return strings.Compare(a.ObligationID, b.ObligationID) })
	nows := map[string]time.Time{}
	var pend []appjournal.Pending
	for _, o := range rows {
		now, ok := nows[o.RunID]
		if !ok {
			if now, err = s.now(ctx, o.RunID); err != nil {
				return rep, err
			}
			nows[o.RunID] = now
		}
		if !o.Due(now) || o.ItemID == "" {
			continue
		}
		rep.Due++
		p, err := s.Codec.Encode(ctx, engineapp.Out{
			EventID: notif.ReachedID(o.ObligationID, o.DueAt), Type: catalog.ObligationDueReached, Kind: catalog.KindService,
			Stream: o.Subject, ItemID: o.ItemID, RunID: o.RunID, OccurredAt: o.DueAt,
			Data: notif.DueReachedData{ObligationID: o.ObligationID, DueAt: notif.FormatTime(o.DueAt)},
		})
		if err != nil {
			return rep, err
		}
		pend = append(pend, p)
	}
	k := s.BatchMax
	if k <= 0 {
		k = 64
	}
	for chunk := range slices.Chunk(pend, k) {
		ids, err := s.append(ctx, fence, chunk)
		rep.Written += len(ids)
		rep.EventIDs = append(rep.EventIDs, ids...)
		if err != nil {
			return rep, err
		}
	}
	return rep, nil
}

// append — пачка «наступил срок»; при дубле — по одной, дубли пропускаются.
func (s *Scheduler) append(ctx context.Context, fence *appjournal.Fence, batch []appjournal.Pending) ([]string, error) {
	_, err := s.Codec.Store.Append(ctx, appjournal.AppendRequest{Batch: batch, Fence: fence})
	if err == nil {
		ids := make([]string, 0, len(batch))
		for _, p := range batch {
			ids = append(ids, p.Entry.EventID)
		}
		return ids, nil
	}
	if !errors.Is(err, appjournal.ErrDuplicate) {
		return nil, err
	}
	var ids []string
	for _, p := range batch {
		_, err := s.Codec.Store.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}, Fence: fence})
		switch {
		case errors.Is(err, appjournal.ErrDuplicate):
			continue
		case err != nil:
			return ids, err
		}
		ids = append(ids, p.Entry.EventID)
	}
	return ids, nil
}

func (s *Scheduler) now(ctx context.Context, runID string) (time.Time, error) {
	if s.Clock == nil {
		return time.Now().UTC(), nil
	}
	if runID != "" {
		ctx = appjournal.WithRun(ctx, runID)
	}
	t, err := s.Clock.Now(ctx)
	return t.UTC(), err
}

// ObjectTasksConsumer — имя глобального потребителя задач по решениям в
// потоках объектов (AD-45: курсор атомарно с записанными задачами).
const ObjectTasksConsumer = "notifications.objects"

// ObjectTasks — обработчик потребителя ObjectTasksConsumer: решения в потоках
// объектов (инцидент: запрос измерения, назначенная мера) → задачи
// task.task.created в тот же поток (notif.ObjectReact, версия 1, id от слота).
// Задача, уже записанная прежде (повтор пачки), не дублируется.
func (s *Scheduler) ObjectTasks(ctx context.Context, batch []jc.JournalEntry) (appjournal.AppendRequest, error) {
	var rq appjournal.AppendRequest
	for _, e := range batch {
		if e.ItemID != nil || !objectTrigger(e.EventType) {
			continue
		}
		d, err := s.Codec.Decode(ctx, e)
		if err != nil {
			s.log().Error("задачи объектов: запись пропущена", "seq", e.Seq, "err", err)
			continue
		}
		for _, re := range notif.ObjectReact(d.Record) {
			id := re.ID(1)
			exists, err := s.written(ctx, re.Slot.Subject, string(re.Type), id)
			if err != nil {
				return rq, err
			}
			if exists {
				continue
			}
			out := engineapp.ReactionOut(engine.Planned{Reaction: re, Change: engine.ChangeNew, Version: 1, EventID: id},
				"", d.Record.RunID, d.Record.CorrelationID, d.Record.EventID, "", d.Record.Seq)
			p, err := s.Codec.Encode(ctx, out)
			if err != nil {
				return rq, err
			}
			rq.Batch = append(rq.Batch, p)
		}
	}
	return rq, nil
}

func objectTrigger(t string) bool {
	switch catalog.Type(t) {
	case catalog.IncidentIncidentOpened, catalog.IncidentCauseConcluded, catalog.IncidentIncidentClosed, // «Разобрать инцидент»
		catalog.IncidentMeasurementRequested, catalog.IncidentMeasurementRecorded, catalog.IncidentActionAssigned, catalog.AnalyzerPassportSuspended,
		catalog.IncidentSuggestionForwarded, // эпик 42: задача ответственному за предложение
		catalog.EquipmentDeviationDetected:  // режим вне уставки: мастеру и руководителю
		return true
	}
	return false
}

// written — запись задачи (поставлена или снята) с этим id уже есть в потоке.
func (s *Scheduler) written(ctx context.Context, stream, eventType, id string) (bool, error) {
	es, err := s.Codec.Store.Read(ctx, appjournal.ReadQuery{Stream: stream, EventType: eventType, Limit: 1000})
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(es, func(e jc.JournalEntry) bool { return e.EventID == id }), nil
}

func (s *Scheduler) log() *slog.Logger {
	if s.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return s.Log
}

// Check — периодическая проверка планировщика (FR-57: целостность, полнота).
type Check struct {
	Name  string
	Every time.Duration
	Run   func(ctx context.Context) error
}

// Loop — работа лидера: проход по срокам каждые every и периодические
// проверки по своему расписанию, до отмены ctx. Ошибка прохода или проверки
// — в лог, работа продолжается; потеря аренды (ErrFenced) — выход.
func (s *Scheduler) Loop(ctx context.Context, fence appjournal.Fence, every time.Duration, checks ...Check) error {
	if every <= 0 {
		every = 5 * time.Second
	}
	last := make([]time.Time, len(checks))
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		rep, err := s.Tick(ctx, &fence)
		switch {
		case errors.Is(err, appjournal.ErrFenced):
			return err
		case err != nil && ctx.Err() == nil:
			s.log().Error("планировщик: проход сроков", "err", err)
		case rep.Written > 0:
			s.log().Info("планировщик: наступил срок", "written", rep.Written, "due", rep.Due)
		}
		for i, c := range checks {
			if time.Since(last[i]) < c.Every {
				continue
			}
			last[i] = time.Now()
			if err := c.Run(ctx); err != nil && ctx.Err() == nil {
				s.log().Error("планировщик: проверка", "check", c.Name, "err", err)
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// Integrity — периодическая проверка целостности основной цепочки журнала
// (AD-8, FR-57): звенья записей после проверенного seq сходятся с формулой
// (dj.VerifyLinks — та же функция, что у хранителя и верификатора). Это
// быстрая проверка сервера; вердикт для проверяющих — отчёт верификатора
// у хранителя (AD-46, эпик 29).
type Integrity struct {
	Journal appjournal.JournalStore
	Log     *slog.Logger
	seq     int
	link    dj.Digest
	broken  error
}

// Check проверяет новые записи основной цепочки. Разрыв — ошибка (и далее
// та же ошибка: цепочка не «чинится» сама).
func (c *Integrity) Check(ctx context.Context) error {
	if c.broken != nil {
		return c.broken
	}
	for {
		es, err := c.Journal.Read(ctx, appjournal.ReadQuery{Chain: string(jc.JournalEntryChainMain), AfterSeq: int64(c.seq), Limit: 1000})
		if err != nil {
			return err
		}
		if len(es) == 0 {
			return nil
		}
		link, err := dj.VerifyLinks(c.link, c.seq, es)
		if err != nil {
			c.broken = fmt.Errorf("целостность журнала: %w", err)
			return c.broken
		}
		c.link, c.seq = link, es[len(es)-1].Seq
		if len(es) < 1000 {
			return nil
		}
	}
}
