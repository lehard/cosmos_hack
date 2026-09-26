package process

import (
	"slices"
	"strings"
	"time"

	"ant/internal/contracts/bpmnext"
)

// Модель описания процесса BPMN 2.0 с расширением urn:ant:bpmn-ext:1
// (AD-17, FR-10, FR-11, FR-12). Строится разбором XML версии (Parse) и
// дальше только читается: исполнитель, гарды, карточки узлов и карта
// работают с одной и той же моделью.

// Типы узлов поддерживаемого подмножества (contracts/bpmn-ext/rules.yaml →
// supported_elements.flow_nodes).
const (
	NodeStartEvent         = "startEvent"
	NodeEndEvent           = "endEvent"
	NodeCatchEvent         = "intermediateCatchEvent"
	NodeThrowEvent         = "intermediateThrowEvent"
	NodeBoundaryEvent      = "boundaryEvent"
	NodeTask               = "task"
	NodeServiceTask        = "serviceTask"
	NodeUserTask           = "userTask"
	NodeExclusiveGateway   = "exclusiveGateway"
	NodeParallelGateway    = "parallelGateway"
	NodeInclusiveGateway   = "inclusiveGateway"
	NodeCallActivity       = "callActivity"
	NodeSubProcess         = "subProcess"
	scopeKindProcess       = "process"
	scopeKindSubProcess    = "subProcess"
	eventMessage           = "message"
	eventTimer             = "timer"
	eventTerminate         = "terminate"
	stepKindOperation      = "operation"
	stepKindAutomated      = "automated_inspection"
	stepKindHuman          = "human_inspection"
	stepKindMovement       = "movement"
	stepKindStorage        = "storage"
	timerScopeUntilStart   = "until_started"
	timerScopeActivity     = "activity"
	reworkScopeItem        = "item"
	reworkScopeZone        = "zone"
	reworkScopeLoop        = "loop"
	preconditionBlock      = "block"
	preconditionRecordViol = "record_violation"
)

// Definition — описание процесса одной версии (bpmn:definitions целиком).
type Definition struct {
	ID                 string `json:"id"`
	ExpressionLanguage string `json:"expression_language,omitempty"`
	// Nodes — узлы всех процессов и подпроцессов по id элемента.
	Nodes map[string]*Node `json:"nodes"`
	// Flows — стрелки по id.
	Flows map[string]*Flow `json:"flows"`
	// Scopes — процессы и встроенные подпроцессы по id.
	Scopes map[string]*Scope `json:"scopes"`
	// Messages — bpmn:message: id → имя.
	Messages map[string]string `json:"messages,omitempty"`
	// Lanes — дорожки-цеха (FR-130) в порядке документа.
	Lanes []Lane `json:"lanes,omitempty"`
	// Main — исполняемый процесс, по которому запускается изделие: процесс,
	// который не вызывается callActivity (первый такой в документе).
	Main string `json:"main"`
	// Order — id узлов в порядке документа (читаемая версия, FR-24).
	Order []string `json:"order"`
	// Steps — step_key → id узла (AD-17: счётчики и карточки — по step_key).
	Steps map[string]string `json:"steps"`
}

// Scope — процесс или встроенный подпроцесс: область видимости узлов и
// стартовых событий.
type Scope struct {
	ID         string   `json:"id"`
	Name       string   `json:"name,omitempty"`
	Kind       string   `json:"kind"`
	Parent     string   `json:"parent,omitempty"`
	Executable bool     `json:"executable,omitempty"`
	Nodes      []string `json:"nodes"`
	Starts     []string `json:"starts"`
}

// Lane — дорожка BPMN = цех со складом (FR-130).
type Lane struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Workshop  string   `json:"workshop,omitempty"`
	Warehouse string   `json:"warehouse,omitempty"`
	Nodes     []string `json:"nodes"`
}

// Node — узел процесса с нашими свойствами (FR-12).
type Node struct {
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	Type          string `json:"type"`
	Scope         string `json:"scope"`
	Documentation string `json:"documentation,omitempty"`
	// Event — определение события: "" | message | timer | terminate.
	Event      string `json:"event,omitempty"`
	MessageRef string `json:"message_ref,omitempty"`
	// Timer — timeDuration как записан (ISO 8601); TimerDur — разобранная длительность.
	Timer    string        `json:"timer,omitempty"`
	TimerDur time.Duration `json:"timer_ns,omitempty"`
	// AttachedTo, CancelActivity — граничное событие (Д-8).
	AttachedTo     string `json:"attached_to,omitempty"`
	CancelActivity bool   `json:"cancel_activity,omitempty"`
	// CalledElement — вызываемый процесс callActivity.
	CalledElement string `json:"called_element,omitempty"`
	// Default — стрелка по умолчанию шлюза.
	Default string `json:"default,omitempty"`
	// In, Out — входящие и исходящие стрелки в порядке документа.
	In  []string `json:"in,omitempty"`
	Out []string `json:"out,omitempty"`
	// Boundaries — граничные события на узле.
	Boundaries []string `json:"boundaries,omitempty"`
	// Lane — id дорожки (цеха).
	Lane string `json:"lane,omitempty"`

	Props         bpmnext.Properties         `json:"props"`
	Inspection    *bpmnext.Inspection        `json:"inspection,omitempty"`
	Requirements  []bpmnext.Requirement      `json:"requirements,omitempty"`
	Zones         []string                   `json:"zones,omitempty"`
	Presentation  *bpmnext.PresentationPoint `json:"presentation,omitempty"`
	Preconditions []bpmnext.Precondition     `json:"preconditions,omitempty"`
	Documents     []string                   `json:"documents,omitempty"`
	ReactionMap   string                     `json:"reaction_map,omitempty"`
	Norm          *bpmnext.Norm              `json:"norm,omitempty"`
	NormRefs      []bpmnext.NormRef          `json:"norm_refs,omitempty"`
}

// Flow — стрелка процесса; Cond — разобранное условие языка urn:ant:expr:1 (Д-7).
type Flow struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Scope    string `json:"scope"`
	CondText string `json:"condition,omitempty"`
	Cond     *Expr  `json:"-"`
}

// StepKey — стабильный ключ шага узла (AD-17).
func (n *Node) StepKey() string { return n.Props.StepKey }

// IsTask — задача (работа, контроль, перемещение, хранение).
func (n *Node) IsTask() bool {
	return n.Type == NodeTask || n.Type == NodeServiceTask || n.Type == NodeUserTask
}

// IsGateway — шлюз.
func (n *Node) IsGateway() bool {
	return n.Type == NodeExclusiveGateway || n.Type == NodeParallelGateway || n.Type == NodeInclusiveGateway
}

// IsActivity — узел, на котором токен ждёт факта или решения (задача,
// вызов подпроцесса, встроенный подпроцесс, промежуточное событие-приём).
func (n *Node) IsActivity() bool {
	return n.IsTask() || n.Type == NodeCallActivity || n.Type == NodeSubProcess || n.Type == NodeCatchEvent
}

// IsPresentationPoint — точка предъявления («шлагбаум», FR-19): задача с
// ant:presentationPoint. Токен проходит её только по подписанному решению.
func (n *Node) IsPresentationPoint() bool { return n.Presentation != nil }

// IsHumanControl — контроль человеком: точка предъявления или задача вида
// human_inspection (FR-13 «закрывающий путь без контроля человеком»).
func (n *Node) IsHumanControl() bool {
	return n.IsPresentationPoint() || n.Props.StepKind == stepKindHuman
}

// Special — шаг — специальный процесс (FR-151).
func (n *Node) Special() bool { return n.Props.SpecialProcess != nil && *n.Props.SpecialProcess }

// ReworkLimit — лимит доработок шага (FR-18); ok=false — не задан.
func (n *Node) ReworkLimit() (int, bool) {
	if n.Props.ReworkLimit == nil {
		return 0, false
	}
	return int(*n.Props.ReworkLimit), true
}

// ClosesZones — зоны, доступ к которым закрывает шаг (FR-20).
func (n *Node) ClosesZones() []string { return strings.Fields(n.Props.ClosesZoneAccess) }

// Node — узел по id (nil — нет).
func (d *Definition) Node(id string) *Node {
	if d == nil {
		return nil
	}
	return d.Nodes[id]
}

// ByStep — узел по step_key (nil — нет).
func (d *Definition) ByStep(stepKey string) *Node {
	if d == nil {
		return nil
	}
	return d.Nodes[d.Steps[stepKey]]
}

// SpecialSteps — step_key шагов-специальных процессов (FR-151): признак
// ant:properties/@specialProcess передаётся модулю machinelogs через Bundle.
func (d *Definition) SpecialSteps() []string {
	if d == nil {
		return nil
	}
	var out []string
	for _, id := range d.Order {
		if n := d.Nodes[id]; n.Special() && n.StepKey() != "" {
			out = append(out, n.StepKey())
		}
	}
	slices.Sort(out)
	return out
}

// Workshop — цех (дорожка) узла: свойство узла или его дорожки.
func (d *Definition) Workshop(n *Node) string {
	if n.Props.Workshop != "" {
		return n.Props.Workshop
	}
	for _, l := range d.Lanes {
		if l.ID == n.Lane {
			return l.Workshop
		}
	}
	return ""
}

// LaneName — имя дорожки узла.
func (d *Definition) LaneName(n *Node) string {
	for _, l := range d.Lanes {
		if l.ID == n.Lane {
			return l.Name
		}
	}
	return ""
}

// Next — узлы, куда ведут исходящие стрелки n.
func (d *Definition) Next(n *Node) []string {
	out := make([]string, 0, len(n.Out))
	for _, f := range n.Out {
		out = append(out, d.Flows[f].Target)
	}
	return out
}
