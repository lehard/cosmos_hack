package reference

import (
	"time"

	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/reference"
)

// Представления записей справочника в ответах API и данные записей из команд.

func itemTypeView(x dom.ItemType) RefItemType {
	v := RefItemType{ItemTypeID: string(x.Data.ItemTypeID), Designation: x.Data.Designation, Name: x.Data.Name,
		Revision: x.Data.Revision, Zones: []RefZone{}, Components: []RefComponent{}, ValidFrom: x.From, ValidUntil: x.Until}
	for _, z := range x.Data.Zones {
		v.Zones = append(v.Zones, RefZone{ZoneID: string(z.ZoneID), Name: z.Name, Kind: string(z.Kind)})
	}
	for _, c := range x.Data.Components {
		v.Components = append(v.Components, RefComponent{ItemTypeID: string(c.ItemTypeID), Quantity: c.Quantity, LotTracked: c.LotTracked})
	}
	if x.Data.Marking != nil {
		v.Marking = string(*x.Data.Marking)
	}
	return v
}

func locationView(x dom.Location) RefLocation {
	v := RefLocation{LocationID: string(x.Data.LocationID), Kind: string(x.Data.Kind), Scope: x.Data.Scope, Name: x.Data.Name,
		ValidFrom: x.From, ValidUntil: x.Until}
	if x.Data.ParentID != nil {
		v.ParentID = string(*x.Data.ParentID)
	}
	if x.Data.WarehouseID != nil {
		v.WarehouseID = string(*x.Data.WarehouseID)
	}
	if x.Data.AccessZoneID != nil {
		v.AccessZoneID = string(*x.Data.AccessZoneID)
	}
	return v
}

func equipmentView(st dom.EquipmentStatus) RefEquipment {
	e := st.Equipment.Data
	v := RefEquipment{EquipmentID: string(e.EquipmentID), Kind: string(e.Kind), LocationID: string(e.LocationID), Name: e.Name,
		IsMeasuringInstrument: e.IsMeasuringInstrument, Usable: st.Valid, UnusableReason: st.Reason}
	if e.SourceID != nil {
		v.SourceID = string(*e.SourceID)
	}
	if st.Verification != nil {
		v.VerificationResult = string(st.Verification.Data.Result)
		if st.Verification.Data.CertificateRef != nil {
			v.CertificateRef = *st.Verification.Data.CertificateRef
		}
	}
	if st.VerifiedUntil != nil {
		// Последняя миллисекунда дня valid_until по местному времени: в
		// интерфейсе — та самая дата «действует до».
		u := st.VerifiedUntil.Add(-time.Millisecond)
		v.VerifiedUntil = &u
	}
	return v
}

func calendarView(v dom.CalendarYearView) RefCalendar {
	out := RefCalendar{CalendarID: v.CalendarID, Year: v.Year, WeeklyDaysOff: []string{}, NonWorkingDays: []string{}, ShortenedDays: []string{}}
	codes := map[time.Weekday]string{time.Monday: "mon", time.Tuesday: "tue", time.Wednesday: "wed", time.Thursday: "thu",
		time.Friday: "fri", time.Saturday: "sat", time.Sunday: "sun"}
	for _, d := range v.DaysOff {
		out.WeeklyDaysOff = append(out.WeeklyDaysOff, codes[d])
	}
	out.NonWorkingDays = append(out.NonWorkingDays, v.NonWorking...)
	out.ShortenedDays = append(out.ShortenedDays, v.Shortened...)
	out.WorkingDays = append(out.WorkingDays, v.Working...)
	return out
}

func lotView(l dom.Lot, usable bool) RefLot {
	d := l.Data
	v := RefLot{LotID: string(d.LotID), ItemTypeID: string(d.ItemTypeID), SupplierID: string(d.SupplierID), Quantity: d.Quantity,
		Usable: usable, ExternalSystem: string(d.ExternalSystem), ExternalNumber: d.ExternalNumber, ReceivedAt: l.At}
	if d.HeatNo != nil {
		v.HeatNo = *d.HeatNo
	}
	if d.CertificateNo != nil {
		v.CertificateNo = *d.CertificateNo
	}
	if d.ExpiryDate != nil {
		v.ExpiryDate = string(*d.ExpiryDate)
		if t, err := time.ParseInLocation(dom.DateLayout, v.ExpiryDate, dom.Local); err == nil {
			end := t.AddDate(0, 0, 1).UTC()
			v.ExpiresAt = &end
		}
	}
	return v
}

func orderView(o dom.Order) RefOrder {
	d := o.Data
	v := RefOrder{OrderID: string(d.OrderID), ExternalSystem: string(d.ExternalSystem), ExternalNumber: d.ExternalNumber,
		ItemTypeID: string(d.ItemTypeID), Quantity: d.Quantity, ReceivedAt: o.At}
	if d.ItemRevision != nil {
		v.ItemRevision = *d.ItemRevision
	}
	if d.DueDate != nil {
		v.DueDate = string(*d.DueDate)
	}
	return v
}

// ── данные записей из команд ──

func ts(t time.Time) ev.Timestamp { return ev.NewTimestamp(t) }

func tsp(t *time.Time) *ev.Timestamp {
	if t == nil {
		return nil
	}
	x := ev.NewTimestamp(*t)
	return &x
}

func oid(s string) *ev.ObjectID {
	if s == "" {
		return nil
	}
	x := ev.ObjectID(s)
	return &x
}

func strp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func itemTypeData(v RefItemType) ev.ReferenceItemTypeDefinedV1 {
	d := ev.ReferenceItemTypeDefinedV1{ItemTypeID: ev.ObjectID(v.ItemTypeID), Designation: v.Designation, Name: v.Name,
		Revision: v.Revision, ValidFrom: ts(v.ValidFrom), ValidUntil: tsp(v.ValidUntil)}
	for _, z := range v.Zones {
		d.Zones = append(d.Zones, ev.ZoneDef{ZoneID: ev.ObjectID(z.ZoneID), Name: z.Name, Kind: ev.ReferenceItemTypeDefinedV1ZonesElemKind(z.Kind)})
	}
	for _, c := range v.Components {
		d.Components = append(d.Components, ev.ComponentDef{ItemTypeID: ev.ObjectID(c.ItemTypeID), Quantity: c.Quantity, LotTracked: c.LotTracked})
	}
	if v.Marking != "" {
		m := ev.CarrierType(v.Marking)
		d.Marking = &m
	}
	return d
}

func locationData(v RefLocation) ev.ReferenceLocationDefinedV1 {
	return ev.ReferenceLocationDefinedV1{LocationID: ev.ObjectID(v.LocationID), Kind: ev.ReferenceLocationDefinedV1Kind(v.Kind),
		ParentID: oid(v.ParentID), Scope: v.Scope, Name: v.Name, WarehouseID: oid(v.WarehouseID), AccessZoneID: oid(v.AccessZoneID),
		ValidFrom: ts(v.ValidFrom), ValidUntil: tsp(v.ValidUntil)}
}

func calendarData(v RefCalendar) ev.ReferenceCalendarDefinedV1 {
	d := ev.ReferenceCalendarDefinedV1{CalendarID: ev.ObjectID(v.CalendarID), Year: v.Year, NonWorkingDays: []ev.Date{}}
	for _, x := range v.WeeklyDaysOff {
		d.WeeklyDaysOff = append(d.WeeklyDaysOff, ev.ReferenceCalendarDefinedV1WeeklyDaysOffElem(x))
	}
	for _, x := range v.NonWorkingDays {
		d.NonWorkingDays = append(d.NonWorkingDays, ev.Date(x))
	}
	for _, x := range v.ShortenedDays {
		d.ShortenedDays = append(d.ShortenedDays, ev.Date(x))
	}
	for _, x := range v.WorkingDays {
		d.WorkingDays = append(d.WorkingDays, ev.Date(x))
	}
	return d
}

func shiftData(v RefShift) ev.ReferenceShiftScheduledV1 {
	d := ev.ReferenceShiftScheduledV1{ShiftID: ev.ObjectID(v.ShiftID), LocationID: ev.ObjectID(v.LocationID), Name: strp(v.Name),
		StartsAt: ts(v.StartsAt), EndsAt: ts(v.EndsAt), RepeatUntil: tsp(v.RepeatUntil)}
	if v.WorkingDaysOnly {
		t := true
		d.WorkingDaysOnly = &t
	}
	return d
}
