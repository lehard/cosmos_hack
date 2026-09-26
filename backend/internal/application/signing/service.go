package signing

import (
	"context"
	"time"

	"ant/internal/application/journal"
	dom "ant/internal/domain/signing"
)

// Config — настройки модуля signing.
type Config struct {
	// Profile — профиль окружения: demo | fixtures | load | prod.
	Profile string
	// AllowUnsigned — неподписанная команда допустима с пометкой «подпись не
	// проверялась» (профили demo и fixtures: демо без агента токена, Д-28, Д-30).
	AllowUnsigned bool
	// Partitions — P движка: записи вне изделия — в партиции P (как у приёма).
	Partitions int
	// DomainBuild — domain_build записей (AD-9).
	DomainBuild string
}

// SourceAPI — source_id актов, принятых операциями API модуля.
const SourceAPI = "ant-api"

// Deps — ведомые порты модуля.
type Deps struct {
	Journal  journal.JournalStore
	Registry *Registry
	// Crypto — проверка подписей (profiles.Verifier над Registry).
	Crypto      Crypto
	Authorities Authorities
	Scans       Scans
	QR          QRReader
	QRWriter    QRWriter
	Alerts      Alerts
	// Now — InfraClock (AD-37): криптографические моменты — по committed_at.
	Now func() time.Time
}

// Service — реализация live ведущих портов модуля signing (AD-36): реестр
// ключей и профилей — свёртка журнала (Registry), акты — записи key.* через
// journal.Append, проверка подписей команд — CheckCommand для всех модулей.
// Без Deps.Registry операции отвечают 501 (Unimplemented).
type Service struct {
	Unimplemented
	cfg Config
	d   Deps
}

// Option — настройка Service.
type Option func(*Service)

// WithConfig задаёт настройки.
func WithConfig(c Config) Option { return func(s *Service) { s.cfg = c } }

// WithDeps подключает ведомые порты.
func WithDeps(d Deps) Option { return func(s *Service) { s.d = d } }

// NewService создаёт реализацию live.
func NewService(opts ...Option) *Service {
	s := &Service{}
	for _, o := range opts {
		o(s)
	}
	if s.d.Now == nil {
		s.d.Now = time.Now
	}
	if s.cfg.Partitions <= 0 {
		s.cfg.Partitions = 16
	}
	if s.d.Registry == nil && s.d.Journal != nil {
		s.d.Registry = NewRegistry(s.d.Journal, nil, nil)
	}
	return s
}

// UseCrypto подключает проверку подписей, которой нужен сам реестр
// (profiles.Verifier{Keys: s.Registry()}) — сборка в cmd/*.
func (s *Service) UseCrypto(c Crypto) { s.d.Crypto = c }

// UseAlerts подключает шину безопасности (мост JournalKeyAlerts или порт эпика 29).
func (s *Service) UseAlerts(a Alerts) { s.d.Alerts = a }

// Registry — реестр ключей модуля (открытые ключи для порта Verifier).
func (s *Service) Registry() *Registry { return s.d.Registry }

// Ready — модуль подключён к журналу и реестру.
func (s *Service) Ready() bool { return s.d.Registry != nil }

func (s *Service) now() time.Time { return s.d.Now().UTC() }

// head — последняя позиция основной цепочки (момент проверки команды).
func (s *Service) head(ctx context.Context) (int64, error) {
	if s.d.Journal == nil {
		return 0, nil
	}
	h, err := s.d.Journal.Head(ctx)
	return h.MainSeq, err
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// SignatureModeFor — допустима ли неподписанная команда в профиле окружения.
func SignatureModeFor(profile string) bool {
	return profile == "demo" || profile == "fixtures" || profile == ""
}

// Level1Actions — перечень действий уровня подписи 1 (для агента токена и столов).
func Level1Actions() []string { return dom.Level1Actions() }
