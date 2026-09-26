package erp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	dom "ant/internal/domain/erp"
)

// TopologyFromBPMN — цеха процесса и их склады из нормативного слоя
// (FR-130): дорожки BPMN с `ant:properties/@workshop` и `@warehouse` в
// порядке laneSet и шаги (step_key) их узлов. По ним модуль erp вычисляет
// склады «смены склада» при передаче изделия между цехами. Узлы без дорожки
// (вызываемый подпроцесс «Брак») в цеха не входят.
func TopologyFromBPMN(src []byte) (dom.Topology, error) {
	dec := xml.NewDecoder(bytes.NewReader(src))
	stepOf := map[string]string{} // id узла → step_key
	type laneRaw struct {
		dom.Lane
		refs []string
	}
	var (
		lanes    []*laneRaw
		stack    []string // id элементов с id
		inLane   *laneRaw
		inRef    bool
		refText  strings.Builder
		depthIDs []bool
	)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return dom.Topology{}, fmt.Errorf("erp: BPMN: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			id := attr(t, "id")
			depthIDs = append(depthIDs, id != "")
			if id != "" {
				stack = append(stack, id)
			}
			switch t.Name.Local {
			case "lane":
				inLane = &laneRaw{Lane: dom.Lane{ID: id}}
				lanes = append(lanes, inLane)
			case "flowNodeRef":
				inRef = inLane != nil
				refText.Reset()
			case "properties":
				if inLane != nil && len(stack) > 0 && stack[len(stack)-1] == inLane.ID {
					inLane.Workshop, inLane.Warehouse = attr(t, "workshop"), attr(t, "warehouse")
				} else if sk := attr(t, "stepKey"); sk != "" && len(stack) > 0 {
					stepOf[stack[len(stack)-1]] = sk
				}
			}
		case xml.CharData:
			if inRef {
				refText.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "lane":
				inLane = nil
			case "flowNodeRef":
				if inRef && inLane != nil {
					inLane.refs = append(inLane.refs, strings.TrimSpace(refText.String()))
				}
				inRef = false
			}
			if n := len(depthIDs); n > 0 {
				if depthIDs[n-1] && len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				depthIDs = depthIDs[:n-1]
			}
		}
	}
	var topo dom.Topology
	for _, l := range lanes {
		for _, r := range l.refs {
			if sk, ok := stepOf[r]; ok {
				l.Steps = append(l.Steps, sk)
			}
		}
		if l.Warehouse == "" {
			return dom.Topology{}, fmt.Errorf("erp: BPMN: у дорожки %s нет склада (ant:properties/@warehouse, FR-130)", l.ID)
		}
		topo.Lanes = append(topo.Lanes, l.Lane)
	}
	if len(topo.Lanes) == 0 {
		return topo, errors.New("erp: BPMN: нет дорожек цехов (FR-130)")
	}
	return topo, nil
}

func attr(t xml.StartElement, name string) string {
	for _, a := range t.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}
