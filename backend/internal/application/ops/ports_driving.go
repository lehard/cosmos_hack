package ops

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля ops (AD-36).
type Queries interface {
	// Health — состояние системы (ops.health.read, FR-127).
	Health(ctx context.Context) (OpsHealth, error)
	// StoppedItems — остановленные изделия (ops.stopped_item.list, AD-45).
	StoppedItems(ctx context.Context, p platform.Page) (StoppedItemList, error)
	// Settings — адаптеры портов и режимы модулей (ops.setting.list, AD-35, AD-36).
	Settings(ctx context.Context) (SettingList, error)
}

// Commands — ведущий порт команд модуля ops.
type Commands interface {
	RetryProcessing(ctx context.Context, itemID string, in RetryProcessing) (platform.Receipt, error)
	DisableSource(ctx context.Context, sourceID string, in SwitchSource) (platform.Receipt, error)
	EnableSource(ctx context.Context, sourceID string, in SwitchSource) (platform.Receipt, error)
}

// Unimplemented — заглушка портов ops: каждая операция отвечает 501.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Health(context.Context) (OpsHealth, error) {
	return OpsHealth{}, ni("ops.health.read")
}
func (Unimplemented) StoppedItems(context.Context, platform.Page) (StoppedItemList, error) {
	return StoppedItemList{}, ni("ops.stopped_item.list")
}
func (Unimplemented) Settings(context.Context) (SettingList, error) {
	return SettingList{}, ni("ops.setting.list")
}
func (Unimplemented) RetryProcessing(context.Context, string, RetryProcessing) (platform.Receipt, error) {
	return platform.Receipt{}, ni("ops.processing.retry")
}
func (Unimplemented) DisableSource(context.Context, string, SwitchSource) (platform.Receipt, error) {
	return platform.Receipt{}, ni("ops.source.disable")
}
func (Unimplemented) EnableSource(context.Context, string, SwitchSource) (platform.Receipt, error) {
	return platform.Receipt{}, ni("ops.source.enable")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
