package process

import "ant/internal/domain/kernel"

// Имена намерений модуля process.
const (
	IntentAdvancePresentation = "advance_presentation"
	IntentIsolate             = "isolate"
)

// Presentation — итог точки предъявления для продвижения токена.
type Presentation struct {
	// StepKey — стабильный ключ шага точки предъявления (AD-17).
	StepKey string
	// Resolution — переменная decision условий BPMN: accept |
	// accept_with_concession | reject | insufficient_data (contracts/bpmn-ext/rules.yaml).
	Resolution string
}

// AdvancePresentation — функция-намерение process (AD-40): поздний модуль
// (quality, nonconformity) продвигает токен точки предъявления; положение в
// процессе меняет только process (AD-30).
func AdvancePresentation(from kernel.Module, p Presentation, causes ...kernel.Record) kernel.Intent {
	return kernel.NewIntent(Module, from, IntentAdvancePresentation, p, causes...)
}

// Isolate — функция-намерение process: изделие переходит в положение «в
// изоляции» (AD-30: изоляция — положение; блок — сдерживание nonconformity).
func Isolate(from kernel.Module, reason string, causes ...kernel.Record) kernel.Intent {
	return kernel.NewIntent(Module, from, IntentIsolate, reason, causes...)
}
