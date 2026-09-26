package notifications

import (
	"time"

	"ant/internal/application/platform"
)

// NotificationSummary — сводка уведомлений пользователя для шапки (FR-57).
type NotificationSummary struct {
	Unread int                        `json:"unread" minimum:"0" doc:"Непрочитанных всего."`
	ByKind *NotificationSummaryByKind `json:"by_kind,omitempty" doc:"По видам: информация, тревога, задача, запрос решения."`
}

// NotificationSummaryByKind — непрочитанные по видам (FR-57).
type NotificationSummaryByKind struct {
	Info            int `json:"info" minimum:"0"`
	Alarm           int `json:"alarm" minimum:"0"`
	Task            int `json:"task" minimum:"0"`
	DecisionRequest int `json:"decision_request" minimum:"0"`
}

// AttentionEntry — строка блока «Требует вашего внимания» (FR-8) поверх живой
// карты. Дискриминированный вид — плоский объект: kind + поля своего вида.
// overdue_decision: target, overdue_minutes, items, operations (цена задержки:
// «просрочено на 37 мин — стоят 18 изделий, 2 операции»);
// unverified_measures, temporary_measures: n.
type AttentionEntry struct {
	Kind           string             `json:"kind" enum:"overdue_decision,unverified_measures,temporary_measures"`
	EntryID        string             `json:"entry_id"`
	Target         *string            `json:"target,omitempty" doc:"overdue_decision: что ждёт решения — изделие, несоответствие, точка предъявления (подпись)."`
	OverdueMinutes *int               `json:"overdue_minutes,omitempty" minimum:"0" doc:"overdue_decision: на сколько просрочено, минуты."`
	Items          *int               `json:"items,omitempty" minimum:"0" doc:"overdue_decision: сколько изделий стоит."`
	Operations     *int               `json:"operations,omitempty" minimum:"0" doc:"overdue_decision: сколько операций стоит."`
	N              *int               `json:"n,omitempty" minimum:"0" doc:"unverified_measures, temporary_measures: сколько мер."`
	Ref            *platform.DrillRef `json:"ref,omitempty" doc:"Куда провалиться (FR-7)."`
}

// AttentionList — блок «Требует вашего внимания».
type AttentionList struct {
	Items []AttentionEntry `json:"items"`
}

// AlertEntry — тревога ленты (FR-8): параметры текста — по виду.
type AlertEntry struct {
	AlertID        string             `json:"alert_id"`
	At             time.Time          `json:"at"`
	Kind           string             `json:"kind" enum:"overdue_isolation,gate_overdue,not_moved_to_isolator,anomaly,escalation,integrity_violation"`
	Item           *string            `json:"item,omitempty" doc:"Изделие (просроченная изоляция, не перемещено в изолятор)."`
	Gate           *string            `json:"gate,omitempty" doc:"Точка предъявления."`
	Node           *string            `json:"node,omitempty" doc:"Узел (step_key) аномалии."`
	NodeName       *string            `json:"node_name,omitempty" doc:"Имя узла — name элемента BPMN действующей версии процесса (для подписи вместо step_key); нет — показывать node."`
	Anomaly        *string            `json:"anomaly,omitempty" doc:"Вид аномалии узла (queue_above_norm, wait_above_norm, downtime_over_threshold, output_spike, defect_rate_out_of_control)."`
	Target         *string            `json:"target,omitempty" doc:"Эскалация: цель."`
	OverdueMinutes *int               `json:"overdue_minutes,omitempty" minimum:"0"`
	Items          *int               `json:"items,omitempty" minimum:"0"`
	Operations     *int               `json:"operations,omitempty" minimum:"0"`
	Ref            *platform.DrillRef `json:"ref,omitempty" doc:"Объект тревоги (FR-7)."`
}

// AlertList — лента тревог.
type AlertList struct {
	Items      []AlertEntry `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

// TaskEntry — задача или уведомление пользователя (FR-57): порождает только
// notifications (task.task.created, AD-40); цена задержки — для эскалаций.
type TaskEntry struct {
	TaskID       string             `json:"task_id"`
	Kind         string             `json:"kind" enum:"physical_move,isolate_move,recheck,decision_required,review_after_new_data,protection_basis_changed,resign,remark_carrier,remove_temporary_carrier,inspection_missing,admin_resend,other"`
	Title        string             `json:"title"`
	State        string             `json:"state" enum:"open,done,accepted,declined,withdrawn"`
	AssigneeRole string             `json:"assignee_role"`
	AssigneeID   *string            `nullable:"true" json:"assignee_id" doc:"Псевдоним исполнителя; null — любой с ролью."`
	LocationID   *string            `json:"location_id,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	DueAt        *time.Time         `nullable:"true" json:"due_at" doc:"Срок по производственному календарю (AD-4); null — без срока."`
	Overdue      bool               `json:"overdue"`
	Ref          *platform.DrillRef `json:"ref,omitempty" doc:"Субъект задачи."`
}

// TaskList — задачи пользователя.
type TaskList struct {
	Items      []TaskEntry `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

// AcknowledgeTask — отметить задачу (task.task.acknowledged): выполнена,
// принята или отклонена с примечанием.
type AcknowledgeTask struct {
	platform.CommandHeader
	Outcome string `json:"outcome" enum:"done,accepted,declined"`
	Note    string `json:"note,omitempty" maxLength:"2000"`
}
