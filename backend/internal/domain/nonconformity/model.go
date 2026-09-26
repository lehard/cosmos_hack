package nonconformity

import (
	"slices"
	"time"

	"ant/internal/contracts/statuses"
)

// Статусы несоответствия «по изделию» (FR-51; перечисление NCCard.status).
const (
	StatusDraft          = "draft"
	StatusConfirmed      = "confirmed"
	StatusDispositionSet = "disposition_set"
	StatusVerified       = "verified"
	StatusClosed         = "closed"
)

// Статусы «системного расследования» (FR-51): закрытие несоответствия по
// изделию не закрывает расследование.
const (
	InvestigationNone   = "none"
	InvestigationOpen   = "open"
	InvestigationClosed = "closed"
)

// Происхождение несоответствия.
const (
	// OriginSignal — черновик по сигналу (quality, machinelogs — намерение Draft).
	OriginSignal = "signal"
	// OriginSpecialProcess — регистрация правилом окна нарушения специального
	// процесса (FR-151); решение — комиссия.
	OriginSpecialProcess = "special_process"
)

// Исходы несоответствия, закрытого без решения по изделию.
const (
	// ResolutionSignalRejected — все сигналы черновика отклонены контролёром.
	ResolutionSignalRejected = "signal_rejected"
)

// Состояние подписей маршрута решения (approvals_status, FR-50 режим 4, AD-43).
const (
	ApprovalsRouteClosed = "route_closed"
	ApprovalsPending     = "pending"
	ApprovalsDemoStub    = "demo_stub"
)

// Кто установил сдерживание.
const (
	ByRule  = "rule"
	ByHuman = "human"
)

// NC — несоответствие изделия в свёртке (FR-51): исходный сигнал (поля
// запроса черновика), решение контролёра и решение по несоответствию —
// раздельно; исходные записи не меняются, каждое решение — своя запись.
type NC struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Origin string `json:"origin"`
	// FoundAt — occurred_at записи, породившей несоответствие.
	FoundAt time.Time `json:"found_at"`
	// Causes — записи-основания черновика или регистрации (event_id, отсортированы).
	Causes []string `json:"causes,omitempty"`
	// From — модуль, выразивший намерение черновика.
	From string `json:"from,omitempty"`

	Draft DraftedData `json:"draft"`
	// Commission — решение принимает комиссия (FR-151).
	Commission bool `json:"commission,omitempty"`
	// WindowEventID — окно нарушения специального процесса (FR-151).
	WindowEventID string `json:"window_event_id,omitempty"`

	Status        string `json:"status"`
	Investigation string `json:"investigation"`
	// Resolution — исход несоответствия, закрытого без решения по изделию.
	Resolution string `json:"resolution,omitempty"`
	// Severity, DefectTypeCode, RequirementRef — по подтверждению контролёра
	// (до него — из сигнала).
	Severity       string `json:"severity"`
	DefectTypeCode string `json:"defect_type_code,omitempty"`
	RequirementRef string `json:"requirement_ref,omitempty"`

	// Решение по несоответствию (FR-53).
	Disposition     string `json:"disposition"`
	ScrapKind       string `json:"scrap_kind,omitempty"`
	ConcessionID    string `json:"concession_id,omitempty"`
	DocumentID      string `json:"document_id,omitempty"`
	ApprovalsStatus string `json:"approvals_status,omitempty"`
	// Executed — решение исполняется: маршрут подписей закрыт (или демо-заглушка);
	// до этого решение режима 4 не исполняется (FR-50).
	Executed bool `json:"executed,omitempty"`

	// SignalID — сигнал quality, по которому построен черновик; AutomationMode —
	// режим правила карты реакций (FR-50).
	SignalID       string `json:"signal_id,omitempty"`
	AutomationMode int    `json:"automation_mode,omitempty"`

	// Решения людей по несоответствию (event_id записей по порядку).
	ConfirmedEventID   string `json:"confirmed_event_id,omitempty"`
	DispositionEventID string `json:"disposition_event_id,omitempty"`
	VerifiedEventID    string `json:"verified_event_id,omitempty"`
	ClosedEventID      string `json:"closed_event_id,omitempty"`
}

// Open — несоответствие ждёт решения по изделию.
func (n NC) Open() bool { return n.Status != StatusClosed }

// ContainmentSource — действующее основание сдерживания изделия: правило или
// человек, запись и уровень. Снимает только человек (AD-27).
type ContainmentSource struct {
	// Key — event_id записи-основания (для правила из намерения — ключ).
	Key      string    `json:"key"`
	Level    string    `json:"level"`
	By       string    `json:"by"`
	Reason   string    `json:"reason,omitempty"`
	At       time.Time `json:"at"`
	Released bool      `json:"released,omitempty"`
	// Rule — id правила (для правила) и несоответствие-основание.
	Rule string `json:"rule,omitempty"`
	NCID string `json:"nc_id,omitempty"`
	// Basis — уровень, который сейчас поддерживает основание правила: ниже
	// Level — основание изменилось (снятие только человеком, AD-27, AD-3);
	// BasisEventID — запись, давшая последнее основание.
	Basis        string `json:"basis,omitempty"`
	BasisEventID string `json:"basis_event_id,omitempty"`
	// Causes — записи-основания намерения (для правила из намерения).
	Causes []string `json:"causes,omitempty"`
	// SignalID — сигнал quality, на котором основано сдерживание правилом.
	SignalID string `json:"signal_id,omitempty"`
	// ReleasedBy — решение человека, снявшее сдерживание.
	ReleasedBy string `json:"released_by,omitempty"`
}

// Isolation — изоляция изделия (FR-55): «изоляция» — положение, ждущее
// решения; физическое перемещение в изолятор подтверждается отдельно.
type Isolation struct {
	EventID            string     `json:"event_id"`
	At                 time.Time  `json:"at"`
	DecisionDueAt      *time.Time `json:"decision_due_at,omitempty"`
	IsolatorLocationID string     `json:"isolator_location_id,omitempty"`
	// PhysicallyMoved — перемещение в изолятор подтверждено фактом приёмки
	// (operation.movement.received, destination_kind = isolator).
	PhysicallyMoved bool   `json:"physically_moved"`
	MovedEventID    string `json:"moved_event_id,omitempty"`
	Released        bool   `json:"released,omitempty"`
}

// Presentation — предъявление изделия на закрывающей точке и решение по нему.
type Presentation struct {
	EventID        string    `json:"event_id"`
	StepKey        string    `json:"step_key"`
	ClosingPoint   string    `json:"closing_point,omitempty"`
	PresentationNo int       `json:"presentation_no"`
	At             time.Time `json:"at"`
	// ResolvedEventID, Resolution — решение контролёра; пусто — ждёт решения.
	ResolvedEventID string `json:"resolved_event_id,omitempty"`
	Resolution      string `json:"resolution,omitempty"`
}

// DecisionRef — решение человека в изделии (для карточки, FR-51).
type DecisionRef struct {
	EventID string    `json:"event_id"`
	Seq     int64     `json:"seq"`
	Type    string    `json:"type"`
	Actor   string    `json:"actor,omitempty"`
	At      time.Time `json:"at"`
	NCID    string    `json:"nc_id,omitempty"`
	Summary string    `json:"summary"`
}

// Rejection — отклонённые контролёром сигналы (FR-52): исходный сигнал не
// меняется, отклонение — отдельная запись.
type Rejection struct {
	EventID   string   `json:"event_id"`
	SignalIDs []string `json:"signal_ids"`
	Reason    string   `json:"reason"`
}

// Recheck — назначенная доп. проверка.
type Recheck struct {
	EventID string `json:"event_id"`
	Method  string `json:"method"`
	DueAt   string `json:"due_at,omitempty"`
	Done    bool   `json:"done,omitempty"`
}

// Effect — что решение текущего шага свёртки поручает ранним модулям через
// их функции-намерения (AD-30, AD-40). Сбрасывается в начале каждого Reduce:
// намерение выражается один раз — на шаге своей записи.
type Effect struct {
	// Kind — set_quality | isolate.
	Kind  string `json:"kind"`
	Value string `json:"value,omitempty"`
	// StepKey — для advance_presentation.
	StepKey string `json:"step_key,omitempty"`
	Cause   string `json:"cause"`
	// CauseAt — occurred_at записи-причины.
	CauseAt time.Time `json:"cause_at"`
}

// State — состояние модуля nonconformity в свёртке одного изделия (AD-5):
// несоответствия, оси «решение по изделию» и «сдерживание» (AD-30),
// изоляция, предъявления и участники изготовления (разделение обязанностей,
// FR-56). Экспортируемые поля сериализуются в хеш состояния (Д-22).
type State struct {
	ItemID string `json:"item_id,omitempty"`
	NCs    []NC   `json:"ncs,omitempty"`

	Rejections    []Rejection         `json:"rejections,omitempty"`
	Containment   []ContainmentSource `json:"containment,omitempty"`
	Isolation     *Isolation          `json:"isolation,omitempty"`
	Rechecks      []Recheck           `json:"rechecks,omitempty"`
	Presentations []Presentation      `json:"presentations,omitempty"`
	Decisions     []DecisionRef       `json:"decisions,omitempty"`

	// Participants — исполнители операций изделия (operation.run.started),
	// отсортированы: они не принимают изделие на точке предъявления (FR-56).
	Participants []string `json:"participants,omitempty"`
	// Processed — изделие уже обрабатывалось (было выполнение операции):
	// «вернуть поставщику» недопустимо (FR-53).
	Processed bool `json:"processed,omitempty"`
	// Inspections — event_id результатов контроля изделия по порядку.
	Inspections []string `json:"inspections,omitempty"`
	// InterventionOpen — открыта запись вмешательства (FR-21).
	InterventionOpen bool `json:"intervention_open,omitempty"`
	// ReworkWaivers — разрешения сверх лимита доработок (FR-18): зона → доп. число.
	ReworkWaivers map[string]int `json:"rework_waivers,omitempty"`

	// Effects — намерения к ранним модулям на текущем шаге (см. Effect).
	Effects []Effect `json:"effects,omitempty"`
	// At — occurred_at записи текущего шага: время черновиков и сдерживания
	// из намерений, применённых в конце шага (AD-37).
	At time.Time `json:"at"`
}

// levelRank — порядок уровней сдерживания (FR-49): наблюдать < доп.
// проверка < блок изделия < блок партии.
func levelRank(l string) int {
	switch statuses.Containment(l) {
	case statuses.ContainmentObserve:
		return 1
	case statuses.ContainmentAdditionalCheck:
		return 2
	case statuses.ContainmentItemHold:
		return 3
	case statuses.ContainmentLotHold:
		return 4
	}
	return 0
}

// ContainmentLevel — ось «сдерживание» (AD-30): наибольший уровень среди
// действующих оснований; «none» — сдерживания нет.
func (s State) ContainmentLevel() string {
	best := string(statuses.ContainmentNone)
	for _, c := range s.Containment {
		if !c.Released && levelRank(c.Level) > levelRank(best) {
			best = c.Level
		}
	}
	return best
}

// Blocked — изделие заблокировано: блок изделия или партии (FR-49);
// «заблокировано» ≠ «признано дефектным» (NFR-UI-4).
func (s State) Blocked() bool {
	return levelRank(s.ContainmentLevel()) >= levelRank(string(statuses.ContainmentItemHold))
}

// Disposition — ось «решение по изделию» (AD-30): решение последнего
// несоответствия с исполняемым решением; «none» — решения нет.
func (s State) Disposition() string {
	d := "none"
	for _, n := range s.NCs {
		if n.Disposition != "" && n.Executed {
			d = n.Disposition
		}
	}
	return d
}

// NC — несоответствие по id.
func (s State) NC(id string) (NC, bool) {
	i := slices.IndexFunc(s.NCs, func(n NC) bool { return n.ID == id })
	if i < 0 {
		return NC{}, false
	}
	return s.NCs[i], true
}

func (s *State) nc(id string) *NC {
	i := slices.IndexFunc(s.NCs, func(n NC) bool { return n.ID == id })
	if i < 0 {
		return nil
	}
	return &s.NCs[i]
}

// SignalRejected — сигнал отклонён контролёром.
func (s State) SignalRejected(id string) bool {
	for _, r := range s.Rejections {
		if slices.Contains(r.SignalIDs, id) {
			return true
		}
	}
	return false
}

// PendingPresentation — последнее предъявление без решения (nil — нет).
func (s State) PendingPresentation() *Presentation {
	for i := len(s.Presentations) - 1; i >= 0; i-- {
		if s.Presentations[i].ResolvedEventID == "" {
			p := s.Presentations[i]
			return &p
		}
	}
	return nil
}

// PhysicallyNotMoved — «изолировано в системе, физически не перемещено» (FR-55).
func (s State) PhysicallyNotMoved() bool {
	return s.Isolation != nil && !s.Isolation.Released && !s.Isolation.PhysicallyMoved
}

// Isolated — изделие в изоляции (решение изолировать действует).
func (s State) Isolated() bool { return s.Isolation != nil && !s.Isolation.Released }

// Participant — сотрудник участвовал в изготовлении изделия (FR-56).
func (s State) Participant(person string) bool {
	_, ok := slices.BinarySearch(s.Participants, person)
	return ok
}

// DefaultClosingPoints — закрывающие точки шагов предъявления фланца
// (normative/process/flange-process.bpmn, ant:properties/@closingPoint).
// Действует, пока нормативный слой (эпик 17) не передаёт их в Env.
var DefaultClosingPoints = map[string]string{
	"incoming.zt1_lot_acceptance":    "ZT-1",
	"machining.zt2_acceptance":       "ZT-2",
	"welding.zt3_acceptance":         "ZT-3",
	"assembly.zt4_zone_presentation": "ZT-4.1",
	"assembly.zt4_acceptance":        "ZT-4.2",
	"testing.zt5_protocol":           "ZT-5",
	"final.zt6_acceptance":           "ZT-6",
	"nc.zt_r_disposition":            "ZT-R",
}

// ClosingPoint — закрывающая точка шага (пусто — шаг не точка предъявления
// или неизвестен).
func (e Env) ClosingPoint(stepKey string) string {
	m := e.ClosingPoints
	if len(m) == 0 {
		m = DefaultClosingPoints
	}
	return m[stepKey]
}
