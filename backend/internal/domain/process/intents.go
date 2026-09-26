package process

import "ant/internal/domain/kernel"

// Имена намерений модуля process (AD-40): словарь намерений принадлежит
// владельцу оси «положение в процессе» (AD-30).
const (
	IntentAdvancePresentation = "advance_presentation"
	IntentIsolate             = "isolate"
)

// Решения точки предъявления — значения переменной decision языка условий
// (contracts/bpmn-ext/rules.yaml → conditions.variables.decision).
const (
	ResolutionAccept               = "accept"
	ResolutionAcceptWithConcession = "accept_with_concession"
	ResolutionReject               = "reject"
	ResolutionInsufficientData     = "insufficient_data"
)

// Presentation — итог точки предъявления для продвижения токена (FR-19, FR-44).
//
// Опубликованная сигнатура для модуля quality (эпик 20) и nonconformity
// (эпик 21); поля только добавляются.
type Presentation struct {
	// StepKey — стабильный ключ шага точки предъявления (AD-17): userTask с
	// ant:presentationPoint (закрывающая точка ЗТ-…).
	StepKey string
	// Resolution — переменная decision условий BPMN: accept |
	// accept_with_concession | reject | insufficient_data.
	Resolution string
	// DecisionEventID — event_id подписанного решения человека
	// (decision.presentation.resolved, вид записи decision, происхождение
	// personal или paper). Без него — или если такой записи нет во входе
	// изделия — токен не проходит точку (FR-19: «без подписи не проходит»);
	// пусто — берётся первая причина намерения.
	DecisionEventID string
	// PresentationNo — номер предъявления, к которому относится решение
	// (FR-19); 0 — текущее предъявление на точке.
	PresentationNo int
	// ConcessionID — разрешение на отклонение для accept_with_concession (FR-54).
	ConcessionID string
}

// AdvancePresentation — функция-намерение process (AD-40): поздний модуль
// (quality после проверки полноты методов, FR-14; nonconformity) продвигает
// токен точки предъявления; положение в процессе меняет только process (AD-30).
//
// Применение (Apply) идемпотентно по DecisionEventID: одно решение продвигает
// токен один раз, даже если process уже продвинул его сам по записи
// decision.presentation.resolved. Намерение без подписанного решения, к
// точке, где изделие не стоит, или при открытом вмешательстве (FR-21) токен
// не двигает — попытка видна в State.Refusals.
func AdvancePresentation(from kernel.Module, p Presentation, causes ...kernel.Record) kernel.Intent {
	return kernel.NewIntent(Module, from, IntentAdvancePresentation, p, causes...)
}

// Isolate — функция-намерение process: изделие переходит в положение «в
// изоляции» (AD-30: изоляция — положение; блок — сдерживание nonconformity).
// reason — код причины (для карточки изделия).
func Isolate(from kernel.Module, reason string, causes ...kernel.Record) kernel.Intent {
	return kernel.NewIntent(Module, from, IntentIsolate, reason, causes...)
}
