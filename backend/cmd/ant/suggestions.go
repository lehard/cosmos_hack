package main

import (
	"context"
	"strconv"
	"strings"

	analyticsapp "ant/internal/application/analytics"
	"ant/internal/application/platform"
	dom "ant/internal/domain/analysis"
)

// analyticsLine — адаптер порта analysis.LineSource (эпик 42, FR-63): вход
// генератора «ограничение линии» — узел-ограничение из счётчиков узлов
// analytics за текущую смену (FR-5, считает сервер).
type analyticsLine struct{ a *analyticsapp.Service }

// Bottleneck — ограничение линии сейчас; nil — ограничения нет.
func (l analyticsLine) Bottleneck(ctx context.Context) (*dom.LineBottleneck, error) {
	set, err := l.a.NodeCounters(ctx, "", analyticsapp.PeriodQuery{Kind: "shift"}, platform.Moment{})
	if err != nil || set.Bottleneck == nil {
		return nil, err
	}
	b := &dom.LineBottleneck{StepKey: set.Bottleneck.StepKey, Period: "смену"}
	if set.Bottleneck.Wait != nil {
		b.WaitSec = waitSeconds(*set.Bottleneck.Wait)
	}
	for _, c := range set.Counters {
		if c.StepKey == b.StepKey {
			b.Queue, b.Passed = c.Queue, c.Passed
		}
	}
	return b, nil
}

// waitSeconds — «37 мин», «2 ч 5 мин», «3 ч» → секунды (текст analytics).
func waitSeconds(s string) int64 {
	var sec int64
	f := strings.Fields(s)
	for i := 0; i+1 < len(f); i += 2 {
		n, err := strconv.ParseInt(f[i], 10, 64)
		if err != nil {
			continue
		}
		switch {
		case strings.HasPrefix(f[i+1], "ч"):
			sec += n * 3600
		case strings.HasPrefix(f[i+1], "мин"):
			sec += n * 60
		}
	}
	return sec
}
