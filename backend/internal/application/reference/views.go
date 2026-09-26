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
	// Usable — оборудование можно использовать на момент ответа (FR-17):
	// есть в справочнике, у средства измерений действует поверка.
	Usable bool `json:"usable" doc:"FR-17: можно использовать на момент ответа (поверка действует)."`
	// UnusableReason — почему нельзя: unknown | not_verified | verification_invalid | verification_expired.
	UnusableReason string `json:"unusable_reason,omitempty" doc:"unknown — нет в справочнике; not_verified — средство измерений без поверки; verification_invalid — непригодно; verification_expired — срок поверки истёк."`
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
	WorkingDays    []string `json:"working_days,omitempty" doc:"Рабочие дни на еженедельных выходных (перенос), YYYY-MM-DD."`
}

// RefShift — смена.
type RefShift struct {
	ShiftID    string    `json:"shift_id"`
	LocationID string    `json:"location_id"`
	Name       string    `json:"name,omitempty"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	// RepeatUntil, WorkingDaysOnly — шаблон: смена повторяется каждый
	// (рабочий) день в то же местное время до RepeatUntil.
	RepeatUntil     *time.Time `json:"repeat_until,omitempty" doc:"Шаблон: повторять каждый день, пока начало не позже этого момента."`
	WorkingDaysOnly bool       `json:"working_days_only,omitempty" doc:"Шаблон: только рабочие дни производственного календаря."`
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
	Effective  bool      `json:"effective,omitempty" doc:"Действующее соответствие ключа (первое по порядку записи); при конфликте остальные только видны."`
}

// RefExternalIDList — соответствия внешних ID.
type RefExternalIDList struct {
	Items      []RefExternalID `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

// RefLot — партия со сроком годности (erp.lot.received; FR-17: сроки годности
// материалов и их партий).
type RefLot struct {
	LotID          string     `json:"lot_id"`
	ItemTypeID     string     `json:"item_type_id"`
	SupplierID     string     `json:"supplier_id"`
	HeatNo         string     `json:"heat_no,omitempty"`
	Quantity       int        `json:"quantity"`
	CertificateNo  string     `json:"certificate_no,omitempty"`
	ExpiryDate     string     `json:"expiry_date,omitempty" doc:"Срок годности YYYY-MM-DD (включительно, местная дата)."`
	Usable         bool       `json:"usable" doc:"Срок годности не истёк на момент ответа."`
	ExternalSystem string     `json:"external_system"`
	ExternalNumber string     `json:"external_number"`
	ReceivedAt     time.Time  `json:"received_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty" doc:"Момент окончания годности (начало следующих местных суток)."`
}

// RefLotList — партии.
type RefLotList struct {
	Items []RefLot `json:"items"`
}

// RefOrder — производственное задание учётной системы (erp.order.received).
type RefOrder struct {
	OrderID        string    `json:"order_id"`
	ExternalSystem string    `json:"external_system"`
	ExternalNumber string    `json:"external_number"`
	ItemTypeID     string    `json:"item_type_id"`
	ItemRevision   string    `json:"item_revision,omitempty"`
	Quantity       int       `json:"quantity"`
	DueDate        string    `json:"due_date,omitempty" doc:"Срок YYYY-MM-DD."`
	ReceivedAt     time.Time `json:"received_at"`
}

// RefOrderList — задания.
type RefOrderList struct {
	Items []RefOrder `json:"items"`
}
