package process

import (
	"strings"
	"testing"
	"time"

	"ant/internal/contracts/errcodes"
)

func hasViolation(vs []Violation, code errcodes.Code, element string) bool {
	for _, v := range vs {
		if v.Code == code && v.Element == element {
			return true
		}
	}
	return false
}

func mutate(t *testing.T, old, new string) []Violation {
	t.Helper()
	if !strings.Contains(miniBPMN, old) {
		t.Fatalf("в учебном процессе нет фрагмента %q", old)
	}
	_, vs := Load([]byte(strings.Replace(miniBPMN, old, new, 1)), LoadOptions{})
	return vs
}

// FR-13: каждое нарушение — код и id элемента.
func TestLoadRejectsWithCodeAndElement(t *testing.T) {
	cases := []struct {
		name, old, new string
		code           errcodes.Code
		element        string
	}{
		{"неподдерживаемый элемент", `<bpmn:sequenceFlow id="f23" sourceRef="E1" targetRef="END"/>`, `<bpmn:sequenceFlow id="f23" sourceRef="E1" targetRef="END"/><bpmn:scriptTask id="SCR"/>`, errcodes.ProcessUnsupportedElement, "SCR"},
		{"неподдерживаемое событие", `<bpmn:terminateEventDefinition id="TX1"/>`, `<bpmn:errorEventDefinition id="TX1"/>`, errcodes.ProcessUnsupportedElement, "X1"},
		{"нет stepKey", `<ant:properties stepKey="welding.kt3" stepKind`, `<ant:properties stepKind`, errcodes.ProcessStepKeyMissing, "K"},
		{"неявное слияние", `<bpmn:sequenceFlow id="f5" sourceRef="WT" targetRef="J_P"/>`, `<bpmn:sequenceFlow id="f5" sourceRef="WT" targetRef="P"/>`, errcodes.ProcessImplicitMerge, "P"},
		{"недостижимый узел", `<bpmn:sequenceFlow id="f23" sourceRef="E1" targetRef="END"/>`,
			`<bpmn:sequenceFlow id="f23" sourceRef="E1" targetRef="END"/><bpmn:task id="LOST"><bpmn:extensionElements><ant:properties stepKey="lost" stepKind="operation"/></bpmn:extensionElements></bpmn:task><bpmn:sequenceFlow id="f99" sourceRef="LOST" targetRef="END2"/><bpmn:endEvent id="END2"><bpmn:extensionElements><ant:properties stepKey="lost.end"/></bpmn:extensionElements></bpmn:endEvent>`,
			errcodes.ProcessUnreachableNode, "LOST"},
		{"точка предъявления без роли", `<ant:presentationPoint authority="qc_acceptance" role="quality_inspector" waitLimitMinutes="60"`, `<ant:presentationPoint authority="qc_acceptance" waitLimitMinutes="60"`, errcodes.ProcessPresentationPointWithoutRole, "G"},
		{"условие не по языку", `decision == 'insufficient_data'`, `decision == 'maybe'`, errcodes.ProcessConditionInvalid, "f10"},
		{"вызов функции в условии", `nc.outcome == 'scrapped'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f15"`, `len(nc.outcome) == 'scrapped'</bpmn:conditionExpression></bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f15"`, errcodes.ProcessConditionInvalid, "f14"},
		{"недостижимый шлюз", `<bpmn:sequenceFlow id="f16" sourceRef="A" targetRef="G2"/>`,
			`<bpmn:sequenceFlow id="f16" sourceRef="A" targetRef="G2"/><bpmn:exclusiveGateway id="SKIP"><bpmn:extensionElements><ant:properties stepKey="skip"/></bpmn:extensionElements></bpmn:exclusiveGateway>`,
			errcodes.ProcessUnreachableNode, "SKIP"},
		{"неизвестный атрибут расширения", `stepKind="operation" closesZoneAccess="Z9"`, `stepKind="operation" closesZoneAccess="Z9" magic="1"`, errcodes.ProcessSchemaViolation, "A"},
		{"недопустимое значение", `timerScope="until_started"`, `timerScope="forever"`, errcodes.ProcessSchemaViolation, "WT"},
		{"таймер без длительности ISO", `>PT8H<`, `>8 часов<`, errcodes.ProcessSchemaViolation, "WT"},
		{"тип события не из каталога", `triggerEventType="erp.order.received"`, `triggerEventType="erp.order.invented"`, errcodes.ProcessSchemaViolation, "S0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vs := mutate(t, c.old, c.new)
			if !hasViolation(vs, c.code, c.element) {
				t.Fatalf("нет нарушения %s у %s: %v", c.code, c.element, vs)
			}
		})
	}
}

func TestClosingPathWithoutHuman(t *testing.T) {
	// Выход «в 1С» сразу после сварки, в обход ЗТ.
	vs := mutate(t, `<bpmn:sequenceFlow id="f4" sourceRef="W" targetRef="J_K"/>`, `<bpmn:sequenceFlow id="f4" sourceRef="W" targetRef="E1"/>`)
	if !hasViolation(vs, errcodes.ProcessClosingPathWithoutHuman, "E1") {
		t.Fatalf("закрывающий путь без контроля человеком не найден: %v", vs)
	}
}

func TestInspectionWithoutRequirementIsWarning(t *testing.T) {
	vs := mutate(t, `<ant:requirement characteristic="шов" tolerance="нет" kdRef="КД"/>`, ``)
	if !hasViolation(vs, errcodes.ProcessInspectionWithoutRequirement, "K") {
		t.Fatalf("нет предупреждения: %v", vs)
	}
	if len(Errors(vs)) != 0 {
		t.Fatalf("предупреждение стало отказом: %v", Errors(vs))
	}
}

func TestBrokenXML(t *testing.T) {
	_, vs := Load([]byte("<bpmn:definitions"), LoadOptions{})
	if len(vs) != 1 || vs[0].Code != errcodes.ProcessSchemaViolation {
		t.Fatalf("битый XML: %v", vs)
	}
}

func TestMiniModel(t *testing.T) {
	d, vs := Load([]byte(miniBPMN), LoadOptions{})
	if len(Errors(vs)) != 0 {
		t.Fatal(vs)
	}
	if d.Main != "Main" || d.ByStep("welding.weld").ID != "W" || !d.ByStep("welding.weld").Special() {
		t.Fatalf("модель: main=%s", d.Main)
	}
	if got := d.SpecialSteps(); len(got) != 1 || got[0] != "welding.weld" {
		t.Fatalf("специальные процессы: %v", got)
	}
	if n := d.Node("WT"); n.TimerDur != 8*time.Hour || n.AttachedTo != "W" || d.Node("W").Boundaries[0] != "WT" {
		t.Fatalf("граничный таймер: %+v", n)
	}
	if d.Workshop(d.Node("W")) != "WS-WC" {
		t.Fatal("цех дорожки")
	}
}

func TestExpr(t *testing.T) {
	vars := map[string]Value{VarDecision: Str("accept"), VarReworkCount: Int(2), VarToolsAccounted: Bool(true)}
	cases := map[string]bool{
		"decision == 'accept'": true,
		"decision == 'accept' or decision == 'accept_with_concession'": true,
		"not (decision == 'reject')":                                   true,
		"rework.count < 3 and decision != 'reject'":                    true,
		"rework.count >= 3":                                            false,
		"tools.accounted == false":                                     false,
		"nc.outcome == 'scrapped'":                                     false, // нет данных — не ветка
	}
	for _, src := range sortedKeys(cases) {
		want := cases[src]
		e, err := ParseExpr(src, nil)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		if got := e.Eval(vars); got != want {
			t.Fatalf("%s = %v, ожидалось %v", src, got, want)
		}
	}
	for _, bad := range []string{"decision || x", "decision == accept", "rework.count == 1.5", "unknown.var == 'x'", "decision < 'accept'",
		"tools.accounted == 'yes'", "f(decision) == 'x'", "decision == 'accept' and", "(decision == 'accept'"} {
		if _, err := ParseExpr(bad, nil); err == nil {
			t.Fatalf("условие %q принято", bad)
		}
	}
}

func TestISODuration(t *testing.T) {
	cases := map[string]time.Duration{"PT8H": 8 * time.Hour, "P1D": 24 * time.Hour, "PT30M": 30 * time.Minute, "P1DT2H": 26 * time.Hour, "P1W": 168 * time.Hour}
	for _, s := range sortedKeys(cases) {
		h := cases[s]
		d, err := ParseISODuration(s)
		if err != nil || d != h {
			t.Fatalf("%s → %v %v", s, d, err)
		}
	}
	for _, s := range []string{"8H", "P1Y", "PT", "P1M", ""} {
		if _, err := ParseISODuration(s); err == nil {
			t.Fatalf("%s принято", s)
		}
	}
}
