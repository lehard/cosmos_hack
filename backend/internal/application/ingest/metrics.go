package ingest

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// Имена метрик приёма в /metrics (FR-41, AD-7): операционные, не проекции.
const (
	MetricMessages           = "ant_ingest_messages_total"             // outcome, code
	MetricLatency            = "ant_ingest_latency_seconds"            // обработка приёма
	MetricDeliveryDelay      = "ant_ingest_delivery_delay_seconds"     // received_at − occurred_at
	MetricQuarantineOpen     = "ant_ingest_quarantine_open"            // объём карантина
	MetricCompleteness       = "ant_ingest_source_completeness_bp"     // source_id
	MetricSecuritySuppressed = "ant_ingest_security_suppressed_total"  // source_id
	MetricUnverified         = "ant_ingest_signature_unverified_total" // принято без проверки подписи
)

// metrics — счётчики для стола администратора (Stats) и проброс в Telemetry.
type metrics struct {
	mu        sync.Mutex
	outcomes  map[Outcome]int64
	rejects   map[string]int64
	unverif   int64
	latencies []time.Duration // последние наблюдения для медианы
	maxLat    time.Duration
	maxDelay  time.Duration
}

func newMetrics() *metrics {
	return &metrics{outcomes: map[Outcome]int64{}, rejects: map[string]int64{}}
}

const latencyWindow = 1024

// observe учитывает итог сообщения.
func (s *Service) observe(r Result, latency, delay time.Duration) {
	m := s.stats
	m.mu.Lock()
	m.outcomes[r.Outcome]++
	if r.Outcome == OutcomeQuarantined || r.Outcome == OutcomeConflict {
		m.rejects[string(r.Code)]++
	}
	if !r.SignatureVerified && (r.Outcome == OutcomeAccepted || r.Outcome == OutcomeAcceptedWithFlag) {
		m.unverif++
	}
	m.latencies = append(m.latencies, latency)
	if len(m.latencies) > latencyWindow {
		m.latencies = m.latencies[len(m.latencies)-latencyWindow:]
	}
	m.maxLat = max(m.maxLat, latency)
	m.maxDelay = max(m.maxDelay, delay)
	m.mu.Unlock()
	if t := s.deps.Telemetry; t != nil {
		t.Counter(MetricMessages, 1, "outcome", string(r.Outcome), "code", string(r.Code))
		t.Observe(MetricLatency, latency)
		if delay > 0 {
			t.Observe(MetricDeliveryDelay, delay, "source_id", r.SourceID)
		}
		if !r.SignatureVerified && (r.Outcome == OutcomeAccepted || r.Outcome == OutcomeAcceptedWithFlag) {
			t.Counter(MetricUnverified, 1, "source_id", r.SourceID)
		}
	}
}

func (s *Service) gauge(name string, v int64, labels ...string) {
	if s.deps.Telemetry != nil {
		s.deps.Telemetry.Gauge(name, v, labels...)
	}
}

// Stats — метрики приёма (FR-41).
func (s *Service) Stats(ctx context.Context) (Stats, error) {
	if err := s.ready(); err != nil {
		return Stats{}, platform.NotImplemented("ingest.stats.read")
	}
	m := s.stats
	m.mu.Lock()
	st := Stats{
		Accepted:           m.outcomes[OutcomeAccepted],
		AcceptedWithFlag:   m.outcomes[OutcomeAcceptedWithFlag],
		Duplicates:         m.outcomes[OutcomeDuplicate],
		Conflicts:          m.outcomes[OutcomeConflict],
		Quarantined:        m.outcomes[OutcomeQuarantined],
		Unverified:         m.unverif,
		Rejects:            maps.Clone(m.rejects),
		LatencyMaxMS:       m.maxLat.Milliseconds(),
		DeliveryDelayMaxMS: m.maxDelay.Milliseconds(),
	}
	if n := len(m.latencies); n > 0 {
		l := slices.Clone(m.latencies)
		slices.Sort(l)
		st.LatencyP50MS = l[n/2].Milliseconds()
	}
	m.mu.Unlock()
	open, err := s.deps.Quarantine.Count(ctx, QuarantineOpen)
	if err != nil {
		return Stats{}, err
	}
	st.QuarantineOpen = open
	srcs, err := s.deps.Registry.Sources(ctx)
	if err != nil {
		return Stats{}, err
	}
	for _, src := range srcs {
		st.Sources = append(st.Sources, SourceStats{SourceID: src.SourceID, HighWater: src.HighWater, Missing: src.Missing(),
			CompletenessBP: src.CompletenessBP(), OpenGaps: len(src.Gaps)})
	}
	return st, nil
}

// statusOf — HTTP-статус кода по contracts/errors.yaml.
func statusOf(c errcodes.Code, def int) int {
	if i, ok := errcodes.Lookup(c); ok {
		return i.Status
	}
	return def
}
