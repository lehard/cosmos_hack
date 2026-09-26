package main

import (
	notificationsapp "ant/internal/application/notifications"
	"context"
	"io/fs"
	"time"

	analyticsapp "ant/internal/application/analytics"
	engineapp "ant/internal/application/engine"
	ingestapp "ant/internal/application/ingest"
	"ant/internal/application/platform"
	processapp "ant/internal/application/process"
	"ant/internal/infrastructure/fixtures/world"
	"ant/internal/infrastructure/storage/journal/clock"
)

// Модуль process (эпик 17) на ядре процесса: хранилище версий в схеме
// process, нормативный слой изделия для воркера, запросов на момент и
// пересборки (Bundles, AD-17), live-реализация операций process.

// processSeedFile — стартовый процесс (FR-10): встроенная копия
// normative/process (world.Inputs; совпадение с репозиторием проверяет тест
// генератора заготовок), пока генезис не пишет его в журнал (эпик 05).
const processSeedFile = "normative/process/flange-process.bpmn"

// bundleSource — нормативный слой изделия для воркера, запросов на момент,
// пересборки и гардов команд: версия процесса, закреплённая при запуске
// изделия (эпик 17), слой quality (эпик 20) и слой notifications (эпик 24:
// описание процесса для сроков окон BPMN и точек предъявления) поверх неё.
func (c *core) bundleSource() engineapp.BundleSource {
	return notificationsapp.Bundles{Next: c.qualityBundles(c.bundles)}
}

// states — запросы состояния изделия на момент с тем же нормативным слоем (AD-22).
func (c *core) states() engineapp.StateQueries {
	return engineapp.StateQueries{Codec: c.codec, Bundles: c.bundleSource()}
}

// ensureProcessSeed — стартовая версия на чистой базе без ручных шагов
// (FR-10); демо-трек: подписана генезисом (AD-33). Без схемы process
// (migrate ещё не прошёл) — предупреждение, повтор при следующем старте роли.
func (c *core) ensureProcessSeed(ctx context.Context, env *environment) {
	c.seedOnce.Do(func() {
		xml, err := fs.ReadFile(world.Inputs(), processSeedFile)
		if err != nil {
			env.log.Error("процесс: нет стартовой версии", "err", err)
			return
		}
		v, err := processapp.EnsureSeed(ctx, c.versions, xml, time.Now().UTC())
		if err != nil {
			env.log.Warn("процесс: стартовая версия не загружена — жду migrate", "err", err)
			return
		}
		env.log.Info("процесс: стартовая версия", "version_id", v.ID, "hash", v.Hash)
	})
}

// processLive — live-реализация операций process для роли api: живая карта и
// карточки узлов по проекциям движка, версии из схемы process, команды
// исполнителя — гард над свёрткой изделия и факт через приём (ingest).
func processLive(ctx context.Context, env *environment, ingest *ingestapp.Service, analytics *analyticsapp.Service) (*processapp.Service, error) {
	c, err := env.readyCore(ctx)
	if err != nil {
		return nil, err
	}
	c.ensureProcessSeed(ctx, env)
	svc := &processapp.LiveService{Store: c.engine, States: c.states(), Library: c.versions, Bundles: c.bundles,
		Clock:    clock.NewJournal(c.journal).Now,
		Recorder: &processapp.Recorder{Journal: c.journal, DomainBuild: c.codec.DomainBuild, Partitions: env.cfg.Engine.Partitions, Now: c.codec.Now}}
	if ingest != nil {
		svc.Facts = ingestFacts{ingest}
	}
	if analytics != nil {
		svc.Counters = analyticsCounters{analytics}
	}
	return svc, nil
}

// ingestFacts — FactWriter над ручным вводом приёма (FR-137, FR-141): схема,
// дубли (event_id = command_id), журнал, пометка источника «ручной ввод».
type ingestFacts struct{ s *ingestapp.Service }

func (f ingestFacts) Submit(ctx context.Context, x processapp.Fact) (platform.Receipt, error) {
	r, err := f.s.SubmitManual(ctx, ingestapp.Cmd[ingestapp.ManualInput]{Meta: x.Meta, Body: ingestapp.ManualInput{
		EventType: string(x.EventType), ItemID: x.ItemID, OccurredAt: x.OccurredAt, Data: x.Data}})
	if err != nil {
		return platform.Receipt{}, err
	}
	if r.Code != "" && r.Outcome != ingestapp.OutcomeAccepted && r.Outcome != ingestapp.OutcomeAcceptedWithFlag && r.Outcome != ingestapp.OutcomeDuplicate {
		e := platform.Fail(r.Code, "field", r.Field, "value", r.Value, "reason", r.Detail)
		e.Detail = r.Detail
		return platform.Receipt{}, e
	}
	return platform.Receipt{CommandID: x.Meta.CommandID, Seq: r.Seq, EventIDs: []string{r.EventID}, Replayed: r.Outcome == ingestapp.OutcomeDuplicate}, nil
}

// analyticsCounters — счётчики узлов живой карты от analytics (эпик 25, AD-21).
type analyticsCounters struct{ s *analyticsapp.Service }

func (a analyticsCounters) NodeCounters(ctx context.Context, versionID string, q processapp.LiveMapQuery, m platform.Moment) (processapp.NodeCounterSet, error) {
	set, err := a.s.NodeCounters(ctx, versionID, analyticsapp.PeriodQuery{Kind: q.Period, From: q.From, To: q.To}, m)
	if err != nil {
		return processapp.NodeCounterSet{}, err
	}
	out := processapp.NodeCounterSet{DataGaps: set.DataGaps, BasisSeq: set.BasisSeq}
	for _, c := range set.Counters {
		out.Counters = append(out.Counters, processapp.MapNodeCounters{StepKey: c.StepKey, Queue: c.Queue, InProgress: c.InProgress,
			Passed: c.Passed, Defects: c.Defects, Nonconformities: c.Nonconformities})
	}
	if b := set.Bottleneck; b != nil {
		out.Bottleneck = &processapp.MapBottleneck{StepKey: b.StepKey}
		if b.Wait != nil {
			out.Bottleneck.Wait = *b.Wait
		}
	}
	for _, an := range set.Anomalies {
		x := processapp.MapNodeAnomaly{StepKey: an.StepKey, Kind: an.Kind}
		if an.Threshold != nil {
			x.Threshold = *an.Threshold
		}
		out.Anomalies = append(out.Anomalies, x)
	}
	return out, nil
}
