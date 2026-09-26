package simulation

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"
)

// TimeLayout — время в событиях и ответах: RFC 3339 UTC, ровно три знака после
// секунд («Соглашения/Время»).
const TimeLayout = "2006-01-02T15:04:05.000Z"

// Week — шаг сдвига прогона: целые недели сохраняют дни недели, смены и
// «рабочие дни» сроков (AD-38: даты определений — смещения от «сейчас»).
const Week = 7 * 24 * time.Hour

// ParseTime разбирает время определения: RFC 3339 с зоной (как в таблицах
// процессной сессии, «+03:00»).
func ParseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("пустое время")
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("время %q: %w", s, err)
	}
	return t.UTC(), nil
}

// FormatTime — время по соглашению контракта.
func FormatTime(t time.Time) string { return t.UTC().Format(TimeLayout) }

// ParseOffset разбирает сдвиг «+7m», «-30s», «+2h», «+1d».
func ParseOffset(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	if strings.HasSuffix(s, "d") {
		d, err := time.ParseDuration(strings.TrimSuffix(s, "d") + "h")
		return d * 24, err
	}
	return time.ParseDuration(strings.TrimPrefix(s, "+"))
}

// WeekShift — сдвиг прогона: наименьшее целое число недель, при котором
// начало прогона (anchor + сдвиг) не раньше доменного «сейчас» now (AD-37:
// recorded_at журнала не убывает; AD-38: даты — смещения от «сейчас»).
func WeekShift(anchor, now time.Time) time.Duration {
	if now.IsZero() || !now.After(anchor) {
		return 0
	}
	d := now.Sub(anchor)
	n := d / Week
	if d%Week != 0 {
		n++
	}
	return n * Week
}

var (
	reDateTime = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:\d{2})$`)
	reDate     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// ShiftValue сдвигает на d все моменты и даты внутри значения (строки RFC 3339
// и YYYY-MM-DD): ожидаемые значения и параметры шагов пишутся в датах
// определения, прогон идёт в своих датах (AD-38). Время приводится к UTC
// формата контракта.
func ShiftValue(v any, d time.Duration) any {
	switch x := v.(type) {
	case string:
		return shiftString(x, d)
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = ShiftValue(x[i], d)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for _, k := range slices.Sorted(maps.Keys(x)) {
			out[k] = ShiftValue(x[k], d)
		}
		return out
	}
	return v
}

func shiftString(s string, d time.Duration) string {
	switch {
	case reDateTime.MatchString(s):
		t, err := time.Parse(time.RFC3339Nano, normalizeSeconds(s))
		if err != nil {
			return s
		}
		return FormatTime(t.Add(d))
	case reDate.MatchString(s):
		if d == 0 {
			return s
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return s
		}
		return t.Add(d).Format("2006-01-02")
	}
	return s
}

// normalizeSeconds добавляет секунды к «ЧЧ:ММ» перед зоной, если их нет.
func normalizeSeconds(s string) string {
	i := strings.IndexByte(s, 'T')
	if i < 0 || len(s) < i+6 {
		return s
	}
	rest := s[i+1:]
	if len(rest) >= 6 && rest[5] != ':' {
		return s[:i+6] + ":00" + s[i+6:]
	}
	return s
}
