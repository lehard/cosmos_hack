package journal

import (
	"context"
	"strings"

	"ant/internal/application/platform"
	jc "ant/internal/contracts/journal"
)

// Service — реализация live ведущих портов модуля journal (AD-36): сценарии
// приложения над журналом. Без хранилища (NewService) — заглушка 501, как в
// волне 1; с хранилищем и сигналом (NewServiceWith) — живые обновления.
type Service struct {
	Unimplemented
	store  JournalStore
	signal Signal
}

// NewService создаёт реализацию live без хранилища (все операции — 501).
func NewService() *Service { return &Service{} }

// NewServiceWith создаёт реализацию live над журналом и сигналом «есть новое».
func NewServiceWith(store JournalStore, signal Signal) *Service {
	return &Service{store: store, signal: signal}
}

// subscribeBatch — сколько записей читается за один шаг подписки.
const subscribeBatch = 256

// Subscribe — живые обновления (journal.stream.subscribe, AD-21, FR-2):
// каждая запись основной цепочки после afterSeq в пределах прогона даёт
// изменение сущности своего потока. Сигнал несёт только seq (AD-6): данные
// подписка читает из журнала сама, поэтому после обрыва догоняет по seq.
func (s *Service) Subscribe(ctx context.Context, afterSeq int64, runID string) (Subscription, error) {
	if s.store == nil || s.signal == nil {
		return s.Unimplemented.Subscribe(ctx, afterSeq, runID)
	}
	return &subscription{s: s, cursor: afterSeq, runID: runID}, nil
}

type subscription struct {
	s       *Service
	cursor  int64
	runID   string
	pending []Change
}

func (sub *subscription) Next(ctx context.Context) (Change, error) {
	for len(sub.pending) == 0 {
		head := sub.s.signal.Head()
		entries, err := sub.s.store.Read(ctx, ReadQuery{AfterSeq: sub.cursor, Limit: subscribeBatch, RunID: sub.runID})
		if err != nil {
			return Change{}, err
		}
		for _, e := range entries {
			sub.cursor = int64(e.Seq)
			if ch, ok := ChangeOf(e); ok {
				sub.pending = append(sub.pending, ch)
			}
		}
		if len(entries) == 0 {
			// Ждём после головы, известной до чтения: запись, пришедшая между
			// чтением и ожиданием, не теряется.
			if _, err := sub.s.signal.Wait(ctx, max(head, sub.cursor)); err != nil {
				return Change{}, err
			}
		}
	}
	ch := sub.pending[0]
	sub.pending = sub.pending[1:]
	return ch, nil
}

func (sub *subscription) Close() {}

// ChangeOf — изменение сущности по записи журнала: вид сущности — вид потока
// записи (`item:‹id›` → item), если такой вид есть у SSE (EntityKinds);
// служебные потоки изменений не дают.
func ChangeOf(e jc.JournalEntry) (Change, bool) {
	kind, id, ok := strings.Cut(e.Stream, ":")
	if !ok || id == "" {
		return Change{}, false
	}
	for _, k := range platform.EntityKinds {
		if string(k) == kind {
			ch := Change{Entity: k, ID: id, Seq: int64(e.Seq), Mode: platform.ModeLive}
			if e.RunID != nil {
				ch.RunID = *e.RunID
			}
			return ch, true
		}
	}
	return Change{}, false
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)
