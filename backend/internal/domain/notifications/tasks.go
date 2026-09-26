package notifications

import (
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// ReviewTask — реакция «задача: пересмотрите» (task.task.created; эмитент
// типа — notifications, AD-40), которую ставят правила движка: «решение
// принято до новых данных» (AD-5, FR-32) и «основание защиты изменилось»
// (AD-3). Функция модуля-владельца типа: движок вызывает её, а не строит
// запись чужого модуля сам. task_id — UUIDv5 от слота: пока причины те же,
// задача вычисляется одинаково; новая причина — следующая версия слота.
// person — исполнитель, если известен (пусто — только роль).
func ReviewTask(slot kernel.Slot, kind ev.TaskTaskCreatedV1Kind, title, roleID, person string, causes ...kernel.Record) (kernel.Reaction, error) {
	data := ev.TaskTaskCreatedV1{
		AssigneeRoleID: ev.ObjectID(roleID),
		Kind:           kind,
		SubjectRef:     ev.StreamRef(slot.Subject),
		TaskID:         ev.ObjectID(kernel.UUIDv5(constants.NsAnt, "task\x1f"+slot.Key())),
		Title:          title,
	}
	if person != "" {
		p := ev.PersonRef(person)
		data.AssigneePersonID = &p
	}
	r, err := kernel.NewReaction(Module, catalog.TaskTaskCreated, slot, data, causes...)
	if err != nil {
		return kernel.Reaction{}, err
	}
	// Задача только сообщает (режим автоматизации 1, FR-50).
	r.AutomationMode = 1
	return r, nil
}
