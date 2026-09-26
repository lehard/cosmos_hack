package notifications

import (
	"strings"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// ObjectReact — задачи по решениям в потоках объектов вне изделия (AD-39:
// инцидент), которые свёртка изделия не видит (FR-57, FR-59, FR-60):
//   - incident.measurement.requested — задача «измерить» исполнителю из
//     запроса, иначе контролёру ОТК (эпик 22: «задачу ставит notifications»);
//   - incident.action.assigned — задача владельцу меры со сроком меры.
//
// Чистая функция записи: слот — (правило, поток объекта, event_id решения),
// версия одна — повтор даёт тот же reaction_id. Исполняет её роль scheduler
// потребителем журнала (application/notifications.ObjectTasks) и пишет
// реакцию в поток объекта.
func ObjectReact(r kernel.Record) []kernel.Reaction {
	if r.Stream == "" || strings.HasPrefix(r.Stream, "item:") {
		return nil
	}
	slot := kernel.Slot{RuleID: RuleObjectTask, Subject: r.Stream, TriggerKey: r.EventID}
	var d TaskData
	switch r.Type {
	case catalog.IncidentMeasurementRequested:
		var m struct {
			IncidentID string `json:"incident_id"`
			What       string `json:"what"`
			AssigneeID string `json:"assignee_id"`
		}
		if !decode(r, &m) || m.What == "" {
			return nil
		}
		d = TaskData{Kind: "recheck", AssigneeRoleID: RoleInspector, AssigneePersonID: m.AssigneeID,
			Title: truncate("Измерение для проверки гипотезы ("+m.IncidentID+"): "+m.What, 256)}
	case catalog.IncidentActionAssigned:
		var m struct {
			IncidentID string `json:"incident_id"`
			ActionID   string `json:"action_id"`
			ActionType string `json:"action_type"`
			OwnerID    string `json:"owner_id"`
			DueAt      string `json:"due_at"`
		}
		if !decode(r, &m) || m.OwnerID == "" {
			return nil
		}
		d = TaskData{Kind: "other", AssigneeRoleID: RoleTechnologist, AssigneePersonID: m.OwnerID,
			Title: truncate("Мера "+m.ActionID+" по инциденту "+m.IncidentID+" ("+actionTitle(m.ActionType)+")", 256)}
		if t, ok := ParseTime(m.DueAt); ok {
			d.DueAt = FormatTime(t)
		}
	default:
		return nil
	}
	d.TaskID, d.SubjectRef = TaskID(slot), r.Stream
	re, err := kernel.NewReaction(Module, catalog.TaskTaskCreated, slot, d, r)
	if err != nil {
		panic(err)
	}
	re.AutomationMode = 1
	return []kernel.Reaction{re}
}

func actionTitle(t string) string {
	switch t {
	case "correction":
		return "коррекция"
	case "corrective_action":
		return "корректирующее действие"
	case "preventive_action":
		return "предупреждающее действие"
	}
	return t
}
