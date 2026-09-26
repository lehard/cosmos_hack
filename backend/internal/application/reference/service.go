package reference

import (
	"context"
	"strconv"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/reference"
)

// Clock — доменные часы (AD-37): «сейчас» ответа и момент решения.
type Clock interface {
	Now(ctx context.Context) (time.Time, error)
}

// Service — реализация live ведущих портов модуля reference (AD-36):
// чтение — срез справочника из журнала на момент (AD-22, AD-31); команды —
// проверка и решение в журнал с проверками AD-39. Без источника (выгрузка
// OpenAPI, тест «каждая операция отвечает») операции отвечают 501.
type Service struct {
	src   BookSource
	w     Writer
	clock Clock
}

// Option — настройка Service.
type Option func(*Service)

// WithSource — справочник из журнала.
func WithSource(s BookSource) Option { return func(x *Service) { x.src = s } }

// WithWriter — запись решений.
func WithWriter(w Writer) Option { return func(x *Service) { x.w = w } }

// WithClock — доменные часы.
func WithClock(c Clock) Option { return func(x *Service) { x.clock = c } }

// NewService создаёт реализацию live.
func NewService(opts ...Option) *Service {
	s := &Service{}
	for _, o := range opts {
		o(s)
	}
	return s
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// view — срез справочника на момент m и сам момент (дата действия).
func (s *Service) view(ctx context.Context, op string, m platform.Moment) (dom.Book, time.Time, error) {
	if s.src == nil {
		return dom.Book{}, time.Time{}, platform.NotImplemented(op)
	}
	at, err := s.now(ctx, m.RunID)
	if err != nil {
		return dom.Book{}, time.Time{}, err
	}
	f := dom.Filter{RunID: m.RunID}
	if m.AsOf != nil {
		at = m.AsOf.UTC()
		if m.Axis == platform.AxisRecorded {
			// «Что мы знали»: только записи, записанные к моменту (AD-22).
			f.RecordedBy = m.AsOf
		}
	}
	b, err := s.src.Book(ctx, 0)
	if err != nil {
		return dom.Book{}, time.Time{}, err
	}
	return b.Slice(f), at, nil
}

// now — доменное «сейчас» (AD-37; в прогоне — часы прогона).
func (s *Service) now(ctx context.Context, run string) (time.Time, error) {
	if s.clock == nil {
		return time.Now().UTC(), nil
	}
	if run != "" {
		ctx = appjournal.WithRun(ctx, run)
	}
	t, err := s.clock.Now(ctx)
	if err != nil {
		// Часы сценария без тика — системное время (прогон ещё не начал время).
		return time.Now().UTC(), nil
	}
	return t.UTC(), nil
}

// ItemTypes — номенклатура на момент.
func (s *Service) ItemTypes(ctx context.Context, m platform.Moment) (RefItemTypeList, error) {
	b, at, err := s.view(ctx, "reference.item_type.list", m)
	if err != nil {
		return RefItemTypeList{}, err
	}
	out := RefItemTypeList{Items: []RefItemType{}}
	for _, x := range b.ItemTypesAt(at) {
		out.Items = append(out.Items, itemTypeView(x))
	}
	return out, nil
}

// Locations — места на момент.
func (s *Service) Locations(ctx context.Context, m platform.Moment) (RefLocationList, error) {
	b, at, err := s.view(ctx, "reference.location.list", m)
	if err != nil {
		return RefLocationList{}, err
	}
	out := RefLocationList{Items: []RefLocation{}}
	for _, x := range b.LocationsAt(at) {
		out.Items = append(out.Items, locationView(x))
	}
	return out, nil
}

// Equipment — оборудование и поверка на момент (FR-17).
func (s *Service) Equipment(ctx context.Context, m platform.Moment) (RefEquipmentList, error) {
	b, at, err := s.view(ctx, "reference.equipment.list", m)
	if err != nil {
		return RefEquipmentList{}, err
	}
	out := RefEquipmentList{Items: []RefEquipment{}}
	for _, id := range b.EquipmentIDs() {
		if _, ok := b.EquipmentAt(id, at); !ok {
			continue
		}
		out.Items = append(out.Items, equipmentView(b.EquipmentStatusAt(id, at)))
	}
	return out, nil
}

// Calendar — производственный календарь года (0 — год момента ответа).
func (s *Service) Calendar(ctx context.Context, year int, m platform.Moment) (RefCalendar, error) {
	b, at, err := s.view(ctx, "reference.calendar.read", m)
	if err != nil {
		return RefCalendar{}, err
	}
	if year == 0 {
		year = at.In(dom.Local).Year()
	}
	v, ok := b.Calendar().Year(year)
	if !ok {
		e := platform.Fail(errcodes.ReferenceNotFound, "field", "year")
		e.Detail = "Производственный календарь на " + strconv.Itoa(year) + " год не задан: сроки считаются «пн–пт»"
		return RefCalendar{}, e
	}
	return calendarView(v), nil
}

// shiftWindow — окно графика смен в ответе: от полусуток до момента до
// полутора суток после (текущая, предыдущая и следующие смены).
const (
	shiftWindowBefore = 12 * time.Hour
	shiftWindowAfter  = 36 * time.Hour
)

// Shifts — смены вокруг момента ответа (FR-81); locationID — место (пусто — все).
func (s *Service) Shifts(ctx context.Context, locationID string, m platform.Moment) (RefShiftList, error) {
	b, at, err := s.view(ctx, "reference.shift.list", m)
	if err != nil {
		return RefShiftList{}, err
	}
	out := RefShiftList{Items: []RefShift{}}
	for _, x := range b.ShiftsBetween(at.Add(-shiftWindowBefore), at.Add(shiftWindowAfter), locationID) {
		out.Items = append(out.Items, RefShift{ShiftID: x.ID, LocationID: x.LocationID, Name: x.Name, StartsAt: x.From, EndsAt: x.To})
	}
	return out, nil
}

// ExternalIDs — соответствия внешних ID (FR-95); курсор — номер строки.
func (s *Service) ExternalIDs(ctx context.Context, system string, m platform.Moment, p platform.Page) (RefExternalIDList, error) {
	b, _, err := s.view(ctx, "reference.external_id.list", m)
	if err != nil {
		return RefExternalIDList{}, err
	}
	all := b.MappingsView(system)
	start, _ := strconv.Atoi(p.Cursor)
	limit := p.Limit
	if limit <= 0 {
		limit = 100
	}
	out := RefExternalIDList{Items: []RefExternalID{}}
	for i := max(start, 0); i < len(all) && len(out.Items) < limit; i++ {
		v := all[i]
		out.Items = append(out.Items, RefExternalID{System: string(v.Data.System), ObjectKind: string(v.Data.ObjectKind),
			ExternalID: v.Data.ExternalID, InternalID: string(v.Data.InternalID), MappedAt: v.At, Conflict: v.Conflict, Effective: v.Effective})
		if len(out.Items) == limit && i+1 < len(all) {
			out.NextCursor = strconv.Itoa(i + 1)
		}
	}
	return out, nil
}

// Lots — партии и сроки годности на момент.
func (s *Service) Lots(ctx context.Context, m platform.Moment) (RefLotList, error) {
	b, at, err := s.view(ctx, "reference.lot.list", m)
	if err != nil {
		return RefLotList{}, err
	}
	out := RefLotList{Items: []RefLot{}}
	for _, l := range b.LotsLatest() {
		usable, _ := b.LotUsableAt(string(l.Data.LotID), at)
		out.Items = append(out.Items, lotView(l, usable))
	}
	return out, nil
}

// Orders — задания учётных систем.
func (s *Service) Orders(ctx context.Context, m platform.Moment) (RefOrderList, error) {
	b, _, err := s.view(ctx, "reference.order.list", m)
	if err != nil {
		return RefOrderList{}, err
	}
	out := RefOrderList{Items: []RefOrder{}}
	for _, o := range b.OrdersLatest() {
		out.Items = append(out.Items, orderView(o))
	}
	return out, nil
}
