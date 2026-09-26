package notifications

import (
	"encoding/json"
	"fmt"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/notifications"
)

// Проекции модуля notifications (AD-45: один писатель — notifications; пишет
// роль projector эффектами в транзакции Append вместе с курсором;
// пересобираются `ant rebuild`).
const (
	// ProjectionObligation — единственная проекция сроков (AD-4): ключ —
	// obligation_id; её читает планировщик (роль scheduler) и по ней
	// считается цена задержки (FR-8, FR-57).
	ProjectionObligation = "notifications.obligation"
	// ProjectionTask — задачи в потоках изделий и объектов; ключ — task_id.
	ProjectionTask = "notifications.task"
	// ProjectionNotice — уведомления; ключ — notification_id.
	ProjectionNotice = "notifications.notice"
)

// Register подключает проекции notifications к реестру движка (одна строка в
// cmd/ant engineRegistry).
func Register(reg *engineapp.Registry) error {
	gs := []engineapp.GlobalProjection{
		{Name: ProjectionObligation, Writer: dom.Module, Keys: dom.ObligationKeys, Step: step(dom.StepObligation), Entity: entity(platform.EntityNotification)},
		{Name: ProjectionTask, Writer: dom.Module, Keys: dom.TaskKeys, Step: step(dom.StepTask), Entity: entity(platform.EntityTask)},
		{Name: ProjectionNotice, Writer: dom.Module, Keys: dom.NoticeKeys, Step: step(dom.StepNotice), Entity: entity(platform.EntityNotification)},
	}
	for _, g := range gs {
		if err := reg.AddGlobal(g); err != nil {
			return fmt.Errorf("проекции notifications: %w", err)
		}
	}
	return nil
}

// step — шаг глобальной проекции над доменной чистой функцией строки.
func step[T any](f func(T, kernel.Record) T) func(string, json.RawMessage, kernel.Record) (json.RawMessage, error) {
	return func(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
		var v T
		if len(prev) > 0 {
			if err := json.Unmarshal(prev, &v); err != nil {
				return nil, err
			}
		}
		return json.Marshal(f(v, r))
	}
}

// entity — сущность SSE для ключа проекции: клиент перечитывает список.
func entity(kind platform.EntityKind) func(string) (platform.EntityKind, string, bool) {
	return func(key string) (platform.EntityKind, string, bool) { return kind, key, true }
}
