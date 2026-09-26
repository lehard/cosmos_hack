package main

import (
	"context"
	"time"

	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
)

// serverNow — «сейчас» сервера для заголовка Ant-Now каждого ответа API
// (AD-37): фронтенд сравнивает с ним время в ответе (checked_at индикатора
// целостности и т. п.), а не часы браузера.
//   - режим ведущих портов fixtures — часы текущего шага мира заготовок
//     (курсор, в том числе прогона);
//   - профиль на часах сценария (demo) — доменное «сейчас» журнала (часы прогона);
//   - иначе — реальное время.
//
// Ошибка источника — реальное время: заголовок есть всегда.
func serverNow(ctx context.Context, env *environment, o apiOptions) func(context.Context) time.Time {
	if o.mode == platform.ModeFixtures {
		return func(ctx context.Context) time.Time {
			rt, err := loader.Default()
			if err != nil {
				return time.Now()
			}
			st, sc, err := rt.State(ctx)
			if err != nil || sc == nil || st.Step < 0 || st.Step >= sc.Steps() {
				return time.Now()
			}
			return sc.Header(st.Step).Clock
		}
	}
	if scenarioClock(env.cfg) {
		c, err := env.core(ctx)
		if err != nil {
			return nil
		}
		dc := c.domainClock()
		return func(ctx context.Context) time.Time {
			t, err := dc.Now(ctx)
			if err != nil {
				return time.Now()
			}
			return t
		}
	}
	return nil
}
