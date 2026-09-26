package nonconformity

import (
	"context"
	"sync"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/domain/engine"
	dom "ant/internal/domain/nonconformity"
)

// Service — реализация live ведущих портов модуля nonconformity (AD-36):
// сценарии приложения над доменом и журналом. Чтение — та же свёртка
// изделия, что у воркера, над входом из журнала на момент (AD-22); команды —
// доменный гард над состоянием изделия, построенным из журнала (а не из
// отстающей проекции, AD-39), и одна запись-решение через journal.Append с
// проверками AD-39 (поток изделия, поток разрешения, атомарный расход лимита).
// Без зависимостей (NewService()) — заглушка 501, как в волне 1.
type Service struct {
	Unimplemented
	d   Deps
	cfg Config

	// ncItems — несоответствие → изделие (соответствие неизменно: id
	// несоответствия детерминирован по изделию, AD-4).
	ncItems sync.Map
}

// Deps — зависимости live-реализации.
type Deps struct {
	// Journal — журнал (AD-44): чтение входа и единственная функция записи.
	Journal appjournal.JournalStore
	// Codec — чтение записей журнала в представление домена (engine, AD-20).
	Codec *engineapp.Codec
	// Bundles — нормативный слой изделия (AD-17); nil — EmptyBundles.
	Bundles engineapp.BundleSource
	// Fold — свёртка изделия; nil — domain/engine.Fold (тесты подставляют
	// свёртку с моделью модуля quality).
	Fold engine.Folder
	// DomainClock — доменное «сейчас» при приёме команды (AD-37).
	DomainClock appjournal.DomainClock
	// Routes — порт «маршрут подписей закрыт» (AD-43); nil — DemoRoutes.
	Routes RouteGate
	// Calendar — производственный календарь (FR-55); nil — WeekdayCalendar.
	Calendar Calendar
	// Now — InfraClock для received_at (AD-37); nil — time.Now.
	Now func() time.Time
}

// Config — параметры модуля.
type Config struct {
	// DomainBuild — domain_build записей (AD-9).
	DomainBuild string
	// Partitions — число партиций P (AD-6).
	Partitions int
	// IsolationWorkingDays — срок решения по изолированному изделию в рабочих
	// днях (FR-55: по умолчанию 3).
	IsolationWorkingDays int
	// ScenarioClock — режим часов scenario: recorded_at решения = доменное
	// «сейчас» (AD-37).
	ScenarioClock bool
	// Env — нормативная часть модуля, если движок её не передаёт (до эпика 17).
	Env dom.Env
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
	if s.cfg.IsolationWorkingDays <= 0 {
		s.cfg.IsolationWorkingDays = 3
	}
	if s.cfg.Partitions <= 0 {
		s.cfg.Partitions = 1
	}
	if s.d.Fold == nil {
		s.d.Fold = engine.Fold
	}
	if s.d.Bundles == nil {
		s.d.Bundles = engineapp.EmptyBundles{}
	}
	if s.d.Routes == nil {
		s.d.Routes = DemoRoutes{}
	}
	if s.d.Calendar == nil {
		s.d.Calendar = WeekdayCalendar{}
	}
	if s.d.Now == nil {
		s.d.Now = time.Now
	}
	return s
}

func (s *Service) live() bool { return s.d.Journal != nil && s.d.Codec != nil }

// now — доменное «сейчас» (AD-37) в прогоне runID (AD-38).
func (s *Service) now(ctx context.Context, runID string) (time.Time, error) {
	if s.d.DomainClock == nil {
		return s.d.Now().UTC(), nil
	}
	if runID != "" {
		ctx = appjournal.WithRun(ctx, runID)
	}
	t, err := s.d.DomainClock.Now(ctx)
	return t.UTC(), err
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)
