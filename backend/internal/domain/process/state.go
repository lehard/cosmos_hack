package process

import (
	"maps"
	"slices"
	"time"

	"ant/internal/domain/kernel"
)

// Состояние исполнителя BPMN в свёртке одного изделия (AD-5, AD-17):
// положение токенов со стеком вызовов подпроцессов, переменные условий,
// выполнения операций со связью повторов (FR-47), счётчики доработок (FR-18),
// точки предъявления со счётом предъявлений (FR-19), зоны и скрытые работы
// (FR-20), вмешательства (FR-21), таймеры-окна (Д-8), отправленные сообщения
// в 1С и нарушения предусловий (FR-17). Поля экспортируемые и
// сериализуемые в JSON — они входят в хеш состояния (Д-22).

// Положения изделия в процессе (PRD §3b; AD-30: ось «положение» — process).
const (
	PosInQueue      = "in_queue"
	PosInProgress   = "in_progress"
	PosAtInspection = "at_inspection"
	PosAtGate       = "at_presentation_point"
	PosInTransit    = "in_transit"
	PosInStorage    = "in_storage"
	PosIsolated     = "isolated"
	PosCompleted    = "completed"
)

// Фазы токена на узле.
const (
	PhaseWaiting    = "waiting"     // ждёт факта или решения
	PhaseInProgress = "in_progress" // операция начата
	PhasePaused     = "paused"
	PhaseInTransit  = "in_transit"
	PhaseBlocked    = "blocked" // предусловие с блокировкой (FR-17, FR-18)
	PhaseStuck      = "stuck"   // ни одна ветка развилки не выбрана — нет данных
)

// Cause — ссылка на запись-причину (AD-3, AD-37).
type Cause struct {
	EventID    string    `json:"event_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

func causeOf(r kernel.Record) Cause { return Cause{EventID: r.EventID, OccurredAt: r.OccurredAt} }

// Record — причина как запись для kernel.NewReaction.
func (c Cause) Record() kernel.Record { return kernel.Record{EventID: c.EventID, OccurredAt: c.OccurredAt} }

// Frame — кадр стека вызовов токена: узел вызова (callActivity или встроенный
// подпроцесс) и номер экземпляра вызова. Возврат — в точку вызова (AD-17).
type Frame struct {
	Call string `json:"call"`
	Inst int    `json:"inst"`
}

// Token — токен изделия на узле процесса.
type Token struct {
	ID      int       `json:"id"`
	Node    string    `json:"node"`
	StepKey string    `json:"step_key"`
	Since   time.Time `json:"since"`
	Phase   string    `json:"phase"`
	RunID   string    `json:"run_id,omitempty"`
	Stack   []Frame   `json:"stack,omitempty"`
	// Vars — переменные условий (decision, nc.outcome, test.result…).
	Vars map[string]Value `json:"vars,omitempty"`
	// Basis — записи последней закрывающей точки на пути токена (closing_basis
	// сообщений в 1С).
	Basis []string `json:"basis,omitempty"`
	// Block — почему токен заблокирован (Phase = blocked).
	Block string `json:"block,omitempty"`
}

// Run — выполнение операции (FR-47): шаг, время, связь повторов.
type Run struct {
	RunID      string     `json:"operation_run_id"`
	StepKey    string     `json:"step_key"`
	Node       string     `json:"node"`
	N          int        `json:"n"` // номер выполнения на шаге: 1 — первое
	ReworkOf   string     `json:"rework_of,omitempty"`
	Inferred   bool       `json:"rework_of_inferred,omitempty"`
	Zones      []string   `json:"zones,omitempty"`
	Equipment  string     `json:"equipment_id,omitempty"`
	Operator   string     `json:"operator_id,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	// SourceStart, SourceEnd — границы переданы источником (не occurred_at).
	SourceStart bool   `json:"source_start,omitempty"`
	SourceEnd   bool   `json:"source_end,omitempty"`
	Completion  string `json:"completion,omitempty"`
	// Detached — выполнение не на шаге токена (вне маршрута).
	Detached bool    `json:"detached,omitempty"`
	Causes   []Cause `json:"causes"`
}

// Gate — точка предъявления на пути изделия (FR-19): счёт предъявлений,
// требуемое полномочие текущего предъявления и решения.
type Gate struct {
	StepKey   string   `json:"step_key"`
	Count     int      `json:"count"`
	Authority string   `json:"authority"`
	Decisions []string `json:"decisions,omitempty"`
	Last      string   `json:"last_resolution,omitempty"`
}

// Decision — подписанное решение человека во входе изделия (FR-19, AD-2).
type Decision struct {
	EventID    string `json:"event_id"`
	Type       string `json:"type"`
	StepKey    string `json:"step_key,omitempty"`
	Provenance string `json:"provenance"`
	Signed     bool   `json:"signed"`
}

// Zone — состояние зоны изделия (FR-20, FR-21, FR-46).
type Zone struct {
	CheckedBy string     `json:"checked_by,omitempty"`
	CheckedAt *time.Time `json:"checked_at,omitempty"`
	// Outdated — результаты контроля устарели после вмешательства (FR-21).
	Outdated bool `json:"outdated,omitempty"`
	Defect   bool `json:"defect,omitempty"`
}

// Intervention — запись вмешательства в собранное изделие (FR-21).
type Intervention struct {
	ID       string    `json:"id"`
	Zones    []string  `json:"zones"`
	OpenedAt time.Time `json:"opened_at"`
	Open     bool      `json:"open"`
}

// Timer — взведённый таймер-окно (Д-8): граничный на задаче или
// промежуточный. ObligationID — id срока для notifications (AD-4).
type Timer struct {
	ObligationID string    `json:"obligation_id"`
	Token        int       `json:"token"`
	Node         string    `json:"node"`
	StepKey      string    `json:"step_key"`
	Host         string    `json:"host,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	ArmedAt      time.Time `json:"armed_at"`
	DueAt        time.Time `json:"due_at"`
}

// Thrown — событие-сообщение «в 1С», которого достигло изделие (AD-18).
type Thrown struct {
	StepKey    string   `json:"step_key"`
	Node       string   `json:"node"`
	MessageRef string   `json:"message_ref"`
	ErpAction  string   `json:"erp_action"`
	Visit      int      `json:"visit"`
	Basis      []string `json:"basis"`
	Causes     []Cause  `json:"causes"`
}

// Breach — нарушение предусловия операции (FR-17, FR-18, FR-20, FR-21):
// реакция operation.precondition.failed.
type Breach struct {
	Key          string  `json:"key"`
	StepKey      string  `json:"step_key"`
	RunID        string  `json:"operation_run_id,omitempty"`
	Precondition string  `json:"precondition"`
	Mode         string  `json:"mode"`
	Subject      string  `json:"subject_ref,omitempty"`
	Detail       string  `json:"detail,omitempty"`
	Causes       []Cause `json:"causes"`
}

// Refusal — попытка, которую исполнитель не принял: решение без подписи,
// продвижение не на той точке, факт вне маршрута (FR-19, FR-44).
type Refusal struct {
	Kind    string `json:"kind"`
	StepKey string `json:"step_key,omitempty"`
	EventID string `json:"event_id"`
	Code    string `json:"code"`
	Detail  string `json:"detail,omitempty"`
}

// Gap — шаг, пройденный без данных (догон по факту дальнего шага): на карте
// «оценка невозможна — нет данных», не «норма» (NFR-UI-4).
type Gap struct {
	StepKey string `json:"step_key"`
	EventID string `json:"event_id"`
}

// Visit — посещение шага (для счётчиков карты и истории изделия).
type Visit struct {
	StepKey string     `json:"step_key"`
	Node    string     `json:"node"`
	Entered time.Time  `json:"entered"`
	Left    *time.Time `json:"left,omitempty"`
	// Via — как покинут: completed | timer | skipped | terminated.
	Via string `json:"via,omitempty"`
}

// State — состояние модуля process в свёртке одного изделия (AD-5).
type State struct {
	ItemID      string     `json:"item_id,omitempty"`
	RunID       string     `json:"run_id,omitempty"`
	VersionID   string     `json:"version_id,omitempty"`
	VersionHash string     `json:"version_hash,omitempty"`
	Refused     string     `json:"refused,omitempty"`
	RefusedWhy  string     `json:"refused_detail,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	Lots        []string   `json:"lots,omitempty"`
	// Now — наибольший occurred_at входа (доменное время свёртки, AD-37).
	Now time.Time `json:"now"`

	Tokens    []Token `json:"tokens,omitempty"`
	NextToken int     `json:"next_token,omitempty"`
	NextInst  int     `json:"next_inst,omitempty"`
	Completed bool    `json:"completed,omitempty"`
	// Outcome — released | terminated.
	Outcome     string     `json:"outcome,omitempty"`
	EndStep     string     `json:"end_step,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Isolated    bool       `json:"isolated,omitempty"`
	Containment string     `json:"containment,omitempty"`

	Visits        map[string]int          `json:"visits,omitempty"`
	History       []Visit                 `json:"history,omitempty"`
	Runs          map[string]Run          `json:"runs,omitempty"`
	Rework        map[string]int          `json:"rework,omitempty"`
	Waivers       map[string]int          `json:"waivers,omitempty"`
	Gates         map[string]Gate         `json:"gates,omitempty"`
	Decisions     map[string]Decision     `json:"decisions,omitempty"`
	Applied       map[string]bool         `json:"applied,omitempty"`
	Zones         map[string]Zone         `json:"zones,omitempty"`
	DefectZones   []string                `json:"defect_zones,omitempty"`
	Interventions map[string]Intervention `json:"interventions,omitempty"`
	Timers        []Timer                 `json:"timers,omitempty"`
	Thrown        []Thrown                `json:"thrown,omitempty"`
	Breaches      []Breach                `json:"breaches,omitempty"`
	Refusals      []Refusal               `json:"refusals,omitempty"`
	Gaps          []Gap                   `json:"gaps,omitempty"`
	// Defects — признаки дефекта по шагу: различные зоны в пределах
	// посещения шага (соглашение «Дефект»: повторные наблюдения не множат).
	Defects    map[string]int    `json:"defects,omitempty"`
	DefectSeen map[string]bool   `json:"defect_seen,omitempty"`
	Errors     map[string]string `json:"errors,omitempty"`

	// env — нормативный слой, с которым свёрнута последняя запись: нужен
	// Apply (намерения приходят без Env). Не входит в JSON и хеш состояния.
	env Env
}

// clone — глубокая копия: прежнее состояние свёртки не меняется (AD-4).
func (s State) clone() State {
	s.Lots = slices.Clone(s.Lots)
	toks := make([]Token, len(s.Tokens))
	for i, t := range s.Tokens {
		t.Stack = slices.Clone(t.Stack)
		t.Vars = maps.Clone(t.Vars)
		t.Basis = slices.Clone(t.Basis)
		toks[i] = t
	}
	s.Tokens = toks
	s.Visits = maps.Clone(s.Visits)
	s.History = slices.Clone(s.History)
	s.Runs = maps.Clone(s.Runs)
	s.Rework = maps.Clone(s.Rework)
	s.Waivers = maps.Clone(s.Waivers)
	s.Gates = maps.Clone(s.Gates)
	s.Decisions = maps.Clone(s.Decisions)
	s.Applied = maps.Clone(s.Applied)
	s.Zones = maps.Clone(s.Zones)
	s.DefectZones = slices.Clone(s.DefectZones)
	s.Interventions = maps.Clone(s.Interventions)
	s.Timers = slices.Clone(s.Timers)
	s.Thrown = slices.Clone(s.Thrown)
	s.Breaches = slices.Clone(s.Breaches)
	s.Refusals = slices.Clone(s.Refusals)
	s.Gaps = slices.Clone(s.Gaps)
	s.Errors = maps.Clone(s.Errors)
	s.Defects = maps.Clone(s.Defects)
	s.DefectSeen = maps.Clone(s.DefectSeen)
	return s
}

// Env — нормативный слой, с которым свёрнута последняя запись изделия
// (проекции карты строят положение по описанию процесса версии изделия).
func (s State) Env() Env { return s.env }

func setMap[V any](m *map[string]V, k string, v V) {
	if *m == nil {
		*m = map[string]V{}
	}
	(*m)[k] = v
}

// TokenAt — токен на шаге stepKey (nil — изделие не на шаге).
func (s State) TokenAt(stepKey string) *Token {
	for i := range s.Tokens {
		if s.Tokens[i].StepKey == stepKey {
			return &s.Tokens[i]
		}
	}
	return nil
}

// Steps — step_key шагов, где сейчас стоит изделие (по токенам).
func (s State) Steps() []string {
	var out []string
	for _, t := range s.Tokens {
		if !contains(out, t.StepKey) {
			out = append(out, t.StepKey)
		}
	}
	return out
}

// OpenInterventions — открытые записи вмешательства (FR-21): приёмка и
// отгрузка запрещены, пока список не пуст.
func (s State) OpenInterventions() []Intervention {
	var out []Intervention
	for _, k := range sortedKeys(s.Interventions) {
		if iv := s.Interventions[k]; iv.Open {
			out = append(out, iv)
		}
	}
	return out
}

// Gate — точка предъявления по step_key (ok=false — изделие её не проходило).
func (s State) Gate(stepKey string) (Gate, bool) {
	g, ok := s.Gates[stepKey]
	return g, ok
}
