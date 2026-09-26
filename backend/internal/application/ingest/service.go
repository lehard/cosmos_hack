package ingest

import (
	"sync"
	"time"

	"ant/internal/application/journal"
	"ant/internal/application/materials"
	"ant/internal/application/platform"
	"ant/internal/application/signing"
	"ant/internal/domain/crossitem"
	dom "ant/internal/domain/ingest"
)

// SignatureMode — режим проверки подписи источника (FR-26, выключатель по профилю).
type SignatureMode string

const (
	// SignatureRequired — подпись обязательна (профиль prod, после эпика 05):
	// неподписанное, неизвестный или отозванный ключ — отказ с кодом, карантин
	// и событие безопасности.
	SignatureRequired SignatureMode = "required"
	// SignatureDemoUnverified — профиль demo до эпика 05: неподписанное событие
	// принимается с явной пометкой «подпись не проверялась»; подписанное
	// проверяется, если подключены Verifier и KeyRegistry.
	SignatureDemoUnverified SignatureMode = "demo_unverified"
)

// SignatureModeFor — режим подписи по профилю: demo — без обязательной подписи,
// остальные — обязательная.
func SignatureModeFor(profile string) SignatureMode {
	if profile == "demo" {
		return SignatureDemoUnverified
	}
	return SignatureRequired
}

// Config — настройки приёма.
type Config struct {
	// Profile — профиль окружения (prod | demo | …): режим подписи по умолчанию.
	Profile string
	// Signature — режим проверки подписи; пусто — по профилю.
	Signature SignatureMode
	// Partitions — число партиций P (AD-6); StagePartition — партиция записей вне изделия.
	Partitions     int
	StagePartition int
	// Clock — пороги часов (FR-33).
	Clock dom.ClockPolicy
	// LossWindow — окно ожидания досылки до сигнала «потеря данных источника» (AD-7).
	LossWindow time.Duration
	// MaxBatch — наибольшее число сообщений в пачке (ingest.batch_too_large).
	MaxBatch int
	// SecurityPerMinute — сколько событий безопасности в минуту на источник
	// записывается при конфликтах и ошибках подписи (AD-7: частота ограничена).
	SecurityPerMinute int
	// DomainBuild — хеш доменного пакета для записей журнала (AD-9).
	DomainBuild string
	// ScenarioClock — журнал в режиме часов scenario (AD-37): recorded_at записей
	// приёма — доменное «сейчас»; иначе пусто — Append ставит committed_at.
	ScenarioClock bool
	// GatewaySource — source_id служебных записей приёма; GatewayKey — ключ шлюза.
	GatewaySource string
	GatewayKey    string
}

// DefaultConfig — настройки по умолчанию.
func DefaultConfig() Config {
	return Config{
		Profile:           "demo",
		Partitions:        16,
		StagePartition:    16,
		Clock:             dom.DefaultClockPolicy,
		LossWindow:        time.Minute,
		MaxBatch:          1000,
		SecurityPerMinute: 10,
		DomainBuild:       dom.Digest([]byte("ant/domain:dev")),
		GatewaySource:     "ant-ingest",
		GatewayKey:        "gateway-ingest@1",
	}
}

// Deps — ведомые порты приёма.
type Deps struct {
	Journal    journal.JournalStore
	Registry   Registry
	Quarantine QuarantineStore
	// Materials — содержимое карантина по адресу H(байты) (AD-2, AD-23); nil —
	// байты хранятся в записи карантина (упрощённый режим).
	Materials materials.MaterialStore
	// Verifier, Keys — проверка подписи и реестр ключей (FR-26); nil в demo — «подпись не проверялась».
	Verifier signing.Verifier
	Keys     KeyRegistry
	// ServerSigner — подпись служебных записей ключом шлюза; nil — без подписи (до эпика 05).
	ServerSigner signing.Signer
	// Security — шина безопасности; nil — мост JournalSecurityBus.
	Security SecurityBus
	// Carriers — реестр носителей на момент (AD-41); nil — носитель не разрешается,
	// событие идёт в поток межизделийной стадии.
	Carriers    crossitem.CarrierRegistry
	DomainClock journal.DomainClock
	InfraClock  journal.InfraClock
	// Telemetry — метрики /metrics (FR-41); nil — только Stats.
	Telemetry platform.Telemetry
}

// Service — реализация live ведущих портов модуля ingest (AD-36).
type Service struct {
	cfg     Config
	deps    Deps
	schemas *schemaSet
	locks   sync.Map // source_id → *sync.Mutex: учёт номеров и реестр — по источнику последовательно
	limiter *rateLimiter
	stats   *metrics
	cmdMu   sync.Mutex
	cmds    map[string]any // command_id → прежний ответ (AD-7)
	started time.Time
}

// Option — настройка Service.
type Option func(*Service)

// WithConfig задаёт настройки приёма.
func WithConfig(c Config) Option { return func(s *Service) { s.cfg = c } }

// WithDeps подключает ведомые порты.
func WithDeps(d Deps) Option { return func(s *Service) { s.deps = d } }

// NewService создаёт реализацию live. Без WithDeps операции отвечают 501:
// приём не подключён к журналу (волна 1, сборка в cmd/ant).
func NewService(opts ...Option) *Service {
	s := &Service{cfg: DefaultConfig(), schemas: newSchemaSet(), stats: newMetrics(), cmds: map[string]any{}}
	for _, o := range opts {
		o(s)
	}
	if s.cfg.Signature == "" {
		s.cfg.Signature = SignatureModeFor(s.cfg.Profile)
	}
	if s.deps.Security == nil {
		s.deps.Security = JournalSecurityBus{Gateway: s}
	}
	s.limiter = newRateLimiter(s.cfg.SecurityPerMinute)
	if s.deps.InfraClock != nil {
		s.started = s.deps.InfraClock.Now()
	}
	return s
}

// ready — подключены ли порты, без которых приём не работает.
func (s *Service) ready() error {
	if s.deps.Journal == nil || s.deps.Registry == nil || s.deps.Quarantine == nil || s.deps.DomainClock == nil || s.deps.InfraClock == nil {
		return platform.NotImplemented("ingest.event.submit")
	}
	return nil
}

func (s *Service) lock(sourceID string) func() {
	m, _ := s.locks.LoadOrStore(sourceID, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// rateLimiter — не больше n событий безопасности в минуту на источник (AD-7);
// время — InfraClock (AD-37).
type rateLimiter struct {
	mu sync.Mutex
	n  int
	w  map[string]window
}

type window struct {
	start time.Time
	count int
}

func newRateLimiter(n int) *rateLimiter { return &rateLimiter{n: n, w: map[string]window{}} }

func (r *rateLimiter) allow(key string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	w := r.w[key]
	if now.Sub(w.start) >= time.Minute {
		w = window{start: now}
	}
	if w.count >= r.n {
		r.w[key] = w
		return false
	}
	w.count++
	r.w[key] = w
	return true
}
