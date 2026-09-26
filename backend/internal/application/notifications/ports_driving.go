package notifications

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля notifications (AD-36): операции чтения API
// опираются только на него. Методы добавляются вместе с операциями контракта.
type Queries interface {
	// Summary — сводка уведомлений для шапки (notifications.summary.read, FR-57).
	Summary(ctx context.Context, m platform.Moment) (NotificationSummary, error)
}

// Commands — ведущий порт команд модуля notifications (AD-36, AD-39): одна операция —
// одна реализация команды в модуле-владельце.
type Commands interface{}

// Unimplemented — заглушка портов notifications: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures, чтобы новые
// операции контракта не ломали сборку (реализация переопределяет метод).
type Unimplemented struct{}

func (Unimplemented) Summary(context.Context, platform.Moment) (NotificationSummary, error) {
	return NotificationSummary{}, platform.NotImplemented("notifications.summary.read")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
