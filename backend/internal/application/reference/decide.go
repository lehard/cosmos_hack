package reference

import (
	"context"
	"regexp"
	"strconv"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/reference"
)

// Команды справочника (AD-31, AD-39): каждая — решение человека с подписью
// (класс personal, AD-2: справочники, от которых зависят предусловия, не
// меняются записями server-attested) в поток `reference:‹вид›:‹id›`. Гард
// проверяет ссылки на действующие записи на дату действия; basis_seq команды
// проверяет journal.Append (AD-39). Изменение с датой действия в прошлом
// доходит до изделий адресованными записями межизделийной стадии (AD-42,
// domain/crossitem → reference.AffectsItem).

// Stream — поток записи справочника `reference:‹вид›:‹id›` (каталог streams.reference).
func Stream(kind, id string) string { return "reference:" + kind + ":" + id }

var scopePattern = regexp.MustCompile(`^[a-z0-9_-]+(/[a-z0-9_-]+)*$`)

func invalid(field, reason string) error {
	e := platform.Fail(errcodes.ApiValidationFailed, "field", field, "reason", reason)
	e.Detail = field + ": " + reason
	return e
}

func notFound(field, id string) error {
	e := platform.Fail(errcodes.ReferenceNotFound, "field", field)
	e.Detail = "Не найдена запись справочника: " + field + " = " + id
	return e
}

// decide — общий сценарий команды: доменное «сейчас», срез справочника,
// гард, запись решения.
func (s *Service) decide(ctx context.Context, op string, h platform.CommandHeader, t catalog.Type, stream string,
	build func(b dom.Book, now time.Time) (any, error)) (platform.Receipt, error) {
	if s.src == nil || s.w == nil {
		return platform.Receipt{}, platform.NotImplemented(op)
	}
	run := appjournal.RunFrom(ctx)
	now, err := s.now(ctx, run)
	if err != nil {
		return platform.Receipt{}, err
	}
	b, err := s.src.Book(ctx, 0)
	if err != nil {
		return platform.Receipt{}, err
	}
	data, err := build(b.Slice(dom.Filter{RunID: run}), now)
	if err != nil {
		return platform.Receipt{}, err
	}
	p := platform.PrincipalFrom(ctx)
	return s.w.Write(ctx, Record{EventID: h.CommandID, Type: t, Stream: stream, RunID: run, OccurredAt: now, Data: data,
		Actor: p.PersonID, BasisSeq: h.BasisSeq, GuardStreams: []string{stream}, PolicySeq: h.PolicySeq, WorkplaceID: h.WorkplaceID})
}

func checkUntil(from time.Time, until *time.Time) error {
	if until != nil && !until.After(from) {
		return invalid("valid_until", "окончание действия не позже начала")
	}
	return nil
}

// DefineItemType — позиция номенклатуры (новая версия с датой действия).
func (s *Service) DefineItemType(ctx context.Context, in DefineItemType) (platform.Receipt, error) {
	return s.decide(ctx, "reference.item_type.define", in.CommandHeader, catalog.ReferenceItemTypeDefined, Stream("item_type", in.ItemTypeID),
		func(b dom.Book, _ time.Time) (any, error) {
			if err := checkUntil(in.ValidFrom, in.ValidUntil); err != nil {
				return nil, err
			}
			seen := map[string]bool{}
			for _, z := range in.Zones {
				if seen[z.ZoneID] {
					return nil, invalid("zones", "зона "+z.ZoneID+" повторяется")
				}
				seen[z.ZoneID] = true
			}
			return itemTypeData(in.RefItemType), nil
		})
}

// DefineLocation — место; родитель должен действовать на дату действия.
func (s *Service) DefineLocation(ctx context.Context, in DefineLocation) (platform.Receipt, error) {
	return s.decide(ctx, "reference.location.define", in.CommandHeader, catalog.ReferenceLocationDefined, Stream("location", in.LocationID),
		func(b dom.Book, _ time.Time) (any, error) {
			if err := checkUntil(in.ValidFrom, in.ValidUntil); err != nil {
				return nil, err
			}
			if !scopePattern.MatchString(in.Scope) {
				return nil, invalid("scope", "путь области: латиница, цифры, «_», «-» через «/»")
			}
			if in.ParentID != "" {
				if in.ParentID == in.LocationID {
					return nil, invalid("parent_id", "место не может быть своим родителем")
				}
				if _, ok := b.LocationAt(in.ParentID, in.ValidFrom); !ok {
					return nil, notFound("parent_id", in.ParentID)
				}
			}
			return locationData(in.RefLocation), nil
		})
}

// DefineEquipment — оборудование; место должно действовать на дату действия.
func (s *Service) DefineEquipment(ctx context.Context, in DefineEquipment) (platform.Receipt, error) {
	return s.decide(ctx, "reference.equipment.define", in.CommandHeader, catalog.ReferenceEquipmentDefined, Stream("equipment", in.EquipmentID),
		func(b dom.Book, _ time.Time) (any, error) {
			if err := checkUntil(in.ValidFrom, in.ValidUntil); err != nil {
				return nil, err
			}
			if _, ok := b.LocationAt(in.LocationID, in.ValidFrom); !ok && len(b.Locations) > 0 {
				return nil, notFound("location_id", in.LocationID)
			}
			d := ev.ReferenceEquipmentDefinedV1{EquipmentID: ev.ObjectID(in.EquipmentID), Kind: ev.ReferenceEquipmentDefinedV1Kind(in.Kind),
				LocationID: ev.ObjectID(in.LocationID), Name: in.Name, IsMeasuringInstrument: in.IsMeasuringInstrument,
				ValidFrom: ts(in.ValidFrom), ValidUntil: tsp(in.ValidUntil)}
			if in.SourceID != "" {
				src := ev.SourceID(in.SourceID)
				d.SourceID = &src
			}
			return d, nil
		})
}

// VerifyEquipment — поверка или калибровка (метролог, FR-17): действует с
// момента записи до конца дня valid_until; «непригодно» прекращает действие
// прежней поверки.
func (s *Service) VerifyEquipment(ctx context.Context, equipmentID string, in VerifyEquipment) (platform.Receipt, error) {
	return s.decide(ctx, "reference.equipment.verify", in.CommandHeader, catalog.ReferenceEquipmentVerified, Stream("equipment", equipmentID),
		func(b dom.Book, now time.Time) (any, error) {
			if _, ok := b.EquipmentAt(equipmentID, now); !ok {
				return nil, notFound("equipment_id", equipmentID)
			}
			d := ev.ReferenceEquipmentVerifiedV1{EquipmentID: ev.ObjectID(equipmentID), Kind: ev.ReferenceEquipmentVerifiedV1Kind(in.Kind),
				Result: ev.ReferenceEquipmentVerifiedV1Result(in.Result), CertificateRef: strp(in.CertificateRef)}
			if in.ValidUntil != "" {
				u, err := time.ParseInLocation(dom.DateLayout, in.ValidUntil, dom.Local)
				if err != nil {
					return nil, invalid("valid_until", "дата YYYY-MM-DD")
				}
				if in.Result == "valid" && !u.AddDate(0, 0, 1).After(now) {
					return nil, invalid("valid_until", "срок поверки уже истёк")
				}
				x := ev.Date(in.ValidUntil)
				d.ValidUntil = &x
			} else if in.Result == "valid" {
				return nil, invalid("valid_until", "у пригодного средства измерений нужен срок действия поверки")
			}
			return d, nil
		})
}

// DefineCalendar — производственный календарь года (заменяет прежний год целиком).
func (s *Service) DefineCalendar(ctx context.Context, in DefineCalendar) (platform.Receipt, error) {
	return s.decide(ctx, "reference.calendar.define", in.CommandHeader, catalog.ReferenceCalendarDefined, Stream("calendar", in.CalendarID),
		func(dom.Book, time.Time) (any, error) {
			if in.Year < 2000 || in.Year > 2100 {
				return nil, invalid("year", "год 2000–2100")
			}
			prefix := strconv.Itoa(in.Year) + "-"
			for field, ds := range map[string][]string{"non_working_days": in.NonWorkingDays, "shortened_days": in.ShortenedDays, "working_days": in.WorkingDays} {
				for _, d := range ds {
					if _, err := time.Parse(dom.DateLayout, d); err != nil || d[:5] != prefix {
						return nil, invalid(field, "дата "+d+" — не YYYY-MM-DD "+strconv.Itoa(in.Year)+" года")
					}
				}
			}
			return calendarData(in.RefCalendar), nil
		})
}

// maxShift — наибольшая длительность смены.
const maxShift = 24 * time.Hour

// ScheduleShift — смена или шаблон смены (FR-81: график смен ведёт мастер участка).
func (s *Service) ScheduleShift(ctx context.Context, in ScheduleShift) (platform.Receipt, error) {
	return s.decide(ctx, "reference.shift.schedule", in.CommandHeader, catalog.ReferenceShiftScheduled, Stream("shift", in.ShiftID),
		func(b dom.Book, _ time.Time) (any, error) {
			if !in.EndsAt.After(in.StartsAt) || in.EndsAt.Sub(in.StartsAt) > maxShift {
				return nil, invalid("ends_at", "конец смены позже начала и не дальше суток")
			}
			if in.RepeatUntil != nil && in.RepeatUntil.Before(in.StartsAt) {
				return nil, invalid("repeat_until", "повтор до начала смены")
			}
			if _, ok := b.LocationAt(in.LocationID, in.StartsAt); !ok && len(b.Locations) > 0 {
				return nil, notFound("location_id", in.LocationID)
			}
			return shiftData(in.RefShift), nil
		})
}
