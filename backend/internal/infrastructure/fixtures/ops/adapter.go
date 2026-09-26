package ops

import (
	"context"

	app "ant/internal/application/ops"
	"ant/internal/application/platform"
)

// Adapter — реализация fixtures ведущих портов модуля ops (AD-36): здоровье
// компонентов, остановленные изделия, настройки — из мира заготовок.
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Health — состояние компонентов (ops.health.read, FR-127).
func (Adapter) Health(ctx context.Context) (app.OpsHealth, error) {
	return respond[app.OpsHealth](ctx, "ops.health.read", nil, nil)
}

// StoppedItems — изделия с остановленной обработкой (ops.stopped_item.list).
func (Adapter) StoppedItems(ctx context.Context, _ platform.Page) (app.StoppedItemList, error) {
	return respond[app.StoppedItemList](ctx, "ops.stopped_item.list", nil, nil)
}

// Settings — настройки (ops.setting.list).
func (Adapter) Settings(ctx context.Context) (app.SettingList, error) {
	return respond[app.SettingList](ctx, "ops.setting.list", nil, nil)
}

// RetryProcessing — повторить обработку изделия (ops.processing.retry).
func (Adapter) RetryProcessing(ctx context.Context, itemID string, in app.RetryProcessing) (platform.Receipt, error) {
	return decide(ctx, "ops.processing.retry", "item", itemID, in.CommandMeta())
}

// DisableSource — отключить источник (ops.source.disable).
func (Adapter) DisableSource(ctx context.Context, sourceID string, in app.SwitchSource) (platform.Receipt, error) {
	return decide(ctx, "ops.source.disable", "quarantine", sourceID, in.CommandMeta())
}

// EnableSource — включить источник (ops.source.enable).
func (Adapter) EnableSource(ctx context.Context, sourceID string, in app.SwitchSource) (platform.Receipt, error) {
	return decide(ctx, "ops.source.enable", "quarantine", sourceID, in.CommandMeta())
}
