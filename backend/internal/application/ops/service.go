package ops

import (
	"context"
	"log/slog"
	"sync"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	dom "ant/internal/domain/ops"
)

// Config — зависимости live-реализации модуля ops.
type Config struct {
	// Journal — журнал: чтение сбоев, повторов, отчётов верификатора, генезиса;
	// запись решений администратора (AD-44: пишет только journal.Append).
	Journal appjournal.JournalStore
	// Codec — разбор записей журнала и сборка служебных записей ops (AD-44).
	Codec *engineapp.Codec
	// Runtime — аренды и курсоры (nil — роли и очереди «unknown»).
	Runtime Runtime
	// Exchanges — очереди исходящих и каналы модулей интеграций.
	Exchanges []Exchange
	// Quarantine — открытый карантин приёма (nil — 0).
	Quarantine Quarantine
	// Database — проверка базы (nil — postgres «unknown»).
	Database Database
	// Partitions — P (AD-6).
	Partitions int
	// Enabled — включённые внешние системы (integrations.enabled).
	Enabled []string
	// Installed — установленные конфигурацией интеграции и заданные режимы
	// (стенд, реальная система) — экран «Интеграции» (FR-157, AD-47).
	Installed []dom.Installed
	// Probe — «проверить соединение» (nil — «у адаптера нет проверки»).
	Probe Probe
	// Switch — порт состояния интеграций этой копии: сбрасывается после решения.
	Switch *IntegrationSwitch
	// Profile, Version, Mode — профиль конфигурации, версия бинарника, режим ведущих портов.
	Profile string
	Version string
	Mode    platform.Mode
	// Adapters — адаптеры ведомых портов по ключу (ports.adapters, AD-35).
	Adapters map[string]string
	// Modules — режимы ведущих портов модулей (AD-36); nil — пусто.
	Modules func() []ModuleMode
	// VerifierInterval — интервал проверок верификатора: отчёт старше двух
	// интервалов — верификатор «degraded» (AD-46).
	VerifierInterval time.Duration
	// Now — InfraClock (AD-37); nil — time.Now.
	Now func() time.Time
	// Clock — доменное «сейчас» для occurred_at решений (AD-37); nil — Now.
	Clock appjournal.DomainClock
	// DomainBuild — хеш доменной сборки для записей-решений (AD-9).
	DomainBuild string
	// Telemetry — показатели ops для /metrics (Collect); nil — не выгружаются.
	Telemetry platform.Telemetry
	Log       *slog.Logger
}

// Service — реализация ведущих портов модуля ops (AD-36): live — над
// журналом, арендами и курсорами (NewLive); без зависимостей — заглушка
// 501 (NewService: выгрузка OpenAPI, тесты контракта).
type Service struct {
	cfg  Config
	live bool

	mu   sync.RWMutex
	self *SelfCheckView
}

// NewService создаёт заглушку: каждая операция отвечает 501.
func NewService() *Service { return &Service{} }

// NewLive создаёт live-реализацию.
func NewLive(cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.DiscardHandler)
	}
	if cfg.VerifierInterval <= 0 {
		cfg.VerifierInterval = 30 * time.Second
	}
	return &Service{cfg: cfg, live: cfg.Journal != nil && cfg.Codec != nil}
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// SetModules задаёт источник режимов ведущих портов модулей (каталог
// операций API собирается после сервиса).
func (s *Service) SetModules(f func() []ModuleMode) {
	s.mu.Lock()
	s.cfg.Modules = f
	s.mu.Unlock()
}

// SetSelfCheck запоминает итог самопроверки этой копии (показывается в Health).
func (s *Service) SetSelfCheck(v SelfCheckView) {
	s.mu.Lock()
	s.self = &v
	s.mu.Unlock()
}

func (s *Service) selfCheck() *SelfCheckView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.self
}

// domainNow — доменное «сейчас» для occurred_at решений (AD-37).
func (s *Service) domainNow(ctx context.Context) time.Time {
	if s.cfg.Clock != nil {
		if t, err := s.cfg.Clock.Now(ctx); err == nil && !t.IsZero() {
			return t.UTC()
		}
	}
	return s.cfg.Now().UTC()
}
