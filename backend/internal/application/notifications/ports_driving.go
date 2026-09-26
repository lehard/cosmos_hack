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
	// Attention — «Требует вашего внимания» (notifications.attention.list, FR-8).
	Attention(ctx context.Context, m platform.Moment) (AttentionList, error)
	// Alerts — лента тревог (notifications.alert.list, FR-8).
	Alerts(ctx context.Context, m platform.Moment, p platform.Page) (AlertList, error)
	// Tasks — задачи и уведомления пользователя (notifications.task.list, FR-57).
	Tasks(ctx context.Context, f TaskFilter, m platform.Moment, p platform.Page) (TaskList, error)
}

// TaskFilter — фильтр задач.
type TaskFilter struct {
	State      string
	LocationID string
}

// Commands — ведущий порт команд модуля notifications (AD-36, AD-39): одна операция —
// одна реализация команды в модуле-владельце.
type Commands interface {
	// AcknowledgeTask — отметить задачу (notifications.task.acknowledge).
	AcknowledgeTask(ctx context.Context, taskID string, in AcknowledgeTask) (platform.Receipt, error)
}

// Unimplemented — заглушка портов notifications: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures, чтобы новые
// операции контракта не ломали сборку (реализация переопределяет метод).
type Unimplemented struct{}

func (Unimplemented) Summary(context.Context, platform.Moment) (NotificationSummary, error) {
	return NotificationSummary{}, platform.NotImplemented("notifications.summary.read")
}

func (Unimplemented) Attention(context.Context, platform.Moment) (AttentionList, error) {
	return AttentionList{}, platform.NotImplemented("notifications.attention.list")
}

func (Unimplemented) Alerts(context.Context, platform.Moment, platform.Page) (AlertList, error) {
	return AlertList{}, platform.NotImplemented("notifications.alert.list")
}

func (Unimplemented) Tasks(context.Context, TaskFilter, platform.Moment, platform.Page) (TaskList, error) {
	return TaskList{}, platform.NotImplemented("notifications.task.list")
}

func (Unimplemented) AcknowledgeTask(context.Context, string, AcknowledgeTask) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("notifications.task.acknowledge")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
