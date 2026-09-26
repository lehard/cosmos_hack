package notifications

import (
	"context"

	engineapp "ant/internal/application/engine"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	notif "ant/internal/domain/notifications"
)

// Bundles — BundleSource движка, дополняющий нормативный слой изделия частью
// модуля notifications (AD-17): описание процесса той же версии (сроки
// исполнителя — окна BPMN и нормы ожидания на точках предъявления, эпик 17) и
// политика сроков (Env; нулевая — умолчания: 3 рабочих дня, календарь пн–пт
// до справочника эпика 19).
type Bundles struct {
	// Next — источник версии; nil — пустой нормативный слой.
	Next engineapp.BundleSource
	// Env — политика сроков и календарь по умолчанию.
	Env notif.Env
}

var _ engineapp.BundleSource = Bundles{}

// Bundle — нормативный слой изделия с частью notifications.
func (b Bundles) Bundle(ctx context.Context, itemID string, input []kernel.Record) (engine.Bundle, string, error) {
	next := b.Next
	if next == nil {
		next = engineapp.EmptyBundles{}
	}
	bd, rev, err := next.Bundle(ctx, itemID, input)
	if err != nil {
		return bd, rev, err
	}
	env := bd.Notifications
	if env.DecisionWorkingDays == 0 && len(env.Ladders) == 0 {
		env = b.Env
	}
	env.Process = bd.Process
	bd.Notifications = env
	return bd, rev, nil
}
