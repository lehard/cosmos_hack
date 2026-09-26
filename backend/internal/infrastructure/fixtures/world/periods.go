package world

import (
	"bytes"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	analyticsapp "ant/internal/application/analytics"
)

// Периоды показателей заготовок (FR-3; UI-15): смена, сутки, неделя, месяц —
// те же окна, что у live (application/analytics: resolvePeriod и shiftWindow).
// Границы суток, недели и месяца — по календарю предприятия (пояс сценария,
// МСК); смена — по шаблонам смен справочника (normative/reference/flange/
// shifts.yaml, FR-81), повторяющимся в рабочие дни. Прошлый период — такое же
// окно того же вида, сдвинутое назад (у смены — та же длительность от начала
// предыдущей смены).

// ShiftsFile — шаблоны смен справочника от корня репозитория (вход генератора).
const ShiftsFile = "normative/reference/flange/shifts.yaml"

// shiftPattern — шаблон смены: начало и конец, местное время суток (минуты от
// полуночи); конец не позже начала — смена через полночь.
type shiftPattern struct {
	id, name     string
	starts, ends int
	// locations — места, где идёт смена (цеха справочника мест).
	locations []string
}

// LoadShifts читает шаблоны смен справочника из fsys (корень репозитория).
func LoadShifts(fsys fs.FS) ([]shiftPattern, error) {
	b, err := fs.ReadFile(fsys, ShiftsFile)
	if err != nil {
		return nil, err
	}
	var f struct {
		Version  int    `yaml:"version"`
		TimeZone string `yaml:"time_zone"`
		Patterns []struct {
			ID        string   `yaml:"id"`
			Name      string   `yaml:"name"`
			Starts    string   `yaml:"starts"`
			Ends      string   `yaml:"ends"`
			Locations []string `yaml:"locations"`
		} `yaml:"patterns"`
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("%s: %w", ShiftsFile, err)
	}
	var out []shiftPattern
	for _, p := range f.Patterns {
		s, err1 := clockMinutes(p.Starts)
		e, err2 := clockMinutes(p.Ends)
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("%s: смена %s: время «чч:мм»", ShiftsFile, p.ID)
		}
		out = append(out, shiftPattern{id: p.ID, name: p.Name, starts: s, ends: e, locations: p.Locations})
	}
	return out, nil
}

func clockMinutes(hhmm string) (int, error) {
	h, m, ok := strings.Cut(hhmm, ":")
	if !ok {
		return 0, fmt.Errorf("время %q", hhmm)
	}
	var hh, mm int
	if _, err := fmt.Sscanf(h+" "+m, "%d %d", &hh, &mm); err != nil {
		return 0, err
	}
	return hh*60 + mm, nil
}

// shiftStart — начало смены, идущей в момент t, или, если t вне смен,
// последней начавшейся до t (как ShiftLookup live). Смены — в рабочие дни
// (пн–пт: в месяцах мира заготовок праздников нет). ok=false — шаблонов нет.
func (m *Model) shiftStart(t time.Time) (time.Time, bool) {
	if len(m.shifts) == 0 {
		return time.Time{}, false
	}
	l := t.In(m.clk.loc)
	day := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, m.clk.loc)
	var best time.Time
	for back := 0; back <= 7; back++ {
		d := day.AddDate(0, 0, -back)
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		for _, p := range m.shifts {
			s := d.Add(time.Duration(p.starts) * time.Minute)
			if !s.After(t) && s.After(best) {
				best = s
			}
		}
		if !best.IsZero() {
			break
		}
	}
	return best.UTC(), !best.IsZero()
}

// aWindow — период показателей и прошлый такой же период.
type aWindow struct {
	analyticsapp.Period
	prevFrom, prevTo time.Time
}

// periodKinds — виды периодов переключателя плиток (FR-3); первый — период
// по умолчанию (без параметра period), как у live.
var periodKinds = []string{"shift", "day", "week", "month"}

// window — период вида kind на часах шага (конец — часы шага).
func (c *Ctx) window(kind string) aWindow {
	loc := c.M.clk.loc
	now := c.T.UTC()
	l := now.In(loc)
	day := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
	var from time.Time
	var back func(time.Time) time.Time
	switch kind {
	case "shift":
		// Смена — по графику справочника (FR-81); прошлая — такая же по
		// длительности часть предыдущей смены.
		if s, ok := c.M.shiftStart(now); ok {
			prev, ok := c.M.shiftStart(s.Add(-time.Nanosecond))
			if !ok {
				prev = s.Add(-8 * time.Hour)
			}
			return aWindow{Period: analyticsapp.Period{Kind: kind, From: s, To: now}, prevFrom: prev, prevTo: prev.Add(now.Sub(s))}
		}
		from = day.Add(time.Duration(l.Hour()/8*8) * time.Hour)
		back = func(t time.Time) time.Time { return t.Add(-8 * time.Hour) }
	case "day":
		from = day
		back = func(t time.Time) time.Time { return t.AddDate(0, 0, -1) }
	case "week":
		wd := (int(l.Weekday()) + 6) % 7 // понедельник — 0
		from = day.AddDate(0, 0, -wd)
		back = func(t time.Time) time.Time { return t.AddDate(0, 0, -7) }
	case "month":
		from = time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, loc)
		back = func(t time.Time) time.Time { return t.AddDate(0, -1, 0) }
	default:
		panic("вид периода заготовок: " + kind)
	}
	from = from.UTC()
	return aWindow{Period: analyticsapp.Period{Kind: kind, From: from, To: now}, prevFrom: back(from).UTC(), prevTo: back(now).UTC()}
}

// periodParams — параметры ответа периода: у периода по умолчанию — без period.
func periodParams(kind string, kv ...string) []string {
	if kind == periodKinds[0] {
		return kv
	}
	return append(kv, "period", kind)
}
