package platform

import (
	"context"
	"time"
)

// PortKey — ключ конфигурации ведомого порта (AD-35): у каждого порта ключ и
// общий контрактный тест для всех адаптеров. Значение ключа — имя адаптера.
type PortKey string

// Ключи конфигурации портов платформы (deploy/config/ant.yaml, секция ports.adapters).
const (
	PortJournalStore     PortKey = "journal_store"     // postgres | (bft — описание)
	PortLeaseStore       PortKey = "lease_store"       // postgres | (k8s — описание)
	PortMaterialStore    PortKey = "material_store"    // volume | (s3 — описание)
	PortWorkFeed         PortKey = "work_feed"         // postgres | (kafka — описание)
	PortPublisher        PortKey = "publisher"         // http | (broker — описание)
	PortTelemetry        PortKey = "telemetry"         // prometheus | (otlp — описание)
	PortSigner           PortKey = "signer"            // token_agent | demo_signer | (pkcs11 — описание)
	PortVerifier         PortKey = "verifier"          // gogost | (skzi — описание)
	PortCipher           PortKey = "cipher"            // gogost | (skzi — описание)
	PortAccessControl    PortKey = "access_control"    // permissive (волна 1) | casbin
	PortIdentityProvider PortKey = "identity_provider" // demo (волна 1) | local | (ldap — описание)
	PortDomainClock      PortKey = "domain_clock"      // system | journal (режим сценария)
	PortInfraClock       PortKey = "infra_clock"       // system
	PortFixtureCursor    PortKey = "fixture_cursor"    // postgres | memory
	PortConsumer         PortKey = "consumer"          // postgres
)

// PortKeys — все ключи портов платформы в порядке таблицы AD-35.
var PortKeys = []PortKey{
	PortJournalStore, PortLeaseStore, PortMaterialStore, PortWorkFeed, PortPublisher,
	PortTelemetry, PortSigner, PortVerifier, PortCipher, PortAccessControl,
	PortIdentityProvider, PortDomainClock, PortInfraClock, PortFixtureCursor, PortConsumer,
}

// Publisher — ведомый порт исходящих сообщений внешним системам (AD-35, AD-18):
// HTTP с повтором в MVP, брокер — замена. Ключ идемпотентности — бизнес-ключ (AD-7).
// Отправляет только роль outbox; при воспроизведении роль не запущена.
type Publisher interface {
	// Publish отправляет сообщение адресату; повтор с тем же ключом не дублирует.
	Publish(ctx context.Context, msg OutboundMessage) (Delivery, error)
}

// OutboundMessage — исходящее сообщение.
type OutboundMessage struct {
	// System — внешняя система (onec, galaktika, mes, partner:‹код›).
	System string
	// BusinessKey — субъект, учётное действие, закрывающая точка (AD-7).
	BusinessKey string
	// ContentType, Body — тело в протоколе внешней системы (готовит адаптер integration).
	ContentType string
	Body        []byte
}

// Delivery — результат отправки: квитанция или транспортная ошибка.
type Delivery struct {
	Accepted  bool
	Receipt   []byte
	Transport string
}

// Telemetry — ведомый порт наблюдаемости (AD-35): Prometheus /metrics в MVP,
// OTLP — замена. Метрики — операционные, не проекции (AD-7).
type Telemetry interface {
	// Counter увеличивает счётчик name с метками (пары ключ, значение).
	Counter(name string, delta int64, labels ...string)
	// Observe записывает наблюдение гистограммы (секунды, байты…).
	Observe(name string, value time.Duration, labels ...string)
	// Gauge устанавливает значение показателя.
	Gauge(name string, value int64, labels ...string)
}

// FixtureCursor — ведомый порт курсора сценария заготовок (AD-36, AD-38):
// общий для копий api (адаптер infrastructure/storage/fixtures, схема fixtures).
// Момент as_of на заготовках — шаг курсора.
type FixtureCursor interface {
	// Current — текущее положение курсора.
	Current(ctx context.Context) (CursorState, error)
	// Move ставит курсор на шаг step сценария scenario (пауза — Paused).
	Move(ctx context.Context, to CursorState) error
}

// CursorState — положение курсора заготовок.
type CursorState struct {
	Scenario string
	RunID    string
	Step     int
	Paused   bool
	Speed    int
	// ClockAt — доменное «сейчас» шага (виртуальные часы сценария, AD-37).
	ClockAt time.Time
}
