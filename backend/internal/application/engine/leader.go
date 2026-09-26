package engine

import (
	"context"
	"log/slog"
	"time"

	appjournal "ant/internal/application/journal"
)

// Leader — копия-лидер по аренде (AD-6): роли crossitem и projector — одна
// копия; аренда продлевается каждые TTL/3 по InfraClock (AD-37); потеря
// аренды отменяет работу лидера, запись без аренды отвергает Append
// (ErrFenced).
type Leader struct {
	Leases appjournal.LeaseStore
	// Name — имя аренды (crossitem, projector).
	Name string
	// Holder — идентификатор копии (хост:pid).
	Holder string
	TTL    time.Duration
	Log    *slog.Logger
}

// Run ждёт аренду, исполняет fn под ней и продлевает её; при потере аренды
// или ошибке fn — снова ждёт аренду. Возвращается при отмене ctx.
func (l Leader) Run(ctx context.Context, fn func(ctx context.Context, fence appjournal.Fence) error) error {
	if l.TTL <= 0 {
		l.TTL = 15 * time.Second
	}
	if l.Log == nil {
		l.Log = slog.New(slog.DiscardHandler)
	}
	for ctx.Err() == nil {
		fence, ok, err := l.Leases.Acquire(ctx, l.Name, l.Holder, l.TTL)
		if err != nil || !ok {
			if err != nil && ctx.Err() == nil {
				l.Log.Warn("аренда лидера", "lease", l.Name, "err", err)
			}
			pause(ctx, l.TTL/3)
			continue
		}
		l.Log.Info("копия стала лидером", "lease", l.Name, "epoch", fence.Epoch)
		lctx, cancel := context.WithCancel(ctx)
		done := make(chan error, 1)
		go func() { done <- fn(lctx, fence) }()
		t := time.NewTicker(l.TTL / 3)
	renew:
		for {
			select {
			case err := <-done:
				if err != nil && ctx.Err() == nil {
					l.Log.Error("работа лидера прервана", "lease", l.Name, "err", err)
				}
				break renew
			case <-t.C:
				f, ok, err := l.Leases.Acquire(ctx, l.Name, l.Holder, l.TTL)
				if err != nil || !ok || f.Epoch != fence.Epoch {
					l.Log.Warn("аренда лидера потеряна", "lease", l.Name, "err", err)
					cancel()
					<-done
					break renew
				}
			}
		}
		t.Stop()
		cancel()
		if ctx.Err() != nil {
			_ = l.Leases.Release(context.WithoutCancel(ctx), fence)
			break
		}
		pause(ctx, l.TTL/3)
	}
	return nil
}
