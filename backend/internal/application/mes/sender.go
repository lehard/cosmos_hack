package mes

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/mes"
)

// Sender — отправка блоков в MES (роль outbox, одна копия-лидер, AD-6,
// AD-18): неподтверждённые блоки из проекции mes.block уходят через порт
// Channel с номером = UUIDv5 бизнес-ключа; квитанция или ошибка данных —
// запись mes.hold.responded. Повтор — только при транспортных ошибках
// (FR-96); после предела блок остаётся неподтверждённым и помечается в
// журнале роли тревогой «блок в MES не доставлен» — изделие остаётся
// заблокированным в ant. При воспроизведении роль не запущена.
type Sender struct {
	Journal appjournal.JournalStore
	Codec   *engineapp.Codec
	Store   engineapp.ProjectionStore
	Channel Channel
	// Clock — доменные часы для occurred_at ответов (AD-37); nil — Now.
	Clock appjournal.DomainClock
	// Now — InfraClock: сроки попыток (AD-37).
	Now func() time.Time
	// Max — предел транспортных попыток на блок (0 — 8).
	Max int
	Log *slog.Logger

	mu    sync.Mutex
	tries map[string]int
}

// SendDue — одна попытка по каждому неподтверждённому блоку; возвращает
// число записанных ответов. Несовместимость контракта прерывает отправку
// (канал degraded — решает вызывающий).
func (s *Sender) SendDue(ctx context.Context, fence appjournal.Fence) (int, error) {
	s.mu.Lock()
	if s.tries == nil {
		s.tries = map[string]int{}
	}
	s.mu.Unlock()
	maxTries := s.Max
	if maxTries <= 0 {
		maxTries = 8
	}
	var keys []string
	raw, ok, err := s.Store.Get(ctx, ProjectionBlockIndex, "all")
	if err != nil {
		return 0, err
	}
	if ok {
		if err := json.Unmarshal(raw, &keys); err != nil {
			return 0, err
		}
	}
	n := 0
	for _, k := range keys {
		raw, ok, err := s.Store.Get(ctx, ProjectionBlock, k)
		if err != nil || !ok {
			if err != nil {
				return n, err
			}
			continue
		}
		var b dom.Block
		if err := json.Unmarshal(raw, &b); err != nil {
			return n, err
		}
		if !b.Pending() {
			continue
		}
		s.mu.Lock()
		tries := s.tries[k]
		s.mu.Unlock()
		if tries >= maxTries {
			continue
		}
		m := HoldMessage{MessageID: dom.MessageID(k), Key: k, Hold: b.Hold, ItemID: b.ItemID, LotID: b.LotID, OccurredAt: b.RequestedAt, Attempt: tries + 1}
		m.Reason = "Снятие блока по решению человека"
		if b.Hold {
			m.Reason = "Блок по сдерживанию: ожидается решение контролёра"
		}
		resp, err := s.Channel.Post(ctx, m)
		if err != nil {
			if _, ok := AsContract(err); ok {
				return n, err
			}
			if _, ok := AsTransport(err); ok {
				s.mu.Lock()
				s.tries[k] = tries + 1
				s.mu.Unlock()
				if tries+1 >= maxTries && s.Log != nil {
					s.Log.Error("MES: блок не доставлен — изделие остаётся заблокированным в ant", "business_key", k, "attempts", tries+1, "err", err)
				}
				continue
			}
			return n, err
		}
		if err := s.respond(ctx, fence, b, resp, b.Attempts); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// respond — ответ MES записью mes.hold.responded (факт роли outbox).
func (s *Sender) respond(ctx context.Context, fence appjournal.Fence, b dom.Block, r Response, generation int) error {
	d := ev.MesHoldRespondedV1{BusinessKey: b.Key, RequestEventID: ev.UUID(b.RequestEventID), Outcome: ev.MesHoldRespondedV1Outcome(r.Outcome)}
	if r.Outcome == string(ev.MesHoldRespondedV1OutcomeRejected) {
		code := r.Code
		if code == "" {
			code = "DATA_ERROR"
		}
		d.ErrorCode = &code
	}
	at := time.Now().UTC()
	if s.Now != nil {
		at = s.Now().UTC()
	}
	if s.Clock != nil {
		if t, err := s.Clock.Now(ctx); err == nil && !t.IsZero() {
			at = t.UTC()
		}
	}
	pend, err := s.Codec.Encode(ctx, engineapp.Out{EventID: dom.ResponseID(b.RequestEventID, generation), Type: catalog.MesHoldResponded,
		Kind: catalog.KindFact, Stream: dom.Stream(b.Key), OccurredAt: at, Causation: b.RequestEventID, Data: d})
	if err != nil {
		return err
	}
	_, err = s.Journal.Append(ctx, appjournal.AppendRequest{Fence: &fence, Batch: []appjournal.Pending{pend}})
	if errors.Is(err, appjournal.ErrDuplicate) {
		return nil
	}
	return err
}
