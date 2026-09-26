package ops

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/ops"
)

// Reporter — порт ops для других модулей (AD-40: тип записи эмитирует
// только модуль-владелец семейства): служебные записи ops.processing.failed
// (потребитель журнала не смог обработать запись, AD-45) и
// ops.integration.degraded (канал обмена с внешней системой перешёл в
// degraded или вернулся, AD-18). Модули erp, mes, engine и crossitem не
// собирают эти записи сами — они сообщают сюда.
type Reporter struct {
	Journal appjournal.JournalStore
	Codec   *engineapp.Codec
	// Clock — доменное «сейчас» для occurred_at (AD-37); nil — Now.
	Clock appjournal.DomainClock
	// Now — InfraClock; nil — time.Now.
	Now func() time.Time
	Log *slog.Logger
}

// FailureReport — ошибка потребителя на записи изделия (AD-45).
type FailureReport struct {
	// Consumer — имя курсора потребителя (engine.worker, projector:‹проекция›, crossitem…).
	Consumer string
	ItemID   string
	// Seq — запись, на которой случилась ошибка.
	Seq int64
	// Trigger — запись журнала (время, прогон, корреляция — от неё).
	Trigger jc.JournalEntry
	Cause   error
}

// ReportFailure — служебная запись ops.processing.failed для транзакции
// потребителя (AD-45: курсор и отметка «обработка остановлена» — одним
// Append): возвращает запрос, который потребитель дополняет курсором и
// записывает сам. Id записи — dom.FailureEventID: повтор после сбоя
// процесса не создаёт второго смысла.
func (r *Reporter) ReportFailure(ctx context.Context, f FailureReport) (appjournal.AppendRequest, error) {
	if f.Cause == nil {
		f.Cause = errors.New("причина не указана")
	}
	r.log().Error("обработка остановлена", "module", "ops", "item_id", f.ItemID, "consumer", f.Consumer, "seq", f.Seq,
		"event_id", f.Trigger.EventID, "correlation_id", f.Trigger.CorrelationID, "run_id", deref(f.Trigger.RunID), "err", f.Cause)
	return engineapp.FailureRequest(ctx, r.Codec, f.Consumer, f.ItemID, f.Seq, f.Trigger, f.Cause)
}

// IntegrationReport — наблюдённое состояние канала обмена (AD-18).
type IntegrationReport struct {
	// System — внешняя система (onec, galaktika, mes, kompas, skud, ca, partner).
	System string
	// State — ok | degraded (disabled не журналируется).
	State  string
	Detail string
}

// integrationSystems — системы, которые знает схема ops.integration.degraded.
var integrationSystems = []string{
	string(ev.OpsIntegrationDegradedV1SystemOnec), string(ev.OpsIntegrationDegradedV1SystemGalaktika),
	string(ev.OpsIntegrationDegradedV1SystemMes), string(ev.OpsIntegrationDegradedV1SystemKompas),
	string(ev.OpsIntegrationDegradedV1SystemSkud), string(ev.OpsIntegrationDegradedV1SystemCa),
	string(ev.OpsIntegrationDegradedV1SystemPartner),
}

// ReportIntegration — ops.integration.degraded по просьбе модуля
// интеграции (эпик 30): пишется только переход (ok → degraded, degraded →
// ok; первое «ok» не пишется, dom.ShouldRecord). Возвращает, записан ли
// переход. Две копии, увидевшие один переход, дают одну запись (id от seq
// предыдущей записи системы).
func (r *Reporter) ReportIntegration(ctx context.Context, rep IntegrationReport) (bool, error) {
	if !slices.Contains(integrationSystems, rep.System) {
		return false, fmt.Errorf("ops.integration.degraded: система %q не из контракта", rep.System)
	}
	stream := "source:" + rep.System
	prev, err := last(ctx, r.Journal, r.Codec, catalog.OpsIntegrationDegraded, stream)
	if err != nil {
		return false, err
	}
	var lastRec *dom.IntegrationRecord
	var after int64
	if prev != nil {
		rec, err := integrationRecord(*prev)
		if err != nil {
			return false, err
		}
		lastRec, after = &rec, rec.Seq
	}
	if !dom.ShouldRecord(lastRec, rep.State) {
		return false, nil
	}
	data := ev.OpsIntegrationDegradedV1{System: ev.OpsIntegrationDegradedV1System(rep.System), State: ev.OpsIntegrationDegradedV1State(rep.State)}
	if rep.Detail != "" {
		d := dom.FailureMessage(rep.Detail)
		data.Detail = &d
	}
	id := dom.DegradedEventID(rep.System, rep.State, after)
	pend, err := r.Codec.Encode(ctx, engineapp.Out{EventID: id, Type: catalog.OpsIntegrationDegraded, Kind: catalog.KindService,
		Stream: stream, OccurredAt: r.now(ctx), RunID: appjournal.RunFrom(ctx), Data: data})
	if err != nil {
		return false, err
	}
	if _, err := r.Journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{pend}}); err != nil {
		if errors.Is(err, appjournal.ErrDuplicate) {
			return false, nil
		}
		return false, err
	}
	lvl := slog.LevelWarn
	if rep.State == dom.IntegrationOK {
		lvl = slog.LevelInfo
	}
	r.log().Log(ctx, lvl, "интеграция: смена состояния канала", "module", "ops", "event_id", id, "system", rep.System, "state", rep.State)
	return true, nil
}

func (r *Reporter) now(ctx context.Context) time.Time {
	if r.Clock != nil {
		if t, err := r.Clock.Now(ctx); err == nil && !t.IsZero() {
			return t.UTC()
		}
	}
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

func (r *Reporter) log() *slog.Logger {
	if r.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return r.Log
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	return t.UTC(), err
}
