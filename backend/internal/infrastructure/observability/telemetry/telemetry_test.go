package telemetry

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"

	"ant/internal/application/platform"
)

// reader — чтение значений адаптера для общего контрактного теста (AD-35).
type reader interface {
	platform.Telemetry
	counter(name string, labels ...string) int64
	gauge(name string, labels ...string) (int64, bool)
	observations(name string, labels ...string) int
}

type memReader struct{ *Memory }

func (m memReader) counter(n string, l ...string) int64       { return m.CounterValue(n, l...) }
func (m memReader) gauge(n string, l ...string) (int64, bool) { return m.GaugeValue(n, l...) }
func (m memReader) observations(n string, l ...string) int    { return len(m.Observations(n, l...)) }
func (p promReader) counter(n string, l ...string) int64 {
	m := p.find(n, l)
	return int64(m.GetCounter().GetValue())
}
func (p promReader) observations(n string, l ...string) int {
	return int(p.find(n, l).GetHistogram().GetSampleCount())
}
func (p promReader) gauge(n string, l ...string) (int64, bool) {
	m := p.find(n, l)
	return int64(m.GetGauge().GetValue()), m.GetGauge() != nil
}

type promReader struct{ *Prometheus }

// find — ряд метрики name с метками labels (пары) из Gather.
func (p promReader) find(name string, labels []string) *dto.Metric {
	fams, err := p.Registry().Gather()
	if err != nil {
		panic(err)
	}
	want := map[string]string{}
	for _, kv := range pairsOf(labels) {
		want[kv[0]] = kv[1]
	}
	for _, f := range fams {
		if f.GetName() != name {
			continue
		}
	next:
		for _, m := range f.GetMetric() {
			got := map[string]string{}
			for _, lp := range m.GetLabel() {
				got[lp.GetName()] = lp.GetValue()
			}
			for k, v := range want {
				if got[k] != v {
					continue next
				}
			}
			return m
		}
	}
	return &dto.Metric{}
}

// AD-35: общий контрактный тест для всех адаптеров порта Telemetry.
func TestTelemetryContract(t *testing.T) {
	for name, r := range map[string]reader{"memory": memReader{NewMemory()}, "prometheus": promReader{NewPrometheus(nil)}} {
		t.Run(name, func(t *testing.T) {
			r.Counter("ant_ingest_messages_total", 1, "outcome", "accepted", "code", "")
			r.Counter("ant_ingest_messages_total", 2, "outcome", "accepted", "code", "")
			r.Counter("ant_ingest_messages_total", 1, "outcome", "duplicate", "code", "")
			r.Counter("ant_ingest_messages_total", -5, "outcome", "accepted", "code", "") // счётчик не убывает
			if got := r.counter("ant_ingest_messages_total", "outcome", "accepted", "code", ""); got != 3 {
				t.Fatalf("счётчик: %d", got)
			}
			r.Observe("ant_event_to_sse_seconds", 300*time.Millisecond, "entity", "item")
			r.Observe("ant_event_to_sse_seconds", 1500*time.Millisecond, "entity", "item")
			if got := r.observations("ant_event_to_sse_seconds", "entity", "item"); got != 2 {
				t.Fatalf("наблюдения: %d", got)
			}
			r.Gauge("ant_ingest_quarantine_open", 4)
			r.Gauge("ant_ingest_quarantine_open", 2)
			if v, ok := r.gauge("ant_ingest_quarantine_open"); !ok || v != 2 {
				t.Fatalf("показатель: %d %v", v, ok)
			}
		})
	}
}

// /metrics в формате Prometheus: имена, метки, гистограмма; несовместимые
// обращения не роняют процесс.
func TestPrometheusHandler(t *testing.T) {
	p := NewPrometheus(nil)
	p.Counter("ant_ingest_messages_total", 1, "outcome", "accepted")
	p.Counter("ant_ingest_messages_total", 1, "code", "x", "outcome", "quarantined") // лишняя метка отброшена
	p.Gauge("ant_ingest_messages_total", 1)                                          // другой вид — пропуск
	p.Observe("ant_event_to_sse_seconds", 900*time.Millisecond, "entity", "item")
	p.Gauge("ant.bad-name", 1, "label-x", "v", "odd")
	rec := httptest.NewRecorder()
	p.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body, _ := io.ReadAll(rec.Body)
	s := string(body)
	for _, want := range []string{
		`ant_ingest_messages_total{outcome="accepted"} 1`,
		`ant_ingest_messages_total{outcome="quarantined"} 1`,
		`ant_event_to_sse_seconds_bucket{entity="item",le="1"} 1`,
		`ant_event_to_sse_seconds_count{entity="item"} 1`,
		`ant_bad_name{label_x="v"} 1`,
		"# HELP ant_event_to_sse_seconds Событие → экран",
		"go_goroutines",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("нет %q в /metrics:\n%s", want, s)
		}
	}
}

func TestNewByKey(t *testing.T) {
	for k, prom := range map[string]bool{"": true, "prometheus": true, "memory": false, "nop": false} {
		tel, p, err := New(k, nil)
		if err != nil || tel == nil || (p != nil) != prom {
			t.Fatalf("%q: %v %v %v", k, tel, p, err)
		}
	}
	if _, _, err := New("otlp", nil); err == nil {
		t.Fatal("otlp — описание, не адаптер MVP")
	}
}
