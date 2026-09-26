package identity

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// limiter — ограничение частоты попыток входа по ключу (адрес клиента,
// логин; golang.org/x/time/rate, AD-15). Время — InfraClock (AD-37).
type limiter struct {
	every rate.Limit
	burst int

	mu sync.Mutex
	m  map[string]*limEntry
}

type limEntry struct {
	lim  *rate.Limiter
	seen time.Time
}

func newLimiter(every time.Duration, burst int) *limiter {
	return &limiter{every: rate.Every(every), burst: burst, m: map[string]*limEntry{}}
}

// allow — попытка по ключу разрешена сейчас.
func (l *limiter) allow(key string, now time.Time) bool {
	if l == nil || key == "" {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.m) > 10000 {
		for k, e := range l.m {
			if now.Sub(e.seen) > 30*time.Minute {
				delete(l.m, k)
			}
		}
	}
	e, ok := l.m[key]
	if !ok {
		e = &limEntry{lim: rate.NewLimiter(l.every, l.burst)}
		l.m[key] = e
	}
	e.seen = now
	return e.lim.AllowN(now, 1)
}
