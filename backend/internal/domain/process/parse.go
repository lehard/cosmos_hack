package process

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/bpmnext"
	"ant/internal/contracts/errcodes"
)

// Разбор BPMN 2.0 XML версии (FR-10): дерево элементов с разрешёнными
// пространствами имён → модель Definition. Наши свойства разбираются в
// сгенерированные из дескриптора moddle структуры contracts/bpmnext (AD-20);
// неизвестные элементы и атрибуты расширения — нарушение схемы с id
// элемента (FR-13).

// NsBPMN — пространство имён модели BPMN 2.0.
const NsBPMN = "http://www.omg.org/spec/BPMN/20100524/MODEL"

// Violation — нарушение описания процесса при загрузке (FR-13): код из
// contracts/errors.yaml и id элемента BPMN; Warning — принято с пометкой
// (process.inspection_without_requirement).
type Violation struct {
	Code    errcodes.Code `json:"code"`
	Element string        `json:"element"`
	Detail  string        `json:"detail,omitempty"`
	Warning bool          `json:"warning,omitempty"`
}

func (v Violation) String() string {
	s := string(v.Code) + " [" + v.Element + "]"
	if v.Detail != "" {
		s += ": " + v.Detail
	}
	return s
}

// xel — элемент XML с разрешённым пространством имён.
type xel struct {
	name  xml.Name
	attrs []xml.Attr
	text  strings.Builder
	kids  []*xel
}

func (x *xel) attr(local string) string {
	for _, a := range x.attrs {
		if a.Name.Local == local && (a.Name.Space == "" || a.Name.Space == x.name.Space) {
			return a.Value
		}
	}
	return ""
}

func (x *xel) id() string { return x.attr("id") }

func parseTree(b []byte) (*xel, error) {
	d := xml.NewDecoder(bytes.NewReader(b))
	var root *xel
	var stack []*xel
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := t.(type) {
		case xml.StartElement:
			e := &xel{name: t.Name, attrs: append([]xml.Attr(nil), t.Attr...)}
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				p.kids = append(p.kids, e)
			} else if root == nil {
				root = e
			}
			stack = append(stack, e)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].text.Write(t)
			}
		}
	}
	if root == nil {
		return nil, fmt.Errorf("пустой документ")
	}
	return root, nil
}

// flowNodeTypes — поддерживаемые узлы (rules.yaml → supported_elements.flow_nodes).
var flowNodeTypes = map[string]bool{
	NodeStartEvent: true, NodeEndEvent: true, NodeCatchEvent: true, NodeThrowEvent: true, NodeBoundaryEvent: true,
	NodeTask: true, NodeServiceTask: true, NodeUserTask: true,
	NodeExclusiveGateway: true, NodeParallelGateway: true, NodeInclusiveGateway: true,
	NodeCallActivity: true, NodeSubProcess: true,
}

// eventDefsAllowed — допустимые определения событий по типу узла (Д-4, Д-5, Д-8).
var eventDefsAllowed = map[string][]string{
	NodeStartEvent:    {"", eventMessage},
	NodeEndEvent:      {"", eventTerminate},
	NodeCatchEvent:    {eventMessage, eventTimer},
	NodeThrowEvent:    {"", eventMessage},
	NodeBoundaryEvent: {eventTimer},
}

// otherElements — прочие поддерживаемые элементы модели BPMN (rules.yaml → supported_elements.other).
var otherElements = map[string]bool{
	"definitions": true, "collaboration": true, "participant": true, "process": true, "laneSet": true, "lane": true,
	"sequenceFlow": true, "message": true, "documentation": true, "extensionElements": true, "conditionExpression": true,
	"timeDuration": true, "incoming": true, "outgoing": true, "flowNodeRef": true, "textAnnotation": true, "association": true,
	"text": true,
}

type parser struct {
	def  *Definition
	viol []Violation
	// flowNodeScope — узел → область (для проверки стрелок).
	called map[string]bool
}

func (p *parser) fail(code errcodes.Code, el, format string, args ...any) {
	p.viol = append(p.viol, Violation{Code: code, Element: el, Detail: fmt.Sprintf(format, args...)})
}

// Parse разбирает BPMN XML версии в модель. Нарушения структуры и схемы
// расширения возвращаются списком с кодом и id элемента (FR-13); модель
// строится и при нарушениях — чтобы показать все сразу. Ошибка XML —
// единственное нарушение process.schema_violation без модели.
func Parse(b []byte) (*Definition, []Violation) {
	root, err := parseTree(b)
	if err != nil {
		return nil, []Violation{{Code: errcodes.ProcessSchemaViolation, Element: "definitions", Detail: "XML не разбирается: " + err.Error()}}
	}
	if root.name.Space != NsBPMN || root.name.Local != "definitions" {
		return nil, []Violation{{Code: errcodes.ProcessSchemaViolation, Element: root.name.Local, Detail: "корень — не bpmn:definitions"}}
	}
	p := &parser{def: &Definition{
		ID: root.id(), ExpressionLanguage: root.attr("expressionLanguage"),
		Nodes: map[string]*Node{}, Flows: map[string]*Flow{}, Scopes: map[string]*Scope{},
		Messages: map[string]string{}, Steps: map[string]string{},
	}, called: map[string]bool{}}
	var processes []*xel
	for _, k := range root.kids {
		if k.name.Space != NsBPMN {
			continue // BPMNDI и чужие расширения — раскладка, не исполняются
		}
		switch k.name.Local {
		case "message":
			p.def.Messages[k.id()] = k.attr("name")
		case "process":
			processes = append(processes, k)
		case "collaboration", "documentation", "extensionElements":
			p.checkKnown(k)
		default:
			p.fail(errcodes.ProcessUnsupportedElement, elemID(k), "элемент bpmn:%s не поддерживается", k.name.Local)
		}
	}
	for _, pr := range processes {
		p.scope(pr, scopeKindProcess, "")
	}
	p.link()
	for _, pr := range processes {
		if !p.called[pr.id()] && p.def.Main == "" && pr.attr("isExecutable") != "false" {
			p.def.Main = pr.id()
		}
	}
	if p.def.Main == "" {
		p.fail(errcodes.ProcessSchemaViolation, p.def.ID, "нет исполняемого процесса, который не вызывается callActivity")
	}
	return p.def, p.viol
}

func elemID(x *xel) string {
	if id := x.id(); id != "" {
		return id
	}
	return x.name.Local
}

// checkKnown — прочие элементы BPMN внутри поддерживаемых: только из перечня.
func (p *parser) checkKnown(x *xel) {
	for _, k := range x.kids {
		if k.name.Space != NsBPMN {
			continue
		}
		if !otherElements[k.name.Local] && !flowNodeTypes[k.name.Local] {
			p.fail(errcodes.ProcessUnsupportedElement, elemID(k), "элемент bpmn:%s не поддерживается", k.name.Local)
			continue
		}
		if k.name.Local != "extensionElements" {
			p.checkKnown(k)
		}
	}
}

// scope — процесс или встроенный подпроцесс.
func (p *parser) scope(x *xel, kind, parent string) {
	s := &Scope{ID: x.id(), Name: x.attr("name"), Kind: kind, Parent: parent, Executable: x.attr("isExecutable") == "true"}
	p.def.Scopes[s.ID] = s
	for _, k := range x.kids {
		if k.name.Space != NsBPMN {
			continue
		}
		switch {
		case k.name.Local == "sequenceFlow":
			f := &Flow{ID: k.id(), Name: k.attr("name"), Source: k.attr("sourceRef"), Target: k.attr("targetRef"), Scope: s.ID}
			for _, c := range k.kids {
				if c.name.Space == NsBPMN && c.name.Local == "conditionExpression" {
					f.CondText = strings.TrimSpace(c.text.String())
				}
			}
			p.def.Flows[f.ID] = f
		case k.name.Local == "laneSet":
			for _, l := range k.kids {
				if l.name.Space != NsBPMN || l.name.Local != "lane" {
					continue
				}
				lane := Lane{ID: l.id(), Name: l.attr("name")}
				for _, c := range l.kids {
					switch {
					case c.name.Space == NsBPMN && c.name.Local == "flowNodeRef":
						lane.Nodes = append(lane.Nodes, strings.TrimSpace(c.text.String()))
					case c.name.Space == NsBPMN && c.name.Local == "extensionElements":
						ext := p.extension(c, lane.ID)
						if len(ext.Properties) > 0 {
							lane.Workshop, lane.Warehouse = ext.Properties[0].Workshop, ext.Properties[0].Warehouse
						}
					}
				}
				p.def.Lanes = append(p.def.Lanes, lane)
			}
		case flowNodeTypes[k.name.Local]:
			p.node(k, s)
		case otherElements[k.name.Local]:
		default:
			p.fail(errcodes.ProcessUnsupportedElement, elemID(k), "элемент bpmn:%s не поддерживается", k.name.Local)
		}
	}
}

func (p *parser) node(x *xel, s *Scope) {
	n := &Node{ID: x.id(), Name: x.attr("name"), Type: x.name.Local, Scope: s.ID}
	if n.ID == "" {
		p.fail(errcodes.ProcessSchemaViolation, x.name.Local, "у узла нет id")
		return
	}
	if _, dup := p.def.Nodes[n.ID]; dup {
		p.fail(errcodes.ProcessSchemaViolation, n.ID, "повтор id узла")
		return
	}
	n.AttachedTo = x.attr("attachedToRef")
	n.CancelActivity = x.attr("cancelActivity") != "false"
	n.CalledElement = x.attr("calledElement")
	n.Default = x.attr("default")
	var defs []string
	for _, k := range x.kids {
		if k.name.Space != NsBPMN {
			continue
		}
		switch k.name.Local {
		case "documentation":
			n.Documentation = strings.TrimSpace(k.text.String())
		case "extensionElements":
			p.apply(n, p.extension(k, n.ID))
		case "incoming", "outgoing":
		case "messageEventDefinition":
			defs = append(defs, eventMessage)
			n.MessageRef = k.attr("messageRef")
		case "terminateEventDefinition":
			defs = append(defs, eventTerminate)
		case "timerEventDefinition":
			defs = append(defs, eventTimer)
			for _, c := range k.kids {
				switch {
				case c.name.Space == NsBPMN && c.name.Local == "timeDuration":
					n.Timer = strings.TrimSpace(c.text.String())
				case c.name.Space == NsBPMN:
					p.fail(errcodes.ProcessUnsupportedElement, n.ID, "таймер bpmn:%s не поддерживается — только timeDuration (Д-8)", c.name.Local)
				}
			}
		default:
			if strings.HasSuffix(k.name.Local, "EventDefinition") {
				p.fail(errcodes.ProcessUnsupportedElement, n.ID, "определение события bpmn:%s не поддерживается", k.name.Local)
				continue
			}
			if n.Type == NodeSubProcess && (flowNodeTypes[k.name.Local] || k.name.Local == "sequenceFlow" || k.name.Local == "laneSet") {
				continue
			}
			if !otherElements[k.name.Local] {
				p.fail(errcodes.ProcessUnsupportedElement, elemID(k), "элемент bpmn:%s не поддерживается", k.name.Local)
			}
		}
	}
	if len(defs) > 1 {
		p.fail(errcodes.ProcessUnsupportedElement, n.ID, "несколько определений события на одном узле не поддерживаются")
	}
	if len(defs) > 0 {
		n.Event = defs[0]
	}
	if allowed, ok := eventDefsAllowed[n.Type]; ok {
		if !contains(allowed, n.Event) {
			ev := n.Event
			if ev == "" {
				ev = "без определения"
			}
			p.fail(errcodes.ProcessUnsupportedElement, n.ID, "%s с событием «%s» не поддерживается", n.Type, ev)
		}
	} else if n.Event != "" {
		p.fail(errcodes.ProcessUnsupportedElement, n.ID, "определение события на %s не поддерживается", n.Type)
	}
	if n.Event == eventTimer {
		d, err := ParseISODuration(n.Timer)
		if err != nil {
			p.fail(errcodes.ProcessSchemaViolation, n.ID, "timeDuration %q: %v", n.Timer, err)
		}
		n.TimerDur = d
	}
	p.def.Nodes[n.ID] = n
	p.def.Order = append(p.def.Order, n.ID)
	s.Nodes = append(s.Nodes, n.ID)
	if n.Type == NodeCallActivity && n.CalledElement != "" {
		p.called[n.CalledElement] = true
	}
	if n.Type == NodeSubProcess {
		p.scope(x, scopeKindSubProcess, s.ID)
	}
}

// apply переносит элементы расширения в узел.
func (p *parser) apply(n *Node, ext bpmnext.ExtensionElements) {
	if len(ext.Properties) > 1 {
		p.fail(errcodes.ProcessSchemaViolation, n.ID, "ant:properties — ровно один на узел")
	}
	if len(ext.Properties) > 0 {
		n.Props = ext.Properties[0]
	}
	if len(ext.Inspection) > 1 {
		p.fail(errcodes.ProcessSchemaViolation, n.ID, "ant:inspection — не больше одного")
	}
	if len(ext.Inspection) > 0 {
		v := ext.Inspection[0]
		n.Inspection = &v
	}
	n.Requirements = ext.Requirement
	for _, z := range ext.ZoneRef {
		n.Zones = append(n.Zones, z.Zone)
	}
	if len(ext.PresentationPoint) > 1 {
		p.fail(errcodes.ProcessSchemaViolation, n.ID, "ant:presentationPoint — не больше одного")
	}
	if len(ext.PresentationPoint) > 0 {
		v := ext.PresentationPoint[0]
		n.Presentation = &v
	}
	n.Preconditions = ext.Precondition
	for _, d := range ext.Document {
		n.Documents = append(n.Documents, d.Template)
	}
	if len(ext.ReactionMap) > 0 {
		n.ReactionMap = ext.ReactionMap[0].Ref
	}
	if len(ext.Norm) > 0 {
		v := ext.Norm[0]
		n.Norm = &v
	}
	n.NormRefs = ext.NormRef
}

// extension разбирает bpmn:extensionElements: элементы пространства
// urn:ant:bpmn-ext:1 — в сгенерированные структуры (по тегам xml полей),
// неизвестный элемент или атрибут расширения — нарушение схемы. Элементы
// других пространств (например, раскладки редакторов) пропускаются.
func (p *parser) extension(x *xel, owner string) bpmnext.ExtensionElements {
	var ext bpmnext.ExtensionElements
	ev := reflect.ValueOf(&ext).Elem()
	et := ev.Type()
	for _, k := range x.kids {
		if k.name.Space != bpmnext.Namespace {
			continue
		}
		fi := -1
		for i := 0; i < et.NumField(); i++ {
			if tagLocal(et.Field(i).Tag.Get("xml")) == k.name.Local {
				fi = i
			}
		}
		if fi < 0 {
			p.fail(errcodes.ProcessSchemaViolation, owner, "неизвестный элемент ant:%s", k.name.Local)
			continue
		}
		slice := ev.Field(fi)
		item := reflect.New(slice.Type().Elem()).Elem()
		p.fill(item, k, owner)
		slice.Set(reflect.Append(slice, item))
	}
	return ext
}

func tagLocal(tag string) string {
	name, _, _ := strings.Cut(tag, ",")
	if i := strings.LastIndexByte(name, ' '); i >= 0 {
		name = name[i+1:]
	}
	return name
}

// fill — атрибуты и дочерние элементы ant:… в структуру по тегам xml.
func (p *parser) fill(v reflect.Value, x *xel, owner string) {
	t := v.Type()
	for _, a := range x.attrs {
		if a.Name.Space != "" && a.Name.Space != bpmnext.Namespace {
			continue
		}
		fi := -1
		for i := 0; i < t.NumField(); i++ {
			tag := t.Field(i).Tag.Get("xml")
			if strings.Contains(tag, ",attr") && tagLocal(tag) == a.Name.Local {
				fi = i
			}
		}
		if fi < 0 {
			p.fail(errcodes.ProcessSchemaViolation, owner, "ant:%s: неизвестный атрибут %s", x.name.Local, a.Name.Local)
			continue
		}
		f := v.Field(fi)
		switch {
		case f.Kind() == reflect.String:
			f.SetString(a.Value)
		case f.Kind() == reflect.Pointer && f.Type().Elem().Kind() == reflect.Int64:
			n, err := strconv.ParseInt(strings.TrimSpace(a.Value), 10, 64)
			if err != nil {
				p.fail(errcodes.ProcessSchemaViolation, owner, "ant:%s/@%s: не целое %q", x.name.Local, a.Name.Local, a.Value)
				continue
			}
			f.Set(reflect.ValueOf(&n))
		case f.Kind() == reflect.Pointer && f.Type().Elem().Kind() == reflect.Bool:
			b, err := strconv.ParseBool(strings.TrimSpace(a.Value))
			if err != nil {
				p.fail(errcodes.ProcessSchemaViolation, owner, "ant:%s/@%s: не логическое %q", x.name.Local, a.Name.Local, a.Value)
				continue
			}
			f.Set(reflect.ValueOf(&b))
		}
	}
	for _, c := range x.kids {
		fi := -1
		for i := 0; i < t.NumField(); i++ {
			tag := t.Field(i).Tag.Get("xml")
			if !strings.Contains(tag, ",attr") && tagLocal(tag) == c.name.Local && t.Field(i).Name != "XMLName" {
				fi = i
			}
		}
		if c.name.Space != bpmnext.Namespace || fi < 0 {
			p.fail(errcodes.ProcessSchemaViolation, owner, "ant:%s: неизвестный элемент %s", x.name.Local, c.name.Local)
			continue
		}
		if v.Field(fi).Kind() == reflect.String {
			v.Field(fi).SetString(strings.TrimSpace(c.text.String()))
		}
	}
}

// link — стрелки к узлам, дорожки, step_key и условия.
func (p *parser) link() {
	d := p.def
	for _, id := range sortedKeys(d.Flows) {
		f := d.Flows[id]
		src, tgt := d.Nodes[f.Source], d.Nodes[f.Target]
		if src == nil || tgt == nil {
			p.fail(errcodes.ProcessSchemaViolation, f.ID, "стрелка ссылается на несуществующий узел")
			continue
		}
		if src.Scope != f.Scope || tgt.Scope != f.Scope {
			p.fail(errcodes.ProcessSchemaViolation, f.ID, "стрелка пересекает границу процесса или подпроцесса")
		}
	}
	// Порядок стрелок у узла — порядок документа (детерминизм выбора ветки).
	for _, fid := range p.flowOrder() {
		f := d.Flows[fid]
		if src, tgt := d.Nodes[f.Source], d.Nodes[f.Target]; src != nil && tgt != nil {
			src.Out = append(src.Out, f.ID)
			tgt.In = append(tgt.In, f.ID)
		}
	}
	for _, l := range d.Lanes {
		for _, nid := range l.Nodes {
			if n := d.Nodes[nid]; n != nil {
				n.Lane = l.ID
			}
		}
	}
	for _, id := range d.Order {
		n := d.Nodes[id]
		if n.Type == NodeBoundaryEvent {
			if host := d.Nodes[n.AttachedTo]; host != nil {
				host.Boundaries = append(host.Boundaries, n.ID)
			} else {
				p.fail(errcodes.ProcessSchemaViolation, n.ID, "граничное событие без узла attachedToRef")
			}
		}
		if n.Type == NodeStartEvent {
			d.Scopes[n.Scope].Starts = append(d.Scopes[n.Scope].Starts, n.ID)
		}
		if k := n.Props.StepKey; k != "" {
			if prev, dup := d.Steps[k]; dup {
				p.fail(errcodes.ProcessSchemaViolation, n.ID, "step_key %q уже у узла %s — ключ уникален во всём файле (AD-17)", k, prev)
			} else {
				d.Steps[k] = n.ID
			}
		}
	}
}

// flowOrder — id стрелок в порядке документа (по порядку узлов-источников,
// затем по id — устойчиво).
func (p *parser) flowOrder() []string {
	ids := sortedKeys(p.def.Flows)
	pos := map[string]int{}
	for i, id := range p.def.Order {
		pos[id] = i
	}
	out := make([]string, 0, len(ids))
	out = append(out, ids...)
	sortStableBy(out, func(a, b string) bool {
		return pos[p.def.Flows[a].Source] < pos[p.def.Flows[b].Source]
	})
	return out
}

// ParseISODuration — длительность ISO 8601 вида PnDTnHnMnS (целые единицы;
// недели — PnW). Годы и месяцы не поддерживаются: их длина зависит от календаря.
func ParseISODuration(s string) (time.Duration, error) {
	if len(s) < 2 || s[0] != 'P' {
		return 0, fmt.Errorf("ожидалась длительность ISO 8601 вида PT8H")
	}
	var total time.Duration
	inTime := false
	num := ""
	for _, c := range s[1:] {
		switch {
		case c >= '0' && c <= '9':
			num += string(c)
		case c == 'T':
			if inTime || num != "" {
				return 0, fmt.Errorf("неверная длительность %q", s)
			}
			inTime = true
		default:
			if num == "" {
				return 0, fmt.Errorf("неверная длительность %q", s)
			}
			n, err := strconv.ParseInt(num, 10, 64)
			if err != nil {
				return 0, err
			}
			num = ""
			var unit time.Duration
			switch {
			case !inTime && c == 'W':
				unit = 7 * 24 * time.Hour
			case !inTime && c == 'D':
				unit = 24 * time.Hour
			case inTime && c == 'H':
				unit = time.Hour
			case inTime && c == 'M':
				unit = time.Minute
			case inTime && c == 'S':
				unit = time.Second
			default:
				return 0, fmt.Errorf("единица %q в %q не поддерживается (годы и месяцы — зависят от календаря)", string(c), s)
			}
			total += time.Duration(n) * unit
		}
	}
	if num != "" || total <= 0 {
		return 0, fmt.Errorf("неверная длительность %q", s)
	}
	return total, nil
}
