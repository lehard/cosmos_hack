package world

import (
	"encoding/hex"
	"encoding/xml"
	"strings"

	"go.stargrave.org/gogost/v7/gost34112012256"
)

// BpmnNode — узел стартового процесса с описанием и нашими свойствами (FR-154, FR-156).
type BpmnNode struct {
	ID, Name, Type, StepKey, Lane, Doc string
	Props                              map[string]string
	Norms                              []BpmnNorm
	Next                               []string
}

// BpmnNorm — нормативная опора узла (ant:normRef).
type BpmnNorm struct{ Standard, Clause, Check, Action string }

type xnode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Content  string     `xml:",chardata"`
	Children []xnode    `xml:",any"`
}

func (x xnode) attr(name string) string {
	for _, a := range x.Attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// ParseBpmn читает узлы процесса: id, имя, тип, step_key, дорожку, документацию, свойства, опоры.
func ParseBpmn(b []byte) (map[string]*BpmnNode, []*BpmnNode, error) {
	var root xnode
	if err := xml.Unmarshal(b, &root); err != nil {
		return nil, nil, err
	}
	byKey := map[string]*BpmnNode{}
	var order []*BpmnNode
	laneOf := map[string]string{}
	byID := map[string]*BpmnNode{}
	var flows [][2]string
	var walk func(x xnode, lane string)
	walk = func(x xnode, lane string) {
		switch x.XMLName.Local {
		case "lane":
			for _, c := range x.Children {
				if c.XMLName.Local == "flowNodeRef" {
					laneOf[strings.TrimSpace(c.Content)] = x.attr("name")
				}
			}
		case "sequenceFlow":
			flows = append(flows, [2]string{x.attr("sourceRef"), x.attr("targetRef")})
		}
		id := x.attr("id")
		var props *xnode
		for i := range x.Children {
			if x.Children[i].XMLName.Local == "extensionElements" {
				for j := range x.Children[i].Children {
					if x.Children[i].Children[j].XMLName.Local == "properties" {
						props = &x.Children[i].Children[j]
					}
				}
			}
		}
		if id != "" && props != nil && props.attr("stepKey") != "" && x.XMLName.Local != "lane" {
			n := &BpmnNode{ID: id, Name: x.attr("name"), Type: x.XMLName.Local, StepKey: props.attr("stepKey"), Props: map[string]string{}}
			for _, a := range props.Attrs {
				if a.Name.Local != "stepKey" {
					n.Props[a.Name.Local] = a.Value
				}
			}
			for _, c := range x.Children {
				switch c.XMLName.Local {
				case "documentation":
					n.Doc = strings.TrimSpace(c.Content)
				case "extensionElements":
					for _, e := range c.Children {
						if e.XMLName.Local != "normRef" {
							continue
						}
						nr := BpmnNorm{Standard: e.attr("standard"), Clause: e.attr("clause")}
						for _, cc := range e.Children {
							switch cc.XMLName.Local {
							case "check":
								nr.Check = strings.TrimSpace(cc.Content)
							case "systemAction":
								nr.Action = strings.TrimSpace(cc.Content)
							}
						}
						n.Norms = append(n.Norms, nr)
					}
				}
			}
			byKey[n.StepKey] = n
			byID[id] = n
			order = append(order, n)
		}
		for _, c := range x.Children {
			walk(c, lane)
		}
	}
	walk(root, "")
	for _, n := range order {
		n.Lane = laneOf[n.ID]
	}
	for _, f := range flows {
		if s, t := byID[f[0]], byID[f[1]]; s != nil && t != nil {
			s.Next = append(s.Next, t.ID)
		}
	}
	return byKey, order, nil
}

// Digest — отпечаток байтов по формату цепочки: streebog256:‹hex› (AD-44).
func Digest(b []byte) string {
	h := gost34112012256.New()
	h.Write(b)
	return "streebog256:" + hex.EncodeToString(h.Sum(nil))
}

// nodeKind — вид узла для карточки и читаемой версии (перечисление контракта).
func nodeKind(n *BpmnNode) string {
	switch n.Type {
	case "serviceTask":
		return "automatedInspection"
	case "userTask":
		if n.Props["stepKind"] == "human_inspection" {
			return "humanInspection"
		}
		return "operation"
	case "task":
		return "operation"
	case "parallelGateway":
		return "parallelGateway"
	case "inclusiveGateway":
		return "inclusiveGateway"
	case "exclusiveGateway":
		return "conditionalFlow"
	case "startEvent":
		return "startEvent"
	case "endEvent":
		return "endEvent"
	case "boundaryEvent":
		return "timer"
	case "callActivity":
		return "callActivity"
	case "subProcess":
		return "subprocess"
	}
	return "intermediateEvent"
}
