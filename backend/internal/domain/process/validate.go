package process

import (
	"fmt"
	"regexp"
	"strings"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
)

// Проверка описания при загрузке (FR-13, contracts/bpmn-ext/rules.yaml):
// схема расширения и допустимые значения, неподдерживаемые элементы, неявные
// слияния (Д-6), язык условий (Д-7), недостижимые узлы, закрывающий путь без
// контроля человеком, точка предъявления без роли. Каждое нарушение — код и
// id элемента.

// LoadOptions — справочные множества для перекрёстных ссылок (пусто — не
// проверяется): коды классификатора дефектов, зоны номенклатуры, полномочия и
// роли политики, нормативные опоры.
type LoadOptions struct {
	Vars        map[string]VarSpec
	DefectCodes []string
	Zones       []string
	Authorities []string
	Roles       []string
	// NormAnchors — стандарт → допустимые пункты (norm-anchors.yaml).
	NormAnchors map[string][]string
}

var stepKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`)

// Перечисления rules.yaml → enums.
var (
	enumStepKind     = []string{stepKindOperation, stepKindAutomated, stepKindHuman, stepKindMovement, stepKindStorage}
	enumInspMethod   = []string{"camera", "cmm", "radiography", "ultrasonic", "penetrant", "leak_test", "torque", "visual_human", "supplier_documents", "laboratory", "other"}
	enumInspPhase    = []string{"incoming", "before_operation", "after_operation", "before_zone_closure", "assembly", "test", "final", "other"}
	enumPrecondKind  = []string{"qualification", "equipment_verification", "document_revision", "material_expiry", "time_window", "zone_check", "item_blocked", "open_intervention", "lot_accepted", "rework_limit", "status_expired"}
	enumPrecondMode  = []string{preconditionBlock, preconditionRecordViol}
	enumReworkScope  = []string{reworkScopeItem, reworkScopeZone, reworkScopeLoop}
	enumErpAction    = []string{"accept_into_work", "warehouse_transfer", "scrap_transfer_rework", "scrap_transfer_writeoff", "scrap_transfer_reprocess", "return_to_supplier", "release"}
	enumTimerScope   = []string{timerScopeUntilStart, timerScopeActivity}
	enumOutcome      = []string{"rework_or_repair", "use_as_is", "scrapped", "returned"}
	enumClosingPoint = []string{"ZT-1", "ZT-2", "ZT-3", "ZT-4.1", "ZT-4.2", "ZT-5", "ZT-6", "ZT-R", "ZT-V"}
)

// Load — загрузка версии: разбор и проверка (FR-10, FR-13). Модель
// возвращается и при нарушениях (для показа); исполнять её можно, только если
// Errors(нарушения) пуст.
func Load(xmlBytes []byte, o LoadOptions) (*Definition, []Violation) {
	d, vs := Parse(xmlBytes)
	if d == nil {
		return nil, vs
	}
	return d, append(vs, Validate(d, o)...)
}

// Errors — нарушения-отказы (без предупреждений).
func Errors(vs []Violation) []Violation {
	var out []Violation
	for _, v := range vs {
		if !v.Warning {
			out = append(out, v)
		}
	}
	return out
}

type checker struct {
	d  *Definition
	o  LoadOptions
	vs []Violation
}

func (c *checker) add(code errcodes.Code, el, format string, args ...any) {
	c.vs = append(c.vs, Violation{Code: code, Element: el, Detail: fmt.Sprintf(format, args...)})
}

func (c *checker) enum(el, what, v string, allowed []string) {
	if v != "" && !contains(allowed, v) {
		c.add(errcodes.ProcessSchemaViolation, el, "%s = %q: допустимо %s", what, v, strings.Join(allowed, ", "))
	}
}

func (c *checker) known(el, what, v string, set []string) {
	if v != "" && len(set) > 0 && !contains(set, v) {
		c.add(errcodes.ProcessSchemaViolation, el, "%s %q нет в справочнике", what, v)
	}
}

// Validate — семантические проверки модели (FR-13).
func Validate(d *Definition, o LoadOptions) []Violation {
	if o.Vars == nil {
		o.Vars = Variables
	}
	c := &checker{d: d, o: o}
	usesOutcome := false
	for _, fid := range sortedKeys(d.Flows) {
		if strings.Contains(d.Flows[fid].CondText, VarNCOutcome) {
			usesOutcome = true
		}
	}
	for _, id := range d.Order {
		c.node(d.Nodes[id], usesOutcome)
	}
	c.flows()
	c.reachability()
	c.closingPaths()
	return c.vs
}

func (c *checker) node(n *Node, usesOutcome bool) {
	d, p := c.d, n.Props
	switch {
	case p.StepKey == "":
		c.add(errcodes.ProcessStepKeyMissing, n.ID, "нет ant:properties/@stepKey")
	case !stepKeyPattern.MatchString(p.StepKey):
		c.add(errcodes.ProcessStepKeyMissing, n.ID, "stepKey %q не по шаблону %s", p.StepKey, stepKeyPattern)
	}
	if n.IsTask() {
		if p.StepKind == "" {
			c.add(errcodes.ProcessSchemaViolation, n.ID, "у задачи нет stepKind")
		}
	}
	c.enum(n.ID, "stepKind", p.StepKind, enumStepKind)
	c.enum(n.ID, "reworkLimitScope", p.ReworkLimitScope, enumReworkScope)
	c.enum(n.ID, "erpAction", p.ErpAction, enumErpAction)
	c.enum(n.ID, "timerScope", p.TimerScope, enumTimerScope)
	c.enum(n.ID, "outcome", p.Outcome, enumOutcome)
	c.enum(n.ID, "closingPoint", p.ClosingPoint, enumClosingPoint)
	if p.ReworkLimit != nil && (*p.ReworkLimit < 0 || p.ReworkLimitScope == "") {
		c.add(errcodes.ProcessSchemaViolation, n.ID, "reworkLimit ≥ 0 и reworkLimitScope обязательны вместе (FR-18)")
	}
	if n.Inspection != nil {
		c.enum(n.ID, "inspection.method", n.Inspection.Method, enumInspMethod)
		c.enum(n.ID, "inspection.phase", n.Inspection.Phase, enumInspPhase)
		for _, code := range strings.Fields(n.Inspection.Coverage) {
			c.known(n.ID, "код дефекта", code, c.o.DefectCodes)
		}
		if len(n.Requirements) == 0 {
			c.vs = append(c.vs, Violation{Code: errcodes.ProcessInspectionWithoutRequirement, Element: n.ID, Warning: true,
				Detail: "контроль без ссылки на требование КД"})
		}
	}
	if n.Type == NodeServiceTask && p.StepKind == stepKindAutomated && n.Inspection == nil {
		c.add(errcodes.ProcessSchemaViolation, n.ID, "автоматизированный контроль без ant:inspection")
	}
	for _, z := range n.Zones {
		c.known(n.ID, "зона", z, c.o.Zones)
	}
	for _, z := range n.ClosesZones() {
		c.known(n.ID, "зона", z, c.o.Zones)
	}
	for _, pc := range n.Preconditions {
		c.enum(n.ID, "precondition.kind", pc.Kind, enumPrecondKind)
		c.enum(n.ID, "precondition.mode", pc.Mode, enumPrecondMode)
		if pc.Kind == "" || pc.Mode == "" {
			c.add(errcodes.ProcessSchemaViolation, n.ID, "у ant:precondition обязательны kind и mode")
		}
		if (pc.Kind == "time_window" || pc.Kind == "status_expired") && pc.Ref != "" {
			if _, _, err := ParseWindowRef(pc.Ref); err != nil {
				c.add(errcodes.ProcessSchemaViolation, n.ID, "precondition %s ref %q: %v", pc.Kind, pc.Ref, err)
			}
		}
	}
	// FR-13, FR-19: точка предъявления без роли или полномочия.
	if p.ClosingPoint != "" && n.Presentation == nil && n.Type == NodeUserTask {
		c.add(errcodes.ProcessPresentationPointWithoutRole, n.ID, "закрывающая точка %s без ant:presentationPoint", p.ClosingPoint)
	}
	if pp := n.Presentation; pp != nil {
		if pp.Authority == "" || pp.Role == "" {
			c.add(errcodes.ProcessPresentationPointWithoutRole, n.ID, "у ant:presentationPoint нужны authority и role")
		}
		c.known(n.ID, "полномочие", pp.Authority, c.o.Authorities)
		c.known(n.ID, "полномочие", pp.RepeatAuthority, c.o.Authorities)
		c.known(n.ID, "роль", pp.Role, c.o.Roles)
		if !n.IsTask() {
			c.add(errcodes.ProcessSchemaViolation, n.ID, "точка предъявления — только на задаче")
		}
	}
	c.known(n.ID, "роль заверителя", p.PaperAttester, c.o.Roles)
	for _, nr := range n.NormRefs {
		if len(c.o.NormAnchors) == 0 {
			break
		}
		if !anchorAllowed(c.o.NormAnchors, nr.Standard, nr.Clause) {
			c.add(errcodes.ProcessSchemaViolation, n.ID, "нормативная опора %s п. %s не из norm-anchors.yaml", nr.Standard, nr.Clause)
		}
	}
	// События-сообщения (Д-5) и выход в 1С.
	if n.Event == eventMessage && (n.Type == NodeStartEvent || n.Type == NodeCatchEvent) {
		if p.TriggerEventType == "" {
			c.add(errcodes.ProcessSchemaViolation, n.ID, "событие-сообщение без triggerEventType (Д-5)")
		} else if _, ok := catalog.Lookup(catalog.Type(p.TriggerEventType)); !ok {
			c.add(errcodes.ProcessSchemaViolation, n.ID, "triggerEventType %q нет в каталоге событий", p.TriggerEventType)
		}
	}
	if n.Event == eventMessage && n.MessageRef != "" {
		if _, ok := d.Messages[n.MessageRef]; !ok {
			c.add(errcodes.ProcessSchemaViolation, n.ID, "messageRef %q не объявлен", n.MessageRef)
		}
	}
	if n.Type == NodeThrowEvent && n.Event == eventMessage && p.ErpAction == "" {
		c.add(errcodes.ProcessSchemaViolation, n.ID, "событие-сообщение «в 1С» без erpAction")
	}
	if n.Type == NodeBoundaryEvent {
		if n.Timer == "" || p.TimerScope == "" {
			c.add(errcodes.ProcessSchemaViolation, n.ID, "граничный таймер требует timeDuration и timerScope (Д-8)")
		}
		if host := d.Nodes[n.AttachedTo]; host != nil && !host.IsActivity() {
			c.add(errcodes.ProcessSchemaViolation, n.ID, "граничное событие — только на задаче или подпроцессе")
		}
		if len(n.Out) != 1 {
			c.add(errcodes.ProcessSchemaViolation, n.ID, "у граничного события ровно одна исходящая стрелка")
		}
	}
	if n.Type == NodeCatchEvent && n.Event == eventTimer && n.Timer == "" {
		c.add(errcodes.ProcessSchemaViolation, n.ID, "таймер без timeDuration")
	}
	if n.Type == NodeCallActivity {
		sc := d.Scopes[n.CalledElement]
		switch {
		case n.CalledElement == "" || sc == nil || sc.Kind != scopeKindProcess:
			c.add(errcodes.ProcessSchemaViolation, n.ID, "calledElement %q — не процесс этого файла", n.CalledElement)
		case n.CalledElement == n.Scope:
			c.add(errcodes.ProcessSchemaViolation, n.ID, "рекурсивный вызов процесса")
		}
	}
	if usesOutcome && n.Type == NodeEndEvent && n.Event == "" && d.Scopes[n.Scope].ID != d.Main && d.Scopes[n.Scope].Kind == scopeKindProcess && p.Outcome == "" {
		c.add(errcodes.ProcessSchemaViolation, n.ID, "конечное событие вызываемого подпроцесса без outcome: условия проверяют nc.outcome")
	}
	// Д-6: неявные слияния запрещены.
	if !n.IsGateway() && len(n.In) > 1 {
		c.add(errcodes.ProcessImplicitMerge, n.ID, "%d входящих стрелки без шлюза — поставьте явный шлюз слияния", len(n.In))
	}
	switch {
	case n.Type == NodeStartEvent && len(n.In) > 0:
		c.add(errcodes.ProcessSchemaViolation, n.ID, "у стартового события не бывает входящих стрелок")
	case n.Type == NodeEndEvent && len(n.Out) > 0:
		c.add(errcodes.ProcessSchemaViolation, n.ID, "у конечного события не бывает исходящих стрелок")
	case n.Type != NodeEndEvent && n.Type != NodeBoundaryEvent && len(n.Out) == 0:
		c.add(errcodes.ProcessSchemaViolation, n.ID, "нет исходящей стрелки — изделие застрянет")
	case n.Type == NodeBoundaryEvent && len(n.In) > 0:
		c.add(errcodes.ProcessSchemaViolation, n.ID, "у граничного события не бывает входящих стрелок")
	}
	if !n.IsGateway() && n.Type != NodeBoundaryEvent && len(n.Out) > 1 {
		c.add(errcodes.ProcessSchemaViolation, n.ID, "несколько исходящих стрелок без шлюза — поставьте явный шлюз развилки")
	}
}

func anchorAllowed(anchors map[string][]string, std, clause string) bool {
	allowed, ok := anchors[std]
	if !ok {
		return false
	}
	for _, part := range strings.Split(clause, ",") {
		part = strings.TrimSpace(part)
		part = strings.TrimSuffix(part, " примечание")
		part = strings.TrimSuffix(part, ", примечание")
		if part == "примечание" {
			continue
		}
		if !contains(allowed, part) {
			return false
		}
	}
	return true
}

// flows — условия на стрелках (Д-7) и развилки шлюзов.
func (c *checker) flows() {
	d := c.d
	for _, fid := range sortedKeys(d.Flows) {
		f := d.Flows[fid]
		src := d.Nodes[f.Source]
		if f.CondText == "" {
			continue
		}
		if src != nil && src.Type != NodeExclusiveGateway && src.Type != NodeInclusiveGateway {
			c.add(errcodes.ProcessConditionInvalid, f.ID, "условие — только на стрелке исключающего или включающего шлюза")
			continue
		}
		if d.ExpressionLanguage != ExprLanguage {
			c.add(errcodes.ProcessConditionInvalid, f.ID, "язык условий %q — нужен %s", d.ExpressionLanguage, ExprLanguage)
			continue
		}
		e, err := ParseExpr(f.CondText, c.o.Vars)
		if err != nil {
			c.add(errcodes.ProcessConditionInvalid, f.ID, "%v", err)
			continue
		}
		f.Cond = e
	}
	for _, id := range d.Order {
		n := d.Nodes[id]
		if (n.Type != NodeExclusiveGateway && n.Type != NodeInclusiveGateway) || len(n.Out) < 2 {
			continue
		}
		for _, fid := range n.Out {
			if fid != n.Default && d.Flows[fid].CondText == "" {
				c.add(errcodes.ProcessConditionInvalid, fid, "ветка развилки %s без условия и не по умолчанию", n.ID)
			}
		}
	}
}

// reachability — недостижимые узлы (FR-13): обход от стартовых событий каждой
// области; граничные события достижимы вместе с узлом, к которому прикреплены.
func (c *checker) reachability() {
	d := c.d
	for _, sid := range sortedKeys(d.Scopes) {
		s := d.Scopes[sid]
		if len(s.Starts) == 0 {
			c.add(errcodes.ProcessSchemaViolation, s.ID, "у процесса нет стартового события")
			continue
		}
		if len(s.Starts) > 1 && s.Kind == scopeKindSubProcess {
			c.add(errcodes.ProcessSchemaViolation, s.ID, "у встроенного подпроцесса одно стартовое событие")
		}
		seen := map[string]bool{}
		queue := append([]string(nil), s.Starts...)
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			if seen[id] {
				continue
			}
			seen[id] = true
			n := d.Nodes[id]
			queue = append(queue, d.Next(n)...)
			queue = append(queue, n.Boundaries...)
		}
		for _, id := range s.Nodes {
			if !seen[id] {
				c.add(errcodes.ProcessUnreachableNode, id, "к узлу нет пути от стартового события")
			}
		}
	}
}

// humanScope — в вызываемой области есть контроль человеком.
func (c *checker) humanScope(scope string) bool {
	for _, id := range c.d.Scopes[scope].Nodes {
		if c.d.Nodes[id].IsHumanControl() {
			return true
		}
	}
	return false
}

// closingPaths — FR-13, FR-44: к закрывающему событию (выход в 1С, выпуск
// изделия) нет пути, минующего контроль человеком.
func (c *checker) closingPaths() {
	d := c.d
	for _, sid := range sortedKeys(d.Scopes) {
		s := d.Scopes[sid]
		seen := map[string]bool{}
		queue := append([]string(nil), s.Starts...)
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			if seen[id] {
				continue
			}
			seen[id] = true
			n := d.Nodes[id]
			if closing(d, n) {
				c.add(errcodes.ProcessClosingPathWithoutHuman, n.ID, "к закрывающему событию есть путь без контроля человеком")
				continue
			}
			if n.IsHumanControl() {
				continue
			}
			if (n.Type == NodeCallActivity && c.humanScope(n.CalledElement)) || (n.Type == NodeSubProcess && c.humanScope(n.ID)) {
				continue
			}
			queue = append(queue, d.Next(n)...)
			queue = append(queue, n.Boundaries...)
		}
	}
}

// closing — закрывающее событие: сообщение «в 1С» с учётным действием или
// обычное конечное событие главного процесса (выпуск).
func closing(d *Definition, n *Node) bool {
	if n.Type == NodeThrowEvent && n.Props.ErpAction != "" {
		return true
	}
	return n.Type == NodeEndEvent && n.Event == "" && n.Scope == d.Main
}
