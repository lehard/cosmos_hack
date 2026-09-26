package main

import (
	"context"
	"sync"
	"time"

	appjournal "ant/internal/application/journal"
	notificationsapp "ant/internal/application/notifications"
	"ant/internal/infrastructure/storage/journal/feed"
	notificationsstore "ant/internal/infrastructure/storage/notifications"
)

// Сборка модуля notifications (эпик 24): сроки, задачи, уведомления,
// эскалации. Реакции (сроки, эскалации, тревоги, задачи изделия) вычисляет
// свёртка изделия у воркера; проекции notifications.* ведёт роль projector
// (engineRegistry); роль scheduler — «наступил срок», задачи по решениям в
// потоках объектов и периодические проверки.

// Периоды роли scheduler (InfraClock, AD-37): проход сроков — часто (срок
// точки предъявления — минуты; в прогоне сценария доменные часы идут
// быстрее), проверки — реже.
const (
	schedulerEvery = 5 * time.Second
	integrityEvery = time.Minute
	lossesEvery    = 30 * time.Second
)

// notificationsLive — live-реализация ведущих портов notifications для роли
// api: чтение — проекции notifications.* на ядре процесса; отметка задачи —
// решение в журнал ядра; доменное «сейчас» — часы журнала (AD-37).
func notificationsLive(ctx context.Context, env *environment) (*notificationsapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	return notificationsapp.NewLive(notificationsapp.Config{
		Projections: notificationsstore.New(c.pool),
		Decisions: notificationsapp.JournalDecisions{Journal: c.journal, DomainBuild: c.codec.DomainBuild,
			Partitions: env.cfg.Engine.Partitions, Now: c.codec.Now},
		Clock: c.domainClock(),
	}), nil
}

// runScheduler — роль scheduler (AD-4, AD-6, AD-25): копия-лидер по аренде
// `scheduler` пишет «наступил срок» по проекции сроков и ведёт периодические
// проверки — целостность основной цепочки и полноту приёма (CheckLosses,
// эпик 06); потребитель notifications.objects (своя аренда) ставит задачи по
// решениям в потоках инцидентов.
func runScheduler(ctx context.Context, env *environment) error {
	c, err := env.readyCore(ctx)
	if err != nil {
		return err
	}
	s := &notificationsapp.Scheduler{
		Projections: notificationsstore.New(c.pool), Codec: c.codec,
		Clock: c.domainClock(), Log: env.moduleLog("notifications"), BatchMax: env.cfg.Journal.BatchMax,
	}
	integrity := &notificationsapp.Integrity{Journal: c.journal, Log: env.moduleLog("notifications")}
	checks := []notificationsapp.Check{{Name: "integrity", Every: integrityEvery, Run: integrity.Check}}
	if ing, err := ingestLive(ctx, env); err != nil {
		env.log.Warn("планировщик: приём недоступен — проверка полноты выключена", "err", err)
	} else {
		checks = append(checks, notificationsapp.Check{Name: "losses", Every: lossesEvery, Run: func(ctx context.Context) error {
			reps, err := ing.CheckLosses(ctx)
			if len(reps) > 0 {
				env.log.Warn("планировщик: потеря данных источника", "reports", len(reps))
			}
			return err
		}})
	}
	objects := feed.NewConsumer(c.journal, c.leases, c.listener, c.feedOptions(env, "scheduler"))
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Go(func() {
		err := objects.Consume(ctx, notificationsapp.ObjectTasksConsumer, appjournal.Scope{Global: true}, s.ObjectTasks)
		if err != nil && ctx.Err() == nil {
			env.log.Error("планировщик: задачи объектов остановлены", "err", err)
		}
	})
	env.log.Info("планировщик: старт", "every", schedulerEvery)
	return c.leader(env, "scheduler").Run(ctx, func(ctx context.Context, fence appjournal.Fence) error {
		return s.Loop(ctx, fence, schedulerEvery, checks...)
	})
}
