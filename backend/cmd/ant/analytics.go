package main

import (
	"context"

	analyticsapp "ant/internal/application/analytics"
	"ant/internal/infrastructure/storage/journal/clock"
	analyticsstore "ant/internal/infrastructure/storage/analytics"
)

// analyticsLive — live-показатели (эпик 25, AD-45) на ядре процесса: строки
// вклада изделий и глобальные проекции analytics, которые пишет движок
// (роли worker и projector); «сейчас» — доменные часы журнала (AD-37:
// в режиме сценария — виртуальные часы прогона).
func analyticsLive(ctx context.Context, env *environment) (*analyticsapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	return analyticsapp.NewService(
		analyticsapp.WithStore(analyticsstore.New(c.pool)),
		analyticsapp.WithClock(clock.NewJournal(c.journal)),
	), nil
}
