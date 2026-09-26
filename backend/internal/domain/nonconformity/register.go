package nonconformity

import (
	"errors"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
)

// RegisterWindowNC — функция-намерение nonconformity для межизделийной стадии
// (FR-151, AD-29, AD-42): регистрирует несоответствие выполнению операции
// специального процесса, попавшему в окно нарушения режима оборудования, —
// даже если дефект при контроле не найден. Строит адресованную запись
// decision.nonconformity.registered в поток изделия; решение по такому
// несоответствию принимает комиссия (маршрут подписей, §11.9) — в свёртке
// изделия несоответствие получает признак Commission.
//
// machinelogs стоит в композиции раньше nonconformity и не импортирует его:
// функцию подставляет сборка стадии domain/crossitem через порт
// machinelogs.Registrar (StageWith).
func RegisterWindowNC(q machinelogs.NCRequest) (kernel.Addressed, error) {
	if q.ItemID == "" || q.OperationRunID == "" || q.WindowEventID == "" {
		return kernel.Addressed{}, errors.New("nonconformity: запрос несоответствия окна без изделия, выполнения или окна")
	}
	ncID := q.NCID
	if ncID == "" {
		ncID = WindowNCID(q.WindowEventID)
	}
	data := ev.DecisionNonconformityRegisteredV1{
		NcID:                   ev.ObjectID(ncID),
		OperationRunID:         ev.ObjectID(q.OperationRunID),
		ViolationWindowEventID: ev.UUID(q.WindowEventID),
	}
	if q.StepKey != "" {
		sk := ev.StepKey(q.StepKey)
		data.StepKey = &sk
	}
	return kernel.NewAddressed(Module, catalog.DecisionNonconformityRegistered, "item:"+q.ItemID, q.Key(), data, q.CauseRecords()...)
}
