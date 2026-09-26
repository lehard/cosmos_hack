package reference

import (
	"context"
	"strconv"
	"sync"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	notif "ant/internal/domain/notifications"
	dp "ant/internal/domain/process"
	dom "ant/internal/domain/reference"
)

// Bundles — BundleSource движка, дополняющий нормативный слой изделия
// срезом справочника (AD-31): на basis_seq входа (наибольший seq записей
// изделия) и в пределах прогона изделия (AD-38). Подставляет:
//   - предусловия процесса (process.Env.Reference, FR-17): интервалы
//     действия оборудования с поверкой и квалификаций исполнителей — процесс
//     сравнивает с ними дату операции;
//   - производственный календарь сроков (notifications.Env.Calendar, FR-55).
//
// Один и тот же вход даёт один и тот же срез: воркер, пересборка, запрос на
// момент, гард команды и верификатор получают одинаковый нормативный слой.
// Внешний слой цепочки BundleSource: подставляет поля поверх слоёв модулей.
type Bundles struct {
	// Next — нормативный слой модулей (process, quality, item, notifications).
	Next engineapp.BundleSource
	// Source — справочник из журнала.
	Source BookSource

	mu    sync.Mutex
	cache map[string]itemRef
}

var _ engineapp.BundleSource = (*Bundles)(nil)

// itemRef — производная среза для свёртки изделия.
type itemRef struct {
	process  dp.Reference
	calendar *notif.Calendar
}

// cacheMax — предел кэша производных срезов (разных basis_seq × прогон).
const cacheMax = 256

// Bundle — нормативный слой изделия со срезом справочника.
func (b *Bundles) Bundle(ctx context.Context, itemID string, input []kernel.Record) (engine.Bundle, string, error) {
	next := b.Next
	if next == nil {
		next = engineapp.EmptyBundles{}
	}
	bd, rev, err := next.Bundle(ctx, itemID, input)
	if err != nil || b.Source == nil {
		return bd, rev, err
	}
	ref, err := b.itemReference(ctx, dom.ItemFilter(input))
	if err != nil {
		return bd, rev, err
	}
	bd.Process.Reference = ref.process
	if ref.calendar != nil {
		bd.Notifications.Calendar = *ref.calendar
	}
	return bd, rev, nil
}

// itemReference — производная среза справочника по фильтру изделия.
func (b *Bundles) itemReference(ctx context.Context, f dom.Filter) (itemRef, error) {
	book, err := b.Source.Book(ctx, f.BasisSeq)
	if err != nil {
		return itemRef{}, err
	}
	sl := book.Slice(f)
	// Ключ кэша — последняя запись справочника в срезе и прогон: срезы на
	// разные basis_seq без новых записей справочника совпадают.
	key := strconv.FormatInt(sl.UpTo, 10) + "|" + f.RunID
	b.mu.Lock()
	if r, ok := b.cache[key]; ok {
		b.mu.Unlock()
		return r, nil
	}
	b.mu.Unlock()
	r := itemRef{process: ProcessReference(sl)}
	if cal := sl.Calendar(); len(cal.Years) > 0 {
		c := notif.Calendar{NonWorkingDates: cal.NonWorkingDates()}
		r.calendar = &c
	}
	b.mu.Lock()
	if b.cache == nil || len(b.cache) >= cacheMax {
		b.cache = map[string]itemRef{}
	}
	b.cache[key] = r
	b.mu.Unlock()
	return r, nil
}

// ProcessReference — вход предусловий процесса из среза (FR-17): оборудование
// — интервалы, когда оно есть в справочнике и (для средств измерений) поверка
// действует; квалификации — интервалы действия по сотрудникам. Нет записей
// вида в срезе — поле пусто: «не проверено», а не «нарушено» (Д-52).
func ProcessReference(sl dom.Book) dp.Reference {
	var r dp.Reference
	if ids := sl.EquipmentIDs(); len(ids) > 0 {
		r.Equipment = map[string][]dp.Validity{}
		for _, id := range ids {
			vs := []dp.Validity{}
			for _, v := range sl.EquipmentValidity(id) {
				vs = append(vs, dp.Validity{From: v.From, To: v.Until})
			}
			r.Equipment[id] = vs
		}
	}
	if qs, ok := sl.QualificationValidity(); ok {
		r.Qualifications = map[string][]dp.Validity{}
		for person, list := range qs {
			for _, q := range list {
				r.Qualifications[person] = append(r.Qualifications[person], dp.Validity{Ref: q.Qualification, From: q.From, To: q.Until})
			}
		}
	}
	return r
}

// WorkingCalendar — производственный календарь для сроков, которые модуль
// считает при приёме команды (nonconformity: срок решения по изолированному
// изделию, FR-55; порт nonconformity.Calendar): всё известное к моменту
// вызова, без прогона. Нет календаря в справочнике — «пн–пт».
type WorkingCalendar struct {
	Source BookSource
	// Timeout — предел чтения справочника; 0 — 5 с.
	Timeout time.Duration
}

// AddWorkingDays — from плюс days рабочих дней производственного календаря.
// Ошибка чтения справочника — календарь «пн–пт» (срок ставится всегда).
func (w WorkingCalendar) AddWorkingDays(from time.Time, days int) time.Time {
	to := w.Timeout
	if to <= 0 {
		to = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), to)
	defer cancel()
	var cal dom.Calendar
	if w.Source != nil {
		if b, err := w.Source.Book(ctx, 0); err == nil {
			cal = b.Slice(dom.Filter{}).Calendar()
		}
	}
	return cal.AddWorkingDays(from, days)
}

// EquipmentNames — названия оборудования для карточки несоответствия
// (порт nonconformity.EquipmentNames: equipment_label рядом с equipment_id).
// Версия, действующая на at; нет такой — последняя версия оборудования; нет
// оборудования в справочнике или ошибка чтения — названия нет.
type EquipmentNames struct {
	Source BookSource
}

// EquipmentName — название оборудования id на момент at.
func (n EquipmentNames) EquipmentName(ctx context.Context, id string, at time.Time) (string, bool) {
	if n.Source == nil || id == "" {
		return "", false
	}
	b, err := n.Source.Book(ctx, 0)
	if err != nil {
		return "", false
	}
	if e, ok := b.EquipmentAt(id, at); ok && e.Data.Name != "" {
		return e.Data.Name, true
	}
	name := ""
	for _, e := range b.Equipment {
		if string(e.Data.EquipmentID) == id && e.Data.Name != "" {
			name = e.Data.Name // версии — по порядку записи: последняя побеждает
		}
	}
	return name, name != ""
}
