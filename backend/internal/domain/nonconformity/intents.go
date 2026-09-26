package nonconformity

import (
	"ant/internal/contracts/statuses"
	"ant/internal/domain/kernel"
)

// IntentContain — имя намерения «сдерживание».
const IntentContain = "contain"

// Containment — уровень сдерживания и его основание.
type Containment struct {
	Level  statuses.Containment
	Reason string
}

// Contain — функция-намерение nonconformity (AD-30, AD-40): поздний модуль
// (analysis — область риска инцидента) выражает намерение сдержать изделие;
// ось «сдерживание» меняет только nonconformity. Защитное действие при
// пересвёртке не снимается автоматически (AD-3, AD-27).
func Contain(from kernel.Module, c Containment, causes ...kernel.Record) kernel.Intent {
	return kernel.NewIntent(Module, from, IntentContain, c, causes...)
}
