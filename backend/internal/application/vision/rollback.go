package vision

import (
	"context"
	"log/slog"
	"slices"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	dom "ant/internal/domain/vision"
)

// Имена потребителя и состояния правила автоотката.
const (
	// ConsumerRollback — глобальный потребитель журнала роли projector (AD-45):
	// правило автоотката версии анализатора (FR-101).
	ConsumerRollback = "vision.rollback"
	// StateWatch — состояние правила (окна контроля дрейфа, приостановки по
	// прогонам); одна строка — ключ StateWatchKey.
	StateWatch    = "vision.watch"
	StateWatchKey = "all"
)

// Rollback — правило автоотката версии анализатора (FR-101, AD-29; роль
// projector, одна копия-лидер, AD-45): по наблюдениям камеры, пропускам брака
// и отчётам проверки ведёт domain/vision.Watch и пишет реакции
// analyzer.passport.suspended («паспорт приостановлен», режим 2) в поток
// паспорта с run_id записи-триггера (прогон сценария не трогает живую работу
// и другие прогоны, AD-38). Состояние и курсор — одной транзакцией Append;
// повторный проход по тем же записям даёт те же id — дубль, не вторая запись.
// Задачу начальнику ОТК ставит notifications (ObjectReact) по самой записи.
type Rollback struct {
	Consumer appjournal.Consumer
	Codec    *engineapp.Codec
	Store    engineapp.ProjectionStore
	Log      *slog.Logger
}

// Run исполняет правило до отмены ctx.
func (r *Rollback) Run(ctx context.Context) error {
	return r.Consumer.Consume(ctx, ConsumerRollback, appjournal.Scope{Global: true}, r.Apply)
}

// load — состояние правила из проекции.
func (r *Rollback) load(ctx context.Context) (dom.Watch, error) {
	if r.Store == nil {
		return dom.UnmarshalState(nil)
	}
	raw, ok, err := r.Store.Get(ctx, StateWatch, StateWatchKey)
	if err != nil || !ok {
		if err != nil {
			return dom.Watch{}, err
		}
		return dom.UnmarshalState(nil)
	}
	return dom.UnmarshalState(raw)
}

// Apply — выход правила на пачку записей: приостановки, новое состояние,
// изменения для SSE (паспорт — сущность analyzer_passport).
func (r *Rollback) Apply(ctx context.Context, batch []jc.JournalEntry) (appjournal.AppendRequest, error) {
	var rq appjournal.AppendRequest
	relevant := false
	for _, e := range batch {
		if slices.Contains(dom.Watches, catalog.Type(e.EventType)) {
			relevant = true
			break
		}
	}
	if !relevant {
		return rq, nil
	}
	w, err := r.load(ctx)
	if err != nil {
		return rq, err
	}
	var changes []engineapp.Change
	for _, e := range batch {
		if !slices.Contains(dom.Watches, catalog.Type(e.EventType)) {
			continue
		}
		d, err := r.Codec.Decode(ctx, e)
		if err != nil {
			if r.Log != nil {
				r.Log.Error("vision: запись пропущена правилом автоотката", "seq", e.Seq, "type", e.EventType, "err", err)
			}
			continue
		}
		next, outs := w.Step(d.Record)
		w = next
		for _, re := range outs {
			id := re.ID(1)
			o := engineapp.ReactionOut(engine.Planned{Reaction: re, Change: engine.ChangeNew, Version: 1, EventID: id},
				"", d.Record.RunID, d.Record.CorrelationID, d.Record.EventID, "", d.Record.Seq)
			p, err := r.Codec.Encode(ctx, o)
			if err != nil {
				return rq, err
			}
			rq.Batch = append(rq.Batch, p)
			changes = append(changes, engineapp.Change{Entity: platform.EntityAnalyzerPassport, ID: re.Slot.Subject[len("analyzer_passport:"):],
				Seq: d.Record.Seq, RunID: d.Record.RunID, ReceivedAt: d.Record.ReceivedAt})
			if r.Log != nil {
				r.Log.Warn("vision: автооткат — паспорт приостановлен", "passport", re.Slot.Subject, "run", d.Record.RunID, "trigger_seq", d.Record.Seq)
			}
		}
	}
	raw, err := w.MarshalState()
	if err != nil {
		return rq, err
	}
	rq.Effects = append(rq.Effects, engineapp.ProjectionPut{Name: StateWatch, Key: StateWatchKey, Value: raw})
	if len(changes) > 0 {
		rq.Effects = append(rq.Effects, engineapp.Notify{Changes: changes})
	}
	return rq, nil
}
