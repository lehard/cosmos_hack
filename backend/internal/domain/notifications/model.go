package notifications

import (
	"slices"
	"time"

	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
)

// Правила модуля — слоты реакций (AD-3): правило, субъект (поток изделия или
// объекта), ключ срабатывания. reaction_id = UUIDv5(слот ‖ версия).
const (
	// RuleObligation — срок обязательства: obligation.due.set, пока
	// обязательство действует, и obligation.due.cleared, когда основание у
	// модуля-владельца исчезло (AD-4). Ключ — obligation_id.
	RuleObligation = "notifications.obligation"
	// RuleEscalation — эскалация уровня лестницы (obligation.escalation.raised);
	// ключ — obligation_id/уровень.
	RuleEscalation = "notifications.escalation"
	// RuleAlarm — тревога владельцу просроченного срока (task.notification.sent,
	// severity alarm); снимается отзывом, когда срок снят.
	RuleAlarm = "notifications.alarm"
	// RuleInfo — информация тому, у кого появилось действие (severity info).
	RuleInfo = "notifications.info"
	// RuleTask — адресная задача по состоянию изделия (task.task.created /
	// task.task.withdrawn); ключ — ключ задачи.
	RuleTask = "notifications.task"
	// RuleObjectTask — задача по решению в потоке объекта вне изделия
	// (инцидент: запрос измерения, назначенная мера) — ObjectReact.
	RuleObjectTask = "notifications.object_task"
)

// Основания сроков (поле basis obligation.due.set): у каждого — модуль-
// владелец состояния, по которому notifications ставит и снимает срок.
const (
	// BasisIsolation — изолированное изделие ждёт решения (nonconformity, FR-55).
	BasisIsolation = "isolation"
	// BasisIsolationMove — «изолировано в системе, физически не перемещено» (FR-55).
	BasisIsolationMove = "isolation_move"
	// BasisNC — подтверждённое несоответствие ждёт решения (FR-51, FR-53).
	BasisNC = "nonconformity"
	// BasisPresentation — изделие ждёт решения на точке предъявления (FR-19).
	BasisPresentation = "presentation"
	// BasisIncidentScope — изделие в области риска инцидента на блоке или доп.
	// проверке ждёт решения по области (FR-62).
	BasisIncidentScope = "incident_scope"
	// BasisRecheck — назначенная доп. проверка не выполнена (FR-52).
	BasisRecheck = "recheck"
	// BasisBPMNTimer — окно BPMN (таймер, Д-8): срок взвёл исполнитель
	// процесса (эпик 17), «наступил срок» с тем же obligation_id его срабатывает.
	BasisBPMNTimer = "bpmn_timer"
)

// Роли адресатов (normative/policy/policy.v1.yaml): задачи и уведомления —
// только тем, у кого есть действие (FR-57).
const (
	RoleInspector         = "quality_inspector"
	RoleHeadOfQC          = "head_of_qc"
	RoleForeman           = "site_foreman"
	RoleHeadOfWorkshop    = "head_of_workshop"
	RoleTechnologist      = "technologist"
	RoleProductionManager = "production_manager"
)

// Cause — запись-основание: event_id и occurred_at (occurred_at реакции —
// наибольший среди причин, AD-37).
type Cause struct {
	EventID string    `json:"event_id"`
	At      time.Time `json:"at"`
}

// Rung — ступень лестницы эскалации: через сколько минут после исходного
// срока и кому эскалируется (FR-57: «запрос решения — срок, эскалация»).
type Rung struct {
	AfterMin int    `json:"after_min"`
	Role     string `json:"role"`
}

// Reach — «наступил срок» уровня лестницы: запись планировщика
// obligation.due.reached (id = UUIDv5(obligation_id, due_at), AD-4).
type Reach struct {
	Level   int       `json:"level"`
	DueAt   time.Time `json:"due_at"`
	EventID string    `json:"event_id"`
	At      time.Time `json:"at"`
}

// Obligation — обязательство изделия со сроком (AD-4): ставит и снимает его
// только notifications по состоянию модуля-владельца основания.
type Obligation struct {
	// ID — obligation_id: UUIDv5 от изделия, основания и ключа основания —
	// стабилен при пересвёртке; он же пространство имён id «наступил срок».
	ID string `json:"id"`
	// Key — ключ основания (например, isolation/‹event_id решения›).
	Key   string `json:"key"`
	Kind  string `json:"kind"`
	Basis string `json:"basis"`
	// Subject — поток изделия; ItemID — изделие.
	Subject string `json:"subject"`
	ItemID  string `json:"item_id"`
	// OwnerRole — роль, у которой действие (FR-57).
	OwnerRole string `json:"owner_role"`
	// StepKey, LocationID — где стоит изделие: операция и место.
	StepKey    string `json:"step_key,omitempty"`
	LocationID string `json:"location_id,omitempty"`
	// WaitsOn — чьего решения ждёт срок (изделие, несоответствие, инцидент):
	// по нему сводится цена задержки (FR-8).
	WaitsOn string `json:"waits_on"`
	Title   string `json:"title"`
	// FirstDue — исходный срок (уровень 1) по производственному календарю.
	FirstDue time.Time `json:"first_due"`
	Ladder   []Rung    `json:"ladder"`
	Causes   []Cause   `json:"causes"`
	Open     bool      `json:"open"`
	// Cleared — запись, на которой основание исчезло (срок снят).
	Cleared *Cause  `json:"cleared,omitempty"`
	Reaches []Reach `json:"reaches,omitempty"`
}

// Armed — у обязательства есть ещё не наступивший уровень лестницы.
func (o Obligation) Armed() bool { return len(o.Reaches) < len(o.Ladder) }

// Level — уровень текущего срока (1 — исходный).
func (o Obligation) Level() int { return min(len(o.Reaches), len(o.Ladder)-1) + 1 }

// DueAt — текущий срок: исходный плюс смещение уровня лестницы.
func (o Obligation) DueAt() time.Time {
	if len(o.Ladder) == 0 {
		return o.FirstDue
	}
	return o.FirstDue.Add(time.Duration(o.Ladder[o.Level()-1].AfterMin) * time.Minute)
}

// Task — адресная задача по состоянию изделия (FR-57): владелец, срок,
// подтверждение. Снимается, когда основание у модуля-владельца исчезло.
type Task struct {
	ID         string     `json:"id"`
	Key        string     `json:"key"`
	Kind       string     `json:"kind"`
	Role       string     `json:"role"`
	Person     string     `json:"person,omitempty"`
	LocationID string     `json:"location_id,omitempty"`
	Title      string     `json:"title"`
	Subject    string     `json:"subject"`
	DueAt      *time.Time `json:"due_at,omitempty"`
	Causes     []Cause    `json:"causes"`
	Open       bool       `json:"open"`
	Closed     *Cause     `json:"closed,omitempty"`
	// ItemLabel — метка изделия для людей (Ф-001, DM-код), не внутренний id.
	ItemLabel string `json:"item_label,omitempty"`
	// Operation, StepKey — задача процесса: какую операцию API нажать и на
	// каком шаге стоит изделие (process.HumanSteps).
	Operation string `json:"operation,omitempty"`
	StepKey   string `json:"step_key,omitempty"`
}

// Where — где изделие физически и на какой операции: из фактов выполнения,
// перемещения и предъявления (до состояний item и process — эпики 17, 18).
type Where struct {
	StepKey    string    `json:"step_key,omitempty"`
	LocationID string    `json:"location_id,omitempty"`
	At         time.Time `json:"at"`
}

// ObligationID — obligation_id обязательства изделия по основанию.
func ObligationID(itemID, key string) string {
	return kernel.UUIDv5(constants.NsAnt, "obligation\x1f"+itemID+"\x1f"+key)
}

// ReachedID — event_id записи «наступил срок» = UUIDv5(obligation_id, due_at)
// (AD-4): повторный запуск планировщика даёт тот же id и не дублирует запись.
func ReachedID(obligationID string, dueAt time.Time) string {
	return kernel.UUIDv5(obligationID, FormatTime(dueAt))
}

// TaskID — task_id задачи по слоту (как у ReviewTask).
func TaskID(slot kernel.Slot) string {
	return kernel.UUIDv5(constants.NsAnt, "task\x1f"+slot.Key())
}

// NotificationID — notification_id уведомления по слоту.
func NotificationID(slot kernel.Slot) string {
	return kernel.UUIDv5(constants.NsAnt, "notification\x1f"+slot.Key())
}

// TimeLayout — время записей по соглашению (три знака после секунд, UTC).
const TimeLayout = "2006-01-02T15:04:05.000Z"

// FormatTime — время по соглашению «Соглашения/Время».
func FormatTime(t time.Time) string { return t.UTC().Truncate(time.Millisecond).Format(TimeLayout) }

// ParseTime — разбор времени RFC 3339 (нулевое время — ошибка разбора).
func ParseTime(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func (s State) clone() State {
	out := s
	out.Obligations = make([]Obligation, len(s.Obligations))
	for i, o := range s.Obligations {
		o.Causes = slices.Clone(o.Causes)
		o.Reaches = slices.Clone(o.Reaches)
		o.Ladder = slices.Clone(o.Ladder)
		if o.Cleared != nil {
			c := *o.Cleared
			o.Cleared = &c
		}
		out.Obligations[i] = o
	}
	out.Tasks = make([]Task, len(s.Tasks))
	for i, t := range s.Tasks {
		t.Causes = slices.Clone(t.Causes)
		if t.Closed != nil {
			c := *t.Closed
			t.Closed = &c
		}
		if t.DueAt != nil {
			d := *t.DueAt
			t.DueAt = &d
		}
		out.Tasks[i] = t
	}
	return out
}

func (s *State) obligation(id string) *Obligation {
	i := slices.IndexFunc(s.Obligations, func(o Obligation) bool { return o.ID == id })
	if i < 0 {
		return nil
	}
	return &s.Obligations[i]
}

func (s *State) task(id string) *Task {
	i := slices.IndexFunc(s.Tasks, func(t Task) bool { return t.ID == id })
	if i < 0 {
		return nil
	}
	return &s.Tasks[i]
}

// records — причины в виде записей для kernel.NewReaction.
func records(cs ...[]Cause) []kernel.Record {
	var out []kernel.Record
	for _, l := range cs {
		for _, c := range l {
			out = append(out, kernel.Record{EventID: c.EventID, OccurredAt: c.At})
		}
	}
	return out
}
