package telemetry

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"ant/internal/application/platform"
)

// Buckets — границы гистограмм длительностей, секунды: от миллисекунд
// (свёртка, приём) через бюджет «событие → экран» 2 с (FR-2) до часа
// (досылка с исходным временем, FR-39).
var Buckets = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 300, 3600}

// Help — пояснения известных метрик (русский, «Соглашения/Язык»); прочие —
// имя метрики.
var Help = map[string]string{
	"ant_event_to_sse_seconds":              "Событие → экран: от received_at записи-триггера до отправки сообщения SSE (FR-2, бюджет 2 с)",
	"ant_fold_seconds":                      "Длительность пересвёртки изделия (AD-6, бюджет 500 мс)",
	"ant_processing_failed_total":           "Изделия с «обработка остановлена» (AD-45)",
	"ant_ingest_messages_total":             "Сообщения приёма по исходу: принято, дубль, отказ, карантин (FR-41)",
	"ant_ingest_latency_seconds":            "Длительность обработки сообщения приёмом (FR-41)",
	"ant_ingest_delivery_delay_seconds":     "Задержка доставки: received_at − occurred_at (FR-41)",
	"ant_ingest_quarantine_open":            "Открытый карантин приёма (FR-41)",
	"ant_ingest_source_completeness_bp":     "Полнота источника: полученные source_seq к ожидаемым, базисные пункты (FR-41)",
	"ant_ingest_security_suppressed_total":  "Подавленные повторы событий безопасности по источнику",
	"ant_ingest_signature_unverified_total": "Принято без проверки подписи (демо, пометка «подпись не проверялась»)",
	"ant_ops_stopped_items":                 "Изделия «обработка остановлена» (FR-127)",
	"ant_ops_consumer_lag_seq":              "Отставание курсора потребителя журнала от головы, записей (AD-45)",
	"ant_ops_queue_pending":                 "Необработанные записи после курсора потребителя",
	"ant_ops_outbox_quarantined":            "Исходящие сообщения в карантине (FR-96)",
	"ant_ops_integration_degraded":          "Канал обмена с внешней системой в degraded (AD-18)",
	"ant_ops_component_up":                  "Роль или сервис в состоянии ok (FR-127)",
	"ant_ops_quarantine_open":               "Открытый карантин приёма глазами ops (FR-127)",
}

type kind int

const (
	kindCounter kind = iota
	kindHistogram
	kindGauge
)

// metric — зарегистрированная метрика: вид и набор меток первого обращения.
type metric struct {
	kind      kind
	labels    []string
	counter   *prometheus.CounterVec
	histogram *prometheus.HistogramVec
	gauge     *prometheus.GaugeVec
}

// Prometheus — адаптер prometheus порта Telemetry (AD-35): своя
// prometheus.Registry на процесс (метрики Go и процесса + метрики модулей),
// Handler — ответ /metrics.
type Prometheus struct {
	reg *prometheus.Registry
	log *slog.Logger

	mu      sync.Mutex
	metrics map[string]*metric
	warned  map[string]bool
}

var _ platform.Telemetry = (*Prometheus)(nil)

// NewPrometheus создаёт адаптер со своей registry; log — предупреждения о
// несовместимых обращениях (nil — молча).
func NewPrometheus(log *slog.Logger) *Prometheus {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	return &Prometheus{reg: reg, log: log, metrics: map[string]*metric{}, warned: map[string]bool{}}
}

// Registry — registry адаптера (тесты, дополнительные коллекторы).
func (p *Prometheus) Registry() *prometheus.Registry { return p.reg }

// Handler — ответ /metrics (текстовый формат Prometheus / OpenMetrics).
func (p *Prometheus) Handler() http.Handler {
	return promhttp.HandlerFor(p.reg, promhttp.HandlerOpts{Registry: p.reg, EnableOpenMetrics: true})
}

// Counter увеличивает счётчик.
func (p *Prometheus) Counter(name string, delta int64, labels ...string) {
	if delta < 0 {
		return
	}
	if m, vals := p.get(name, kindCounter, labels); m != nil {
		m.counter.WithLabelValues(vals...).Add(float64(delta))
	}
}

// Observe записывает наблюдение гистограммы в секундах.
func (p *Prometheus) Observe(name string, value time.Duration, labels ...string) {
	if m, vals := p.get(name, kindHistogram, labels); m != nil {
		m.histogram.WithLabelValues(vals...).Observe(value.Seconds())
	}
}

// Gauge устанавливает показатель.
func (p *Prometheus) Gauge(name string, value int64, labels ...string) {
	if m, vals := p.get(name, kindGauge, labels); m != nil {
		m.gauge.WithLabelValues(vals...).Set(float64(value))
	}
}

// get — метрика name вида k (регистрирует при первом обращении) и значения
// её меток по парам labels. Несовместимый вид — nil (предупреждение один раз).
func (p *Prometheus) get(name string, k kind, labels []string) (*metric, []string) {
	name = sanitize(name, true)
	pairs := pairsOf(labels)
	p.mu.Lock()
	defer p.mu.Unlock()
	m, ok := p.metrics[name]
	if !ok {
		keys := make([]string, 0, len(pairs))
		for _, kv := range pairs {
			if !slices.Contains(keys, kv[0]) {
				keys = append(keys, kv[0])
			}
		}
		slices.Sort(keys)
		m = &metric{kind: k, labels: keys}
		help := Help[name]
		if help == "" {
			help = name
		}
		var c prometheus.Collector
		switch k {
		case kindCounter:
			m.counter = prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: help}, keys)
			c = m.counter
		case kindHistogram:
			m.histogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name, Help: help, Buckets: Buckets}, keys)
			c = m.histogram
		case kindGauge:
			m.gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: help}, keys)
			c = m.gauge
		}
		if err := p.reg.Register(c); err != nil {
			p.warnOnce(name, "метрика не зарегистрирована", err)
			return nil, nil
		}
		p.metrics[name] = m
	}
	if m.kind != k {
		p.warnOnce(name, "метрика уже зарегистрирована другого вида", nil)
		return nil, nil
	}
	vals := make([]string, len(m.labels))
	for i, key := range m.labels {
		for _, kv := range pairs {
			if kv[0] == key {
				vals[i] = kv[1]
			}
		}
	}
	return m, vals
}

func (p *Prometheus) warnOnce(name, msg string, err error) {
	if p.warned[name] {
		return
	}
	p.warned[name] = true
	p.log.Warn("телеметрия: "+msg, "metric", name, "err", err)
}

// pairsOf — пары (ключ, значение) из плоского списка; имена меток приведены
// к правилам Prometheus, непарный хвост отброшен.
func pairsOf(labels []string) [][2]string {
	out := make([][2]string, 0, len(labels)/2)
	for i := 0; i+1 < len(labels); i += 2 {
		out = append(out, [2]string{sanitize(labels[i], false), labels[i+1]})
	}
	return out
}

// sanitize — имя метрики или метки по правилам Prometheus: латиница, цифры,
// «_» (и «:» у метрик); прочее — «_».
func sanitize(s string, metricName bool) string {
	var b strings.Builder
	for i, r := range s {
		ok := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (i > 0 && r >= '0' && r <= '9') || (metricName && r == ':')
		if ok {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "_"
	}
	return b.String()
}
