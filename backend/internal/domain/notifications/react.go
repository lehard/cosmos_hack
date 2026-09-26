package notifications

import (
	"slices"
	"strconv"
	"strings"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Данные записей модуля — по схемам contracts/events/obligation и task.
// Время — строкой по соглашению (сгенерированный events.Timestamp пока без
// методов JSON, Д-42).

// DueSetData — obligation.due.set (AD-4).
type DueSetData struct {
	ObligationID string `json:"obligation_id"`
	Kind         string `json:"kind"`
	SubjectRef   string `json:"subject_ref"`
	DueAt        string `json:"due_at"`
	OwnerRoleID  string `json:"owner_role_id,omitempty"`
	StepKey      string `json:"step_key,omitempty"`
	Level        int    `json:"level,omitempty"`
	FirstDueAt   string `json:"first_due_at,omitempty"`
	WaitsOn      string `json:"waits_on,omitempty"`
	Basis        string `json:"basis,omitempty"`
	Title        string `json:"title,omitempty"`
}

// DueClearedData — obligation.due.cleared.
type DueClearedData struct {
	ObligationID string `json:"obligation_id"`
	Cause        string `json:"cause"`
}

// DueReachedData — obligation.due.reached (пишет роль scheduler).
type DueReachedData struct {
	ObligationID string `json:"obligation_id"`
	DueAt        string `json:"due_at"`
}

// EscalationData — obligation.escalation.raised: просрочка с ценой задержки
// (FR-57, FR-8).
type EscalationData struct {
	ObligationID      string `json:"obligation_id"`
	Level             int    `json:"level"`
	EscalateToRoleID  string `json:"escalate_to_role_id,omitempty"`
	OverdueMinutes    int    `json:"overdue_minutes"`
	BlockedItems      *int   `json:"blocked_items,omitempty"`
	BlockedOperations *int   `json:"blocked_operations,omitempty"`
}

// NotificationData — task.notification.sent.
type NotificationData struct {
	NotificationID    string            `json:"notification_id"`
	Severity          string            `json:"severity"`
	RecipientRoleID   string            `json:"recipient_role_id,omitempty"`
	RecipientPersonID string            `json:"recipient_person_id,omitempty"`
	SubjectRef        string            `json:"subject_ref"`
	TextKey           string            `json:"text_key"`
	Params            map[string]string `json:"params,omitempty"`
}

// NotificationWithdrawnData — task.notification.withdrawn.
type NotificationWithdrawnData struct {
	NotificationID string `json:"notification_id"`
}

// TaskData — task.task.created.
type TaskData struct {
	TaskID           string `json:"task_id"`
	Kind             string `json:"kind"`
	AssigneeRoleID   string `json:"assignee_role_id"`
	AssigneePersonID string `json:"assignee_person_id,omitempty"`
	LocationID       string `json:"location_id,omitempty"`
	DueAt            string `json:"due_at,omitempty"`
	SubjectRef       string `json:"subject_ref"`
	Title            string `json:"title"`
	ItemID           string `json:"item_id,omitempty"`
	ItemLabel        string `json:"item_label,omitempty"`
	OperationID      string `json:"operation_id,omitempty"`
	StepKey          string `json:"step_key,omitempty"`
}

// ReasonData — причина (common/defs reason).
type ReasonData struct {
	Code string `json:"code,omitempty"`
	Text string `json:"text"`
}

// TaskWithdrawnData — task.task.withdrawn.
type TaskWithdrawnData struct {
	TaskID string      `json:"task_id"`
	Reason *ReasonData `json:"reason,omitempty"`
}

// AckData — task.task.acknowledged (решение человека).
type AckData struct {
	TaskID  string `json:"task_id"`
	Outcome string `json:"outcome"`
	Note    string `json:"note,omitempty"`
}

// DueSet — реакция obligation.due.set (эмитент типа — notifications, AD-40)
// в слоте slot: срок обязательства с текущим уровнем лестницы. Функция
// модуля-владельца: правила модуля и тесты движка строят запись через неё,
// а не вызывают kernel.NewReaction от имени notifications сами.
func DueSet(slot kernel.Slot, d DueSetData, causes ...kernel.Record) (kernel.Reaction, error) {
	return kernel.NewReaction(Module, catalog.ObligationDueSet, slot, d, causes...)
}

// DueCleared — реакция obligation.due.cleared (эмитент — notifications,
// AD-40) в слоте slot: срок снят (основание исчезло или обязательство
// исполнено, d.Cause).
func DueCleared(slot kernel.Slot, d DueClearedData, causes ...kernel.Record) (kernel.Reaction, error) {
	return kernel.NewReaction(Module, catalog.ObligationDueCleared, slot, d, causes...)
}

// obligationReactions — реакции обязательства (AD-3, AD-4, FR-57):
//   - срок: due.set с текущим уровнем лестницы, пока основание есть; due.cleared,
//     когда исчезло;
//   - на каждый наступивший уровень — эскалация с ценой задержки: на сколько
//     просрочено к этому уровню, стоит ли изделие и сколько операций;
//   - тревога владельцу, пока срок просрочен; снятие срока отзывает её;
//   - информация тому, у кого появилось действие: изделие перемещено в
//     изолятор — технолог может принимать решение.
func obligationReactions(s State, o Obligation) []kernel.Reaction {
	var out []kernel.Reaction
	must := func(r kernel.Reaction, err error) {
		if err != nil {
			panic(err)
		}
		out = append(out, r)
	}
	reachCauses := make([]Cause, 0, len(o.Reaches))
	for _, r := range o.Reaches {
		reachCauses = append(reachCauses, Cause{EventID: r.EventID, At: r.At})
	}
	slot := kernel.Slot{RuleID: RuleObligation, Subject: o.Subject, TriggerKey: o.ID}
	if o.Open {
		d := DueSetData{ObligationID: o.ID, Kind: o.Kind, SubjectRef: o.Subject, DueAt: FormatTime(o.DueAt()), OwnerRoleID: o.OwnerRole,
			StepKey: o.StepKey, Level: o.Level(), FirstDueAt: FormatTime(o.FirstDue), WaitsOn: o.WaitsOn, Basis: o.Basis, Title: truncate(o.Title, 256)}
		must(DueSet(slot, d, records(o.Causes, reachCauses)...))
	} else {
		must(DueCleared(slot, DueClearedData{ObligationID: o.ID, Cause: "fulfilled"},
			records(o.Causes, reachCauses, []Cause{*o.Cleared})...))
	}

	for i, r := range o.Reaches {
		rung := o.Ladder[min(i, len(o.Ladder)-1)]
		items, ops := 1, 0
		if o.StepKey != "" {
			ops = 1
		}
		d := EscalationData{ObligationID: o.ID, Level: r.Level, EscalateToRoleID: rung.Role,
			OverdueMinutes: int(r.DueAt.Sub(o.FirstDue).Minutes()), BlockedItems: &items, BlockedOperations: &ops}
		es := kernel.Slot{RuleID: RuleEscalation, Subject: o.Subject, TriggerKey: o.ID + "/" + strconv.Itoa(r.Level)}
		must(kernel.NewReaction(Module, catalog.ObligationEscalationRaised, es, d, records(o.Causes, []Cause{{EventID: r.EventID, At: r.At}})...))
	}

	if len(o.Reaches) > 0 {
		as := kernel.Slot{RuleID: RuleAlarm, Subject: o.Subject, TriggerKey: o.ID}
		id := NotificationID(as)
		if o.Open {
			last := o.Reaches[len(o.Reaches)-1]
			d := NotificationData{NotificationID: id, Severity: "alarm", RecipientRoleID: o.OwnerRole, SubjectRef: o.Subject,
				TextKey: "notifications.overdue." + o.Basis,
				Params: map[string]string{"title": truncate(o.Title, 256), "level": strconv.Itoa(last.Level), "first_due_at": FormatTime(o.FirstDue),
					"item": o.ItemID, "step_key": o.StepKey, "obligation_id": o.ID}}
			must(kernel.NewReaction(Module, catalog.TaskNotificationSent, as, d, records(o.Causes, reachCauses)...))
		} else {
			must(kernel.NewReaction(Module, catalog.TaskNotificationWithdrawn, as, NotificationWithdrawnData{NotificationID: id},
				records(o.Causes, reachCauses, []Cause{*o.Cleared})...))
		}
	}

	// Информация: изделие в изоляторе, решение по нему ещё не принято.
	if o.Basis == BasisIsolationMove && !o.Open {
		decision := slices.IndexFunc(s.Obligations, func(x Obligation) bool {
			return x.Basis == BasisIsolation && x.Open && x.Key == BasisIsolation+"/"+o.Key[len(BasisIsolationMove)+1:]
		})
		if decision >= 0 {
			x := s.Obligations[decision]
			is := kernel.Slot{RuleID: RuleInfo, Subject: o.Subject, TriggerKey: o.ID}
			d := NotificationData{NotificationID: NotificationID(is), Severity: "info", RecipientRoleID: x.OwnerRole, SubjectRef: o.Subject,
				TextKey: "notifications.info.moved_to_isolator", Params: map[string]string{"item": o.ItemID, "title": x.Title, "due_at": FormatTime(x.FirstDue)}}
			must(kernel.NewReaction(Module, catalog.TaskNotificationSent, is, d, records(o.Causes, []Cause{*o.Cleared})...))
		}
	}
	return out
}

// taskReaction — задача поставлена (пока основание есть) или снята
// (основание у владельца исчезло — выполнено), AD-3.
func taskReaction(t Task) (kernel.Reaction, error) {
	slot := kernel.Slot{RuleID: RuleTask, Subject: t.Subject, TriggerKey: t.Key}
	if !t.Open {
		return kernel.NewReaction(Module, catalog.TaskTaskWithdrawn, slot,
			TaskWithdrawnData{TaskID: t.ID, Reason: &ReasonData{Code: "fulfilled", Text: "Основание задачи снято: " + t.Title}},
			records(t.Causes, []Cause{*t.Closed})...)
	}
	d := TaskData{TaskID: t.ID, Kind: t.Kind, AssigneeRoleID: t.Role, AssigneePersonID: t.Person, LocationID: t.LocationID,
		SubjectRef: t.Subject, Title: truncate(t.Title, 256), ItemLabel: t.ItemLabel, OperationID: t.Operation, StepKey: t.StepKey}
	if strings.HasPrefix(t.Subject, "item:") {
		d.ItemID = strings.TrimPrefix(t.Subject, "item:")
	}
	if t.DueAt != nil {
		d.DueAt = FormatTime(*t.DueAt)
	}
	return kernel.NewReaction(Module, catalog.TaskTaskCreated, slot, d, records(t.Causes)...)
}

// truncate — не длиннее n символов (maxLength схемы).
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
