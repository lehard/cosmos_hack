package mes

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/mes"
)

// Имена потребителя и состояния реакции «блок передаётся в MES».
const (
	// ConsumerHolds — потребитель журнала роли projector (охват global).
	ConsumerHolds = "mes.holds"
	// StateHolds — блок в MES по субъекту (изделие или партия).
	StateHolds = "mes.hold_state"
)

// HoldReactor — реакция модуля mes «блок передаётся в MES» (роль projector,
// одна копия-лидер, AD-45): по записям сдерживания изделия или партии
// формирует mes.hold.requested — блок или снятие (FR-93, AD-30). Записи —
// реакции (UUIDv5 слота), состояние блоков и курсор — одной транзакцией
// Append. Наружу ничего не отправляет: отправку делает Sender роли outbox;
// повторный проход по тем же записям ничего не добавляет.
type HoldReactor struct {
	Consumer appjournal.Consumer
	Codec    *engineapp.Codec
	Store    engineapp.ProjectionStore
	Log      *slog.Logger
}

// Run исполняет реакцию до отмены ctx.
func (r *HoldReactor) Run(ctx context.Context) error {
	return r.Consumer.Consume(ctx, ConsumerHolds, appjournal.Scope{Global: true}, r.Apply)
}

// Apply — выход реакции на пачку: записи mes.hold.requested и новое состояние блоков.
func (r *HoldReactor) Apply(ctx context.Context, batch []jc.JournalEntry) (appjournal.AppendRequest, error) {
	var rq appjournal.AppendRequest
	holds := map[string]dom.Hold{}
	for _, e := range batch {
		if !slices.Contains(dom.Triggers, catalog.Type(e.EventType)) {
			continue
		}
		d, err := r.Codec.Decode(ctx, e)
		if err != nil {
			if r.Log != nil {
				r.Log.Error("mes: запись пропущена", "seq", e.Seq, "err", err)
			}
			continue
		}
		subject, lot, err := dom.HoldSubject(d.Record)
		if err != nil || subject == "" {
			continue
		}
		sk := stateKey(subject, lot)
		h, ok := holds[sk]
		if !ok {
			if h, err = r.load(ctx, sk); err != nil {
				return rq, err
			}
			h.Subject, h.Lot = subject, lot
		}
		h, drafts, err := dom.PlanHold(h, d.Record)
		if err != nil {
			return rq, err
		}
		holds[sk] = h
		for _, dr := range drafts {
			rx, err := dom.HoldReaction(dr)
			if err != nil {
				return rq, err
			}
			meta := &engineapp.ReactionMeta{RuleID: rx.Slot.RuleID, AutomationMode: rx.AutomationMode,
				Slot:    engineapp.SlotMeta{RuleID: rx.Slot.RuleID, Subject: rx.Slot.Subject, TriggerKey: rx.Slot.TriggerKey},
				Version: 1, Causes: rx.Causes, BasisSeq: d.Record.Seq}
			pend, err := r.Codec.Encode(ctx, engineapp.Out{EventID: rx.ID(1), Type: catalog.MesHoldRequested, Kind: catalog.KindReaction,
				Stream: dom.Stream(dr.Key), RunID: d.Record.RunID, OccurredAt: rx.OccurredAt, Correlation: d.Record.CorrelationID,
				Causation: d.Record.EventID, BasisSeq: d.Record.Seq, Reaction: meta, Data: rx.Data})
			if err != nil {
				return rq, err
			}
			rq.Batch = append(rq.Batch, pend)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(holds)) {
		b, err := json.Marshal(holds[k])
		if err != nil {
			return rq, err
		}
		rq.Effects = append(rq.Effects, engineapp.ProjectionPut{Name: StateHolds, Key: k, Value: b})
	}
	return rq, nil
}

func stateKey(subject string, lot bool) string {
	if lot {
		return "lot:" + subject
	}
	return "item:" + subject
}

func (r *HoldReactor) load(ctx context.Context, key string) (dom.Hold, error) {
	var h dom.Hold
	if r.Store == nil {
		return h, nil
	}
	raw, ok, err := r.Store.Get(ctx, StateHolds, key)
	if err != nil || !ok {
		return h, err
	}
	if err := json.Unmarshal(raw, &h); err != nil {
		return h, fmt.Errorf("%s %s: %w", StateHolds, key, err)
	}
	return h, nil
}
