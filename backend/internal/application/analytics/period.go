package analytics

import (
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// Периоды счётчиков и показателей (FR-3): смена, сутки, неделя, месяц,
// произвольный интервал. Границы смен и суток — по местному времени
// предприятия (Service.Location, по умолчанию МСК); смены — по 8 ч с 00:00,
// 08:00, 16:00, пока производственный календарь и график смен (reference,
// эпик 19) не дают своих границ.

// Moscow — часовой пояс предприятия по умолчанию (UTC+3, без перехода).
var Moscow = time.FixedZone("MSK", 3*3600)

// window — период [From, To] и длина для сравнения с прошлым периодом.
type window struct {
	Period
	prevFrom, prevTo time.Time
}

// resolvePeriod — период показателей на момент now (конец периода — now;
// для произвольного периода — to, не позже now).
func resolvePeriod(q PeriodQuery, now time.Time, loc *time.Location) (window, error) {
	if loc == nil {
		loc = Moscow
	}
	now = now.UTC()
	local := now.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	kind := q.Kind
	if kind == "" {
		kind = "shift"
	}
	var from time.Time
	var shift func(time.Time) time.Time
	switch kind {
	case "shift":
		from = day.Add(time.Duration(local.Hour()/8*8) * time.Hour)
		shift = func(t time.Time) time.Time { return t.Add(-8 * time.Hour) }
	case "day":
		from = day
		shift = func(t time.Time) time.Time { return t.AddDate(0, 0, -1) }
	case "week":
		wd := (int(local.Weekday()) + 6) % 7 // понедельник — 0
		from = day.AddDate(0, 0, -wd)
		shift = func(t time.Time) time.Time { return t.AddDate(0, 0, -7) }
	case "month":
		from = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
		shift = func(t time.Time) time.Time { return t.AddDate(0, -1, 0) }
	case "custom":
		to := now
		if q.To != nil && q.To.Before(now) {
			to = q.To.UTC()
		}
		if q.From == nil {
			e := platform.Fail(errcodes.ApiValidationFailed, "field", "from", "reason", "для произвольного периода нужно начало")
			return window{}, e
		}
		from = q.From.UTC()
		if from.After(to) {
			return window{}, platform.Fail(errcodes.ApiValidationFailed, "field", "from", "reason", "начало позже конца")
		}
		l := to.Sub(from)
		return window{Period: Period{Kind: kind, From: from, To: to}, prevFrom: from.Add(-l), prevTo: from}, nil
	default:
		return window{}, platform.Fail(errcodes.ApiValidationFailed, "field", "period", "reason", "допустимы shift, day, week, month, custom")
	}
	from = from.UTC()
	return window{Period: Period{Kind: kind, From: from, To: now}, prevFrom: shift(from).UTC(), prevTo: shift(now).UTC()}, nil
}

// bucket — подгруппа контрольной карты по длине периода: смена — час,
// сутки — 2 ч, дольше — сутки.
func bucket(p Period) time.Duration {
	switch l := p.To.Sub(p.From); {
	case l <= 8*time.Hour:
		return time.Hour
	case l <= 24*time.Hour:
		return 2 * time.Hour
	}
	return 24 * time.Hour
}
