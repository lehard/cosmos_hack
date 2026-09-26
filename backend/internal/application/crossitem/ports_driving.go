package crossitem

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля crossitem (AD-36).
type Queries interface {
	// Lots — партии (FR-15) (crossitem.lot.list).
	Lots(ctx context.Context, status string, m platform.Moment, p platform.Page) (LotList, error)
	// Lot — карточка партии (crossitem.lot.read).
	Lot(ctx context.Context, lotID string, m platform.Moment) (LotCard, error)
	// Groups — временные группы изделий (crossitem.group.list).
	Groups(ctx context.Context, m platform.Moment) (ItemGroupList, error)
}

// Commands — ведущий порт команд модуля crossitem (AD-39).
type Commands interface {
	// RegisterLot — crossitem.lot.register.
	RegisterLot(ctx context.Context, lotID string, in RegisterLot) (platform.Receipt, error)
	// IssueLot — crossitem.lot.issue.
	IssueLot(ctx context.Context, lotID string, in IssueLot) (platform.Receipt, error)
	// AssignBinding — crossitem.binding.assign.
	AssignBinding(ctx context.Context, in AssignBinding) (platform.Receipt, error)
}

// Unimplemented — заглушка портов crossitem: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func (Unimplemented) Lots(context.Context, string, platform.Moment, platform.Page) (LotList, error) {
	return LotList{}, platform.NotImplemented("crossitem.lot.list")
}

func (Unimplemented) Lot(context.Context, string, platform.Moment) (LotCard, error) {
	return LotCard{}, platform.NotImplemented("crossitem.lot.read")
}

func (Unimplemented) Groups(context.Context, platform.Moment) (ItemGroupList, error) {
	return ItemGroupList{}, platform.NotImplemented("crossitem.group.list")
}

func (Unimplemented) RegisterLot(context.Context, string, RegisterLot) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("crossitem.lot.register")
}

func (Unimplemented) IssueLot(context.Context, string, IssueLot) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("crossitem.lot.issue")
}

func (Unimplemented) AssignBinding(context.Context, AssignBinding) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("crossitem.binding.assign")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
