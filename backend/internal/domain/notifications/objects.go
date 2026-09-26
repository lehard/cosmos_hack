package notifications

import (
	"strconv"
	"strings"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
)

// ObjectReact — задачи по решениям в потоках объектов вне изделия (AD-39:
// инцидент), которые свёртка изделия не видит (FR-57, FR-59, FR-60):
//   - incident.measurement.requested — задача «измерить» исполнителю из
//     запроса, иначе контролёру ОТК (эпик 22: «задачу ставит notifications»);
//     incident.measurement.recorded её снимает;
//   - incident.action.assigned — задача владельцу меры со сроком меры;
//   - analyzer.passport.suspended — задача начальнику ОТК «решить о возврате
//     анализатора» после автоотката (эпик 40, FR-101);
//   - incident.suggestion.forwarded — задача ответственному рассмотреть
//     предложение (эпик 42, FR-63, UJ-1);
//   - equipment.deviation.detected «вне уставки» — тревога мастеру участка
//     (решить по посту) и руководителю производства (FR-148, FR-151; показ
//     SHOW-IS2, шаг 5).
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
	case catalog.EquipmentDeviationDetected:
		return deviationTasks(r, slot)
	case catalog.IncidentIncidentOpened:
		// Инцидент открыт системой (эпик 22): технологу — «Разобрать
		// инцидент»; снимается выводом о причине или закрытием инцидента.
		var m struct {
			IncidentID   string `json:"incident_id"`
			CommonFactor string `json:"common_factor"`
			FactorRef    string `json:"factor_ref"`
		}
		if !decode(r, &m) || m.IncidentID == "" {
			return nil
		}
		title := "Разобрать инцидент"
		if m.FactorRef != "" {
			title += " " + m.FactorRef
		}
		title += ": область риска, причина и решение по несоответствиям"
		d = TaskData{Kind: "decision_required", AssigneeRoleID: RoleTechnologist, Title: truncate(title, 256)}
		slot.TriggerKey = investigateKey
	case catalog.IncidentCauseConcluded, catalog.IncidentIncidentClosed:
		open := kernel.Slot{RuleID: RuleObjectTask, Subject: r.Stream, TriggerKey: investigateKey}
		slot.TriggerKey = investigateKey + "/done"
		re, err := kernel.NewReaction(Module, catalog.TaskTaskWithdrawn, slot,
			TaskWithdrawnData{TaskID: TaskID(open), Reason: &ReasonData{Code: "fulfilled", Text: "Причина инцидента установлена или инцидент закрыт"}}, r)
		if err != nil {
			panic(err)
		}
		re.AutomationMode = 1
		return []kernel.Reaction{re}
	case catalog.IncidentMeasurementRecorded:
		// Результат измерения записан — задача «измерить» по запросу снята.
		var m struct {
			RequestEventID string `json:"request_event_id"`
		}
		if !decode(r, &m) || m.RequestEventID == "" {
			return nil
		}
		open := kernel.Slot{RuleID: RuleObjectTask, Subject: r.Stream, TriggerKey: m.RequestEventID}
		re, err := kernel.NewReaction(Module, catalog.TaskTaskWithdrawn, slot,
			TaskWithdrawnData{TaskID: TaskID(open), Reason: &ReasonData{Code: "fulfilled", Text: "Результат измерения записан"}}, r)
		if err != nil {
			panic(err)
		}
		re.AutomationMode = 1
		return []kernel.Reaction{re}
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

// deviationTasks — отклонение режима «вне уставки» на оборудовании (FR-148,
// FR-151): мастеру участка — решить по посту (остановить или перевести
// работу), руководителю производства — тревога для сведения. Только
// действующее отклонение (конца нет): закончившееся — например, из
// опоздавшего журнала — разбирается окном нарушения и областью риска, пост
// останавливать уже поздно. Задача не привязана к изделию: изделия окна
// определит разбор.
func deviationTasks(r kernel.Record, slot kernel.Slot) []kernel.Reaction {
	ev, ok, err := machinelogs.ParseEvent(r)
	if !ok || err != nil || ev.EquipmentID == "" || ev.End != nil || !machinelogs.ViolatesRegime(ev.DeviationKind) {
		return nil
	}
	what := "Режим вне уставки на " + ev.EquipmentID
	if ev.Parameter != "" {
		what += ": " + paramText(ev.Parameter)
		if ev.Value != nil {
			what += " " + measureText(*ev.Value)
		}
		if sp := ev.Setpoint; sp != nil && sp.Lower != nil && sp.Upper != nil {
			what += " при уставке " + measureText(*sp.Lower) + "…" + measureText(*sp.Upper)
		}
	}
	var out []kernel.Reaction
	for _, t := range []struct{ key, role, kind, title string }{
		{"deviation/foreman", RoleForeman, "decision_required", what + ". Решить по посту: остановить или перевести работу"},
		{"deviation/manager", RoleProductionManager, "other", "Тревога: " + what},
	} {
		s := slot
		s.TriggerKey = r.EventID + "/" + t.key
		d := TaskData{Kind: t.kind, AssigneeRoleID: t.role, Title: truncate(t.title, 256), TaskID: TaskID(s), SubjectRef: r.Stream,
			LocationID: ev.StationID} // участок поста: задачу видит мастер своего участка
		re, err := kernel.NewReaction(Module, catalog.TaskTaskCreated, s, d, r)
		if err != nil {
			panic(err)
		}
		re.AutomationMode = 1
		out = append(out, re)
	}
	return out
}

// paramText — параметр режима словами (строчными, внутри фразы).
func paramText(p string) string {
	switch p {
	case "current":
		return "ток"
	case "voltage":
		return "напряжение"
	case "wire_feed":
		return "подача проволоки"
	}
	return p
}

// measureText — значение с масштабом и единицей: 1785, 1, A → «178,5 A».
func measureText(m machinelogs.Measure) string {
	v, sign := m.Value, ""
	if v < 0 {
		sign, v = "-", -v
	}
	s := strconv.FormatInt(v, 10)
	if m.Scale > 0 {
		for len(s) <= m.Scale {
			s = "0" + s
		}
		s = s[:len(s)-m.Scale] + "," + s[len(s)-m.Scale:]
	}
	if m.Unit != "" {
		s += " " + m.Unit
	}
	return sign + s
}

// investigateKey — ключ слота задачи «Разобрать инцидент» в потоке инцидента:
// одна задача на инцидент, снятие — тем же task_id.
const investigateKey = "investigate"

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
