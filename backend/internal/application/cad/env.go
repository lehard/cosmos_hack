package cad

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	dom "ant/internal/domain/cad"
)

// EnvFromBPMN — нормативный слой для перевода связей сборки (AD-17) из
// версии процесса: у шагов `ant:properties` берутся лимит ремонтов с
// областью «зона» (`reworkLimit`, `reworkLimitScope="zone"`) и его зоны
// (`ant:zoneRef`, участки W-1.U1… относятся к связи W-1), шаги, закрывающие
// доступ к зонам (`closesZoneAccess`), и операции, работающие с зоной. Так
// импорт КОМПАС сверяет связи сборки с утверждённым ТП: лимит ремонтов шва,
// шаг скрытых работ, шаг установки уплотнения.
func EnvFromBPMN(src []byte) (dom.Env, error) {
	env := dom.Env{ReworkLimits: map[string]dom.Limit{}, ClosingSteps: map[string]string{}, ZoneSteps: map[string][]string{}}
	dec := xml.NewDecoder(bytes.NewReader(src))
	type step struct {
		key, kind, scope, closes string
		limit                    int
		zones                    []string
	}
	var cur *step
	finish := func() {
		if cur == nil {
			return
		}
		for _, z := range cur.zones {
			link, _, _ := strings.Cut(z, ".")
			if cur.scope == "zone" && cur.limit > 0 {
				if _, ok := env.ReworkLimits[link]; !ok {
					env.ReworkLimits[link] = dom.Limit{Value: cur.limit, StepKey: cur.key}
				}
			}
			if cur.kind == "operation" && !slices.Contains(env.ZoneSteps[link], cur.key) {
				env.ZoneSteps[link] = append(env.ZoneSteps[link], cur.key)
			}
		}
		for _, z := range strings.Fields(cur.closes) {
			if _, ok := env.ClosingSteps[z]; !ok {
				env.ClosingSteps[z] = cur.key
			}
		}
		cur = nil
	}
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return dom.Env{}, fmt.Errorf("cad: BPMN: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "properties":
				if k := attr(t, "stepKey"); k != "" {
					finish()
					cur = &step{key: k, kind: attr(t, "stepKind"), scope: attr(t, "reworkLimitScope"), closes: attr(t, "closesZoneAccess")}
					cur.limit, _ = strconv.Atoi(attr(t, "reworkLimit"))
				}
			case "zoneRef":
				if cur != nil {
					cur.zones = append(cur.zones, attr(t, "zone"))
				}
			}
		case xml.EndElement:
			if t.Name.Local == "extensionElements" {
				finish()
			}
		}
	}
	finish()
	return env, nil
}

func attr(t xml.StartElement, name string) string {
	for _, a := range t.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}
