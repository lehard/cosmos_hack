package mes

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	appjournal "ant/internal/application/journal"
)

// Состояния канала MES (AD-18).
const (
	ChannelOK       = "ok"
	ChannelDegraded = "degraded"
)

// Outbox — часть роли outbox для MES (одна копия-лидер по аренде, AD-6):
// сверка канала, отправка неподтверждённых блоков (Sender) и опрос входящих
// (Gateway). При несовместимости контракта канал degraded — отправки нет до
// успешной сверки (AD-18, FR-111). При воспроизведении роль не запущена.
type Outbox struct {
	Sender  *Sender
	Gateway *Gateway
	// Poll — период отправки; PullEvery — опрос входящих; Recheck — сверка при ok.
	Poll, PullEvery, Recheck time.Duration
	// Now — InfraClock (AD-37); nil — time.Now.
	Now func() time.Time
	Log *slog.Logger

	mu    sync.Mutex
	state string
}

// State — состояние канала после последней сверки.
func (o *Outbox) State() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.state
}

// Check — сверка ответной стороны; итог — состояние канала.
func (o *Outbox) Check(ctx context.Context) string {
	st := ChannelOK
	if _, err := o.Sender.Channel.Check(ctx); err != nil {
		st = ChannelDegraded
		if o.Log != nil {
			o.Log.Warn("канал MES degraded", "err", err)
		}
	}
	o.mu.Lock()
	o.state = st
	o.mu.Unlock()
	return st
}

// Run исполняет роль под арендой fence до отмены ctx.
func (o *Outbox) Run(ctx context.Context, fence appjournal.Fence) error {
	now := o.Now
	if now == nil {
		now = time.Now
	}
	poll, pull, recheck := o.Poll, o.PullEvery, o.Recheck
	if poll <= 0 {
		poll = time.Second
	}
	if pull <= 0 {
		pull = 30 * time.Second
	}
	if recheck <= 0 {
		recheck = 5 * time.Minute
	}
	// Сверка: при ok — раз в recheck, при degraded — чаще (10 периодов отправки).
	st := o.Check(ctx)
	checked, nextPull := now(), now()
	t := time.NewTicker(poll)
	defer t.Stop()
	for {
		every := recheck
		if st != ChannelOK {
			every = 10 * poll
		}
		if now().Sub(checked) >= every {
			st, checked = o.Check(ctx), now()
		}
		if st == ChannelOK {
			if _, err := o.Sender.SendDue(ctx, fence); err != nil {
				if errors.Is(err, appjournal.ErrFenced) || ctx.Err() != nil {
					return err
				}
				if _, ok := AsContract(err); ok {
					st = ChannelDegraded
				} else if o.Log != nil {
					o.Log.Error("MES: отправка блоков", "err", err)
				}
			}
			if o.Gateway != nil && now().After(nextPull) {
				if res, err := o.Gateway.PullOnce(ctx); err != nil && ctx.Err() == nil && o.Log != nil {
					o.Log.Warn("MES: опрос входящих", "err", err)
				} else if res.Submitted > 0 && o.Log != nil {
					o.Log.Info("MES: приняты задания и события операций", "count", res.Submitted, "deferred", len(res.Deferred))
				}
				nextPull = now().Add(pull)
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}
