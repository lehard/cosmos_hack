package access

import (
	"context"
	"log/slog"
	"time"
)

// Приём СКУД (FR-82, эпик 37): адаптер внешней системы за ведомым портом
// SKUDFeed; опросчик переводит проходы в факты access.zone.passed и после
// приёма поднимает отклонения присутствия. Проверка «по графику должен быть,
// ключа нет» идёт и без новых проходов — раз в CheckEvery.

// SKUDEvent — событие журнала СКУД: номер в журнале СКУД и проход.
type SKUDEvent struct {
	Seq  int64
	Pass ZonePassIn
}

// SKUDFeed — ведомый порт журнала проходов СКУД (адаптер —
// infrastructure/integration/access/skud; в демо — stand роли stands).
type SKUDFeed interface {
	// Events — события после номера after и номер последнего события журнала СКУД.
	Events(ctx context.Context, after int64) ([]SKUDEvent, int64, error)
}

// SKUDIntake — куда опросчик отдаёт проходы (Service).
type SKUDIntake interface {
	IngestZonePasses(ctx context.Context, passes []ZonePassIn) (int, error)
	CheckPresence(ctx context.Context) error
}

// SKUDPoller — опрос журнала СКУД. Номер прочитанного держится в памяти:
// после перезапуска журнал читается сначала — повтор прохода журнал не
// запишет (однозначный event_id); журнал СКУД стал короче (stand
// перезапущен) — чтение сначала.
type SKUDPoller struct {
	Feed   SKUDFeed
	Intake SKUDIntake
	// Every — период опроса (0 — 1 с); CheckEvery — период проверки присутствия (0 — 30 с).
	Every      time.Duration
	CheckEvery time.Duration
	Log        *slog.Logger

	cursor    int64
	lastCheck time.Time
	failing   bool
}

// Run — опрос до отмены ctx.
func (p *SKUDPoller) Run(ctx context.Context) error {
	every := p.Every
	if every <= 0 {
		every = time.Second
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		p.Tick(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// Tick — один опрос: новые проходы и, по сроку, проверка присутствия.
func (p *SKUDPoller) Tick(ctx context.Context) {
	evs, last, err := p.Feed.Events(ctx, p.cursor)
	switch {
	case err != nil:
		if !p.failing && p.Log != nil {
			p.Log.Warn("СКУД: журнал проходов недоступен", "err", err)
		}
		p.failing = true
	case last < p.cursor:
		p.cursor = 0
	default:
		if p.failing && p.Log != nil {
			p.Log.Info("СКУД: журнал проходов снова доступен")
		}
		p.failing = false
		if len(evs) > 0 {
			passes := make([]ZonePassIn, len(evs))
			for i, e := range evs {
				passes[i] = e.Pass
			}
			if n, err := p.Intake.IngestZonePasses(ctx, passes); err != nil {
				if p.Log != nil {
					p.Log.Warn("СКУД: проходы не записаны", "err", err, "written", n)
				}
				return
			}
			p.cursor = evs[len(evs)-1].Seq
			p.lastCheck = time.Now()
		}
	}
	check := p.CheckEvery
	if check <= 0 {
		check = 30 * time.Second
	}
	if time.Since(p.lastCheck) >= check {
		p.lastCheck = time.Now()
		if err := p.Intake.CheckPresence(ctx); err != nil && p.Log != nil {
			p.Log.Warn("СКУД: проверка присутствия", "err", err)
		}
	}
}
