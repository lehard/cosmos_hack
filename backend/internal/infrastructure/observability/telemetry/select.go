package telemetry

import (
	"fmt"
	"log/slog"
	"time"

	"ant/internal/application/platform"
)

// Адаптеры порта Telemetry по ключу ports.adapters.telemetry (AD-35).
const (
	AdapterPrometheus = "prometheus"
	AdapterMemory     = "memory"
	AdapterNop        = "nop"
)

// Nop — телеметрия-пустышка.
type Nop struct{}

// Counter ничего не делает.
func (Nop) Counter(string, int64, ...string) {}

// Observe ничего не делает.
func (Nop) Observe(string, time.Duration, ...string) {}

// Gauge ничего не делает.
func (Nop) Gauge(string, int64, ...string) {}

// New — адаптер по ключу конфигурации; пусто — prometheus. Для prometheus
// возвращается и *Prometheus (обработчик /metrics), иначе nil.
func New(adapter string, log *slog.Logger) (platform.Telemetry, *Prometheus, error) {
	switch adapter {
	case "", AdapterPrometheus:
		p := NewPrometheus(log)
		return p, p, nil
	case AdapterMemory:
		return NewMemory(), nil, nil
	case AdapterNop:
		return Nop{}, nil, nil
	case "otlp":
		return nil, nil, fmt.Errorf("telemetry: адаптер otlp описан (FR-113), в MVP не собран — используйте prometheus")
	}
	return nil, nil, fmt.Errorf("telemetry: неизвестный адаптер %q (prometheus | memory | nop)", adapter)
}
