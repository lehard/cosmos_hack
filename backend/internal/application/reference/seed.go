package reference

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	jc "ant/internal/contracts/journal"
	"ant/internal/contracts/normative"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/reference"
)

// Затравка справочников (AD-33): стартовые справочники демо-изделия из
// normative/reference/flange/*.yaml — записи журнала с происхождением genesis,
// которые migrate пишет в профилях затравки (демо-трек, до роли init эпика 05),
// а заготовки (fixtures) читают как готовую книгу. Повтор — дубль, а не вторая
// запись: event_id — UUIDv5 от версии затравки и ключа записи.

// Файлы справочников демо-изделия (пути от корня репозитория).
const (
	PathItemTypes = "normative/reference/flange/item-types.yaml"
	PathLocations = "normative/reference/flange/locations.yaml"
	PathEquipment = "normative/reference/flange/equipment.yaml"
	PathCalendar  = "normative/reference/flange/calendar.yaml"
	PathShifts    = "normative/reference/flange/shifts.yaml"
)

// SeedFiles — файлы затравки справочников.
var SeedFiles = []string{PathItemTypes, PathLocations, PathEquipment, PathCalendar, PathShifts}

// SeedEpoch — дата действия затравки: 1 января 2026 года, 00:00 МСК.
// Записи генезиса действуют с неё: прогоны сценариев и изделия демо позже.
var SeedEpoch = time.Date(2026, 1, 1, 0, 0, 0, 0, dom.Local).UTC()

// seedVersion — версия затравки в event_id (меняется вместе с составом).
const seedVersion = 1

func seedID(kind, key string) string {
	return kernel.UUIDv5(constants.NsAnt, fmt.Sprintf("reference.seed/v%d/%s/%s", seedVersion, kind, key))
}

// decodeYAML — YAML в сгенерированный тип через JSON (обязательные поля
// проверяет сгенерированный UnmarshalJSON).
func decodeYAML[T any](fsys fs.FS, name string) (T, error) {
	var out T
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return out, err
	}
	var raw any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return out, fmt.Errorf("%s: %w", name, err)
	}
	j, err := json.Marshal(raw)
	if err != nil {
		return out, fmt.Errorf("%s: %w", name, err)
	}
	if err := json.Unmarshal(j, &out); err != nil {
		return out, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// SeedRecords — записи затравки справочников из файлов fsys (корень — корень
// репозитория или встроенная копия): номенклатура, места, оборудование и
// поверки, календарь, шаблоны смен.
func SeedRecords(fsys fs.FS) ([]Record, error) {
	var out []Record
	add := func(t catalog.Type, kind, key string, data any) {
		out = append(out, Record{EventID: seedID(kind, key), Type: t, Stream: Stream(kind, key), OccurredAt: SeedEpoch, Data: data,
			Actor: "genesis", Provenance: jc.JournalEntryProvenanceClassGenesis, SourceID: SourceGenesis})
	}
	from := ev.NewTimestamp(SeedEpoch)

	its, err := decodeYAML[normative.ItemTypesSeed](fsys, PathItemTypes)
	if err != nil {
		return nil, err
	}
	for _, it := range its.ItemTypes {
		m := ev.CarrierType(it.Marking)
		d := ev.ReferenceItemTypeDefinedV1{ItemTypeID: ev.ObjectID(it.ID), Designation: it.Designation, Name: it.Name,
			Revision: it.Revision, Marking: &m, ValidFrom: from}
		for _, z := range it.Zones {
			d.Zones = append(d.Zones, ev.ZoneDef{ZoneID: ev.ObjectID(z.ZoneID), Name: z.Name, Kind: ev.ReferenceItemTypeDefinedV1ZonesElemKind(z.Kind)})
		}
		for _, c := range it.Components {
			d.Components = append(d.Components, ev.ComponentDef{ItemTypeID: ev.ObjectID(c.ItemTypeID), Quantity: c.Quantity, LotTracked: c.LotTracked})
		}
		add(catalog.ReferenceItemTypeDefined, "item_type", it.ID, d)
	}

	locs, err := decodeYAML[normative.LocationsSeed](fsys, PathLocations)
	if err != nil {
		return nil, err
	}
	for _, l := range locs.Locations {
		d := ev.ReferenceLocationDefinedV1{LocationID: ev.ObjectID(l.ID), Kind: ev.ReferenceLocationDefinedV1Kind(l.Kind),
			Scope: l.Scope, Name: l.Name, ValidFrom: from}
		if l.Parent != nil {
			d.ParentID = oid(*l.Parent)
		}
		if l.WarehouseID != nil {
			d.WarehouseID = oid(*l.WarehouseID)
		}
		if l.AccessZoneID != nil {
			d.AccessZoneID = oid(*l.AccessZoneID)
		}
		add(catalog.ReferenceLocationDefined, "location", l.ID, d)
	}

	eqs, err := decodeYAML[normative.EquipmentSeed](fsys, PathEquipment)
	if err != nil {
		return nil, err
	}
	for _, e := range eqs.Equipment {
		d := ev.ReferenceEquipmentDefinedV1{EquipmentID: ev.ObjectID(e.ID), Kind: ev.ReferenceEquipmentDefinedV1Kind(e.Kind),
			LocationID: ev.ObjectID(e.LocationID), Name: e.Name, IsMeasuringInstrument: e.Measuring, ValidFrom: from}
		if e.SourceID != nil {
			s := ev.SourceID(*e.SourceID)
			d.SourceID = &s
		}
		add(catalog.ReferenceEquipmentDefined, "equipment", e.ID, d)
	}
	for _, v := range eqs.Verifications {
		u := ev.Date(v.ValidUntil)
		d := ev.ReferenceEquipmentVerifiedV1{EquipmentID: ev.ObjectID(v.EquipmentID), Kind: ev.ReferenceEquipmentVerifiedV1Kind(v.Kind),
			Result: ev.ReferenceEquipmentVerifiedV1Result(v.Result), ValidUntil: &u, CertificateRef: v.CertificateRef}
		r := Record{EventID: seedID("verification", v.EquipmentID+"/"+v.ValidUntil), Type: catalog.ReferenceEquipmentVerified,
			Stream: Stream("equipment", v.EquipmentID), OccurredAt: SeedEpoch, Data: d, Actor: "genesis",
			Provenance: jc.JournalEntryProvenanceClassGenesis, SourceID: SourceGenesis}
		out = append(out, r)
	}

	cal, err := decodeYAML[normative.CalendarSeed](fsys, PathCalendar)
	if err != nil {
		return nil, err
	}
	lastYear := 0
	for _, y := range cal.Years {
		d := ev.ReferenceCalendarDefinedV1{CalendarID: ev.ObjectID(cal.CalendarID), Year: y.Year, NonWorkingDays: []ev.Date{}}
		for _, w := range y.WeeklyDaysOff {
			d.WeeklyDaysOff = append(d.WeeklyDaysOff, ev.ReferenceCalendarDefinedV1WeeklyDaysOffElem(w))
		}
		for _, x := range y.Holidays {
			d.NonWorkingDays = append(d.NonWorkingDays, ev.Date(x))
		}
		for _, x := range y.WorkingDays {
			d.WorkingDays = append(d.WorkingDays, ev.Date(x))
		}
		for _, x := range y.ShortenedDays {
			d.ShortenedDays = append(d.ShortenedDays, ev.Date(x))
		}
		r := Record{EventID: seedID("calendar", fmt.Sprintf("%s/%d", cal.CalendarID, y.Year)), Type: catalog.ReferenceCalendarDefined,
			Stream: Stream("calendar", cal.CalendarID), OccurredAt: SeedEpoch, Data: d, Actor: "genesis",
			Provenance: jc.JournalEntryProvenanceClassGenesis, SourceID: SourceGenesis}
		out = append(out, r)
		lastYear = max(lastYear, y.Year)
	}

	sh, err := decodeYAML[normative.ShiftsSeed](fsys, PathShifts)
	if err != nil {
		return nil, err
	}
	// Шаблоны смен повторяются в рабочие дни от даты затравки до конца
	// последнего года календаря.
	until := ev.NewTimestamp(time.Date(max(lastYear, SeedEpoch.In(dom.Local).Year()), 12, 31, 23, 59, 0, 0, dom.Local))
	yes := true
	for _, p := range sh.Patterns {
		start, err := localClock(SeedEpoch, p.Starts)
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", PathShifts, p.ID, err)
		}
		end, err := localClock(SeedEpoch, p.Ends)
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", PathShifts, p.ID, err)
		}
		if !end.After(start) {
			end = end.Add(24 * time.Hour) // смена через полночь
		}
		for _, loc := range p.Locations {
			name := p.Name
			d := ev.ReferenceShiftScheduledV1{ShiftID: ev.ObjectID(p.ID), LocationID: ev.ObjectID(loc), Name: &name,
				StartsAt: ev.NewTimestamp(start), EndsAt: ev.NewTimestamp(end), RepeatUntil: &until, WorkingDaysOnly: &yes}
			r := Record{EventID: seedID("shift", p.ID+"/"+loc), Type: catalog.ReferenceShiftScheduled, Stream: Stream("shift", p.ID),
				OccurredAt: SeedEpoch, Data: d, Actor: "genesis", Provenance: jc.JournalEntryProvenanceClassGenesis, SourceID: SourceGenesis}
			out = append(out, r)
		}
	}
	return out, nil
}

// localClock — местное время суток hh:mm в день момента day.
func localClock(day time.Time, hhmm string) (time.Time, error) {
	h, m, ok := strings.Cut(hhmm, ":")
	if !ok {
		return time.Time{}, fmt.Errorf("время %q: ожидается hh:mm", hhmm)
	}
	var hh, mm int
	if _, err := fmt.Sscanf(h+" "+m, "%d %d", &hh, &mm); err != nil {
		return time.Time{}, fmt.Errorf("время %q: %w", hhmm, err)
	}
	l := day.In(dom.Local)
	return time.Date(l.Year(), l.Month(), l.Day(), hh, mm, 0, 0, dom.Local).UTC(), nil
}

// Seed записывает затравку в журнал. Возвращает число новых записей: уже
// записанные (тот же event_id) — повтор, пропускаются.
func Seed(ctx context.Context, w Writer, recs []Record) (int, error) {
	n := 0
	for _, r := range recs {
		rc, err := w.Write(ctx, r)
		if err != nil {
			return n, fmt.Errorf("затравка справочников: %s %s: %w", r.Type, r.Stream, err)
		}
		if !rc.Replayed {
			n++
		}
	}
	return n, nil
}

// SeedBook — книга справочника из записей затравки без журнала (заготовки,
// тесты): seq — порядковый номер записи.
func SeedBook(recs []Record) (dom.Book, error) {
	var b dom.Book
	for i, r := range recs {
		data, err := json.Marshal(r.Data)
		if err != nil {
			return b, err
		}
		kr := kernel.Record{Seq: int64(i + 1), EventID: r.EventID, Type: r.Type, Kind: catalog.KindDecision, Stream: r.Stream,
			RunID: r.RunID, OccurredAt: r.OccurredAt, RecordedAt: r.OccurredAt, Provenance: string(r.Provenance), Data: data}
		if b, err = dom.Apply(b, kr); err != nil {
			return b, err
		}
	}
	return b, nil
}
