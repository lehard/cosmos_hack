package main

import (
	"context"
	"errors"
	"time"

	"ant/cmd/internal/db"
	analysisapp "ant/internal/application/analysis"
	crossitemapp "ant/internal/application/crossitem"
	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	machinelogsapp "ant/internal/application/machinelogs"
	mldomain "ant/internal/domain/machinelogs"
	"ant/internal/infrastructure/storage/journal/clock"
	"ant/internal/infrastructure/storage/journal/feed"
)

// Тела ролей движка (эпик 07) поверх журнала на Postgres (эпик 04):
// worker — свёртка изделий по арендованным партициям; crossitem —
// межизделийная стадия, одна копия-лидер; projector — глобальные проекции,
// одна копия-лидер; rebuild — пересборка проекций из журнала.

// engineRegistry — реестр проекций движка: проекция engine.item_state и то,
// что подключают модули. Модули волны 4 регистрируют здесь свои проекции
// изделия (AddItem), глобальные (AddGlobal) и вклады показателей
// (AddContributor) — одна строка на модуль, как в buildAPI.
func engineRegistry() *engineapp.Registry {
	r := engineapp.NewRegistry()
	// machinelogs (эпик 23): профили выполнения изделия, индекс выполнений,
	// состояние и журнал оборудования, окна нарушений специального процесса.
	if err := machinelogsapp.RegisterProjections(r, mldomain.Env{}); err != nil {
		panic(err)
	}
	// analysis (эпик 22): разбор обстоятельств изделия, инциденты и версии
	// области риска, несоответствия для гипотез и общих факторов.
	analysisapp.MustRegister(r)
	return r
}

// runWorker — роль worker (AD-5, AD-6, AD-45): партиции hash(item_id) mod P
// с арендой и эпохой делятся между копиями; вход изделия сворачивается
// целиком, разница реакций, проекции, вклады и курсор — одним Append.
func runWorker(ctx context.Context, env *environment) error {
	c, err := env.readyCore(ctx)
	if err != nil {
		return err
	}
	wf := feed.NewWorkFeed(c.journal, c.leases, c.listener, env.cfg.Engine.Partitions, c.feedOptions(env, "worker"))
	w := engineapp.NewWorker(engineapp.WorkerConfig{
		Feed: wf, Codec: c.codec, Projections: c.registry, Log: env.log, Now: c.codec.Now,
		// Аренды партиций продлевает WorkFeed.Partitions — не реже TTL/3.
		Refresh: c.ttl / 3,
	})
	env.log.Info("воркер: старт", "partitions", env.cfg.Engine.Partitions, "holder", c.holder)
	return w.Run(ctx)
}

// runCrossItem — роль crossitem (AD-6, AD-42): межизделийная стадия под
// арендой лидера `crossitem`; курсор, состояние стадии и адресованные записи
// — одним Append под арендой потребителя.
func runCrossItem(ctx context.Context, env *environment) error {
	c, err := env.readyCore(ctx)
	if err != nil {
		return err
	}
	stage := &crossitemapp.StageRunner{
		Consumer: feed.NewConsumer(c.journal, c.leases, c.listener, c.feedOptions(env, "crossitem")),
		Codec:    c.codec, Store: c.engine, Log: env.log,
	}
	return c.leader(env, "crossitem").Run(ctx, stage.Run)
}

// runProjector — роль projector (AD-45): глобальные проекции под арендой
// лидера `projector`, у каждой свой курсор.
func runProjector(ctx context.Context, env *environment) error {
	c, err := env.readyCore(ctx)
	if err != nil {
		return err
	}
	if len(c.registry.Globals()) == 0 {
		env.log.Info("проектор: глобальных проекций пока нет — ожидаю остановки")
		<-ctx.Done()
		return nil
	}
	p := &engineapp.Projector{
		Consumer: feed.NewConsumer(c.journal, c.leases, c.listener, c.feedOptions(env, "projector")),
		Codec:    c.codec, Store: c.engine, Registry: c.registry, Log: env.log,
	}
	return c.leader(env, "projector").Run(ctx, p.Run)
}

// runRebuild — разовая роль rebuild (FR-115, FR-124, AD-45): `ant rebuild`
// — все проекции заново из журнала (при остановленных worker и projector);
// `ant rebuild -item ‹id›` — повтор изделия с «обработка остановлена» или
// пересборка его проекций. Журнал не меняется, наружу ничего не уходит.
func runRebuild(ctx context.Context, env *environment) error {
	c, err := env.readyCore(ctx)
	if err != nil {
		return err
	}
	rb := &engineapp.Rebuilder{Codec: c.codec, Registry: c.registry}
	var rep engineapp.RebuildReport
	if env.item != "" {
		rep, err = rb.RebuildItem(ctx, env.item, env.reason)
	} else {
		rep, err = rb.RebuildAll(ctx)
	}
	if err != nil {
		var perr *engineapp.ProcessingError
		if errors.As(err, &perr) {
			env.log.Error("rebuild: изделие не сворачивается", "item_id", perr.ItemID, "seq", perr.Seq, "err", perr.Err)
		}
		return err
	}
	env.log.Info("rebuild: готово", "item_id", env.item, "items", rep.Items, "globals", rep.Globals,
		"retried", rep.Retried, "rebuild_hash", rep.RebuildHash)
	return nil
}

// readyCore — ядро процесса после ответа БД (роли движка стартуют вместе с
// postgres; migrate к этому времени завершён — порядок compose).
func (e *environment) readyCore(ctx context.Context) (*core, error) {
	c, err := e.core(ctx)
	if err != nil {
		return nil, err
	}
	if err := db.WaitReady(ctx, c.pool, 2*time.Minute); err != nil {
		return nil, err
	}
	return c, nil
}

// journalLive — live-реализация ведущих портов journal для роли api:
// журнал и сигнал эпика 04, живые обновления — публикатор движка
// (LiveUpdates над журналом изменений engine.changes). Публикатор
// запускается здесь и живёт до отмены ctx.
func journalLive(ctx context.Context, env *environment) (*appjournal.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	live := engineapp.NewLiveUpdates(engineapp.LiveConfig{Log: c.engine, Now: c.codec.Now, Logger: env.log})
	env.coreH.bg.Go(func() { _ = live.Run(ctx) })
	return appjournal.NewServiceWith(c.journal, c.listener, appjournal.WithLive(live)), nil
}

// machinelogsLive — live-реализация ведущих портов machinelogs для роли api
// (эпик 23): операции чтения над проекциями модуля на ядре процесса;
// состояние изделия на момент — та же свёртка, что у воркера (AD-22).
func machinelogsLive(ctx context.Context, env *environment) (*machinelogsapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	return machinelogsapp.NewLiveService(c.engine, engineapp.StateQueries{Codec: c.codec}, mldomain.Env{}), nil
}

// analysisLive — live-реализация ведущих портов analysis для роли api (эпик
// 22): чтение — проекции analysis.* на ядре процесса; команды — гард над
// состоянием инцидента и решение в журнал ядра; доменное «сейчас» — часы
// журнала (AD-37).
func analysisLive(ctx context.Context, env *environment) (*analysisapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	return analysisapp.NewLive(analysisapp.Config{
		Projections: c.engine,
		Decisions: analysisapp.JournalDecisions{Journal: c.journal, DomainBuild: c.codec.DomainBuild,
			Partitions: env.cfg.Engine.Partitions, Now: c.codec.Now},
		Clock: clock.NewJournal(c.journal),
	}), nil
}
