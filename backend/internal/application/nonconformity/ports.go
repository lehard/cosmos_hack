package nonconformity

import (
	"context"
	"time"

	dom "ant/internal/domain/nonconformity"
)

// Ведомые порты модуля nonconformity.

// RouteGate — порт «маршрут подписей закрыт» (AD-43): решения режима 4
// (ремонт, «как есть») и режима 5 (разрешение на отклонение) исполняются
// только после закрытия маршрута подписей документа решения; маршрут
// считает модуль documents (эпик 28, второй слой) реакцией
// document.route.closed. Порт отвечает, в каком состоянии подписи на момент
// записи решения: route_closed | pending | demo_stub.
type RouteGate interface {
	Approvals(ctx context.Context, action, documentID string) (string, error)
}

// DemoRoutes — разрешающая заглушка порта для демо-профиля (демо-трек,
// эпик 28 — второй слой): решение исполняется сразу, в записи и в карточке
// помечено approvals_status = demo_stub — «подписи маршрута не проверялись».
type DemoRoutes struct{}

// Approvals — всегда demo_stub.
func (DemoRoutes) Approvals(context.Context, string, string) (string, error) {
	return dom.ApprovalsDemoStub, nil
}

// PendingRoutes — строгая заглушка: подписи не собраны, решение ждёт
// document.route.closed (профиль prod до эпика 28; тесты режима 4).
type PendingRoutes struct{}

// Approvals — всегда pending.
func (PendingRoutes) Approvals(context.Context, string, string) (string, error) {
	return dom.ApprovalsPending, nil
}

// Calendar — производственный календарь (FR-55, AD-37): срок решения по
// изолированному изделию — N рабочих дней. Реализация справочника календаря
// и смен — эпик 19; до него — WeekdayCalendar.
type Calendar interface {
	AddWorkingDays(from time.Time, days int) time.Time
}

// WeekdayCalendar — календарь «пн–пт рабочие» без праздников (заглушка до
// справочника reference, эпик 19).
type WeekdayCalendar struct{}

// AddWorkingDays — from плюс days рабочих дней (суббота и воскресенье пропускаются).
func (WeekdayCalendar) AddWorkingDays(from time.Time, days int) time.Time {
	t := from
	for days > 0 {
		t = t.AddDate(0, 0, 1)
		if wd := t.Weekday(); wd != time.Saturday && wd != time.Sunday {
			days--
		}
	}
	return t
}
