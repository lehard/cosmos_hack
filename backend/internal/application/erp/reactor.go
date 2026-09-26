package erp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/erp"
	"ant/internal/domain/kernel"
)

// Имена потребителя и состояния реакции «учётное сообщение сформировано».
const (
	// ConsumerPostings — потребитель журнала роли projector (охват global),
	// формирующий erp.posting.requested (AD-45, AD-18).
	ConsumerPostings = "erp.postings"
	// StateBooks — учёт субъектов (изделие, партия) глазами erp: ключ — субъект.
	StateBooks = "erp.books"
	// StateSent — последняя версия сообщения по бизнес-ключу (AD-7).
	StateSent = "erp.sent"
)

// Reactor — реакция модуля erp «учётное сообщение сформировано» (роль
// projector, одна копия-лидер, AD-45): по событиям-сообщениям процесса «в
// 1С», решениям на закрывающих точках и по партиям формирует исходящие
// учётные сообщения с бизнес-ключом и версией (AD-7, AD-18). Записи — реакции
// (UUIDv5 слота и версии), состояние erp.books / erp.sent и курсор — одной
// транзакцией Append. Наружу ничего не отправляет: отправку делает роль
// outbox; при пересборке и воспроизведении реакция не запущена, а повторный
// проход по тем же записям ничего не добавляет (версия не меняется).
type Reactor struct {
	Consumer appjournal.Consumer
	Codec    *engineapp.Codec
	Store    engineapp.ProjectionStore
	Env      dom.Env
	Log      *slog.Logger
}

// Run исполняет реакцию до отмены ctx.
func (r *Reactor) Run(ctx context.Context) error {
	return r.Consumer.Consume(ctx, ConsumerPostings, appjournal.Scope{Global: true}, r.Apply)
}

type reactorCache struct {
	books   map[string]dom.Books
	sent    map[string]*dom.Sent
	touched map[string]bool
	sentHit map[string]bool
}

// Apply — выход реакции на пачку записей журнала: записи erp.posting.requested,
// новое состояние учёта и последних версий, изменения для SSE.
func (r *Reactor) Apply(ctx context.Context, batch []jc.JournalEntry) (appjournal.AppendRequest, error) {
	var rq appjournal.AppendRequest
	c := &reactorCache{books: map[string]dom.Books{}, sent: map[string]*dom.Sent{}, touched: map[string]bool{}, sentHit: map[string]bool{}}
	var changes []engineapp.Change
	for _, e := range batch {
		if !dom.Is(catalog.Type(e.EventType), dom.Triggers) {
			continue
		}
		d, err := r.Codec.Decode(ctx, e)
		if err == nil {
			var outs []engineapp.Out
			outs, err = r.step(ctx, c, d)
			for _, o := range outs {
				pend, perr := r.Codec.Encode(ctx, o)
				if perr != nil {
					return rq, perr
				}
				rq.Batch = append(rq.Batch, pend)
				changes = append(changes, engineapp.Change{Entity: platform.EntityErpMessage, ID: o.Stream[len("erp_message:"):],
					Seq: d.Record.Seq, RunID: d.Record.RunID, ReceivedAt: d.Record.ReceivedAt})
			}
		}
		if err == nil {
			continue
		}
		if e.ItemID == nil {
			if r.Log != nil {
				r.Log.Error("erp: запись пропущена", "seq", e.Seq, "type", e.EventType, "err", err)
			}
			continue
		}
		f, ferr := engineapp.FailureRequest(ctx, r.Codec, ConsumerPostings, *e.ItemID, int64(e.Seq), e, err)
		if ferr != nil {
			return rq, ferr
		}
		rq.Batch = append(rq.Batch, f.Batch...)
		rq.Effects = append(rq.Effects, f.Effects...)
	}
	for _, k := range slices.Sorted(maps.Keys(c.touched)) {
		b, err := json.Marshal(c.books[k])
		if err != nil {
			return rq, err
		}
		rq.Effects = append(rq.Effects, engineapp.ProjectionPut{Name: StateBooks, Key: k, Value: b})
	}
	for _, k := range slices.Sorted(maps.Keys(c.sentHit)) {
		b, err := json.Marshal(c.sent[k])
		if err != nil {
			return rq, err
		}
		rq.Effects = append(rq.Effects, engineapp.ProjectionPut{Name: StateSent, Key: k, Value: b})
	}
	if len(changes) > 0 {
		rq.Effects = append(rq.Effects, engineapp.Notify{Changes: changes})
	}
	return rq, nil
}

// step — одна запись-триггер: учёт субъекта, черновики, версии, реакции.
func (r *Reactor) step(ctx context.Context, c *reactorCache, d engineapp.Decoded) ([]engineapp.Out, error) {
	subject, _, err := dom.Subject(d.Record)
	if err != nil || subject == "" {
		return nil, err
	}
	b, ok := c.books[subject]
	if !ok {
		if b, err = r.loadBooks(ctx, subject); err != nil {
			return nil, err
		}
	}
	t := dom.Trigger{Record: d.Record}
	if d.Reaction != nil {
		t.Slot = kernel.Slot{RuleID: d.Reaction.Slot.RuleID, Subject: d.Reaction.Slot.Subject, TriggerKey: d.Reaction.Slot.TriggerKey}.Key()
	}
	drafts, nb, err := dom.Plan(r.Env, b, t)
	if err != nil {
		return nil, err
	}
	c.books[subject], c.touched[subject] = nb, true
	var outs []engineapp.Out
	for _, dr := range drafts {
		prev, err := r.sentOf(ctx, c, dr.Key)
		if err != nil {
			return nil, err
		}
		next, emit := dom.Version(prev, dr)
		if !emit {
			continue
		}
		rx, err := dom.Reaction(dr, next)
		if err != nil {
			return nil, err
		}
		id := rx.ID(next.Version)
		meta := &engineapp.ReactionMeta{RuleID: rx.Slot.RuleID, AutomationMode: rx.AutomationMode,
			Slot:    engineapp.SlotMeta{RuleID: rx.Slot.RuleID, Subject: rx.Slot.Subject, TriggerKey: rx.Slot.TriggerKey},
			Version: next.Version, Causes: rx.Causes, BasisSeq: d.Record.Seq}
		if prev != nil {
			s := prev.EventID
			meta.Supersedes = &s
		}
		next.EventID = id
		c.sent[dr.Key], c.sentHit[dr.Key] = &next, true
		outs = append(outs, engineapp.Out{
			EventID: id, Type: catalog.ErpPostingRequested, Kind: catalog.KindReaction, Stream: dom.Stream(dr.Key),
			RunID: d.Record.RunID, OccurredAt: rx.OccurredAt, Correlation: d.Record.CorrelationID, Causation: d.Record.EventID,
			BasisSeq: d.Record.Seq, Reaction: meta, Data: rx.Data,
		})
	}
	return outs, nil
}

func (r *Reactor) loadBooks(ctx context.Context, subject string) (dom.Books, error) {
	b := dom.Books{Subject: subject}
	if r.Store == nil {
		return b, nil
	}
	raw, ok, err := r.Store.Get(ctx, StateBooks, subject)
	if err != nil || !ok {
		return b, err
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		return b, fmt.Errorf("erp.books %s: %w", subject, err)
	}
	return b, nil
}

func (r *Reactor) sentOf(ctx context.Context, c *reactorCache, key string) (*dom.Sent, error) {
	if s, ok := c.sent[key]; ok {
		return s, nil
	}
	if r.Store == nil {
		return nil, nil
	}
	raw, ok, err := r.Store.Get(ctx, StateSent, key)
	if err != nil || !ok {
		return nil, err
	}
	var s dom.Sent
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("erp.sent %s: %w", key, err)
	}
	c.sent[key] = &s
	return &s, nil
}
