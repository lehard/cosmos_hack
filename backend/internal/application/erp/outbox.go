package erp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/erp"
)

// ConsumerOutbox — потребитель журнала роли outbox: очередь исходящих —
// проекция записей обмена (AD-18, AD-45).
const ConsumerOutbox = "erp.outbox"

// Состояния канала обмена (AD-18).
const (
	ChannelOK       = "ok"
	ChannelDegraded = "degraded"
	ChannelDisabled = "disabled"
)

// RetryPolicy — повторы только при транспортных ошибках (FR-96): задержки по
// номеру попытки (последняя — для всех следующих) и предел попыток, после
// которого сообщение уходит в карантин.
type RetryPolicy struct {
	Delays []time.Duration
	Max    int
}

// DefaultRetry — 1 с, 5 с, 30 с, 2 мин, 10 мин, далее 30 мин; 8 попыток
// (docs/integrations/1c.md, §8).
func DefaultRetry() RetryPolicy {
	return RetryPolicy{Delays: []time.Duration{time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute, 10 * time.Minute, 30 * time.Minute}, Max: 8}
}

// Delay — задержка после tries неудачных попыток.
func (p RetryPolicy) Delay(tries int) time.Duration {
	if len(p.Delays) == 0 {
		return time.Second
	}
	return p.Delays[min(max(tries-1, 0), len(p.Delays)-1)]
}

// Outbox — роль outbox (AD-6, AD-7, AD-18): одна копия-лидер отправляет
// исходящие учётные сообщения через порт учёта и пишет ответ событием.
//
//   - Очередь — проекция журнала: потребитель erp.outbox применяет записи
//     обмена (dom.View) к строкам очереди в той же транзакции, что и курсор.
//   - Ключ идемпотентности — бизнес-ключ; номер сообщения — UUIDv5 от ключа и
//     версии: повтор, переотправка и «потерянный ответ» не задваивают учёт.
//   - Повтор — только при транспортных ошибках, с растущей задержкой; после
//     предела и при ошибке данных — карантин исходящих (erp.posting.quarantined);
//     дальше — только ручная переотправка (erp.posting.resend).
//   - Новая версия содержимого после квитанции — карантин «ждёт решения»:
//     исправление (сторно + новое) — только по решению человека.
//   - Квитанция или ошибка — запись журнала erp.posting.responded (ось «учёт в 1С»).
//   - При старте и по расписанию — сверка ответной стороны: расхождение
//     контракта — канал degraded, отправки нет (FR-111, О8).
//   - Роль не запускается при воспроизведении и пересборке; отправитель
//     шлёт только поколения, по которым ответа в журнале ещё нет.
type Outbox struct {
	Journal  appjournal.JournalStore
	Consumer appjournal.Consumer
	Codec    *engineapp.Codec
	Store    OutboxStore
	Ledger   Ledger
	// Intake — вход входящих фактов порта учёта (nil — шлюз входящих выключен).
	Intake Intake
	// Clock — доменные часы для occurred_at ответов (AD-37); nil — Now.
	Clock appjournal.DomainClock
	// Now — InfraClock: повторы, сроки попыток, сверки (AD-37).
	Now   func() time.Time
	Retry RetryPolicy
	// Poll — период опроса очереди; Recheck — период сверки ответной
	// стороны при ok (при degraded — Poll×10); PullEvery — опрос входящих.
	Poll, Recheck, PullEvery time.Duration
	Log                      *slog.Logger

	once sync.Once
	wake chan struct{}
}

func (o *Outbox) init() {
	o.once.Do(func() {
		o.wake = make(chan struct{}, 1)
		if o.Now == nil {
			o.Now = time.Now
		}
		if o.Retry.Max == 0 {
			o.Retry = DefaultRetry()
		}
		if o.Poll <= 0 {
			o.Poll = time.Second
		}
		if o.Recheck <= 0 {
			o.Recheck = 5 * time.Minute
		}
		if o.PullEvery <= 0 {
			o.PullEvery = 30 * time.Second
		}
		if o.Log == nil {
			o.Log = slog.New(slog.DiscardHandler)
		}
	})
}

func (o *Outbox) system() string { return o.Ledger.Info().System }

// Run исполняет роль под арендой лидера fence до отмены ctx: сверка канала,
// потребитель очереди, отправитель, шлюз входящих.
func (o *Outbox) Run(ctx context.Context, fence appjournal.Fence) error {
	o.init()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errc := make(chan error, 4)
	var wg sync.WaitGroup
	wg.Go(func() {
		if err := o.Consumer.Consume(ctx, ConsumerOutbox, appjournal.Scope{Global: true}, o.Apply); err != nil && ctx.Err() == nil {
			errc <- fmt.Errorf("очередь исходящих: %w", err)
			cancel()
		}
	})
	wg.Go(func() {
		if err := o.sendLoop(ctx, fence); err != nil && ctx.Err() == nil {
			errc <- err
			cancel()
		}
	})
	if o.Intake != nil {
		wg.Go(func() { o.pullLoop(ctx) })
	}
	wg.Wait()
	close(errc)
	return <-errc
}

// Apply — выход потребителя очереди на пачку: строки очереди по записям
// обмена своей системы (эффект OutboxView в транзакции курсора).
func (o *Outbox) Apply(ctx context.Context, batch []jc.JournalEntry) (appjournal.AppendRequest, error) {
	o.init()
	var rq appjournal.AppendRequest
	rows := map[string]*OutboxRow{}
	var order []string
	sys := o.system()
	for _, e := range batch {
		if !dom.Is(catalog.Type(e.EventType), dom.Exchange) {
			continue
		}
		d, err := o.Codec.Decode(ctx, e)
		if err != nil {
			o.Log.Error("очередь исходящих: запись пропущена", "seq", e.Seq, "err", err)
			continue
		}
		key, err := dom.Key(d.Record)
		if err != nil || key == "" {
			continue
		}
		row, ok := rows[key]
		if !ok {
			r, found, err := o.Store.Row(ctx, key)
			if err != nil {
				return rq, err
			}
			if !found && d.Record.Type != catalog.ErpPostingRequested {
				continue
			}
			row = &r
			rows[key] = row
			order = append(order, key)
		}
		v, err := row.View.Apply(d.Record)
		if err != nil {
			return rq, err
		}
		if v.System != sys {
			delete(rows, key)
			continue
		}
		row.View, row.Token = v, TokenOf(v)
	}
	for _, k := range order {
		if r, ok := rows[k]; ok {
			rq.Effects = append(rq.Effects, OutboxView{View: r.View, Token: r.Token})
		}
	}
	if len(rq.Effects) > 0 {
		o.kick()
	}
	return rq, nil
}

func (o *Outbox) kick() {
	select {
	case o.wake <- struct{}{}:
	default:
	}
}

// sendLoop — сверка канала и отправка по сроку или сигналу потребителя.
func (o *Outbox) sendLoop(ctx context.Context, fence appjournal.Fence) error {
	ch := o.CheckChannel(ctx)
	next := o.Now().Add(o.recheck(ch))
	t := time.NewTicker(o.Poll)
	defer t.Stop()
	for {
		if o.Now().After(next) {
			ch = o.CheckChannel(ctx)
			next = o.Now().Add(o.recheck(ch))
		}
		if ch.State == ChannelOK {
			if _, err := o.SendDue(ctx, fence); err != nil {
				if errors.Is(err, appjournal.ErrFenced) || ctx.Err() != nil {
					return err
				}
				o.Log.Error("исходящие: отправка", "err", err)
			}
			// Несовместимость контракта, найденная при отправке, меняет канал.
			if c, ok, err := o.Store.Channel(ctx, o.system()); err == nil && ok {
				ch = c
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		case <-o.wake:
		}
	}
}

func (o *Outbox) recheck(ch Channel) time.Duration {
	if ch.State == ChannelOK {
		return o.Recheck
	}
	return 10 * o.Poll
}

// CheckChannel — сверка ответной стороны (AD-18): метаданные и версия
// контракта; итог — состояние канала в хранилище.
func (o *Outbox) CheckChannel(ctx context.Context) Channel {
	o.init()
	info := o.Ledger.Info()
	ch := Channel{System: info.System, Endpoint: info.Endpoint, Stand: info.Stand, ContractVersion: info.ContractVersion, CheckedAt: o.Now().UTC()}
	if prev, ok, err := o.Store.Channel(ctx, info.System); err == nil && ok {
		ch.LastExchangeAt = prev.LastExchangeAt
	}
	c, err := o.Ledger.Check(ctx)
	switch ce, isContract := AsContract(err); {
	case err == nil:
		ch.State, ch.Detail = ChannelOK, c.Detail
	case isContract:
		ch.State, ch.Detail = ChannelDegraded, ce.Detail
		o.Log.Warn("канал обмена degraded: контракт не совпал", "system", info.System, "detail", ce.Detail)
	default:
		ch.State, ch.Detail = ChannelDegraded, "ответная сторона недоступна, сверка повторится: "+err.Error()
		o.Log.Warn("канал обмена: сверка не удалась", "system", info.System, "err", err)
	}
	if err := o.Store.SetChannel(ctx, ch); err != nil {
		o.Log.Error("канал обмена: состояние не сохранено", "err", err)
	}
	return ch
}

// SendDue — отправить сообщения, срок которых наступил; возвращает число
// обработанных.
func (o *Outbox) SendDue(ctx context.Context, fence appjournal.Fence) (int, error) {
	o.init()
	rows, err := o.Store.Due(ctx, o.system(), o.Now(), 16)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, r := range rows {
		if err := o.Send(ctx, fence, r); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// Send — одна попытка по строке очереди и её итог в журнал.
func (o *Outbox) Send(ctx context.Context, fence appjournal.Fence, row OutboxRow) error {
	o.init()
	v := row.View
	tries := 0
	if row.RetryToken == row.Token {
		tries = row.Tries
	}
	if v.NeedsDecision() {
		return o.quarantine(ctx, fence, row, tries, ev.ErpPostingQuarantinedV1CauseCorrectionPending, string(errcodes.ErpCorrectionPending),
			"содержимое изменилось после подтверждения учётной системой — нужно решение: исправить (сторно + новое) или оставить как отправлено")
	}
	out := Outgoing{MessageID: v.MessageID, Version: v.Version, Request: v.Request, OccurredAt: v.RequestedAt, Attempt: tries + 1}
	if v.Correction && v.AckedVersion > 0 {
		out.CorrectsMessageID = dom.MessageID(v.Key, v.AckedVersion)
	}
	resp, err := o.Ledger.Post(ctx, out)
	if err == nil {
		return o.respond(ctx, fence, row, tries+1, resp)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if te, ok := AsTransport(err); ok {
		tries++
		if tries >= o.Retry.Max {
			return o.quarantine(ctx, fence, row, tries, ev.ErpPostingQuarantinedV1CauseTransportExhausted, string(errcodes.ErpUnavailable),
				fmt.Sprintf("повторы исчерпаны (%d попыток): %v", tries, te))
		}
		o.Log.Info("исходящие: транспортная ошибка — повтор с тем же номером", "business_key", v.Key, "attempt", tries, "err", te)
		_, err := o.Journal.Append(ctx, appjournal.AppendRequest{Fence: &fence, Effects: []appjournal.Effect{OutboxTransport{
			Key: v.Key, Handled: row.Handled, RetryToken: row.Token, Tries: tries, NextAt: o.Now().Add(o.Retry.Delay(tries)), LastError: te.Error()}}})
		return err
	}
	if ce, ok := AsContract(err); ok {
		if ce.Local {
			// Ошибка интеграции поймана до отправки (FR-111): сообщение не по
			// схеме контракта — в учётную систему ничего не ушло.
			return o.quarantine(ctx, fence, row, tries, ev.ErpPostingQuarantinedV1CauseContractIncompatible, string(errcodes.ErpContractIncompatible), ce.Detail)
		}
		ch := Channel{System: o.system(), State: ChannelDegraded, Detail: ce.Detail, CheckedAt: o.Now().UTC()}
		info := o.Ledger.Info()
		ch.Endpoint, ch.Stand, ch.ContractVersion = info.Endpoint, info.Stand, info.ContractVersion
		return o.Store.SetChannel(ctx, ch)
	}
	return err
}

// respond — ответ учётной системы: запись erp.posting.responded (и карантин
// при ошибке данных) и отметка поколения — одной транзакцией.
func (o *Outbox) respond(ctx context.Context, fence appjournal.Fence, row OutboxRow, attempt int, resp Response) error {
	v := row.View
	d := ev.ErpPostingRespondedV1{BusinessKey: v.Key, RequestEventID: ev.UUID(v.RequestEventID), Outcome: resp.Outcome}
	act := ev.ErpPostingRespondedV1Action(v.Action)
	d.Action = &act
	mid := ev.UUID(v.MessageID)
	d.MessageID = &mid
	d.ItemID, d.LotID = v.Request.ItemID, v.Request.LotID
	d.Attempt = &attempt
	d.Receipt, d.ExternalDocumentRef = strp(resp.Receipt), strp(resp.DocumentRef)
	if resp.HTTPStatus > 0 {
		hs := resp.HTTPStatus
		d.HTTPStatus = &hs
	}
	if resp.Outcome == ev.ErpPostingRespondedV1OutcomeRejected {
		code := resp.ErrorCode
		if code == "" {
			code = string(errcodes.ErpDataError)
		}
		d.ErrorCode = &code
		msg := resp.Message
		if resp.Code != "" {
			msg = resp.Code + ": " + msg
		}
		if resp.Field != "" {
			msg += " (поле " + resp.Field + ")"
		}
		d.ErrorMessage = strp(msg)
	} else if ax, ok := dom.Axis(v.Action); ok {
		d.ResultingStatus = &ax
	}
	at, err := o.occurred(ctx)
	if err != nil {
		return err
	}
	pend, err := o.Codec.Encode(ctx, engineapp.Out{EventID: dom.ResponseID(v.RequestEventID, v.Generation), Type: catalog.ErpPostingResponded,
		Kind: catalog.KindFact, Stream: dom.Stream(v.Key), RunID: v.RunID, OccurredAt: at, Causation: v.RequestEventID, Data: d})
	if err != nil {
		return err
	}
	rq := appjournal.AppendRequest{Fence: &fence, Batch: []appjournal.Pending{pend}}
	if resp.Outcome == ev.ErpPostingRespondedV1OutcomeRejected {
		// Ошибка данных — без автоповтора: карантин исходящих, задача администратору.
		q, err := o.quarantineOut(ctx, row, attempt, ev.ErpPostingQuarantinedV1CauseDataError, *d.ErrorCode, *d.ErrorMessage, at)
		if err != nil {
			return err
		}
		rq.Batch = append(rq.Batch, q)
	}
	rq.Effects = append(rq.Effects, OutboxTransport{Key: v.Key, Handled: row.Token, RetryToken: row.Token, Tries: attempt, NextAt: o.Now()})
	if _, err := o.Journal.Append(ctx, rq); err != nil && !errors.Is(err, appjournal.ErrDuplicate) {
		return err
	}
	o.touch(ctx)
	return nil
}

// quarantine — карантин исходящих без ответа учётной системы.
func (o *Outbox) quarantine(ctx context.Context, fence appjournal.Fence, row OutboxRow, tries int, cause ev.ErpPostingQuarantinedV1Cause, code, msg string) error {
	at, err := o.occurred(ctx)
	if err != nil {
		return err
	}
	q, err := o.quarantineOut(ctx, row, tries, cause, code, msg, at)
	if err != nil {
		return err
	}
	o.Log.Warn("исходящие: карантин", "business_key", row.View.Key, "cause", cause, "code", code)
	_, err = o.Journal.Append(ctx, appjournal.AppendRequest{Fence: &fence, Batch: []appjournal.Pending{q},
		Effects: []appjournal.Effect{OutboxTransport{Key: row.View.Key, Handled: row.Token, RetryToken: row.Token, Tries: tries, NextAt: o.Now(), LastError: msg}}})
	if errors.Is(err, appjournal.ErrDuplicate) {
		return nil
	}
	return err
}

func (o *Outbox) quarantineOut(ctx context.Context, row OutboxRow, tries int, cause ev.ErpPostingQuarantinedV1Cause, code, msg string, at time.Time) (appjournal.Pending, error) {
	v := row.View
	d := ev.ErpPostingQuarantinedV1{BusinessKey: v.Key, RequestEventID: ev.UUID(v.RequestEventID), Attempts: tries,
		Cause: &cause, LastErrorCode: strp(code), ErrorMessage: strp(trim(msg, 4000)), ItemID: v.Request.ItemID, LotID: v.Request.LotID}
	ver := v.Version
	d.MessageVersion = &ver
	return o.Codec.Encode(ctx, engineapp.Out{EventID: dom.QuarantineID(v.RequestEventID, v.Generation), Type: catalog.ErpPostingQuarantined,
		Kind: catalog.KindService, Stream: dom.Stream(v.Key), RunID: v.RunID, OccurredAt: at, Causation: v.RequestEventID, Data: d})
}

// occurred — occurred_at ответа: доменное «сейчас» (в сценарии — виртуальное, AD-37).
func (o *Outbox) occurred(ctx context.Context) (time.Time, error) {
	if o.Clock != nil {
		t, err := o.Clock.Now(ctx)
		if err == nil && !t.IsZero() {
			return t.UTC(), nil
		}
	}
	return o.Now().UTC(), nil
}

// touch — время последнего обмена канала.
func (o *Outbox) touch(ctx context.Context) {
	ch, ok, err := o.Store.Channel(ctx, o.system())
	if err != nil || !ok {
		return
	}
	now := o.Now().UTC()
	ch.LastExchangeAt = &now
	_ = o.Store.SetChannel(ctx, ch)
}

func strp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ErrNotQuarantined — команда над сообщением, которое не ждёт решения.
func errNotQuarantined(key string, st dom.Status) error {
	e := platform.Fail(errcodes.ErpNotQuarantined, "business_key", key, "status", string(st))
	e.Detail = "Учётное сообщение " + key + " в состоянии «" + string(st) + "» — переотправка и исправление только из карантина"
	return e
}
