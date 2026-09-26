package ops

import (
	"context"

	dom "ant/internal/domain/ops"
)

// Ведомые порты модуля ops. Модуль не читает чужих таблиц (AD-1): аренды и
// курсоры отдаёт хранение журнала, очереди исходящих и каналы — модуль-владелец
// интеграции, карантин — приём; всё это собирает cmd/ant.

// Runtime — аренды ролей и курсоры потребителей журнала (AD-6, AD-45).
type Runtime interface {
	// Leases — все аренды, включая истёкшие.
	Leases(ctx context.Context) ([]dom.Lease, error)
	// Backlogs — курсоры воркера по партициям 0…partitions-1 и глобальных
	// потребителей с головой и числом записей после курсора.
	Backlogs(ctx context.Context, partitions int) ([]Backlog, error)
}

// Backlog — курсор потребителя и отставание: Head — голова того, что он
// читает (глобальный — основная цепочка, воркер — последний триггер партиции).
type Backlog struct {
	Consumer  string
	Partition int
	Seq       int64
	Head      int64
	Pending   int64
}

// Exchange — очереди исходящих и каналы обмена модуля-владельца интеграции
// (erp — 1С и Галактика; mes — MES): операционное состояние роли outbox.
type Exchange interface {
	Exchange(ctx context.Context) ([]ExchangeQueue, []dom.Channel, error)
}

// ExchangeQueue — очередь исходящих одной системы.
type ExchangeQueue struct {
	System      string
	Queued      int64
	Quarantined int64
	// Consumer — потребитель журнала, наполняющий очередь (его отставание — отставание очереди).
	Consumer string
}

// Quarantine — открытый карантин приёма (FR-30, FR-41).
type Quarantine interface {
	QuarantineOpen(ctx context.Context) (int64, error)
}

// Database — проверки базы для состояния и самопроверки (AD-1).
type Database interface {
	// Ping — база отвечает.
	Ping(ctx context.Context) error
	// Migrations — состояние миграций модулей сборки.
	Migrations(ctx context.Context) []dom.Migration
	// DBRoles — отсутствующие роли БД и проверка их прав на журнал.
	DBRoles(ctx context.Context) (missing []string, privs []dom.Privilege, err error)
}
