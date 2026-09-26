package reference

import (
	"time"

	"ant/internal/application/platform"
)

// Справочники, от которых зависят права и предусловия, не меняются фактами
// server-attested (AD-2): их меняют решения администратора с подписью.

// DefineItemType — позиция номенклатуры (reference.item_type.defined).
type DefineItemType struct {
	platform.CommandHeader
	RefItemType
}

// DefineLocation — место (reference.location.defined).
type DefineLocation struct {
	platform.CommandHeader
	RefLocation
}

// DefineEquipment — оборудование (reference.equipment.defined).
type DefineEquipment struct {
	platform.CommandHeader
	EquipmentID           string     `json:"equipment_id" maxLength:"128"`
	Kind                  string     `json:"kind" enum:"cnc_machine,welding_source,camera,cmm,leak_tester,torque_wrench,xray,test_bench,other"`
	LocationID            string     `json:"location_id" maxLength:"128"`
	Name                  string     `json:"name" maxLength:"256"`
	IsMeasuringInstrument bool       `json:"is_measuring_instrument"`
	SourceID              string     `json:"source_id,omitempty" maxLength:"128"`
	ValidFrom             time.Time  `json:"valid_from"`
	ValidUntil            *time.Time `json:"valid_until,omitempty"`
}

// VerifyEquipment — поверка или калибровка (reference.equipment.verified; метролог).
type VerifyEquipment struct {
	platform.CommandHeader
	Kind           string `json:"kind" enum:"verification,calibration"`
	Result         string `json:"result" enum:"valid,invalid"`
	ValidUntil     string `json:"valid_until,omitempty" pattern:"^[0-9]{4}-[0-9]{2}-[0-9]{2}$" doc:"Дата YYYY-MM-DD."`
	CertificateRef string `json:"certificate_ref,omitempty" maxLength:"256"`
}

// DefineCalendar — производственный календарь (reference.calendar.defined).
type DefineCalendar struct {
	platform.CommandHeader
	RefCalendar
}

// ScheduleShift — смена (reference.shift.scheduled).
type ScheduleShift struct {
	platform.CommandHeader
	RefShift
}
