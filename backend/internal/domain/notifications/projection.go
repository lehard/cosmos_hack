package notifications

import (
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Глобальные проекции модуля (AD-45: писатель — notifications, роль projector)
// — чистые свёртки записей журнала по ключу:
//   - ObligationRecord — единственная проекция сроков (AD-4): её читает
//     планировщик и по ней считается цена задержки;
//   - TaskRecord — задачи (в потоках изделий и объектов);
//   - NoticeRecord — уведомления (информация, тревога, запрос решения).

// Состояния срока в проекции.
const (
	ObligationOpen    = "open"
	ObligationCleared = "cleared"
)

// ReachRecord — наступивший уровень срока.
type ReachRecord struct {
	DueAt   time.Time `json:"due_at"`
	EventID string    `json:"event_id"`
	Seq     int64     `json:"seq"`
}

// EscalationRecord — эскалация уровня с ценой задержки на момент записи.
type EscalationRecord struct {
	Level          int       `json:"level"`
	Role           string    `json:"role,omitempty"`
	OverdueMinutes int       `json:"overdue_minutes"`
	At             time.Time `json:"at"`
}

// ObligationRecord — строка проекции сроков.
type ObligationRecord struct {
	ObligationID string    `json:"obligation_id"`
	Kind         string    `json:"kind"`
	Basis        string    `json:"basis,omitempty"`
	Subject      string    `json:"subject"`
	ItemID       string    `json:"item_id,omitempty"`
	RunID        string    `json:"run_id,omitempty"`
	OwnerRole    string    `json:"owner_role,omitempty"`
	StepKey      string    `json:"step_key,omitempty"`
	WaitsOn      string    `json:"waits_on,omitempty"`
	Title        string    `json:"title,omitempty"`
	FirstDueAt   time.Time `json:"first_due_at"`
	DueAt        time.Time `json:"due_at"`
	Level        int       `json:"level"`
	State        string    `json:"state"`
	SetAt        time.Time `json:"set_at"`
	ClearedAt    time.Time `json:"cleared_at"`
	// Reached — записанные «наступил срок» (по одной на уровень).
	Reached     []ReachRecord      `json:"reached,omitempty"`
	Escalations []EscalationRecord `json:"escalations,omitempty"`
	Seq         int64              `json:"seq"`
}

// ReachedDue — «наступил срок» для срока due уже записан.
func (o ObligationRecord) ReachedDue(due time.Time) bool {
	return slices.ContainsFunc(o.Reached, func(r ReachRecord) bool { return r.DueAt.Equal(due) })
}

// Due — срок наступил к now и ещё не отмечен: планировщик пишет «наступил
// срок» (AD-4).
func (o ObligationRecord) Due(now time.Time) bool {
	return o.State == ObligationOpen && !o.DueAt.IsZero() && !o.DueAt.After(now) && !o.ReachedDue(o.DueAt)
}

// Overdue — срок просрочен к now (от исходного срока).
func (o ObligationRecord) Overdue(now time.Time) bool {
	return o.State == ObligationOpen && !o.FirstDueAt.IsZero() && now.After(o.FirstDueAt)
}

// OverdueMinutes — на сколько просрочено к now, минут.
func (o ObligationRecord) OverdueMinutes(now time.Time) int {
	if !o.Overdue(now) {
		return 0
	}
	return int(now.Sub(o.FirstDueAt) / time.Minute)
}

// ObligationKeys — ключи проекции сроков для записи (obligation_id).
func ObligationKeys(r kernel.Record) []string {
	switch r.Type {
	case catalog.ObligationDueSet, catalog.ObligationDueCleared, catalog.ObligationDueReached, catalog.ObligationEscalationRaised:
		var d struct {
			ObligationID string `json:"obligation_id"`
		}
		if decode(r, &d) && d.ObligationID != "" {
			return []string{d.ObligationID}
		}
	}
	return nil
}

// StepObligation — шаг проекции сроков.
func StepObligation(v ObligationRecord, r kernel.Record) ObligationRecord {
	v.Seq = r.Seq
	if v.ItemID == "" {
		v.ItemID = r.ItemID
	}
	if v.RunID == "" {
		v.RunID = r.RunID
	}
	switch r.Type {
	case catalog.ObligationDueSet:
		var d DueSetData
		if !decode(r, &d) {
			return v
		}
		v.ObligationID, v.Kind, v.Basis, v.Subject = d.ObligationID, d.Kind, d.Basis, d.SubjectRef
		v.OwnerRole, v.StepKey, v.WaitsOn, v.Title = d.OwnerRoleID, d.StepKey, d.WaitsOn, d.Title
		v.DueAt, _ = ParseTime(d.DueAt)
		v.FirstDueAt, _ = ParseTime(d.FirstDueAt)
		if v.FirstDueAt.IsZero() {
			v.FirstDueAt = v.DueAt
		}
		v.Level = max(d.Level, 1)
		if v.WaitsOn == "" {
			v.WaitsOn = v.Subject
		}
		if v.State != ObligationOpen {
			v.SetAt = r.OccurredAt
		}
		v.State, v.ClearedAt = ObligationOpen, time.Time{}
	case catalog.ObligationDueCleared:
		var d DueClearedData
		if decode(r, &d) {
			v.ObligationID = d.ObligationID
			v.State, v.ClearedAt = ObligationCleared, r.OccurredAt
		}
	case catalog.ObligationDueReached:
		var d DueReachedData
		if decode(r, &d) {
			v.ObligationID = d.ObligationID
			if due, ok := ParseTime(d.DueAt); ok && !v.ReachedDue(due) {
				v.Reached = append(slices.Clone(v.Reached), ReachRecord{DueAt: due, EventID: r.EventID, Seq: r.Seq})
			}
		}
	case catalog.ObligationEscalationRaised:
		var d EscalationData
		if decode(r, &d) {
			v.ObligationID = d.ObligationID
			i := slices.IndexFunc(v.Escalations, func(e EscalationRecord) bool { return e.Level == d.Level })
			e := EscalationRecord{Level: d.Level, Role: d.EscalateToRoleID, OverdueMinutes: d.OverdueMinutes, At: r.OccurredAt}
			v.Escalations = slices.Clone(v.Escalations)
			if i >= 0 {
				v.Escalations[i] = e
			} else {
				v.Escalations = append(v.Escalations, e)
			}
		}
	}
	return v
}

// Состояния задачи в проекции (перечисление TaskEntry.state).
const (
	TaskOpen      = "open"
	TaskWithdrawn = "withdrawn"
)

// TaskRecord — строка проекции задач.
type TaskRecord struct {
	TaskID     string     `json:"task_id"`
	Kind       string     `json:"kind"`
	Title      string     `json:"title"`
	Role       string     `json:"role"`
	Person     string     `json:"person,omitempty"`
	LocationID string     `json:"location_id,omitempty"`
	Subject    string     `json:"subject"`
	ItemID     string     `json:"item_id,omitempty"`
	RunID      string     `json:"run_id,omitempty"`
	DueAt      *time.Time `json:"due_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	State      string     `json:"state"`
	ClosedAt   time.Time  `json:"closed_at"`
	Note       string     `json:"note,omitempty"`
	Seq        int64      `json:"seq"`
	// ItemLabel, Operation, StepKey — метка изделия и действие задачи процесса.
	ItemLabel string `json:"item_label,omitempty"`
	Operation string `json:"operation,omitempty"`
	StepKey   string `json:"step_key,omitempty"`
}

// TaskKeys — ключи проекции задач (task_id).
func TaskKeys(r kernel.Record) []string {
	switch r.Type {
	case catalog.TaskTaskCreated, catalog.TaskTaskWithdrawn, catalog.TaskTaskAcknowledged:
		var d struct {
			TaskID string `json:"task_id"`
		}
		if decode(r, &d) && d.TaskID != "" {
			return []string{d.TaskID}
		}
	}
	return nil
}

// StepTask — шаг проекции задач: поставлена (новая версия — обновлена),
// снята, отмечена человеком (выполнена / принята / отклонена).
func StepTask(v TaskRecord, r kernel.Record) TaskRecord {
	v.Seq = r.Seq
	if v.ItemID == "" {
		v.ItemID = r.ItemID
	}
	if v.RunID == "" {
		v.RunID = r.RunID
	}
	switch r.Type {
	case catalog.TaskTaskCreated:
		var d TaskData
		if !decode(r, &d) {
			return v
		}
		v.TaskID, v.Kind, v.Title, v.Role, v.Person, v.LocationID, v.Subject = d.TaskID, d.Kind, d.Title, d.AssigneeRoleID, d.AssigneePersonID, d.LocationID, d.SubjectRef
		v.ItemLabel, v.Operation, v.StepKey = d.ItemLabel, d.OperationID, d.StepKey
		if d.ItemID != "" {
			v.ItemID = d.ItemID
		}
		v.DueAt = nil
		if t, ok := ParseTime(d.DueAt); ok {
			v.DueAt = &t
		}
		if v.CreatedAt.IsZero() || v.State == TaskWithdrawn {
			v.CreatedAt = r.OccurredAt
		}
		// Новая версия задачи (адресат сменился) не отменяет отметку человека.
		if v.State == "" || v.State == TaskWithdrawn {
			v.State = TaskOpen
		}
	case catalog.TaskTaskWithdrawn:
		var d TaskWithdrawnData
		if decode(r, &d) {
			v.TaskID = d.TaskID
			if v.State == "" || v.State == TaskOpen {
				v.State, v.ClosedAt = TaskWithdrawn, r.OccurredAt
			}
		}
	case catalog.TaskTaskAcknowledged:
		var d AckData
		if decode(r, &d) {
			v.TaskID, v.State, v.ClosedAt, v.Note = d.TaskID, d.Outcome, r.OccurredAt, d.Note
		}
	}
	return v
}

// Состояния уведомления.
const (
	NoticeActive    = "active"
	NoticeWithdrawn = "withdrawn"
)

// NoticeRecord — строка проекции уведомлений.
type NoticeRecord struct {
	NotificationID string            `json:"notification_id"`
	Severity       string            `json:"severity"`
	Role           string            `json:"role,omitempty"`
	Person         string            `json:"person,omitempty"`
	Subject        string            `json:"subject"`
	ItemID         string            `json:"item_id,omitempty"`
	RunID          string            `json:"run_id,omitempty"`
	TextKey        string            `json:"text_key"`
	Params         map[string]string `json:"params,omitempty"`
	At             time.Time         `json:"at"`
	State          string            `json:"state"`
	Seq            int64             `json:"seq"`
}

// NoticeKeys — ключи проекции уведомлений (notification_id).
func NoticeKeys(r kernel.Record) []string {
	switch r.Type {
	case catalog.TaskNotificationSent, catalog.TaskNotificationWithdrawn:
		var d struct {
			NotificationID string `json:"notification_id"`
		}
		if decode(r, &d) && d.NotificationID != "" {
			return []string{d.NotificationID}
		}
	}
	return nil
}

// StepNotice — шаг проекции уведомлений.
func StepNotice(v NoticeRecord, r kernel.Record) NoticeRecord {
	v.Seq = r.Seq
	if v.ItemID == "" {
		v.ItemID = r.ItemID
	}
	if v.RunID == "" {
		v.RunID = r.RunID
	}
	switch r.Type {
	case catalog.TaskNotificationSent:
		var d NotificationData
		if decode(r, &d) {
			v.NotificationID, v.Severity, v.Role, v.Person, v.Subject, v.TextKey, v.Params = d.NotificationID, d.Severity, d.RecipientRoleID, d.RecipientPersonID, d.SubjectRef, d.TextKey, d.Params
			v.At, v.State = r.OccurredAt, NoticeActive
		}
	case catalog.TaskNotificationWithdrawn:
		var d NotificationWithdrawnData
		if decode(r, &d) {
			v.NotificationID, v.State = d.NotificationID, NoticeWithdrawn
		}
	}
	return v
}

// Cost — цена задержки группы (FR-8, FR-57: «просрочено 37 мин — стоят 18
// изделий, 2 операции»): изделия, чьи действующие сроки ждут того же решения
// (waits_on), и операции, на которых они стоят. Чистая функция строк
// проекции сроков.
func Cost(rows []ObligationRecord, waitsOn string) (items, operations int) {
	is, ops := map[string]bool{}, map[string]bool{}
	for _, o := range rows {
		if o.State != ObligationOpen || o.WaitsOn != waitsOn {
			continue
		}
		if o.ItemID != "" {
			is[o.ItemID] = true
		}
		if o.StepKey != "" {
			ops[o.StepKey] = true
		}
	}
	return len(is), len(ops)
}

// GuardAcknowledge — гард отметки задачи (AD-39): задача открыта; итог —
// из перечисления контракта. Ошибка — текст отказа.
func GuardAcknowledge(t TaskRecord, outcome string) error {
	switch outcome {
	case "done", "accepted", "declined":
	default:
		return refusal("итог отметки — done, accepted или declined")
	}
	if t.State != TaskOpen {
		return refusal("задача уже закрыта (" + t.State + ")")
	}
	return nil
}

type refusal string

func (r refusal) Error() string { return string(r) }
