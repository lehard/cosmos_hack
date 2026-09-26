package notifications

import (
	"context"
	"encoding/json"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	notif "ant/internal/domain/notifications"
)

// AcknowledgeTask — отметить задачу (notifications.task.acknowledge, FR-57):
// выполнена, принята или отклонена с примечанием — решение человека
// task.task.acknowledged в поток субъекта задачи (изделие или объект).
// Гард — задача есть и открыта, задача процесса отметкой не закрывается
// (notif.GuardAcknowledge над проекцией задач).
func (s *Service) AcknowledgeTask(ctx context.Context, taskID string, in AcknowledgeTask) (platform.Receipt, error) {
	if !s.live() || s.cfg.Decisions == nil {
		return s.Unimplemented.AcknowledgeTask(ctx, taskID, in)
	}
	raw, ok, err := s.cfg.Projections.Get(ctx, ProjectionTask, taskID)
	if err != nil {
		return platform.Receipt{}, err
	}
	var t notif.TaskRecord
	if ok {
		if err := json.Unmarshal(raw, &t); err != nil {
			return platform.Receipt{}, err
		}
	}
	if !ok || t.TaskID == "" {
		e := platform.Fail(errcodes.ApiNotFound, "object", "Задача", "id", taskID)
		e.Detail = "Задача «" + taskID + "» не найдена"
		return platform.Receipt{}, e
	}
	if err := notif.GuardAcknowledge(t, in.Outcome); err != nil {
		// Отказ с кодом каталога (задача процесса закрывается действием) — как есть.
		if r, ok := err.(*kernel.Refusal); ok {
			return platform.Receipt{}, r
		}
		e := platform.Fail(errcodes.ApiValidationFailed, "field", "task_id", "reason", err.Error())
		e.Detail = err.Error()
		return platform.Receipt{}, e
	}
	now, err := s.now(ctx, platform.Moment{RunID: t.RunID})
	if err != nil {
		return platform.Receipt{}, err
	}
	return s.cfg.Decisions.Write(ctx, Decision{Type: catalog.TaskTaskAcknowledged, Stream: t.Subject, ItemID: t.ItemID, RunID: t.RunID,
		Data: notif.AckData{TaskID: t.TaskID, Outcome: in.Outcome, Note: in.Note}, Meta: in.CommandMeta(),
		Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now, GuardStreams: []string{t.Subject}})
}
