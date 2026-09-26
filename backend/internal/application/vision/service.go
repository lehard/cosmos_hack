package vision

import (
	"context"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
)

// Service — реализация live ведущих портов модуля vision (AD-36): чтение
// паспортов допуска и проверок анализаторов — свёртка записей семейства
// analyzer из журнала на момент (AD-22, domain/vision.Fold), без своих таблиц
// проекций; команды допуска, возврата и вывода — доменный гард над тем же
// реестром и одна запись-решение через journal.Append с проверкой AD-39
// потока паспорта. Без зависимостей (NewService()) — заглушка 501.
type Service struct {
	Unimplemented
	d   Deps
	cfg Config
}

// Deps — зависимости live-реализации.
type Deps struct {
	// Journal — журнал (AD-44): чтение записей analyzer.* и единственная функция записи.
	Journal appjournal.JournalStore
	// Codec — чтение записей журнала в представление домена (AD-20).
	Codec *engineapp.Codec
	// DomainClock — доменное «сейчас» при приёме команды (AD-37).
	DomainClock appjournal.DomainClock
	// Routes — порт «маршрут протокола допуска закрыт» (AD-43); nil — PendingRoutes.
	Routes RouteGate
	// Now — InfraClock для received_at (AD-37); nil — time.Now.
	Now func() time.Time
	// Watch — проекции движка: состояние правила автоотката (vision.watch,
	// контроль дрейфа на странице адаптации); nil — без контроля дрейфа.
	Watch engineapp.ProjectionStore
}

// Config — параметры модуля.
type Config struct {
	// DomainBuild — domain_build записей (AD-9).
	DomainBuild string
	// Partitions — число партиций P (AD-6): записи вне изделия — в партиции P.
	Partitions int
	// ScenarioClock — режим часов scenario: recorded_at решения = доменное «сейчас» (AD-37).
	ScenarioClock bool
}

// Option — настройка Service.
type Option func(*Service)

// WithDeps — зависимости live-реализации.
func WithDeps(d Deps) Option { return func(s *Service) { s.d = d } }

// WithConfig — параметры модуля.
func WithConfig(c Config) Option { return func(s *Service) { s.cfg = c } }

// NewService создаёт реализацию live; без WithDeps — заглушка 501.
func NewService(opts ...Option) *Service {
	s := &Service{}
	for _, o := range opts {
		o(s)
	}
	if s.cfg.Partitions <= 0 {
		s.cfg.Partitions = 1
	}
	if s.d.Routes == nil {
		s.d.Routes = PendingRoutes{}
	}
	if s.d.Now == nil {
		s.d.Now = time.Now
	}
	return s
}

func (s *Service) live() bool { return s.d.Journal != nil && s.d.Codec != nil }

// now — доменное «сейчас» (AD-37).
func (s *Service) now(ctx context.Context) (time.Time, error) {
	if s.d.DomainClock == nil {
		return s.d.Now().UTC(), nil
	}
	t, err := s.d.DomainClock.Now(ctx)
	return t.UTC(), err
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)
