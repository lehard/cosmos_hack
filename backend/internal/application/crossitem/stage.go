package crossitem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/crossitem"
	"ant/internal/domain/kernel"
)

// Имена потребителя и проекции стадии (AD-42, AD-45).
const (
	// ConsumerStage — потребитель журнала роли crossitem (охват global).
	ConsumerStage = "crossitem"
	// StateProjection — состояние стадии (писатель crossitem): восстанавливается
	// при смене лидера, пишется в транзакции с курсором и адресованными записями.
	StateProjection = "crossitem.stage"
	stateKey        = "state"
	// CarrierProjection — реестр носителей стадии для приёма (AD-41): ключ
	// `‹тип›:‹значение›`, значение — интервалы действия носителя у изделий
	// (domain/crossitem.CarrierSpan). Пишется в той же транзакции, что и
	// состояние стадии.
	CarrierProjection = "crossitem.carrier"
)

// IsStageInput — запись — вход межизделийной стадии (AD-42): факты и команды
// без изделия; записи изделий с пометкой publish: stage; не собственные
// адресованные записи стадии (роль crossitem).
func IsStageInput(e jc.JournalEntry) bool {
	info, ok := catalog.Lookup(catalog.Type(e.EventType))
	if !ok || info.Role == ConsumerStage {
		return false
	}
	return info.PublishStage || e.ItemID == nil || *e.ItemID == ""
}

// StageRunner — роль crossitem (AD-6, AD-42): одна копия-лидер по аренде
// потребляет журнал, сворачивает вход стадии функцией domain/crossitem.Fold
// (подключённые функции модулей в фиксированном порядке + неподвижная точка)
// и пишет адресованные записи с occurred_at = наибольший среди причин и
// basis_seq; состояние стадии и курсор — в той же транзакции.
type StageRunner struct {
	Consumer appjournal.Consumer
	Codec    *engineapp.Codec
	Store    engineapp.ProjectionStore
	// Fold — шаг стадии; nil — crossitem.Fold.
	Fold func(crossitem.Stage, kernel.Record) (crossitem.Stage, []kernel.Addressed)
	Log  *slog.Logger
}

// Run исполняет стадию под арендой лидера fence до отмены ctx или ошибки
// (после ошибки лидер перезапускает Run — состояние перечитывается).
func (s *StageRunner) Run(ctx context.Context, fence appjournal.Fence) error {
	st, err := s.load(ctx)
	if err != nil {
		return err
	}
	// Эпик 35: состояние стадии — один ключ в сотни КБ; записи без изделия
	// (тики часов прогона) — вход стадии, но состояние обычно не меняют.
	// Неизменившееся состояние не переписывается: меньше записи в Postgres
	// на каждую запись журнала, смысл тот же.
	last, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return s.Consumer.Consume(ctx, ConsumerStage, appjournal.Scope{Global: true}, func(ctx context.Context, batch []jc.JournalEntry) (appjournal.AppendRequest, error) {
		next, rq, err := s.Apply(ctx, st, batch)
		if err != nil {
			return rq, err
		}
		st = next
		rq.Effects = slices.DeleteFunc(rq.Effects, func(e appjournal.Effect) bool {
			put, ok := e.(engineapp.ProjectionPut)
			if !ok || put.Name != StateProjection || put.Key != stateKey {
				return false
			}
			if bytes.Equal(put.Value, last) {
				return true
			}
			last = put.Value
			return false
		})
		rq.Fence = &fence
		return rq, nil
	})
}

func (s *StageRunner) load(ctx context.Context) (crossitem.Stage, error) {
	var st crossitem.Stage
	if s.Store == nil {
		return st, nil
	}
	raw, ok, err := s.Store.Get(ctx, StateProjection, stateKey)
	if err != nil || !ok {
		return st, err
	}
	if err := json.Unmarshal(raw, &st); err != nil {
		return st, fmt.Errorf("состояние стадии: %w", err)
	}
	return st, nil
}

// Apply — шаг стадии над пачкой: адресованные записи, новое состояние,
// изменения для SSE. Чистое относительно st: при ошибке Append состояние не
// сдвигается.
func (s *StageRunner) Apply(ctx context.Context, st crossitem.Stage, batch []jc.JournalEntry) (crossitem.Stage, appjournal.AppendRequest, error) {
	fold := s.Fold
	if fold == nil {
		fold = crossitem.Fold
	}
	var rq appjournal.AppendRequest
	var changes []engineapp.Change
	touched := false
	carrierItems := map[string]bool{}
	for _, e := range batch {
		if !IsStageInput(e) {
			continue
		}
		d, err := s.Codec.Decode(ctx, e)
		if err != nil {
			if e.ItemID == nil {
				if s.Log != nil {
					s.Log.Error("стадия: запись пропущена", "seq", e.Seq, "err", err)
				}
				continue
			}
			f, ferr := engineapp.FailureRequest(ctx, s.Codec, ConsumerStage, *e.ItemID, int64(e.Seq), e, err)
			if ferr != nil {
				return st, rq, ferr
			}
			rq.Batch = append(rq.Batch, f.Batch...)
			rq.Effects = append(rq.Effects, f.Effects...)
			continue
		}
		var out []kernel.Addressed
		st, out = fold(st, d.Record)
		touched = true
		switch d.Record.Type {
		case catalog.ItemCarrierApplied, catalog.ItemCarrierRemoved, catalog.ItemItemRegistered:
			carrierItems[d.Record.ItemID] = true
		}
		for _, a := range out {
			pend, err := s.Codec.Encode(ctx, addressedOut(a, d.Record))
			if err != nil {
				return st, rq, err
			}
			rq.Batch = append(rq.Batch, pend)
			if kind, id, ok := strings.Cut(a.Stream, ":"); ok {
				changes = append(changes, engineapp.Change{Entity: platform.EntityKind(kind), ID: id, Seq: d.Record.Seq,
					RunID: d.Record.RunID, ReceivedAt: d.Record.ReceivedAt})
			}
		}
	}
	if touched {
		raw, err := json.Marshal(st)
		if err != nil {
			return st, rq, err
		}
		rq.Effects = append(rq.Effects, engineapp.ProjectionPut{Name: StateProjection, Key: stateKey, Value: raw})
		carriers, err := carrierEffects(st.Own.Genealogy, carrierItems)
		if err != nil {
			return st, rq, err
		}
		rq.Effects = append(rq.Effects, carriers...)
	}
	if len(changes) > 0 {
		rq.Effects = append(rq.Effects, engineapp.Notify{Changes: changes})
	}
	return st, rq, nil
}

// addressedOut — запись адресованного выхода стадии (AD-42): id =
// AddressedID, occurred_at = наибольший среди причин, basis_seq — seq
// записи-основания, causation — она же.
func addressedOut(a kernel.Addressed, cause kernel.Record) engineapp.Out {
	info, _ := catalog.Lookup(a.Type)
	id := crossitem.AddressedID(a)
	o := engineapp.Out{
		EventID: id, Type: a.Type, Kind: info.Kind, Stream: a.Stream, RunID: cause.RunID,
		OccurredAt: a.OccurredAt, Correlation: cause.CorrelationID, Causation: cause.EventID, BasisSeq: cause.Seq, Data: a.Data,
	}
	if o.OccurredAt.IsZero() {
		o.OccurredAt = cause.OccurredAt
	}
	if item, ok := strings.CutPrefix(a.Stream, "item:"); ok {
		o.ItemID = item
	}
	if info.Kind == catalog.KindReaction {
		causes := a.Causes
		if causes == nil {
			causes = []string{}
		}
		rule := "crossitem." + string(a.Type)
		o.Reaction = &engineapp.ReactionMeta{
			RuleID: rule, AutomationMode: 1, Version: 1, Causes: causes, BasisSeq: cause.Seq,
			Slot: engineapp.SlotMeta{RuleID: rule, Subject: a.Stream, TriggerKey: a.Key},
		}
	}
	return o
}

// carrierEffects — строки реестра носителей для изделий, чьи носители
// изменились в пачке (AD-41).
func carrierEffects(g crossitem.Genealogy, items map[string]bool) ([]appjournal.Effect, error) {
	if len(items) == 0 {
		return nil, nil
	}
	var out []appjournal.Effect
	for _, key := range sortedKeys(g.Carriers) {
		spans := g.Carriers[key]
		if !slices.ContainsFunc(spans, func(sp crossitem.CarrierSpan) bool { return items[sp.ItemID] }) {
			continue
		}
		raw, err := json.Marshal(spans)
		if err != nil {
			return nil, err
		}
		out = append(out, engineapp.ProjectionPut{Name: CarrierProjection, Key: key, Value: raw})
	}
	return out, nil
}

// ProjectedCarriers — реестр носителей на момент для приёма (порт
// domain/crossitem.CarrierRegistry, AD-41): строки проекции crossitem.carrier,
// которые пишет стадия. Носителя в реестре ещё нет (стадия отстаёт) — пусто:
// событие идёт в поток стадии, и стадия разрешает его своим реестром в
// порядке журнала.
type ProjectedCarriers struct {
	Store engineapp.ProjectionStore
	// Timeout — предел чтения (0 — 2 с).
	Timeout time.Duration
}

var _ crossitem.CarrierRegistry = ProjectedCarriers{}

// Lookup — изделия, у которых носитель ref действовал в момент at.
func (p ProjectedCarriers) Lookup(ref crossitem.CarrierRef, at time.Time) []string {
	if p.Store == nil {
		return nil
	}
	t := p.Timeout
	if t == 0 {
		t = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), t)
	defer cancel()
	key := ref.Type + ":" + ref.Value
	raw, ok, err := p.Store.Get(ctx, CarrierProjection, key)
	if err != nil || !ok {
		return nil
	}
	var spans []crossitem.CarrierSpan
	if json.Unmarshal(raw, &spans) != nil {
		return nil
	}
	return crossitem.Genealogy{Carriers: map[string][]crossitem.CarrierSpan{key: spans}}.CarrierAt(key, at)
}
