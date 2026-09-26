package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	dops "ant/internal/domain/ops"
)

// RoleWorker — роль-эмитент записей воркера в каталоге (AD-40): реакции
// свёртки и служебные записи воркера; триггером свёртки они не являются (AD-5).
// Курсор воркера в consumer_offsets — journal.WorkerConsumer, аренда
// партиции — journal.PartitionLease (их же читает WorkFeed).
const RoleWorker = "worker"

// Метрики воркера и живых обновлений (AD-6, FR-2).
const (
	// MetricEventToSSE — «событие → экран»: от received_at записи-триггера до
	// отправки сообщения SSE; бюджет ≤ 2 с (FR-2, AD-6).
	MetricEventToSSE = "ant_event_to_sse_seconds"
	// MetricFold — длительность пересвёртки изделия (бюджет ≤ 500 мс, AD-6).
	MetricFold = "ant_fold_seconds"
	// MetricProcessingFailed — изделия с «обработка остановлена» (AD-45).
	MetricProcessingFailed = "ant_processing_failed_total"
)

// WorkerConfig — зависимости сценария воркера.
type WorkerConfig struct {
	Feed  WorkFeed
	Codec *Codec
	// Fold — свёртка изделия; nil — domain/engine.Fold.
	Fold engine.Folder
	// Bundles — нормативный слой; nil — EmptyBundles.
	Bundles BundleSource
	// Projections — проекции изделия и вклады; nil — NewRegistry().
	Projections *Registry
	Telemetry   platform.Telemetry
	Log         *slog.Logger
	// Now — InfraClock для метрик и пауз (AD-37); nil — time.Now.
	Now func() time.Time
	// Refresh — как часто перечитывать список арендованных партиций.
	Refresh time.Duration
	// Backoff — пауза после ошибки инфраструктуры.
	Backoff time.Duration
}

// WorkerService — сценарий воркера (AD-5, AD-6, AD-40, AD-45).
type WorkerService struct {
	cfg WorkerConfig
}

var _ Worker = (*WorkerService)(nil)

// NewWorker создаёт воркер с умолчаниями.
func NewWorker(cfg WorkerConfig) *WorkerService {
	if cfg.Fold == nil {
		cfg.Fold = engine.Fold
	}
	if cfg.Bundles == nil {
		cfg.Bundles = EmptyBundles{}
	}
	if cfg.Projections == nil {
		cfg.Projections = NewRegistry()
	}
	if cfg.Telemetry == nil {
		cfg.Telemetry = NopTelemetry{}
	}
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.DiscardHandler)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Refresh <= 0 {
		cfg.Refresh = 5 * time.Second
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = time.Second
	}
	return &WorkerService{cfg: cfg}
}

// Run обрабатывает арендованные партиции до отмены ctx: по горутине на
// партицию; список партиций перечитывается каждые Refresh (перераспределение
// при изменении N, AD-6).
func (w *WorkerService) Run(ctx context.Context) error {
	type running struct {
		epoch  int64
		cancel context.CancelFunc
	}
	active := map[int]running{}
	var wg sync.WaitGroup
	defer func() {
		for _, r := range active {
			r.cancel()
		}
		wg.Wait()
	}()
	t := time.NewTicker(w.cfg.Refresh)
	defer t.Stop()
	for {
		parts, err := w.cfg.Feed.Partitions(ctx)
		if err != nil && ctx.Err() == nil {
			w.cfg.Log.Warn("воркер: список партиций", "err", err)
		}
		seen := map[int]bool{}
		for _, p := range parts {
			seen[p.Number] = true
			if r, ok := active[p.Number]; ok && r.epoch == p.Epoch {
				continue
			} else if ok {
				r.cancel()
			}
			pctx, cancel := context.WithCancel(ctx)
			active[p.Number] = running{epoch: p.Epoch, cancel: cancel}
			wg.Go(func() { w.runPartition(pctx, p) })
		}
		for n, r := range active {
			if !seen[n] {
				r.cancel()
				delete(active, n)
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

func (w *WorkerService) runPartition(ctx context.Context, p Partition) {
	log := w.cfg.Log.With("partition", p.Number, "epoch", p.Epoch)
	for ctx.Err() == nil {
		works, err := w.cfg.Feed.Next(ctx, p)
		if err == nil {
			err = w.Process(ctx, p, works)
		}
		switch {
		case err == nil:
			continue
		case ctx.Err() != nil:
			return
		case errors.Is(err, appjournal.ErrFenced):
			// Аренда потеряна: запись от копии без аренды невозможна (AD-6).
			log.Warn("воркер: аренда партиции потеряна", "err", err)
			return
		default:
			log.Error("воркер: ошибка инфраструктуры, повтор", "err", err)
			pause(ctx, w.cfg.Backoff)
		}
	}
}

// pause ждёт d или отмены ctx (application, не домен: часы здесь допустимы).
func pause(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// ProcessingError — ошибка обработки записи изделия (декодирование,
// свёртка, сравнение, проекция): изделие «обработка остановлена», партиция
// продолжает (AD-45). Прочие ошибки — инфраструктура: повтор всей пачки.
type ProcessingError struct {
	ItemID string
	Seq    int64
	Err    error
}

func (e *ProcessingError) Error() string {
	return fmt.Sprintf("обработка изделия %s на seq %d: %v", e.ItemID, e.Seq, e.Err)
}

func (e *ProcessingError) Unwrap() error { return e.Err }

// Process обрабатывает пачку работ партиции: по каждому изделию —
// пересвёртка целиком и Append разницы с basis_seq, проекциями и вкладами;
// курсор партиции сдвигается в транзакции последней записи пачки (AD-45).
// Повтор пачки после сбоя безопасен: пересвёртка идемпотентна (AD-5).
func (w *WorkerService) Process(ctx context.Context, p Partition, works []Work) error {
	if len(works) == 0 {
		return nil
	}
	works = slices.Clone(works)
	slices.SortStableFunc(works, func(a, b Work) int {
		switch {
		case a.UpToSeq < b.UpToSeq:
			return -1
		case a.UpToSeq > b.UpToSeq:
			return 1
		}
		return 0
	})
	cursor := works[len(works)-1].UpToSeq
	fence := &appjournal.Fence{Lease: appjournal.PartitionLease(p.Number), Epoch: p.Epoch}
	for i, wk := range works {
		rq, err := w.planItem(ctx, wk)
		var perr *ProcessingError
		switch {
		case errors.As(err, &perr):
			w.cfg.Log.Error("обработка остановлена", "item_id", wk.ItemID, "seq", wk.UpToSeq, "event_id", wk.Trigger.EventID,
				"correlation_id", wk.Trigger.CorrelationID, "run_id", runOf(wk.Trigger), "err", perr.Err)
			w.cfg.Telemetry.Counter(MetricProcessingFailed, 1)
			if rq, err = w.failure(ctx, wk, perr); err != nil {
				return err
			}
		case err != nil:
			return err
		}
		rq.Fence = fence
		if i == len(works)-1 {
			rq.Consumer = &appjournal.CursorAdvance{Name: appjournal.WorkerConsumer, Partition: p.Number, Seq: cursor}
		}
		if len(rq.Batch) == 0 && len(rq.Effects) == 0 && rq.Consumer == nil {
			continue
		}
		if _, err := w.cfg.Codec.Store.Append(ctx, rq); err != nil {
			return err
		}
	}
	return nil
}

// ItemInput — поток изделия, разобранный для свёртки (AD-3, AD-5, AD-45).
type ItemInput struct {
	ItemID string
	// Input — вход свёртки: всё, кроме записей роли worker (реакции изделия
	// в свёртку не входят, AD-3).
	Input []kernel.Record
	// Recorded — записанные версии слотов реакций изделия.
	Recorded []engine.Recorded
	// Stopped — последняя ops.processing.failed не снята ops.processing.retried (AD-45).
	Stopped bool
	// Last — последняя по seq запись входа.
	Last kernel.Record
	// FailureID — event_id последней ops.processing.failed (для «повтор обработки»).
	FailureID string
}

// LoadItem читает поток изделия до upTo (0 — весь) и раскладывает его на
// вход свёртки и записанные реакции. Ошибки чтения — инфраструктура;
// ошибки разбора записи — *ProcessingError.
func (c *Codec) LoadItem(ctx context.Context, itemID string, upTo int64) (ItemInput, error) {
	in := ItemInput{ItemID: itemID}
	var entries []jc.JournalEntry
	after := int64(0)
	for {
		page, err := c.Store.Read(ctx, appjournal.ReadQuery{ItemID: itemID, AfterSeq: after, Limit: 1000})
		if err != nil {
			return in, err
		}
		entries = append(entries, page...)
		if len(page) < 1000 {
			break
		}
		after = int64(page[len(page)-1].Seq)
	}
	for _, e := range entries {
		if e.Chain != "" && e.Chain != jc.JournalEntryChainMain {
			continue
		}
		// Вход — до upTo; записанные реакции — все: сравнение идёт с последней
		// версией слота, даже если она записана после upTo.
		beyond := upTo > 0 && int64(e.Seq) > upTo
		if beyond && !isOwnOutput(e) {
			continue
		}
		d, err := c.Decode(ctx, e)
		if err != nil {
			return in, &ProcessingError{ItemID: itemID, Seq: int64(e.Seq), Err: err}
		}
		switch {
		case d.Info.Type == catalog.OpsProcessingFailed:
			// Останавливает свёртку изделия только сбой воркера; сбой
			// проекции или стадии на записи изделия — их отметка (AD-45).
			if failedConsumer(d.Record) == appjournal.WorkerConsumer {
				in.Stopped, in.FailureID = true, d.Record.EventID
			}
			continue
		case d.Info.Type == catalog.OpsProcessingRetried:
			in.Stopped = false
		case d.Info.Role == RoleWorker:
			if r, ok := d.Recorded(); ok {
				in.Recorded = append(in.Recorded, r)
			}
			continue
		}
		in.Input = append(in.Input, d.Record)
		if d.Record.Seq >= in.Last.Seq {
			in.Last = d.Record
		}
	}
	return in, nil
}

// isOwnOutput — запись эмитирует сам воркер: не триггер свёртки (AD-5).
func isOwnOutput(e jc.JournalEntry) bool {
	info, ok := catalog.Lookup(catalog.Type(e.EventType))
	return ok && info.Role == RoleWorker
}

// failedConsumer — потребитель, записавший ops.processing.failed.
func failedConsumer(r kernel.Record) string {
	var d ev.OpsProcessingFailedV1
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return ""
	}
	return d.Consumer
}

// planItem — пересвёртка изделия целиком и разница с записанным (AD-5).
func (w *WorkerService) planItem(ctx context.Context, wk Work) (rq appjournal.AppendRequest, err error) {
	if wk.Trigger.EventID != "" && isOwnOutput(wk.Trigger) {
		return rq, nil
	}
	in, err := w.cfg.Codec.LoadItem(ctx, wk.ItemID, wk.UpToSeq)
	if err != nil {
		return rq, err
	}
	if in.Stopped || len(in.Input) == 0 {
		// «Обработка остановлена» до `ant rebuild --item` (AD-45).
		return rq, nil
	}
	fail := func(err error) (appjournal.AppendRequest, error) {
		return appjournal.AppendRequest{}, &ProcessingError{ItemID: wk.ItemID, Seq: in.Last.Seq, Err: err}
	}
	bundle, rev, err := w.cfg.Bundles.Bundle(ctx, wk.ItemID, in.Input)
	if err != nil {
		return rq, err
	}
	started := w.cfg.Now()
	snap, reactions, err := safeFold(w.cfg.Fold, bundle, in.Input)
	w.cfg.Telemetry.Observe(MetricFold, w.cfg.Now().Sub(started))
	if err != nil {
		return fail(err)
	}
	normalize(reactions)
	trigger := in.Last
	if wk.Trigger.EventID != "" {
		for _, r := range in.Input {
			if r.EventID == wk.Trigger.EventID {
				trigger = r
			}
		}
	}
	plan, err := engine.Diff(reactions, in.Recorded, engine.Trigger{EventID: trigger.EventID, OccurredAt: trigger.OccurredAt})
	if err != nil {
		return fail(err)
	}
	for _, p := range plan {
		pend, err := w.cfg.Codec.Encode(ctx, ReactionOut(p, wk.ItemID, trigger.RunID, trigger.CorrelationID, trigger.EventID, rev, snap.BasisSeq))
		if err != nil {
			return rq, err
		}
		rq.Batch = append(rq.Batch, pend)
	}
	effects, err := w.cfg.Projections.ItemEffects(wk.ItemID, snap, reactions, in.Input)
	if err != nil {
		return fail(err)
	}
	rq.Effects = append(effects, Notify{Changes: itemChanges(wk.ItemID, snap.BasisSeq, trigger)})
	return rq, nil
}

// itemChanges — изменения для SSE после пересвёртки изделия: паспорт
// изделия и живая карта (счётчики узлов считает сервер, AD-21).
func itemChanges(itemID string, seq int64, trigger kernel.Record) []Change {
	return []Change{
		{Entity: platform.EntityItem, ID: itemID, Seq: seq, RunID: trigger.RunID, ReceivedAt: trigger.ReceivedAt},
		{Entity: platform.EntityLiveMap, ID: "global", Seq: seq, RunID: trigger.RunID, ReceivedAt: trigger.ReceivedAt},
	}
}

// safeFold — свёртка с перехватом паники доменного кода: паника правила
// модуля — ошибка обработки изделия, а не падение воркера (AD-45).
func safeFold(fold engine.Folder, b engine.Bundle, in []kernel.Record) (s engine.Snapshot, rs []kernel.Reaction, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("паника свёртки: %v", p)
		}
	}()
	s, rs = fold(b, in)
	return s, rs, nil
}

// normalize — режим автоматизации 0 (не задан правилом) записывается как 1
// «только сообщить» (FR-50): конверт требует 1–5, а отпечаток сравнения
// должен совпасть с записанным.
func normalize(rs []kernel.Reaction) {
	for i := range rs {
		if rs[i].AutomationMode == 0 {
			rs[i].AutomationMode = 1
		}
	}
}

// failure — служебная запись ops.processing.failed по ошибке пересвёртки изделия.
func (w *WorkerService) failure(ctx context.Context, wk Work, perr *ProcessingError) (appjournal.AppendRequest, error) {
	seq := perr.Seq
	if seq == 0 {
		seq = wk.UpToSeq
	}
	return FailureRequest(ctx, w.cfg.Codec, appjournal.WorkerConsumer, wk.ItemID, seq, wk.Trigger, perr.Err)
}

// FailureRequest — служебная запись ops.processing.failed (эмитент ops, роль
// worker, AD-45): изделие «обработка остановлена», партиция и потребитель
// продолжают. Идентичность и текст записи — правила модуля ops
// (domain/ops.FailureEventID, FailureMessage): повтор после сбоя не создаёт
// второго смысла. Другие модули сообщают через порт ops.Reporter.ReportFailure.
func FailureRequest(ctx context.Context, c *Codec, consumer, itemID string, seq int64, trig jc.JournalEntry, cause error) (appjournal.AppendRequest, error) {
	msg := dops.FailureMessage(cause.Error())
	id := dops.FailureEventID(consumer, itemID, seq)
	occurred, _ := parseTime(trig.OccurredAt)
	received, _ := parseTime(trig.ReceivedAt)
	var run string
	if trig.RunID != nil {
		run = *trig.RunID
	}
	pend, err := c.Encode(ctx, Out{
		EventID: id, Type: catalog.OpsProcessingFailed, Kind: catalog.KindService, Stream: "item:" + itemID,
		ItemID: itemID, RunID: run, OccurredAt: occurred, Correlation: trig.CorrelationID, Causation: trig.EventID,
		BasisSeq: seq, Data: ev.OpsProcessingFailedV1{Consumer: consumer, FailedSeq: ev.Seq(seq), Error: msg},
	})
	if err != nil {
		return appjournal.AppendRequest{}, err
	}
	return appjournal.AppendRequest{
		Batch:   []appjournal.Pending{pend},
		Effects: []appjournal.Effect{Notify{Changes: []Change{{Entity: platform.EntityItem, ID: itemID, Seq: seq, RunID: run, ReceivedAt: received}}}},
	}, nil
}

// runOf — run_id записи для логов (пусто вне прогона).
func runOf(e jc.JournalEntry) string {
	if e.RunID == nil {
		return ""
	}
	return *e.RunID
}

// NopTelemetry — телеметрия-пустышка.
type NopTelemetry struct{}

// Counter ничего не делает.
func (NopTelemetry) Counter(string, int64, ...string) {}

// Observe ничего не делает.
func (NopTelemetry) Observe(string, time.Duration, ...string) {}

// Gauge ничего не делает.
func (NopTelemetry) Gauge(string, int64, ...string) {}
