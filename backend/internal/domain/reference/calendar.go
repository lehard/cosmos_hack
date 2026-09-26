package reference

import (
	"slices"
	"time"

	ev "ant/internal/contracts/events"
)

// Производственный календарь (FR-55, AD-4): сроки в рабочих днях считаются
// по нему, а не по «пн–пт». Даты календаря — местные даты предприятия.

// Local — местное время предприятия: МСК, UTC+3 без перехода на летнее время
// (с 2014 года). Фиксированный пояс, а не база tzdata: домен не читает файлов
// (AD-4), и результат одинаков у воркера и верификатора.
var Local = time.FixedZone("MSK", 3*3600)

// DateLayout — дата справочника YYYY-MM-DD.
const DateLayout = "2006-01-02"

// defaultDaysOff — еженедельные выходные, если календарь года не задан или
// не задал их: суббота и воскресенье (ТК РФ, ст. 111 — пятидневка).
var defaultDaysOff = []time.Weekday{time.Saturday, time.Sunday}

var weekdayCodes = map[ev.ReferenceCalendarDefinedV1WeeklyDaysOffElem]time.Weekday{
	"mon": time.Monday, "tue": time.Tuesday, "wed": time.Wednesday, "thu": time.Thursday,
	"fri": time.Friday, "sat": time.Saturday, "sun": time.Sunday,
}

// CalendarYearView — действующий календарь одного года.
type CalendarYearView struct {
	CalendarID string         `json:"calendar_id"`
	Year       int            `json:"year"`
	DaysOff    []time.Weekday `json:"days_off"`
	NonWorking []string       `json:"non_working"`
	// Working — рабочие дни на еженедельных выходных (перенос).
	Working   []string `json:"working,omitempty"`
	Shortened []string `json:"shortened"`
	// EventID — запись справочника, из которой взят год.
	EventID string `json:"event_id"`
}

// Calendar — производственный календарь по годам (срез справочника).
// Нулевое значение — «пн–пт» без праздников.
type Calendar struct {
	Years []CalendarYearView `json:"years,omitempty"`
}

// Calendar — действующий календарь среза: на каждый год — запись с
// наибольшим seq (календарь года заменяется целиком новой записью).
func (b Book) Calendar() Calendar {
	byYear := map[int]CalendarYear{}
	var years []int
	for _, c := range b.Calendars {
		y := c.Data.Year
		if cur, ok := byYear[y]; !ok || c.Seq > cur.Seq {
			if !ok {
				years = append(years, y)
			}
			byYear[y] = c
		}
	}
	slices.Sort(years)
	var out Calendar
	for _, y := range years {
		c := byYear[y]
		v := CalendarYearView{CalendarID: string(c.Data.CalendarID), Year: y, EventID: c.EventID}
		for _, d := range c.Data.WeeklyDaysOff {
			if wd, ok := weekdayCodes[d]; ok && !slices.Contains(v.DaysOff, wd) {
				v.DaysOff = append(v.DaysOff, wd)
			}
		}
		if len(c.Data.WeeklyDaysOff) == 0 {
			v.DaysOff = slices.Clone(defaultDaysOff)
		}
		slices.Sort(v.DaysOff)
		for _, d := range c.Data.NonWorkingDays {
			v.NonWorking = append(v.NonWorking, string(d))
		}
		for _, d := range c.Data.ShortenedDays {
			v.Shortened = append(v.Shortened, string(d))
		}
		for _, d := range c.Data.WorkingDays {
			v.Working = append(v.Working, string(d))
		}
		slices.Sort(v.Working)
		slices.Sort(v.NonWorking)
		slices.Sort(v.Shortened)
		out.Years = append(out.Years, v)
	}
	return out
}

// Year — календарь года y; ok=false — год не задан (действует «пн–пт»).
func (c Calendar) Year(y int) (CalendarYearView, bool) {
	for _, v := range c.Years {
		if v.Year == y {
			return v, true
		}
	}
	return CalendarYearView{}, false
}

// LocalDate — местная дата момента t (YYYY-MM-DD).
func LocalDate(t time.Time) string { return t.In(Local).Format(DateLayout) }

// Working — рабочий ли местный день, в который попадает момент t.
func (c Calendar) Working(t time.Time) bool {
	l := t.In(Local)
	v, ok := c.Year(l.Year())
	if !ok {
		return !slices.Contains(defaultDaysOff, l.Weekday())
	}
	day := l.Format(DateLayout)
	if _, hit := slices.BinarySearch(v.NonWorking, day); hit {
		return false
	}
	if slices.Contains(v.DaysOff, l.Weekday()) {
		_, hit := slices.BinarySearch(v.Working, day)
		return hit
	}
	return true
}

// Shortened — предпраздничный (сокращённый на час) ли день момента t (ТК РФ, ст. 95).
func (c Calendar) Shortened(t time.Time) bool {
	l := t.In(Local)
	v, ok := c.Year(l.Year())
	if !ok {
		return false
	}
	_, hit := slices.BinarySearch(v.Shortened, l.Format(DateLayout))
	return hit
}

// AddWorkingDays — from плюс days рабочих дней производственного календаря
// (FR-55: «3 рабочих дня по производственному календарю»): местное время
// суток сохраняется, нерабочие дни (выходные, праздники, перенесённые
// выходные) пропускаются. days ≤ 0 — from без изменений.
func (c Calendar) AddWorkingDays(from time.Time, days int) time.Time {
	t := from
	for days > 0 {
		// МСК без перехода: сутки — ровно 24 часа.
		t = t.Add(24 * time.Hour)
		if c.Working(t) {
			days--
		}
	}
	return t
}

// NonWorkingDates — нерабочие даты сверх еженедельных выходных (праздники и
// перенесённые выходные) всех лет: вход календаря модуля notifications
// (Env.Calendar), который считает субботу и воскресенье выходными сам.
func (c Calendar) NonWorkingDates() []string {
	var out []string
	for _, v := range c.Years {
		out = append(out, v.NonWorking...)
	}
	slices.Sort(out)
	return slices.Compact(out)
}
