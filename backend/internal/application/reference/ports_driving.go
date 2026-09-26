package reference

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля reference (AD-36).
type Queries interface {
	// ItemTypes — номенклатура (reference.item_type.list).
	ItemTypes(ctx context.Context, m platform.Moment) (RefItemTypeList, error)
	// Locations — места (reference.location.list).
	Locations(ctx context.Context, m platform.Moment) (RefLocationList, error)
	// Equipment — оборудование с поверкой (reference.equipment.list).
	Equipment(ctx context.Context, m platform.Moment) (RefEquipmentList, error)
	// Calendar — производственный календарь (reference.calendar.read).
	Calendar(ctx context.Context, year int, m platform.Moment) (RefCalendar, error)
	// Shifts — смены (reference.shift.list).
	Shifts(ctx context.Context, locationID string, m platform.Moment) (RefShiftList, error)
	// ExternalIDs — соответствия внешних ID (reference.external_id.list).
	ExternalIDs(ctx context.Context, system string, m platform.Moment, p platform.Page) (RefExternalIDList, error)
}

// Commands — ведущий порт команд модуля reference (AD-39).
type Commands interface {
	// DefineItemType — reference.item_type.define.
	DefineItemType(ctx context.Context, in DefineItemType) (platform.Receipt, error)
	// DefineLocation — reference.location.define.
	DefineLocation(ctx context.Context, in DefineLocation) (platform.Receipt, error)
	// DefineEquipment — reference.equipment.define.
	DefineEquipment(ctx context.Context, in DefineEquipment) (platform.Receipt, error)
	// VerifyEquipment — reference.equipment.verify.
	VerifyEquipment(ctx context.Context, equipmentID string, in VerifyEquipment) (platform.Receipt, error)
	// DefineCalendar — reference.calendar.define.
	DefineCalendar(ctx context.Context, in DefineCalendar) (platform.Receipt, error)
	// ScheduleShift — reference.shift.schedule.
	ScheduleShift(ctx context.Context, in ScheduleShift) (platform.Receipt, error)
}

// Unimplemented — заглушка портов reference: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func (Unimplemented) ItemTypes(context.Context, platform.Moment) (RefItemTypeList, error) {
	return RefItemTypeList{}, platform.NotImplemented("reference.item_type.list")
}

func (Unimplemented) Locations(context.Context, platform.Moment) (RefLocationList, error) {
	return RefLocationList{}, platform.NotImplemented("reference.location.list")
}

func (Unimplemented) Equipment(context.Context, platform.Moment) (RefEquipmentList, error) {
	return RefEquipmentList{}, platform.NotImplemented("reference.equipment.list")
}

func (Unimplemented) Calendar(context.Context, int, platform.Moment) (RefCalendar, error) {
	return RefCalendar{}, platform.NotImplemented("reference.calendar.read")
}

func (Unimplemented) Shifts(context.Context, string, platform.Moment) (RefShiftList, error) {
	return RefShiftList{}, platform.NotImplemented("reference.shift.list")
}

func (Unimplemented) ExternalIDs(context.Context, string, platform.Moment, platform.Page) (RefExternalIDList, error) {
	return RefExternalIDList{}, platform.NotImplemented("reference.external_id.list")
}

func (Unimplemented) DefineItemType(context.Context, DefineItemType) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("reference.item_type.define")
}

func (Unimplemented) DefineLocation(context.Context, DefineLocation) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("reference.location.define")
}

func (Unimplemented) DefineEquipment(context.Context, DefineEquipment) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("reference.equipment.define")
}

func (Unimplemented) VerifyEquipment(context.Context, string, VerifyEquipment) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("reference.equipment.verify")
}

func (Unimplemented) DefineCalendar(context.Context, DefineCalendar) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("reference.calendar.define")
}

func (Unimplemented) ScheduleShift(context.Context, ScheduleShift) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("reference.shift.schedule")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
