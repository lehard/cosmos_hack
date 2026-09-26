package reference

import (
	"slices"
	"strings"
	"time"
)

// График смен (FR-81): запись reference.shift.scheduled — одна смена или
// шаблон, повторяющийся каждый (рабочий) день до repeat_until. Смена на
// момент — вход среза «смена» показателей (FR-3, кейс §2.4: длительность
// операций по исполнителю и смене) и границ периода «смена».

// Shift — смена-экземпляр: шаблон на конкретную местную дату.
type Shift struct {
	// ID — `‹shift_id›@YYYY-MM-DD` у повторяющегося шаблона, иначе shift_id.
	ID         string    `json:"id"`
	ShiftID    string    `json:"shift_id"`
	LocationID string    `json:"location_id"`
	Name       string    `json:"name,omitempty"`
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
	// EventID — запись справочника, из которой смена.
	EventID string `json:"event_id"`
}

// maxRepeatDays — предел повторов одного шаблона при переборе (защита от
// бесконечного графика; ~11 лет).
const maxRepeatDays = 4000

// shiftDefs — действующие записи графика: на пару (shift_id, место) —
// запись с наибольшим seq.
func (b Book) shiftDefs() []ShiftDef {
	return latest(b.Shifts, func(x ShiftDef) string { return string(x.Data.ShiftID) + "|" + string(x.Data.LocationID) },
		func(x ShiftDef) Meta { return x.Meta })
}

// ShiftsBetween — смены, пересекающие [from, to), места location (пусто — все
// места), по началу и id.
func (b Book) ShiftsBetween(from, to time.Time, location string) []Shift {
	cal := b.Calendar()
	var out []Shift
	for _, d := range b.shiftDefs() {
		if location != "" && string(d.Data.LocationID) != location {
			continue
		}
		start, end := d.Data.StartsAt.Time().UTC(), d.Data.EndsAt.Time().UTC()
		if !end.After(start) {
			continue
		}
		name := ""
		if d.Data.Name != nil {
			name = *d.Data.Name
		}
		mk := func(s, e time.Time, id string) Shift {
			return Shift{ID: id, ShiftID: string(d.Data.ShiftID), LocationID: string(d.Data.LocationID), Name: name, From: s, To: e, EventID: d.EventID}
		}
		if d.Data.RepeatUntil == nil {
			if start.Before(to) && end.After(from) {
				out = append(out, mk(start, end, string(d.Data.ShiftID)))
			}
			continue
		}
		until := d.Data.RepeatUntil.Time().UTC()
		dur := end.Sub(start)
		// Первый день, чья смена может пересечь окно: с дня начала окна минус
		// длительность смены (МСК без перехода — сутки ровно 24 ч).
		first := 0
		if lead := from.Add(-dur).Sub(start); lead > 0 {
			first = int(lead / (24 * time.Hour))
		}
		for i := first; i < first+maxRepeatDays; i++ {
			s := start.Add(time.Duration(i) * 24 * time.Hour)
			if s.After(until) || !s.Before(to) {
				break
			}
			if d.Data.WorkingDaysOnly != nil && *d.Data.WorkingDaysOnly && !cal.Working(s) {
				continue
			}
			e := s.Add(dur)
			if e.After(from) {
				out = append(out, mk(s, e, string(d.Data.ShiftID)+"@"+LocalDate(s)))
			}
		}
	}
	slices.SortFunc(out, func(a, c Shift) int {
		if !a.From.Equal(c.From) {
			return a.From.Compare(c.From)
		}
		if a.ID != c.ID {
			return strings.Compare(a.ID, c.ID)
		}
		return strings.Compare(a.LocationID, c.LocationID)
	})
	return out
}

// ShiftAt — смена, идущая в момент t (FR-81): сначала смена самого места
// или ближайшего его предка (участок → цех → корпус), без места — любая
// смена предприятия (первая по началу и id). ok=false — t вне смен.
func (b Book) ShiftAt(t time.Time, location string) (Shift, bool) {
	all := b.ShiftsBetween(t, t.Add(time.Nanosecond), "")
	if len(all) == 0 {
		return Shift{}, false
	}
	if location != "" {
		for _, l := range b.Ancestors(location, t) {
			for _, s := range all {
				if s.LocationID == l {
					return s, true
				}
			}
		}
	}
	return all[0], true
}

// ShiftBefore — последняя смена, начавшаяся не позже t (для периода «смена»
// вне рабочего времени: смена, которая была последней). ok=false — не было.
func (b Book) ShiftBefore(t time.Time, location string) (Shift, bool) {
	ss := b.ShiftsBetween(t.Add(-7*24*time.Hour), t.Add(time.Nanosecond), location)
	for i := len(ss) - 1; i >= 0; i-- {
		if !ss[i].From.After(t) {
			return ss[i], true
		}
	}
	return Shift{}, false
}
