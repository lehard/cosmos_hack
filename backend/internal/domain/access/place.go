package access

import (
	"strings"
	"time"
)

// Место и время запроса и полномочия — «домен» Casbin (AD-15): одна строка
// `‹область›|‹с›|‹по›` у назначения роли и `‹место›|‹момент›` у запроса.
// Сопоставляет их одна зарегистрированная функция PlaceMatch: путь области
// (ScopeCovers) и срок полномочия (ValidAt) по доменному времени запроса
// (AD-37). Встроенные timeMatch и условия связей Casbin не используются.

// TimeLayout — время в строках политики: RFC 3339 UTC, три знака после секунд.
const TimeLayout = "2006-01-02T15:04:05.000Z07:00"

// AnyPlace — домен связей наследования ролей: действует в любом месте и в любое время.
const AnyPlace = "*"

// GrantPlace — домен назначения роли: область и срок (until нулевой — бессрочно).
func GrantPlace(scope string, from, until time.Time) string {
	return scope + "|" + formatTime(from) + "|" + formatTime(until)
}

// RequestPlace — домен запроса: место операции (пусто — место не определено) и
// доменное время запроса.
func RequestPlace(scope string, at time.Time) string {
	return scope + "|" + formatTime(at)
}

// PlaceMatch — назначение с доменом grant действует для запроса с доменом
// request: область назначения включает место запроса и момент запроса — в
// сроке назначения. Место запроса не определено (пусто) — область не сужает:
// действие решает роль. Нулевой момент запроса — срок не проверяется.
func PlaceMatch(request, grant string) bool {
	if grant == AnyPlace {
		return true
	}
	rs, rat, ok := strings.Cut(request, "|")
	if !ok {
		return false
	}
	gp := strings.Split(grant, "|")
	if len(gp) != 3 {
		return false
	}
	if rs != "" && !ScopeCovers(gp[0], rs) {
		return false
	}
	at, err := parseTime(rat)
	if err != nil {
		return false
	}
	from, err1 := parseTime(gp[1])
	until, err2 := parseTime(gp[2])
	if err1 != nil || err2 != nil {
		return false
	}
	return ValidAt(at, from, until)
}

// ValidAt — момент at попадает в срок [from, until): нулевой from — с начала,
// нулевой until — бессрочно; нулевой at — срок не проверяется. Сроки
// полномочий и клейм — по доменному времени записи (AD-37).
func ValidAt(at, from, until time.Time) bool {
	if at.IsZero() {
		return true
	}
	if !from.IsZero() && at.Before(from) {
		return false
	}
	if !until.IsZero() && !at.Before(until) {
		return false
	}
	return true
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(TimeLayout)
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, s)
}
