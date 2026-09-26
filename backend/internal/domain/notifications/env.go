package notifications

import (
	"slices"
	"time"
)

// Env — закреплённая при запуске изделия часть нормативного слоя, нужная
// модулю notifications: сроки по политике (FR-55: по умолчанию 3 рабочих дня)
// и лестницы эскалации, и срез производственного календаря (AD-4, AD-31).
// Нулевое значение — умолчания (Defaults): пока справочник календаря и смен
// (эпик 19) и политика сроков не приходят в пакет версии.
type Env struct {
	// DecisionWorkingDays — срок решения по изолированному изделию,
	// несоответствию и изделию в области риска, рабочих дней (FR-55).
	DecisionWorkingDays int `json:"decision_working_days,omitempty"`
	// RecheckWorkingDays — срок доп. проверки без явного срока, рабочих дней.
	RecheckWorkingDays int `json:"recheck_working_days,omitempty"`
	// MoveWithinMin — срок физического перемещения в изолятор, минут (FR-55).
	MoveWithinMin int `json:"move_within_min,omitempty"`
	// PresentationWaitMin — норма ожидания решения на точке предъявления,
	// минут (FR-8: «сроки на точках предъявления»; норма узла по умолчанию, Д-44).
	PresentationWaitMin int `json:"presentation_wait_min,omitempty"`
	// Calendar — производственный календарь; нулевой — пн–пт без праздников.
	Calendar Calendar `json:"calendar"`
	// Ladders — лестницы эскалации по основанию срока; нет — DefaultLadders.
	Ladders map[string][]Rung `json:"ladders,omitempty"`
}

// Calendar — производственный календарь (срез справочника reference на
// occurred_at, AD-31): нерабочие даты сверх субботы и воскресенья. До
// справочника календаря (эпик 19) — пн–пт без праздников.
type Calendar struct {
	// NonWorkingDates — нерабочие даты YYYY-MM-DD (праздники, переносы).
	NonWorkingDates []string `json:"non_working_dates,omitempty"`
}

// Working — рабочий ли день у даты t (UTC).
func (c Calendar) Working(t time.Time) bool {
	if wd := t.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return false
	}
	return !slices.Contains(c.NonWorkingDates, t.UTC().Format("2006-01-02"))
}

// AddWorkingDays — from плюс days рабочих дней (FR-55: «3 рабочих дня по
// производственному календарю»): время суток сохраняется, нерабочие дни
// пропускаются.
func (c Calendar) AddWorkingDays(from time.Time, days int) time.Time {
	t := from
	for days > 0 {
		t = t.AddDate(0, 0, 1)
		if c.Working(t) {
			days--
		}
	}
	return t
}

// DefaultLadders — лестницы эскалации по умолчанию (FR-57): первый уровень
// наступает в срок и уходит руководителю владельца, следующие — через
// смещение от исходного срока выше. Спайн и PRD значения не задают — это
// проектное допущение, меняется политикой.
var DefaultLadders = map[string][]Rung{
	BasisIsolation:     {{AfterMin: 0, Role: RoleHeadOfQC}, {AfterMin: 8 * 60, Role: RoleProductionManager}},
	BasisNC:            {{AfterMin: 0, Role: RoleHeadOfQC}, {AfterMin: 8 * 60, Role: RoleProductionManager}},
	BasisIsolationMove: {{AfterMin: 0, Role: RoleHeadOfWorkshop}, {AfterMin: 120, Role: RoleHeadOfQC}},
	BasisPresentation:  {{AfterMin: 0, Role: RoleHeadOfQC}, {AfterMin: 30, Role: RoleProductionManager}},
	BasisIncidentScope: {{AfterMin: 0, Role: RoleHeadOfQC}, {AfterMin: 8 * 60, Role: RoleProductionManager}},
	BasisRecheck:       {{AfterMin: 0, Role: RoleHeadOfQC}},
}

// Defaults — Env с умолчаниями вместо нулевых полей.
func (e Env) Defaults() Env {
	if e.DecisionWorkingDays <= 0 {
		e.DecisionWorkingDays = 3
	}
	if e.RecheckWorkingDays <= 0 {
		e.RecheckWorkingDays = 1
	}
	if e.MoveWithinMin <= 0 {
		e.MoveWithinMin = 120
	}
	if e.PresentationWaitMin <= 0 {
		e.PresentationWaitMin = 30
	}
	return e
}

// ladder — лестница эскалации основания (копия).
func (e Env) ladder(basis string) []Rung {
	if l, ok := e.Ladders[basis]; ok && len(l) > 0 {
		return slices.Clone(l)
	}
	return slices.Clone(DefaultLadders[basis])
}
