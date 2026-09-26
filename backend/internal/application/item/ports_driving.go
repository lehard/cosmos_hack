package item

import (
	"context"

	"ant/internal/application/platform"
)

// ItemFilter — фильтр списка изделий.
type ItemFilter struct {
	StepKey string
	Summary string
	LotID   string
	OrderID string
}

// Queries — ведущий порт чтения модуля item (AD-36).
type Queries interface {
	// Lookup — изделие по номеру детали или содержимому DataMatrix (item.item.lookup):
	// разрешение носителя на момент (AD-41); не найдено — api.not_found.
	Lookup(ctx context.Context, q string, m platform.Moment) (ItemLookup, error)
	// List — изделия по шагу, статусу, партии, заданию (item.item.list).
	List(ctx context.Context, f ItemFilter, m platform.Moment, p platform.Page) (ItemList, error)
	// Passport — паспорт изделия (item.passport.read, FR-42).
	Passport(ctx context.Context, itemID string, m platform.Moment) (ItemPassport, error)
	// History — журнал изменений паспорта (item.history.list, FR-43).
	History(ctx context.Context, itemID string, m platform.Moment, p platform.Page) (ItemHistory, error)
	// Genealogy — генеалогия изделия через порт стадии (item.genealogy.read, FR-45, AD-42).
	Genealogy(ctx context.Context, itemID string, m platform.Moment) (ItemGenealogy, error)
}

// Commands — ведущий порт команд модуля item (AD-39).
type Commands interface {
	Register(ctx context.Context, in RegisterItem) (platform.Receipt, error)
	ApplyCarrier(ctx context.Context, itemID string, in ApplyCarrier) (platform.Receipt, error)
	RemoveCarrier(ctx context.Context, itemID string, in RemoveCarrier) (platform.Receipt, error)
	RecordPresentation(ctx context.Context, itemID string, in RecordPresentation) (platform.Receipt, error)
	OpenIntervention(ctx context.Context, itemID string, in OpenIntervention) (platform.Receipt, error)
	CloseIntervention(ctx context.Context, itemID, interventionID string, in CloseIntervention) (platform.Receipt, error)
	ConfirmIdentification(ctx context.Context, itemID string, in ConfirmIdentification) (platform.Receipt, error)
	RecordAssembly(ctx context.Context, itemID string, in RecordAssembly) (platform.Receipt, error)
	RecordRelease(ctx context.Context, itemID string, in RecordRelease) (platform.Receipt, error)
	// Split — разделение 1→N с переносом происхождения (item.item.split, FR-15).
	Split(ctx context.Context, itemID string, in SplitItem) (platform.Receipt, error)
}

// Unimplemented — заглушка портов item: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Lookup(context.Context, string, platform.Moment) (ItemLookup, error) {
	return ItemLookup{}, ni("item.item.lookup")
}
func (Unimplemented) List(context.Context, ItemFilter, platform.Moment, platform.Page) (ItemList, error) {
	return ItemList{}, ni("item.item.list")
}
func (Unimplemented) Passport(context.Context, string, platform.Moment) (ItemPassport, error) {
	return ItemPassport{}, ni("item.passport.read")
}
func (Unimplemented) History(context.Context, string, platform.Moment, platform.Page) (ItemHistory, error) {
	return ItemHistory{}, ni("item.history.list")
}
func (Unimplemented) Genealogy(context.Context, string, platform.Moment) (ItemGenealogy, error) {
	return ItemGenealogy{}, ni("item.genealogy.read")
}
func (Unimplemented) Register(context.Context, RegisterItem) (platform.Receipt, error) {
	return platform.Receipt{}, ni("item.item.register")
}
func (Unimplemented) ApplyCarrier(context.Context, string, ApplyCarrier) (platform.Receipt, error) {
	return platform.Receipt{}, ni("item.carrier.apply")
}
func (Unimplemented) RemoveCarrier(context.Context, string, RemoveCarrier) (platform.Receipt, error) {
	return platform.Receipt{}, ni("item.carrier.remove")
}
func (Unimplemented) RecordPresentation(context.Context, string, RecordPresentation) (platform.Receipt, error) {
	return platform.Receipt{}, ni("item.presentation.record")
}
func (Unimplemented) OpenIntervention(context.Context, string, OpenIntervention) (platform.Receipt, error) {
	return platform.Receipt{}, ni("item.intervention.open")
}
func (Unimplemented) CloseIntervention(context.Context, string, string, CloseIntervention) (platform.Receipt, error) {
	return platform.Receipt{}, ni("item.intervention.close")
}
func (Unimplemented) ConfirmIdentification(context.Context, string, ConfirmIdentification) (platform.Receipt, error) {
	return platform.Receipt{}, ni("item.identification.confirm")
}
func (Unimplemented) RecordAssembly(context.Context, string, RecordAssembly) (platform.Receipt, error) {
	return platform.Receipt{}, ni("item.assembly.record")
}
func (Unimplemented) RecordRelease(context.Context, string, RecordRelease) (platform.Receipt, error) {
	return platform.Receipt{}, ni("item.release.record")
}
func (Unimplemented) Split(context.Context, string, SplitItem) (platform.Receipt, error) {
	return platform.Receipt{}, ni("item.item.split")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
