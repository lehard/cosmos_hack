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
//   - incident.action.assigned — задача владельцу меры со сроком меры;
//   - analyzer.passport.suspended — задача начальнику ОТК «решить о возврате
//     анализатора» после автоотката (эпик 40, FR-101);
//   - incident.suggestion.forwarded — задача ответственному рассмотреть
//     предложение (эпик 42, FR-63, UJ-1).
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
	case catalog.IncidentSuggestionForwarded:
		// Эпик 42 (FR-63, UJ-1): руководитель передал предложение — задача
		// ответственному; система сама ничего не меняет.
		var m struct {
			SuggestionID    string `json:"suggestion_id"`
			ResponsibleID   string `json:"responsible_id"`
			ResponsibleRole string `json:"responsible_role"`
			Title           string `json:"title"`
		}
		if !decode(r, &m) || m.ResponsibleID == "" {
			return nil
		}
		role := m.ResponsibleRole
		if role == "" {
			role = RoleForeman
		}
		title := m.Title
		if title == "" {
			title = m.SuggestionID
		}
		d = TaskData{Kind: "other", AssigneeRoleID: role, AssigneePersonID: m.ResponsibleID,
			Title: truncate("Предложение "+m.SuggestionID+": "+title+" — рассмотреть и принять решение", 256)}
	case catalog.AnalyzerPassportSuspended:
		// Эпик 40 (FR-101): автооткат анализатора — задача начальнику ОТК:
		// контроль стал строже, вернуть анализатор может только он.
		var m struct {
			PassportID string `json:"passport_id"`
			Trigger    string `json:"trigger"`
			Fallback   string `json:"fallback"`
		}
		if !decode(r, &m) || m.PassportID == "" {
			return nil
		}
		d = TaskData{Kind: "decision_required", AssigneeRoleID: RoleHeadOfQC,
			Title: truncate("Анализатор приостановлен автооткатом ("+rollbackTrigger(m.Trigger)+"): паспорт "+m.PassportID+
				", сейчас "+rollbackFallback(m.Fallback)+". Вернуть в работу — только ваше решение", 256)}
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

// rollbackTrigger — триггер автоотката по-русски.
func rollbackTrigger(t string) string {
	switch t {
	case "drift":
		return "дрейф"
	case "reference_set_failed":
		return "провал эталонного набора"
	case "disagreement_growth":
		return "рост расхождений с людьми"
	case "escape_detected":
		return "пропуск брака"
	}
	return t
}

// rollbackFallback — что действует вместо приостановленного паспорта.
func rollbackFallback(f string) string {
	if f == "previous_passport" {
		return "действует предыдущая допущенная версия"
	}
	return "100 % ручной контроль"
}
