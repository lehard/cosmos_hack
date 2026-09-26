package quality

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/normative"
	"ant/internal/domain/quality"
)

// Отображение выводов домена в формы ответов (views.go). Смысл — только
// контрактный (NFR-UI-4): «оценка невозможна» ≠ «годно», уверенность ≠
// вероятность брака, сигнал ≠ брак.

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func ptr[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// itemLabel — метка изделия для списка: локальная часть идентификатора.
func itemLabel(itemID string) string {
	if _, local, ok := strings.Cut(itemID, ":"); ok {
		return local
	}
	return itemID
}

func observationOf(st quality.State, id string) (quality.Observation, bool) {
	for _, o := range st.Observations {
		if o.EventID == id {
			return o, true
		}
	}
	return quality.Observation{}, false
}

func stagesView(ss []quality.Stage) []QualityAnalyzerStage {
	out := []QualityAnalyzerStage{}
	for _, s := range ss {
		out = append(out, QualityAnalyzerStage{Stage: s.Stage, Version: s.Version, ConfidenceBP: s.ConfidenceBP, OutputNote: s.Note})
	}
	return out
}

func signalView(rec ItemRecord, sg quality.Signal) QualitySignal {
	a := sg.Assessment
	v := QualitySignal{
		SignalID: sg.SignalID, ItemID: rec.ItemID, ItemLabel: itemLabel(rec.ItemID), StepKey: sg.StepKey, BasisKind: sg.Basis,
		DefectID: ptr(sg.DefectID), ZoneID: ptr(sg.Zone), DefectTypeCode: ptr(sg.TypeCode), DefectTypeKnown: sg.TypeKnown,
		Severity: firstNonEmpty(sg.Severity, "unknown"), AnalyzerConfidenceBP: sg.ConfidenceBP, ObservationQualityBP: sg.QualityBP,
		ReactionOutcome: a.Outcome, ReactionMapRef: a.MapRef, TrustLevel: sg.TrustLevel, State: sg.State, RaisedAt: sg.RaisedAt,
		ObservationEventID: ptr(sg.ObservationID), Stages: []QualityAnalyzerStage{}, EvidenceRefs: []string{}, BasisSeq: rec.BasisSeq,
		UnableToAssess: sg.Unable, UnableReason: sg.UnableReason, RequirementRef: sg.RequirementRef, RuleTitle: a.RuleTitle,
		AutomationMode: a.Mode, ProposedOutcome: a.Proposed, Limits: a.Limits, Containment: string(a.Containment),
		ObservationIDs: sg.Observations,
	}
	if o, ok := observationOf(rec.State, sg.ObservationID); ok {
		v.Stages = stagesView(o.Stages)
		v.Versions = o.Versions
		if o.Evidence != nil {
			v.EvidenceRefs = o.Evidence
		}
	}
	return v
}

// sourceKinds — допустимые пометки источника (FR-140).
var sourceKinds = []string{"manual_entry", "machine", "sensor", "camera", "external_system", "import"}

func inspectionView(itemID string, o quality.Observation) InspectionResult {
	v := InspectionResult{
		EventID: o.EventID, ItemID: itemID, StepKey: o.StepKey, OperationRunID: ptr(o.OperationRunID), Method: o.Method,
		Outcome: o.Outcome, ProcessingState: o.ProcessingState, Defects: []InspectionDefect{}, AnalyzerConfidenceBP: o.ConfidenceBP,
		ObservationQualityBP: o.QualityBP, Limitations: o.Limitations, SourceKind: o.SourceKind, Reliability: "unknown",
		OccurredAt: o.OccurredAt, EvidenceRefs: []string{}, ReportedOutcome: o.Reported, UnableReason: o.UnableReason,
		Reinterpreted: o.Reinterpreted, Analyzer: o.Analyzer, TrustNote: o.TrustNote, Stages: stagesView(o.Stages), Superseded: o.Superseded,
	}
	if o.Seq > 0 {
		v.Seq = &o.Seq
	}
	if !slices.Contains(sourceKinds, v.SourceKind) {
		v.SourceKind = "external_system"
		if o.Analyzer {
			v.SourceKind = "camera"
		}
	}
	if o.Analyzer {
		lvl := o.TrustLevel
		v.TrustLevel = &lvl
	}
	if o.Evidence != nil {
		v.EvidenceRefs = o.Evidence
	}
	for _, d := range o.Defects {
		v.Defects = append(v.Defects, InspectionDefect{DefectTypeCode: ptr(d.TypeCode), Description: d.Description, ZoneID: ptr(d.Zone),
			Location: d.Location, Severity: firstNonEmpty(d.Severity, "unknown")})
	}
	for _, m := range o.Measurements {
		v.Measurements = append(v.Measurements, InspectionMeasurement{Characteristic: m.Characteristic, Value: m.Value, Tolerance: m.Tolerance,
			SourceVerdict: m.SourceVerdict, Verdict: m.Verdict})
	}
	return v
}

// outcomeClass — класс действия исхода карты реакций (AD-27).
func outcomeClass(o string) string {
	switch o {
	case quality.ReactPassToNext:
		return "permissive"
	case quality.ReactIsolate, quality.ReactManual:
		return "protective"
	}
	return "record"
}

// reactionMapView — карта реакций с режимом, классом действия, владельцем и
// условием по-русски (FR-48, FR-50).
func reactionMapView(env quality.Env) ReactionMap {
	m := env.ReactionMap
	out := ReactionMap{Ref: m.ID + "@" + strconv.Itoa(m.Version), Version: strconv.Itoa(m.Version), Rules: []ReactionRule{}}
	var until *time.Time
	if m.ValidUntil != nil {
		if t, err := time.Parse(time.RFC3339Nano, *m.ValidUntil); err == nil {
			until = &t
		}
	}
	rules := slices.Clone(m.Rules)
	slices.SortStableFunc(rules, func(a, b normative.ReactionMapRulesElem) int {
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		return strings.Compare(a.ID, b.ID)
	})
	for _, r := range rules {
		rr := ReactionRule{RuleID: r.ID, Title: r.Title, Trigger: trigger(r.Match), Outcome: string(r.Reaction.Outcome),
			AutomationMode: r.AutomationMode, ActionClass: outcomeClass(string(r.Reaction.Outcome)), Owner: m.OwnerRole,
			ApprovedBy: ptr(m.ApprovedBy), ValidUntil: until, ThresholdBP: r.Match.ConfidenceMinBp}
		if len(r.Match.DefectTypes) == 1 {
			rr.DefectTypeCode = &r.Match.DefectTypes[0]
		}
		if len(r.Match.Severity) == 1 {
			sv := string(r.Match.Severity[0])
			rr.Severity = &sv
		}
		out.Rules = append(out.Rules, rr)
	}
	return out
}

// trigger — условие правила по-русски.
func trigger(m normative.ReactionMapRulesElemMatch) string {
	parts := []string{}
	join := func(label string, vs []string) {
		if len(vs) > 0 {
			parts = append(parts, label+": "+strings.Join(vs, ", "))
		}
	}
	str := func(in any) []string {
		out := []string{}
		switch x := in.(type) {
		case []normative.ReactionMapRulesElemMatchOutcomeElem:
			for _, v := range x {
				out = append(out, string(v))
			}
		case []normative.ReactionMapRulesElemMatchSeverityElem:
			for _, v := range x {
				out = append(out, string(v))
			}
		case []normative.ReactionMapRulesElemMatchBasisKindElem:
			for _, v := range x {
				out = append(out, string(v))
			}
		case []normative.ReactionMapRulesElemMatchDeviationKindsElem:
			for _, v := range x {
				out = append(out, string(v))
			}
		case []string:
			out = x
		}
		return out
	}
	join("исход", str(m.Outcome))
	join("основание", str(m.BasisKind))
	join("вид", str(m.DefectTypes))
	join("тяжесть", str(m.Severity))
	join("отклонение", str(m.DeviationKinds))
	if m.ConfidenceMinBp != nil {
		parts = append(parts, "уверенность ≥ "+strconv.Itoa(*m.ConfidenceMinBp)+" б. п.")
	}
	if m.TrustLevelMin != nil {
		parts = append(parts, "уровень доверия паспорта ≥ "+strconv.Itoa(*m.TrustLevelMin))
	}
	if m.NoRequirement != nil && *m.NoRequirement {
		parts = append(parts, "нет требования КД")
	}
	if m.ObservationQualityBelowRecipe != nil && *m.ObservationQualityBelowRecipe {
		parts = append(parts, "качество наблюдения ниже порога карты контроля")
	}
	if m.SpecialProcess != nil && *m.SpecialProcess {
		parts = append(parts, "специальный процесс")
	}
	if len(parts) == 0 {
		return "всегда"
	}
	return strings.Join(parts, "; ")
}
