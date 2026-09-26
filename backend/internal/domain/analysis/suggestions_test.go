package analysis_test

import (
	"strings"
	"testing"
	"time"

	"ant/internal/domain/analysis"
)

var t0 = time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC)

// Ограничение линии (UJ-1): оценка потерь в минутах и деталях; без очереди и ожидания — предложения нет.
func TestBottleneckProposal(t *testing.T) {
	p, ok := analysis.BottleneckProposal(analysis.LineBottleneck{StepKey: "quality.zt3", WaitSec: 600, Queue: 4, Passed: 40, Period: "смену"})
	if !ok || p.Kind != analysis.SuggestBottleneck || p.ResponsibleRole != analysis.RoleSiteForeman || p.StepKey != "quality.zt3" {
		t.Fatalf("%+v", p)
	}
	if !strings.Contains(p.Estimate, "в среднем 10 мин") || !strings.Contains(p.Estimate, "около 6 ч 40 мин") {
		t.Fatalf("оценка: %s", p.Estimate)
	}
	if _, ok := analysis.BottleneckProposal(analysis.LineBottleneck{StepKey: "x"}); ok {
		t.Fatal("пустой узел — не ограничение")
	}
	id1, id2 := analysis.SuggestionID(p.Generator, p.DedupKey), analysis.SuggestionID("rules.bottleneck", p.DedupKey)
	if id1 != id2 || !strings.HasPrefix(id1, "SUG-") || id1 == analysis.SuggestionID(p.Generator, "другой ключ") {
		t.Fatal("id предложения не стабилен")
	}
}

func incident(id string, initial int, versions ...analysis.VersionRecord) analysis.IncidentRecord {
	v := analysis.IncidentRecord{IncidentID: id, Factor: "equipment", FactorValue: "IS-2", InitialSize: initial, Versions: versions,
		Members: map[string]analysis.MemberRecord{}, TriggerEventIDs: []string{"00000000-0000-7000-8000-000000000001"}}
	return v
}

// Область риска: без сужений — предложение сузить; с сужением — нет; ушедшие дальше — известить.
func TestRiskScopeProposals(t *testing.T) {
	a := incident("RS-1", 5, analysis.VersionRecord{Version: 1, Change: analysis.ChangeComputed, Size: 5, EventID: "e1",
		Breakdown: analysis.Breakdown{InProduction: 3, Shipped: 2}})
	for i := range 5 {
		a.Members["ENT01:F-0"+string(rune('1'+i))] = analysis.MemberRecord{Status: "suspected"}
	}
	ps := analysis.RiskScopeProposals([]analysis.IncidentRecord{a})
	if len(ps) != 2 || ps[0].IncidentID != "RS-1" || ps[0].ResponsibleRole != analysis.RoleTechnologist || len(ps[0].Basis) != 2 {
		t.Fatalf("%+v", ps)
	}
	if !strings.Contains(ps[1].Statement, "отгружены 2") {
		t.Fatalf("ушедшие дальше: %s", ps[1].Statement)
	}
	a.Versions = append(a.Versions, analysis.VersionRecord{Version: 2, Change: analysis.ChangeNarrowed, Size: 3, EventID: "e2"})
	a.Closed = false
	if ps := analysis.RiskScopeProposals([]analysis.IncidentRecord{a}); len(ps) != 0 {
		t.Fatalf("после сужения без ушедших: %+v", ps)
	}
}

// Кандидаты в правила реакции: три одинаковых ручных решения — кандидат; два — нет.
func TestReactionRuleCandidates(t *testing.T) {
	mk := func(id string, n int) analysis.IncidentRecord {
		v := incident(id, 3)
		v.Decisions = map[string]analysis.DecisionRecord{}
		for i := range n {
			v.Decisions[id+"-d"+string(rune('0'+i))] = analysis.DecisionRecord{Type: "incident.scope.narrowed", Reason: analysis.ReasonRecord{Code: "log_in_setpoint", Text: "Журнал в уставке"}}
		}
		return v
	}
	if ps := analysis.ReactionRuleCandidates([]analysis.IncidentRecord{mk("RS-1", 2)}); len(ps) != 0 {
		t.Fatalf("два решения — ещё не правило: %+v", ps)
	}
	ps := analysis.ReactionRuleCandidates([]analysis.IncidentRecord{mk("RS-1", 2), mk("RS-2", 1)})
	if len(ps) != 1 || ps[0].Kind != analysis.SuggestReactionRule || len(ps[0].Basis) != 3 || !strings.Contains(ps[0].Statement, "RS-1, RS-2") {
		t.Fatalf("%+v", ps)
	}
}

// Адаптация VisionQC: камера «признаков нет» в окне, затем находка другим методом — пропуск.
func TestAnalyzerAdaptationProposals(t *testing.T) {
	a := analysis.Analysis{NCID: "NC-1", Window: &analysis.Window{Start: t0, End: t0.Add(time.Hour)},
		Profile: analysis.Profile{StepKey: "welding.weld", DefectType: "crack"},
		Records: []analysis.Mark{
			{EventID: "cam", Lane: analysis.LaneItem, Variant: "no_defect_indicated", OccurredAt: t0.Add(10 * time.Minute), Params: map[string]string{"method": "camera"}},
			{EventID: "vis", Lane: analysis.LaneItem, Variant: "defect_indicated", OccurredAt: t0.Add(50 * time.Minute), Params: map[string]string{"method": "visual_human"}},
		}}
	ps := analysis.AnalyzerAdaptationProposals([]analysis.Analysis{a})
	if len(ps) != 1 || ps[0].ResponsibleRole != analysis.RoleHeadOfQC || len(ps[0].Basis) != 2 {
		t.Fatalf("%+v", ps)
	}
	a.Records[1].Params["method"] = "camera"
	if ps := analysis.AnalyzerAdaptationProposals([]analysis.Analysis{a}); len(ps) != 0 {
		t.Fatalf("находка камерой — не пропуск: %+v", ps)
	}
}

// Карта дефицита (FR-143): число расследований по виду сведений, где, оценка
// сужения по сужениям с основаниями; без сужений — оценки нет.
func TestDeficitMap(t *testing.T) {
	an := func(nc, eq string, missing ...string) analysis.Analysis {
		return analysis.Analysis{NCID: nc, Missing: missing, Profile: analysis.Profile{Equipment: eq, StepKey: "machining.turn"}}
	}
	analyses := []analysis.Analysis{an("NC-1", "M-17", analysis.MissingTool), an("NC-2", "M-18", analysis.MissingTool, analysis.MissingOperator),
		an("NC-3", "M-17", analysis.MissingTool), an("NC-4", "M-20")}
	items := map[string]string{"NC-1": "ENT01:A", "NC-2": "ENT01:B", "NC-3": "ENT01:C", "NC-4": "ENT01:D"}
	narrowed := incident("RS-1", 13, analysis.VersionRecord{Version: 1, Change: analysis.ChangeComputed, Size: 13},
		analysis.VersionRecord{Version: 2, Change: analysis.ChangeNarrowed, Size: 4})
	narrowed.Members = map[string]analysis.MemberRecord{"ENT01:A": {Status: "suspected"}, "ENT01:X1": {Status: "suspected"}, "ENT01:X2": {Status: "suspected"},
		"ENT01:X3": {Status: "suspected"}}
	for i := range 9 {
		narrowed.Members["ENT01:E"+string(rune('0'+i))] = analysis.MemberRecord{Status: analysis.StatusExcluded}
	}
	m := analysis.BuildDeficitMap(analyses, []analysis.IncidentRecord{narrowed}, items)
	if m.Investigations != 4 || len(m.Rows) != 2 || m.Rows[0].Kind != analysis.MissingTool || m.Rows[0].Investigations != 3 {
		t.Fatalf("%+v", m)
	}
	if p := m.Rows[0].Places; len(p) != 2 || p[0].Place != "M-17" || p[0].Count != 2 {
		t.Fatalf("места: %+v", p)
	}
	e := m.Rows[0].Estimate
	if e == nil || e.FromTenths != 130 || e.Incidents != 1 {
		t.Fatalf("оценка: %+v", e)
	}
	if m.Rows[1].Estimate != nil {
		t.Fatalf("без связанного сужения оценки нет: %+v", m.Rows[1].Estimate)
	}
	ps := analysis.DeficitProposals(m)
	if len(ps) != 1 || !strings.Contains(ps[0].Statement, "В 3 расследованиях из 4 неизвестен инструмент") ||
		!strings.Contains(ps[0].Estimate, "в среднем с 13 до") {
		t.Fatalf("%+v", ps)
	}
}

// Меры: без плана — отказ; «эффективно» — только после окна; провал — переоткрыта.
func TestActionGuardsAndFlags(t *testing.T) {
	if err := analysis.GuardPlan(analysis.EffectivenessPlan{Metric: "m"}); err == nil {
		t.Fatal("план без базового уровня, окна и критерия принят")
	}
	impl := t0
	a := analysis.CorrectiveAction{ActionID: "ACT-1", Status: analysis.ActionImplemented, ImplementedAt: &impl, AssignedAt: t0.Add(-48 * time.Hour),
		Plan: analysis.EffectivenessPlan{WindowDays: 7, EnhancedControl: "100 %"}}
	if err := analysis.GuardEvaluate(a, "effective", t0.Add(24*time.Hour)); err == nil {
		t.Fatal("«эффективно» до конца окна принято")
	}
	if err := analysis.GuardEvaluate(a, "failed", t0.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := analysis.GuardEvaluate(a, "effective", t0.Add(8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	f := analysis.ActionFlags(a, t0.Add(8*24*time.Hour))
	if len(f) != 2 || f[0] != analysis.FlagHangingControl || f[1] != analysis.FlagEvaluationDue {
		t.Fatalf("флаги: %v", f)
	}
	if err := analysis.GuardImplement(a); err == nil {
		t.Fatal("повторное внедрение принято")
	}
}
