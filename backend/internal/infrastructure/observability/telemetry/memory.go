package telemetry

import (
	"slices"
	"strings"
	"sync"
	"time"

	"ant/internal/application/platform"
)

// Memory — экспортёр в память (AD-35: «тест с экспортёром в память»):
// счётчики и показатели по имени и меткам, наблюдения — списком.
type Memory struct {
	mu       sync.Mutex
	counters map[string]int64
	gauges   map[string]int64
	observed map[string][]time.Duration
}

var _ platform.Telemetry = (*Memory)(nil)

// NewMemory создаёт пустой экспортёр.
func NewMemory() *Memory {
	return &Memory{counters: map[string]int64{}, gauges: map[string]int64{}, observed: map[string][]time.Duration{}}
}

// Key — ключ ряда: имя и пары меток, отсортированные по ключу: `name{a=1,b=2}`.
func Key(name string, labels ...string) string {
	ps := pairsOf(labels)
	slices.SortFunc(ps, func(a, b [2]string) int { return strings.Compare(a[0], b[0]) })
	parts := make([]string, 0, len(ps))
	for _, kv := range ps {
		parts = append(parts, kv[0]+"="+kv[1])
	}
	return sanitize(name, true) + "{" + strings.Join(parts, ",") + "}"
}

// Counter увеличивает счётчик.
func (m *Memory) Counter(name string, delta int64, labels ...string) {
	if delta < 0 {
		return
	}
	m.mu.Lock()
	m.counters[Key(name, labels...)] += delta
	m.mu.Unlock()
}

// Observe записывает наблюдение.
func (m *Memory) Observe(name string, v time.Duration, labels ...string) {
	m.mu.Lock()
	k := Key(name, labels...)
	m.observed[k] = append(m.observed[k], v)
	m.mu.Unlock()
}

// Gauge устанавливает показатель.
func (m *Memory) Gauge(name string, v int64, labels ...string) {
	m.mu.Lock()
	m.gauges[Key(name, labels...)] = v
	m.mu.Unlock()
}

// CounterValue — значение счётчика ряда.
func (m *Memory) CounterValue(name string, labels ...string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[Key(name, labels...)]
}

// GaugeValue — значение показателя ряда и есть ли он.
func (m *Memory) GaugeValue(name string, labels ...string) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.gauges[Key(name, labels...)]
	return v, ok
}

// Observations — наблюдения ряда.
func (m *Memory) Observations(name string, labels ...string) []time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.observed[Key(name, labels...)])
}
