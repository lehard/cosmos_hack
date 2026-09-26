package reference

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Справочники — версионируемые данные журнала (AD-31, FR-17, FR-81, FR-95).
//
// Book — все известные версии справочников: каждая запись помнит свой seq
// (порядок знания), прогон (AD-38), время записи и дату действия. Срез на
// basis_seq — Slice (только записи с seq ≤ basis_seq); срез на дату события —
// функции «…At(t)» над срезом. Одна и та же чистая функция Apply строит
// справочник у воркера, в API, у стадии и у верификатора (AD-4, AD-9):
// правило «кто прав» — ниже, в governing.
//
// Правило версий: у записи справочника с датой действия (номенклатура, место,
// оборудование) на момент t действует версия с наибольшим seq среди версий с
// valid_from ≤ t, и только если t < valid_until. Позднее знание побеждает:
// исправление задним числом (новая запись с прежней датой действия) заменяет
// прежнюю, вывод из эксплуатации — запись с valid_until.

// Meta — координаты записи справочника в журнале (AD-37).
type Meta struct {
	Seq     int64  `json:"seq"`
	EventID string `json:"event_id"`
	// RunID — прогон сценария (AD-38); пусто — общий справочник (генезис, prod).
	RunID string `json:"run_id,omitempty"`
	// At — occurred_at записи (доменное время решения или факта).
	At time.Time `json:"at"`
	// Recorded — recorded_at записи: ось «что мы знали» (AD-22).
	Recorded time.Time `json:"recorded"`
	// Provenance — класс происхождения подписи (AD-2).
	Provenance string `json:"provenance,omitempty"`
}

// Validity — интервал действия записи [From, Until); Until nil — бессрочно.
type Validity struct {
	From  time.Time  `json:"from"`
	Until *time.Time `json:"until,omitempty"`
}

// Contains — момент t внутри интервала.
func (v Validity) Contains(t time.Time) bool {
	return !t.Before(v.From) && (v.Until == nil || t.Before(*v.Until))
}

// ItemType — версия позиции номенклатуры (reference.item_type.defined).
type ItemType struct {
	Meta
	Validity
	Data ev.ReferenceItemTypeDefinedV1 `json:"data"`
}

// Location — версия места (reference.location.defined).
type Location struct {
	Meta
	Validity
	Data ev.ReferenceLocationDefinedV1 `json:"data"`
}

// Equipment — версия оборудования (reference.equipment.defined).
type Equipment struct {
	Meta
	Validity
	Data ev.ReferenceEquipmentDefinedV1 `json:"data"`
}

// Verification — поверка или калибровка (reference.equipment.verified):
// действует с момента записи (At) до конца дня valid_until по местному
// времени предприятия.
type Verification struct {
	Meta
	Data ev.ReferenceEquipmentVerifiedV1 `json:"data"`
}

// CalendarYear — производственный календарь года (reference.calendar.defined).
type CalendarYear struct {
	Meta
	Data ev.ReferenceCalendarDefinedV1 `json:"data"`
}

// ShiftDef — смена или шаблон смены (reference.shift.scheduled).
type ShiftDef struct {
	Meta
	Data ev.ReferenceShiftScheduledV1 `json:"data"`
}

// Mapping — соответствие внешнего ID (reference.external_id.mapped, AD-18).
type Mapping struct {
	Meta
	Data ev.ReferenceExternalIDMappedV1 `json:"data"`
}

// Order — производственное задание учётной системы (erp.order.received).
type Order struct {
	Meta
	Data ev.ErpOrderReceivedV1 `json:"data"`
}

// Nomenclature — позиции номенклатуры учётной системы (erp.nomenclature.synced).
type Nomenclature struct {
	Meta
	Data ev.ErpNomenclatureSyncedV1 `json:"data"`
}

// Lot — поступившая партия со сроком годности (erp.lot.received).
type Lot struct {
	Meta
	Data ev.ErpLotReceivedV1 `json:"data"`
}

// Qualification — квалификация сотрудника (access.qualification.granted):
// данные модуля access, которые предусловия читают из среза (FR-17, FR-80).
type Qualification struct {
	Meta
	Data ev.AccessQualificationGrantedV1 `json:"data"`
}

// QualificationRevocation — отзыв квалификации (access.qualification.revoked).
type QualificationRevocation struct {
	Meta
	Data ev.AccessQualificationRevokedV1 `json:"data"`
}

// Book — справочники: все версии, известные к UpTo (AD-31).
type Book struct {
	ItemTypes      []ItemType                `json:"item_types,omitempty"`
	Locations      []Location                `json:"locations,omitempty"`
	Equipment      []Equipment               `json:"equipment,omitempty"`
	Verifications  []Verification            `json:"verifications,omitempty"`
	Calendars      []CalendarYear            `json:"calendars,omitempty"`
	Shifts         []ShiftDef                `json:"shifts,omitempty"`
	Mappings       []Mapping                 `json:"mappings,omitempty"`
	Orders         []Order                   `json:"orders,omitempty"`
	Nomenclature   []Nomenclature            `json:"nomenclature,omitempty"`
	Lots           []Lot                     `json:"lots,omitempty"`
	Qualifications []Qualification           `json:"qualifications,omitempty"`
	Revocations    []QualificationRevocation `json:"revocations,omitempty"`
	// UpTo — наибольший seq применённой записи.
	UpTo int64 `json:"up_to"`
}

// Types — типы записей журнала, из которых строится справочник: свои
// (reference.*, кроме адресованной reference.change.affects_item) и чужие
// факты, которые справочник только читает — задания, номенклатура и партии
// учётных систем (erp, FR-91), квалификации (access, FR-80).
var Types = []catalog.Type{
	catalog.ReferenceItemTypeDefined,
	catalog.ReferenceLocationDefined,
	catalog.ReferenceEquipmentDefined,
	catalog.ReferenceEquipmentVerified,
	catalog.ReferenceCalendarDefined,
	catalog.ReferenceShiftScheduled,
	catalog.ReferenceExternalIdMapped,
	catalog.ErpOrderReceived,
	catalog.ErpNomenclatureSynced,
	catalog.ErpLotReceived,
	catalog.AccessQualificationGranted,
	catalog.AccessQualificationRevoked,
}

// Relevant — запись входит в справочник.
func Relevant(t catalog.Type) bool { return slices.Contains(Types, t) }

func meta(r kernel.Record) Meta {
	return Meta{Seq: r.Seq, EventID: r.EventID, RunID: r.RunID, At: r.OccurredAt.UTC(), Recorded: r.RecordedAt.UTC(), Provenance: r.Provenance}
}

func validity(from ev.Timestamp, until *ev.Timestamp) Validity {
	v := Validity{From: from.Time().UTC()}
	if until != nil {
		u := until.Time().UTC()
		v.Until = &u
	}
	return v
}

func decode[T any](r kernel.Record) (T, error) {
	var v T
	if err := json.Unmarshal(r.Data, &v); err != nil {
		return v, fmt.Errorf("reference: запись %s (%s): %w", r.EventID, r.Type, err)
	}
	return v, nil
}

// Apply — запись журнала в справочник (чистая функция, AD-4). Записи других
// типов не меняют справочник. Записи применяются в порядке seq; повтор уже
// применённой записи (seq ≤ UpTo) пропускается.
func Apply(b Book, r kernel.Record) (Book, error) {
	if !Relevant(r.Type) || (r.Seq > 0 && r.Seq <= b.UpTo) {
		return b, nil
	}
	m := meta(r)
	switch r.Type {
	case catalog.ReferenceItemTypeDefined:
		d, err := decode[ev.ReferenceItemTypeDefinedV1](r)
		if err != nil {
			return b, err
		}
		b.ItemTypes = append(b.ItemTypes, ItemType{Meta: m, Validity: validity(d.ValidFrom, d.ValidUntil), Data: d})
	case catalog.ReferenceLocationDefined:
		d, err := decode[ev.ReferenceLocationDefinedV1](r)
		if err != nil {
			return b, err
		}
		b.Locations = append(b.Locations, Location{Meta: m, Validity: validity(d.ValidFrom, d.ValidUntil), Data: d})
	case catalog.ReferenceEquipmentDefined:
		d, err := decode[ev.ReferenceEquipmentDefinedV1](r)
		if err != nil {
			return b, err
		}
		b.Equipment = append(b.Equipment, Equipment{Meta: m, Validity: validity(d.ValidFrom, d.ValidUntil), Data: d})
	case catalog.ReferenceEquipmentVerified:
		d, err := decode[ev.ReferenceEquipmentVerifiedV1](r)
		if err != nil {
			return b, err
		}
		b.Verifications = append(b.Verifications, Verification{Meta: m, Data: d})
	case catalog.ReferenceCalendarDefined:
		d, err := decode[ev.ReferenceCalendarDefinedV1](r)
		if err != nil {
			return b, err
		}
		b.Calendars = append(b.Calendars, CalendarYear{Meta: m, Data: d})
	case catalog.ReferenceShiftScheduled:
		d, err := decode[ev.ReferenceShiftScheduledV1](r)
		if err != nil {
			return b, err
		}
		b.Shifts = append(b.Shifts, ShiftDef{Meta: m, Data: d})
	case catalog.ReferenceExternalIdMapped:
		d, err := decode[ev.ReferenceExternalIDMappedV1](r)
		if err != nil {
			return b, err
		}
		b.Mappings = append(b.Mappings, Mapping{Meta: m, Data: d})
	case catalog.ErpOrderReceived:
		d, err := decode[ev.ErpOrderReceivedV1](r)
		if err != nil {
			return b, err
		}
		b.Orders = append(b.Orders, Order{Meta: m, Data: d})
	case catalog.ErpNomenclatureSynced:
		d, err := decode[ev.ErpNomenclatureSyncedV1](r)
		if err != nil {
			return b, err
		}
		b.Nomenclature = append(b.Nomenclature, Nomenclature{Meta: m, Data: d})
	case catalog.ErpLotReceived:
		d, err := decode[ev.ErpLotReceivedV1](r)
		if err != nil {
			return b, err
		}
		b.Lots = append(b.Lots, Lot{Meta: m, Data: d})
	case catalog.AccessQualificationGranted:
		d, err := decode[ev.AccessQualificationGrantedV1](r)
		if err != nil {
			return b, err
		}
		b.Qualifications = append(b.Qualifications, Qualification{Meta: m, Data: d})
	case catalog.AccessQualificationRevoked:
		d, err := decode[ev.AccessQualificationRevokedV1](r)
		if err != nil {
			return b, err
		}
		b.Revocations = append(b.Revocations, QualificationRevocation{Meta: m, Data: d})
	}
	if r.Seq > b.UpTo {
		b.UpTo = r.Seq
	}
	return b, nil
}

// Build — справочник из записей (в любом порядке: применяются по seq).
func Build(records []kernel.Record) (Book, error) {
	rs := slices.Clone(records)
	slices.SortStableFunc(rs, func(a, b kernel.Record) int { return cmpInt(a.Seq, b.Seq) })
	var b Book
	for _, r := range rs {
		var err error
		if b, err = Apply(b, r); err != nil {
			return b, err
		}
	}
	return b, nil
}

func cmpInt(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// Filter — что входит в срез справочника.
type Filter struct {
	// BasisSeq — только записи с seq ≤ BasisSeq (0 — все известные): вход
	// свёртки изделия берёт срез на своём basis_seq (AD-31, AD-45).
	BasisSeq int64
	// RecordedBy — только записи с recorded_at ≤ RecordedBy: ось «что мы
	// знали» (AD-22, AD-37); nil — без ограничения.
	RecordedBy *time.Time
	// RunID — прогон (AD-38): общий справочник плюс записи этого прогона;
	// пусто — только общий справочник.
	RunID string
}

func (f Filter) keep(m Meta) bool {
	if f.BasisSeq > 0 && m.Seq > f.BasisSeq {
		return false
	}
	if f.RecordedBy != nil && !m.Recorded.IsZero() && m.Recorded.After(*f.RecordedBy) {
		return false
	}
	return m.RunID == "" || m.RunID == f.RunID
}

func keep[T any](xs []T, m func(T) Meta, f Filter) []T {
	var out []T
	for _, x := range xs {
		if f.keep(m(x)) {
			out = append(out, x)
		}
	}
	return out
}

// Slice — срез справочника: записи, известные на basis_seq и времени записи,
// общего справочника и прогона (AD-31, AD-38).
func (b Book) Slice(f Filter) Book {
	out := Book{
		ItemTypes:      keep(b.ItemTypes, func(x ItemType) Meta { return x.Meta }, f),
		Locations:      keep(b.Locations, func(x Location) Meta { return x.Meta }, f),
		Equipment:      keep(b.Equipment, func(x Equipment) Meta { return x.Meta }, f),
		Verifications:  keep(b.Verifications, func(x Verification) Meta { return x.Meta }, f),
		Calendars:      keep(b.Calendars, func(x CalendarYear) Meta { return x.Meta }, f),
		Shifts:         keep(b.Shifts, func(x ShiftDef) Meta { return x.Meta }, f),
		Mappings:       keep(b.Mappings, func(x Mapping) Meta { return x.Meta }, f),
		Orders:         keep(b.Orders, func(x Order) Meta { return x.Meta }, f),
		Nomenclature:   keep(b.Nomenclature, func(x Nomenclature) Meta { return x.Meta }, f),
		Lots:           keep(b.Lots, func(x Lot) Meta { return x.Meta }, f),
		Qualifications: keep(b.Qualifications, func(x Qualification) Meta { return x.Meta }, f),
		Revocations:    keep(b.Revocations, func(x QualificationRevocation) Meta { return x.Meta }, f),
	}
	for _, m := range out.metas() {
		if m.Seq > out.UpTo {
			out.UpTo = m.Seq
		}
	}
	return out
}

// metas — координаты всех записей среза.
func (b Book) metas() []Meta {
	var ms []Meta
	for _, x := range b.ItemTypes {
		ms = append(ms, x.Meta)
	}
	for _, x := range b.Locations {
		ms = append(ms, x.Meta)
	}
	for _, x := range b.Equipment {
		ms = append(ms, x.Meta)
	}
	for _, x := range b.Verifications {
		ms = append(ms, x.Meta)
	}
	for _, x := range b.Calendars {
		ms = append(ms, x.Meta)
	}
	for _, x := range b.Shifts {
		ms = append(ms, x.Meta)
	}
	for _, x := range b.Mappings {
		ms = append(ms, x.Meta)
	}
	for _, x := range b.Orders {
		ms = append(ms, x.Meta)
	}
	for _, x := range b.Nomenclature {
		ms = append(ms, x.Meta)
	}
	for _, x := range b.Lots {
		ms = append(ms, x.Meta)
	}
	for _, x := range b.Qualifications {
		ms = append(ms, x.Meta)
	}
	for _, x := range b.Revocations {
		ms = append(ms, x.Meta)
	}
	return ms
}

// Empty — в срезе нет ни одной записи справочника.
func (b Book) Empty() bool { return len(b.metas()) == 0 }

// governing — действующая на момент t версия ключа key среди versions:
// наибольший seq среди версий с valid_from ≤ t; ok=false — нет версии или
// действие закончилось (t ≥ valid_until).
func governing[T any](versions []T, key string, keyOf func(T) string, meta func(T) Meta, val func(T) Validity, t time.Time) (T, bool) {
	var best T
	var bestSeq int64 = -1
	found := false
	for _, v := range versions {
		if keyOf(v) != key || val(v).From.After(t) {
			continue
		}
		if s := meta(v).Seq; !found || s > bestSeq {
			best, bestSeq, found = v, s, true
		}
	}
	if !found {
		return best, false
	}
	if u := val(best).Until; u != nil && !t.Before(*u) {
		return best, false
	}
	return best, true
}

// sortedKeys — ключи версий без повторов, по возрастанию.
func sortedKeys[T any](versions []T, keyOf func(T) string) []string {
	var ks []string
	for _, v := range versions {
		ks = append(ks, keyOf(v))
	}
	slices.Sort(ks)
	return slices.Compact(ks)
}

// ItemTypeAt — позиция номенклатуры, действующая на t.
func (b Book) ItemTypeAt(id string, t time.Time) (ItemType, bool) {
	return governing(b.ItemTypes, id, func(x ItemType) string { return string(x.Data.ItemTypeID) },
		func(x ItemType) Meta { return x.Meta }, func(x ItemType) Validity { return x.Validity }, t)
}

// ItemTypesAt — номенклатура на t (по id).
func (b Book) ItemTypesAt(t time.Time) []ItemType {
	var out []ItemType
	for _, id := range sortedKeys(b.ItemTypes, func(x ItemType) string { return string(x.Data.ItemTypeID) }) {
		if x, ok := b.ItemTypeAt(id, t); ok {
			out = append(out, x)
		}
	}
	return out
}

// LocationAt — место, действующее на t.
func (b Book) LocationAt(id string, t time.Time) (Location, bool) {
	return governing(b.Locations, id, func(x Location) string { return string(x.Data.LocationID) },
		func(x Location) Meta { return x.Meta }, func(x Location) Validity { return x.Validity }, t)
}

// LocationsAt — места на t (по id).
func (b Book) LocationsAt(t time.Time) []Location {
	var out []Location
	for _, id := range sortedKeys(b.Locations, func(x Location) string { return string(x.Data.LocationID) }) {
		if x, ok := b.LocationAt(id, t); ok {
			out = append(out, x)
		}
	}
	return out
}

// Ancestors — место id и его предки на t (от места к корню), без циклов.
func (b Book) Ancestors(id string, t time.Time) []string {
	var out []string
	for id != "" && !slices.Contains(out, id) {
		out = append(out, id)
		l, ok := b.LocationAt(id, t)
		if !ok || l.Data.ParentID == nil {
			break
		}
		id = string(*l.Data.ParentID)
	}
	return out
}

// OrdersLatest — задания учётных систем: последняя запись по order_id.
func (b Book) OrdersLatest() []Order {
	return latest(b.Orders, func(x Order) string { return string(x.Data.OrderID) }, func(x Order) Meta { return x.Meta })
}

// LotsLatest — партии: последняя запись по lot_id.
func (b Book) LotsLatest() []Lot {
	return latest(b.Lots, func(x Lot) string { return string(x.Data.LotID) }, func(x Lot) Meta { return x.Meta })
}

// latest — по ключу — запись с наибольшим seq; результат по ключу.
func latest[T any](xs []T, keyOf func(T) string, meta func(T) Meta) []T {
	by := map[string]T{}
	for _, x := range xs {
		k := keyOf(x)
		if cur, ok := by[k]; !ok || meta(x).Seq > meta(cur).Seq {
			by[k] = x
		}
	}
	out := make([]T, 0, len(by))
	for _, k := range sortedKeys(xs, keyOf) {
		out = append(out, by[k])
	}
	return out
}

// ItemFilter — срез справочника для входа свёртки изделия (AD-31, AD-45):
// basis_seq — наибольший seq входа (записи справочника позже него изделие
// увидит только новой записью своего потока, в том числе адресованной
// reference.change.affects_item), прогон — run_id входа (AD-38).
func ItemFilter(input []kernel.Record) Filter {
	var f Filter
	for _, r := range input {
		if r.Seq > f.BasisSeq {
			f.BasisSeq = r.Seq
		}
		if f.RunID == "" && r.RunID != "" {
			f.RunID = r.RunID
		}
	}
	// BasisSeq = 0 (вход без позиций журнала) — всё известное.
	return f
}
