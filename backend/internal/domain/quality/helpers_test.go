package quality

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/normative"
	"ant/internal/domain/kernel"
)

// Опора тестов домена: компактный нормативный слой по образцу
// normative/reactions/reaction-map.v1.yaml и classifier.v1.yaml (полные файлы
// проверяет application/quality — домену ввод-вывод запрещён, AD-4).

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

func ip(v int) *int    { return &v }
func bptr(v bool) *bool { return &v }

func rule(id string, prio, mode int, m normative.ReactionMapRulesElemMatch, outcome string, cont string, draft bool, task string) normative.ReactionMapRulesElem {
	r := normative.ReactionMapRulesElem{ID: id, Title: id, Priority: prio, AutomationMode: mode, Match: m,
		Reaction: normative.ReactionMapRulesElemReaction{Outcome: normative.ReactionMapRulesElemReactionOutcome(outcome),
			ContainmentLevel: normative.ReactionMapRulesElemReactionContainmentLevel(cont), DraftNc: draft}}
	if task != "" {
		tk := normative.ReactionMapRulesElemReactionTask(task)
		r.Reaction.Task = &tk
	}
	return r
}

type mo = normative.ReactionMapRulesElemMatchOutcomeElem
type ms = normative.ReactionMapRulesElemMatchSeverityElem

func testEnv() Env {
	defect := []mo{"defect_indicated"}
	rm := normative.ReactionMap{ID: "flange-reactions", Version: 1, Rules: []normative.ReactionMapRulesElem{
		rule("R-01", 10, 1, normative.ReactionMapRulesElemMatch{Outcome: []mo{"unable_to_assess"}}, ReactManual, "additional_check", false, "recheck"),
		rule("R-02", 20, 1, normative.ReactionMapRulesElemMatch{Outcome: []mo{"no_defect_indicated"}, ObservationQualityBelowRecipe: bptr(true)}, ReactManual, "additional_check", false, "recheck"),
		rule("R-07", 30, 2, normative.ReactionMapRulesElemMatch{Outcome: defect, DefectTypes: []string{"W-BURNTHRU", "W-UNDERCUT"}}, ReactIsolate, "item_hold", true, "isolate_move"),
		rule("R-03", 40, 2, normative.ReactionMapRulesElemMatch{Outcome: defect, Severity: []ms{"critical"}}, ReactIsolate, "item_hold", true, "isolate_move"),
		rule("R-04", 50, 2, normative.ReactionMapRulesElemMatch{Outcome: defect, Severity: []ms{"major"}, ConfidenceMinBp: ip(6000)}, ReactIsolate, "item_hold", true, "isolate_move"),
		rule("R-05", 60, 1, normative.ReactionMapRulesElemMatch{Outcome: defect}, ReactManual, "additional_check", true, "decision_required"),
		rule("R-06", 5, 1, normative.ReactionMapRulesElemMatch{Outcome: defect, NoRequirement: bptr(true)}, ReactQuestion, "observe", false, "decision_required"),
		rule("R-08", 90, 2, normative.ReactionMapRulesElemMatch{Outcome: []mo{"no_defect_indicated"}, ConfidenceMinBp: ip(9000), TrustLevelMin: ip(4)}, ReactPassToNext, "none", false, ""),
		rule("R-22", 12, 1, normative.ReactionMapRulesElemMatch{BasisKind: []normative.ReactionMapRulesElemMatchBasisKindElem{"check_skipped"}}, ReactManual, "additional_check", false, "inspection_missing"),
	}}
	rm.Rules[7].Reaction.SampledRecheckBp = ip(1000)
	type dt = normative.DefectClassifierDefectTypesElem
	type mm = normative.DefectClassifierDefectTypesElemMethodsElem
	type zk = normative.DefectClassifierDefectTypesElemZoneKindsElem
	cl := normative.DefectClassifier{Version: 1, ItemTypeID: "FL-100.00.000", DefectTypes: []dt{
		{Code: "M-DIM", Name: "Размер вне допуска", Severity: "major", Measurable: true, Methods: []mm{"cmm"}, ZoneKinds: []zk{"surface"}},
		{Code: "M-DENT", Name: "Забоина", Severity: "major", Methods: []mm{"camera", "visual_human"}, ZoneKinds: []zk{"surface"}},
		{Code: "W-UNDERCUT", Name: "Подрез", Severity: "major", Methods: []mm{"camera", "visual_human"}, ZoneKinds: []zk{"weld_section"}},
		{Code: "W-PORE-S", Name: "Поры", Severity: "major", Methods: []mm{"camera", "visual_human", "radiography"}, ZoneKinds: []zk{"weld_section"}},
		{Code: "W-BURNTHRU", Name: "Прожог", Severity: "critical", Methods: []mm{"camera", "visual_human", "radiography"}, ZoneKinds: []zk{"weld_section"}},
		{Code: "W-LOF", Name: "Несплавление", Severity: "critical", Methods: []mm{"radiography", "ultrasonic"}, ZoneKinds: []zk{"weld_section"}},
		{Code: "T-LEAK", Name: "Течь", Severity: "critical", Measurable: true, Methods: []mm{"leak_test"}, ZoneKinds: []zk{"weld_section"}},
	}}
	steps := []StepSpec{
		{StepKey: "machining.turning", Order: 1, Stage: "machining", StepKind: "operation"},
		{StepKey: "machining.kt2_cmm", Order: 2, Stage: "machining", StepKind: "automated_inspection", InspectionPoint: "KT-2",
			Inspections:  []InspectionSpec{{Method: "cmm", Coverage: []string{"M-DIM"}}},
			Requirements: []Requirement{{Characteristic: "Размеры", KDRef: "ФЛ-100.01.001"}}},
		{StepKey: "welding.weld", Order: 10, Stage: "welding", StepKind: "operation"},
		{StepKey: "welding.kt3_camera", Order: 11, Stage: "welding", StepKind: "automated_inspection", InspectionPoint: "KT-3",
			Inspections: []InspectionSpec{{Method: "camera", Coverage: []string{"W-UNDERCUT", "W-PORE-S", "W-BURNTHRU"}, RecipeRef: "kt3-weld@1", ObservationQualityMinBP: 6000}},
			Zones:       []string{"W-1"}},
		{StepKey: "welding.kt3_radiography", Order: 12, Stage: "welding", StepKind: "automated_inspection", InspectionPoint: "KT-3",
			Inspections: []InspectionSpec{{Method: "radiography", Coverage: []string{"W-PORE-S", "W-LOF", "W-BURNTHRU"}}}, Zones: []string{"W-1"}},
		{StepKey: "welding.zt3_acceptance", Order: 13, Stage: "welding", StepKind: "human_inspection", ClosingPoint: "ZT-3"},
	}
	return Env{RuleRev: "rev-1", ReactionMap: rm, Classifier: cl, Steps: steps,
		Zones: []ZoneSpec{{ID: "W-1", Kind: "weld_section"}, {ID: "F-FACE", Kind: "surface"}}}
}

// withPassport — паспорт карты контроля КТ-3 с уровнем доверия lvl.
func withPassport(e Env, lvl int) Env {
	e.Passports = []Passport{{PassportID: "PP-KT3", RecipeRef: "kt3-weld@1", AnalyzerVersion: "vqc-weld 2.3.1", Stage: "active", TrustLevel: lvl,
		History: []PassportStatus{{At: t0.Add(-24 * time.Hour), Active: true, EventID: "pp-admitted", Note: "admitted"}}}}
	return e
}

// builder — вход изделия: записи с растущими occurred_at и seq.
type builder struct {
	n   int
	out []kernel.Record
}

func (b *builder) add(t catalog.Type, data any) kernel.Record {
	b.n++
	raw, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	info, _ := catalog.Lookup(t)
	r := kernel.Record{Seq: int64(b.n), EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", b.n), Type: t, SchemaVersion: 1,
		Kind: info.Kind, ItemID: "ENT01:F-017", Stream: "item:ENT01:F-017", OccurredAt: t0.Add(time.Duration(b.n) * time.Minute),
		ReceivedAt: t0.Add(time.Duration(b.n) * time.Minute), Data: raw}
	b.out = append(b.out, r)
	return r
}

func (b *builder) run(id, step string) kernel.Record {
	return b.add(catalog.OperationRunStarted, map[string]any{"operation_run_id": id, "operation_code": "OP", "step_key": step, "operator_id": "O17"})
}

// camera — результат камеры КТ-3 со ступенями анализатора.
func (b *builder) camera(outcome string, qualityBP, confBP int, defects ...map[string]any) kernel.Record {
	d := map[string]any{"method": "camera", "phase": "after_operation", "outcome": outcome, "processing_state": "completed",
		"step_key": "welding.kt3_camera", "inspection_point": "KT-3", "zone_ids": []string{"W-1"},
		"observation_quality_bp": qualityBP, "analyzer_confidence_bp": confBP,
		"stages": []map[string]any{{"stage": "localize", "version": "vqc-weld 2.3.1", "confidence_bp": confBP}, {"stage": "classify", "version": "vqc-weld 2.3.1", "confidence_bp": confBP}},
		"versions": map[string]any{"recipe_ref": "kt3-weld@1", "analyzer_version": "vqc-weld 2.3.1", "contract_version": "1.0"}}
	if len(defects) > 0 {
		d["defects"] = defects
	}
	return b.add(catalog.InspectionResultRecorded, d)
}

func (b *builder) xray(outcome string, defects ...map[string]any) kernel.Record {
	d := map[string]any{"method": "radiography", "phase": "after_operation", "outcome": outcome, "processing_state": "completed",
		"step_key": "welding.kt3_radiography", "inspection_point": "KT-3", "zone_ids": []string{"W-1"}, "conclusion_ref": "РК-17"}
	if len(defects) > 0 {
		d["defects"] = defects
	}
	return b.add(catalog.InspectionResultRecorded, d)
}

func defect(code, zone, loc, sev string, conf int) map[string]any {
	d := map[string]any{"zone_id": zone, "severity": sev}
	if code != "" {
		d["defect_type_code"] = code
	}
	if loc != "" {
		d["location"] = loc
	}
	if conf > 0 {
		d["stage_confidence_bp"] = conf
	}
	return d
}

// fold — свёртка модуля quality по входу (как Step движка, без поздних модулей).
func fold(env Env, in []kernel.Record) (State, []kernel.Output) {
	var s State
	outs := []kernel.Output{}
	for _, r := range in {
		s = Reduce(s, r, env, Upstream{})
		out := React(s, env, Upstream{})
		for _, it := range out.Intents {
			if it.Target == Module {
				s = Apply(s, it)
			}
		}
		outs = append(outs, out)
	}
	return s, outs
}

// reactionsOf — реакции типа t на последнем шаге.
func reactionsOf(outs []kernel.Output, t catalog.Type) []kernel.Reaction {
	res := []kernel.Reaction{}
	if len(outs) == 0 {
		return res
	}
	for _, r := range outs[len(outs)-1].Reactions {
		if r.Type == t {
			res = append(res, r)
		}
	}
	return res
}

func requireN[T any](t *testing.T, what string, got []T, n int) {
	t.Helper()
	if len(got) != n {
		t.Fatalf("%s: %d, ожидали %d: %+v", what, len(got), n, got)
	}
}
