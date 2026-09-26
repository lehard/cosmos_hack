package notifications

import (
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	"ant/internal/domain/process"
)

// GuardProcessStep — задача шага процесса (kind process_step) закрывается
// только своим действием (FR-57, AD-39): операция задачи продвигает токен, и
// задача снимается свёрткой (task.task.withdrawn). Отметка «Выполнено» /
// «Принято» её не закрывает — отказ process.task_closed_by_action с глаголом
// действия. Прочие задачи — nil.
func GuardProcessStep(kind, operation string) error {
	if kind != KindProcessStep {
		return nil
	}
	action := StepAction(operation)
	r := kernel.Refuse(errcodes.ProcessTaskClosedByAction, "action", action)
	r.Detail = "Задача процесса закрывается действием: " + action
	return r
}

// StepAction — действие шага процесса для людей (глагол кнопки): «Принять в
// цех», «Отправить», «Начать», «Завершить»; неизвестная операция — её id.
func StepAction(operation string) string {
	switch operation {
	case process.OpMovementReceive:
		return "«Принять в цех»"
	case process.OpMovementSend:
		return "«Отправить»"
	case process.OpOperationStart:
		return "«Начать»"
	case process.OpOperationFinish:
		return "«Завершить»"
	case "":
		return "действие шага"
	}
	return operation
}
