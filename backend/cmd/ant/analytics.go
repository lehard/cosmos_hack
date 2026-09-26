package main

import (
	"context"

	analyticsapp "ant/internal/application/analytics"
	analyticsstore "ant/internal/infrastructure/storage/analytics"
	"ant/internal/infrastructure/storage/journal/clock"
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
		// Период и срез «смена» — по графику смен справочника (эпик 19, FR-81).
		analyticsapp.WithShifts(referenceShifts{c.refSource}),
	), nil
}
