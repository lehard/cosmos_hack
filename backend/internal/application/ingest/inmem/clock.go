package inmem

import (
	"context"
	"sync"
	"time"

	"ant/internal/application/platform"
)

// Clock — часы для тестов и демо: доменные и инфраструктурные сразу; Advance
// двигает время вперёд.
type Clock struct {
	mu sync.Mutex
	t  time.Time
}

// NewClock создаёт часы на момент t.
func NewClock(t time.Time) *Clock { return &Clock{t: t} }

// Now — доменное «сейчас» (DomainClock).
func (c *Clock) Now(context.Context) (time.Time, error) { return c.At(), nil }

// At — текущее время часов.
func (c *Clock) At() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

// Advance двигает часы на d.
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// Infra — те же часы как InfraClock.
func (c *Clock) Infra() InfraClock { return InfraClock{c} }

// InfraClock — обёртка Clock под порт InfraClock.
type InfraClock struct{ c *Clock }

// Now — время.
func (i InfraClock) Now() time.Time { return i.c.At() }

// SystemClock — системные часы как DomainClock и InfraClock (демо без сценария).
type SystemClock struct{}

// Now — доменное «сейчас».
func (SystemClock) Now(context.Context) (time.Time, error) { return time.Now().UTC(), nil }

// SystemInfra — системные часы как InfraClock.
type SystemInfra struct{}

// Now — время.
func (SystemInfra) Now() time.Time { return time.Now() }

var _ platform.Telemetry = (*Telemetry)(nil)
