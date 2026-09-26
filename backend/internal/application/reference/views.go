package reference

import "time"

// Справочники — версионируемые данные журнала с датой действия (AD-31):
// ответы — срез на момент (axis, as_of).

// RefZone — зона изделия по КД.
type RefZone struct {
	ZoneID string `json:"zone_id"`
	Name   string `json:"name"`
	Kind   string `json:"kind,omitempty"`
}

// RefComponent — состав изделия.
type RefComponent struct {
	ItemTypeID string `json:"item_type_id"`
	Quantity   int    `json:"quantity" minimum:"1"`
	LotTracked bool   `json:"lot_tracked"`
}

// RefItemType — номенклатура: обозначение, ревизия, зоны, состав, маркировка.
type RefItemType struct {
	ItemTypeID  string         `json:"item_type_id"`
	Designation string         `json:"designation"`
	Name        string         `json:"name"`
	Revision    string         `json:"revision"`
	Zones       []RefZone      `json:"zones"`
	Components  []RefComponent `json:"components"`
	Marking     string         `json:"marking,omitempty" doc:"Ожидаемый носитель идентификатора (AD-16)."`
	ValidFrom   time.Time      `json:"valid_from"`
	ValidUntil  *time.Time     `json:"valid_until,omitempty"`
}

// RefItemTypeList — номенклатура.
type RefItemTypeList struct {
	Items []RefItemType `json:"items"`
}

// RefLocation — место: здание, цех, линия, участок, рабочее место, склад, изолятор.
type RefLocation struct {
	LocationID   string     `json:"location_id"`
	Kind         string     `json:"kind" enum:"building,workshop,line,station,workplace,warehouse,isolator,storage"`
	ParentID     string     `json:"parent_id,omitempty"`
	Scope        string     `json:"scope" doc:"Путь области для прав (AD-15)."`
	Name         string     `json:"name"`
	WarehouseID  string     `json:"warehouse_id,omitempty"`
	AccessZoneID string     `json:"access_zone_id,omitempty" doc:"Зона СКУД."`
	ValidFrom    time.Time  `json:"valid_from"`
	ValidUntil   *time.Time `json:"valid_until,omitempty"`
}

// RefLocationList — места.
type RefLocationList struct {
	Items []RefLocation `json:"items"`
}

// RefEquipment — оборудование с поверкой.
type RefEquipment struct {
	EquipmentID           string     `json:"equipment_id"`
	Kind                  string     `json:"kind" enum:"cnc_machine,welding_source,camera,cmm,leak_tester,torque_wrench,xray,test_bench,other"`
	LocationID            string     `json:"location_id"`
	Name                  string     `json:"name"`
	IsMeasuringInstrument bool       `json:"is_measuring_instrument"`
	SourceID              string     `json:"source_id,omitempty" doc:"Источник событий (edge-агент)."`
	VerificationResult    string     `json:"verification_result,omitempty" enum:"valid,invalid"`
	VerifiedUntil         *time.Time `json:"verified_until,omitempty"`
	CertificateRef        string     `json:"certificate_ref,omitempty"`
}

// RefEquipmentList — оборудование.
type RefEquipmentList struct {
	Items []RefEquipment `json:"items"`
}

// RefCalendar — производственный календарь года.
type RefCalendar struct {
	CalendarID     string   `json:"calendar_id"`
	Year           int      `json:"year"`
	WeeklyDaysOff  []string `json:"weekly_days_off" enum:"mon,tue,wed,thu,fri,sat,sun"`
	NonWorkingDays []string `json:"non_working_days" doc:"Даты YYYY-MM-DD."`
	ShortenedDays  []string `json:"shortened_days" doc:"Даты YYYY-MM-DD."`
}

// RefShift — смена.
type RefShift struct {
	ShiftID    string    `json:"shift_id"`
	LocationID string    `json:"location_id"`
	Name       string    `json:"name,omitempty"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
}

// RefShiftList — смены.
type RefShiftList struct {
	Items []RefShift `json:"items"`
}

// RefExternalID — соответствие внешнего ID внутреннему (reference.external_id.mapped, AD-18).
type RefExternalID struct {
	System     string    `json:"system" enum:"onec,galaktika,mes,kompas,skud,partner,other"`
	ObjectKind string    `json:"object_kind" enum:"item_type,order,lot,supplier,warehouse,item,person,equipment,contract"`
	ExternalID string    `json:"external_id"`
	InternalID string    `json:"internal_id"`
	MappedAt   time.Time `json:"mapped_at"`
	Conflict   bool      `json:"conflict" doc:"Конфликт соответствий — сигнал, не перезапись."`
}

// RefExternalIDList — соответствия внешних ID.
type RefExternalIDList struct {
	Items      []RefExternalID `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}
