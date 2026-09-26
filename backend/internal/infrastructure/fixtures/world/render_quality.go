package world

import (
	"slices"
	"strings"
	"time"

	qualityapp "ant/internal/application/quality"
	"ant/internal/infrastructure/fixtures/loader"
)

// Сигнал о признаке дефекта ≠ брак; уверенность анализатора ≠ вероятность брака;
// «оценка невозможна» ≠ «годно» (NFR-UI-4).

// SignalView — сигнал мира: из несоответствия или отдельный (S08).
type SignalView struct {
	ID, Item, Step, Zone, Kind, Source string
	At, Closed                         time.Time
	ClosedAs                           string // confirmed | rejected
	Confidence, Quality                int
	NC                                 *NC
}

// Signals — сигналы мира по времени.
func (m *Model) Signals() []SignalView {
	var out []SignalView
	for _, s := range m.Spec.Signals {
		if s.Unable {
			continue
		}
		out = append(out, SignalView{ID: s.ID, Item: s.Item, Step: "assembly.kt4_camera", Zone: s.Zone, Kind: s.Kind, Source: s.Source, At: s.At.Time(), Closed: s.Rejected.Time(), ClosedAs: "rejected", Confidence: s.ConfidenceBP, Quality: s.QualityBP})
	}
	for _, n := range m.NCs {
		if len(n.Spec.Items) > 0 || len(n.Items) == 0 {
			continue
		}
		sv := SignalView{ID: n.SignalID, Item: n.Items[0].ID, Step: n.StepKey, Source: n.Spec.Source, At: n.SignalAt, Closed: n.ConfirmedAt, ClosedAs: "confirmed", NC: n}
		if len(n.Spec.Defects) > 0 {
			sv.Zone, sv.Kind = n.Spec.Defects[0].Zone, n.Spec.Defects[0].Kind
		}
		if n.Spec.Source == "CAM-KT3" {
			sv.Confidence, sv.Quality = 8600, 9000
		}
		out = append(out, sv)
	}
	slices.SortStableFunc(out, func(a, b SignalView) int { return a.At.Compare(b.At) })
	return out
}

func (c *Ctx) signalView(s SignalView) qualityapp.QualitySignal {
	it := c.M.itemByID[s.Item]
	q := qualityapp.QualitySignal{SignalID: s.ID, ItemID: FullID(it.ID), ItemLabel: it.Label, StepKey: s.Step, BasisKind: "inspection_result",
		DefectTypeKnown: s.Kind != "", Severity: "major", ReactionMapRef: "reaction-map@1", State: "open", RaisedAt: s.At,
		Stages: []qualityapp.QualityAnalyzerStage{}, EvidenceRefs: []string{}, BasisSeq: c.ItemSeq(it)}
	if s.Zone != "" {
		q.ZoneID = ptr(s.Zone)
	}
	if s.Kind != "" {
		q.DefectTypeCode = ptr(s.Kind)
	}
	q.ReactionOutcome = "manual_review"
	if s.Kind == "burn_through" || s.Kind == "undercut" {
		q.ReactionOutcome = "isolate"
	}
	if s.Confidence > 0 {
		q.AnalyzerConfidenceBP, q.ObservationQualityBP, q.TrustLevel = ptr(s.Confidence), ptr(s.Quality), ptr(2)
		q.Stages = []qualityapp.QualityAnalyzerStage{
			{Stage: "где дефект", Version: "vqc-weld 2.3.1", ConfidenceBP: ptr(s.Confidence), OutputNote: "Место: " + zoneTitle(s.Zone)},
			{Stage: "какой тип", Version: "vqc-weld 2.3.1", ConfidenceBP: ptr(s.Confidence - 300), OutputNote: "Вид: " + defectTitle(s.Kind)},
		}
		q.Versions = map[string]string{"contract": "1.0", "analyzer": "vqc-weld 2.3.1", "item_revision": "Б", "recipe": "КТ-3 рецепт 3", "camera": s.Source}
	}
	if !s.Closed.After(c.T) {
		q.State = s.ClosedAs
	}
	for _, e := range c.M.Events {
		if e.Item == it && e.Type == "inspection.result.recorded" && e.Params["outcome"] == "defect_indicated" && !e.Occurred.After(s.At) && s.At.Sub(e.Occurred) < 5*time.Minute {
			q.ObservationEventID = ptr(e.ID)
		}
	}
	return q
}

func methodOf(e *Event) string {
	m := e.Params["method"]
	switch m {
	case "camera", "cmm", "radiography", "leak_test", "torque":
		return m
	}
	return "other"
}

// renderQuality — сигналы, результаты контроля, полнота, дефекты, пропуски, карта реакций (FR-35…FR-38, FR-48).
func renderQuality(c *Ctx) []loader.Response {
	var out []loader.Response
	list := qualityapp.QualitySignalList{Items: []qualityapp.QualitySignal{}}
	for _, s := range c.M.Signals() {
		if s.At.After(c.T) {
			continue
		}
		v := c.signalView(s)
		list.Items = append(list.Items, v)
		out = append(out, resp("quality.signal.read", v, "signal_id", s.ID))
	}
	out = append(out, resp("quality.signal.list", list))
	all := qualityapp.QualityDefectList{Items: []qualityapp.QualityDefect{}}
	withDef := map[string]bool{}
	for _, it := range c.Existing() {
		insp := qualityapp.InspectionResultList{Items: []qualityapp.InspectionResult{}}
		for _, e := range c.itemEvents(it) {
			if e.Type != "inspection.result.recorded" {
				continue
			}
			r := qualityapp.InspectionResult{EventID: e.ID, Seq: ptr(e.Seq), ItemID: FullID(it.ID), StepKey: e.StepKey, Method: methodOf(e), Outcome: e.Params["outcome"],
				ProcessingState: "completed", Defects: []qualityapp.InspectionDefect{}, SourceKind: sourceKindOf(e), Reliability: "high", OccurredAt: e.Occurred, EvidenceRefs: []string{}}
			if r.SourceKind == "" {
				r.SourceKind, r.Reliability = "camera", "medium"
			}
			if e.Kind == "reaction" {
				r.Limitations = []string{"Производный результат: качество наблюдения ниже порога рецепта 0,6"}
			}
			if v := e.Params["analyzer_confidence_bp"]; v != "" {
				r.AnalyzerConfidenceBP = ptr(atoi(v))
			}
			if v := e.Params["observation_quality_bp"]; v != "" {
				r.ObservationQualityBP = ptr(atoi(v))
			}
			if r.Outcome == "defect_indicated" {
				r.Defects = append(r.Defects, qualityapp.InspectionDefect{Description: e.Summary, Severity: "major"})
			}
			insp.Items = append(insp.Items, r)
		}
		out = append(out, resp("quality.inspection.list", insp, "item_id", FullID(it.ID)))
		out = append(out, resp("quality.coverage.read", c.coverage(it), "item_id", FullID(it.ID)))
		defs := qualityapp.QualityDefectList{Items: []qualityapp.QualityDefect{}}
		for _, n := range c.M.NCs {
			if len(n.Spec.Items) > 0 || !slices.Contains(n.Items, it) || n.SignalAt.After(c.T) {
				continue
			}
			for i, d := range n.Spec.Defects {
				obs := 1
				if n.ID == "NC-01" {
					obs = 2 // два ракурса — один дефект (S06)
				}
				fe := ""
				for _, e := range c.M.Events {
					if e.Item == it && e.Type == "inspection.result.recorded" && e.Params["outcome"] == "defect_indicated" && !e.Occurred.After(n.SignalAt) {
						fe = e.ID
					}
				}
				qd := qualityapp.QualityDefect{DefectID: n.ID + "-D" + string(rune('1'+i)), ItemID: FullID(it.ID), ZoneID: d.Zone, Location: zoneTitle(d.Zone), DefectTypeCode: ptr(d.Kind), FirstObservationEventID: fe, Observations: obs, IdentifiedAt: n.SignalAt}
				defs.Items = append(defs.Items, qd)
				all.Items = append(all.Items, qd)
				withDef[it.ID] = true
			}
		}
		defs.DefectCount, defs.ItemsWithDefect = len(defs.Items), min(1, len(defs.Items))
		out = append(out, resp("quality.defect.list", defs, "item_id", FullID(it.ID)))
	}
	all.DefectCount, all.ItemsWithDefect = len(all.Items), len(withDef)
	out = append(out, resp("quality.defect.list", all))
	esc := qualityapp.QualityEscapeList{Items: []qualityapp.QualityEscape{}}
	if n := c.M.NC("NC-04"); n != nil && !n.ConfirmedAt.After(c.T) {
		esc.Items = append(esc.Items, qualityapp.QualityEscape{EventID: c.M.eventID("escape/NC-04"), DefectID: "NC-04-D1", ItemID: FullID(n.Items[0].ID),
			MissedObservationEventIDs: []string{}, MethodCoversDefect: false, RecordedAt: n.ConfirmedAt})
	}
	out = append(out, resp("quality.escape.list", esc))
	if c.N == 0 {
		out = append(out, resp("quality.reaction_map.read", reactionMap()))
	}
	return out
}

func (c *Ctx) coverage(it *Item) qualityapp.InspectionCoverage {
	cov := qualityapp.InspectionCoverage{ItemID: FullID(it.ID), Points: []qualityapp.CoveragePoint{}, BasisSeq: c.ItemSeq(it)}
	events := c.itemEvents(it)
	st := c.S(it)
	order := []string{"machining", "welding", "assembly", "testing", "final"}
	stage := 0
	for i, p := range order {
		if strings.HasPrefix(st.Step, p) {
			stage = i
		}
	}
	if st.Position == "completed" {
		stage = len(order)
	}
	complete := true
	for _, p := range []struct{ step, point, method string }{
		{"machining.kt2_camera", "KT-2", "camera"}, {"machining.kt2_cmm", "KT-2", "cmm"}, {"welding.kt3_camera", "KT-3", "camera"},
		{"welding.kt3_radiography", "KT-3", "radiography"}, {"assembly.kt4_camera", "KT-4", "camera"}, {"testing.leak_test", "ZT-5", "leak_test"}, {"final.kt5_camera", "KT-5", "camera"},
	} {
		cp := qualityapp.CoveragePoint{StepKey: p.step, InspectionPoint: p.point, Method: p.method, Required: true, Status: "pending"}
		for _, e := range events {
			if e.Type == "inspection.result.recorded" && e.StepKey == p.step && e.Params["method"] == p.method {
				cp.Status, cp.EventID = "received", ptr(e.ID)
			}
		}
		idx := slices.Index(order, strings.SplitN(p.step, ".", 2)[0])
		if cp.Status == "pending" && idx < stage && p.step != "assembly.kt4_camera" {
			cp.Status, cp.MissingReason = "missing", ptr("result_not_received")
		}
		if cp.Status != "received" {
			complete = false
		}
		cov.Points = append(cov.Points, cp)
	}
	cov.Complete = complete
	return cov
}

func reactionMap() qualityapp.ReactionMap {
	return qualityapp.ReactionMap{Ref: "reaction-map@1", Version: "1", Rules: []qualityapp.ReactionRule{
		{RuleID: "R-07", Title: "Прожог или подрез на шве", Trigger: "Сигнал «прожог» или «подрез» в зоне шва W-1", DefectTypeCode: ptr("burn_through"), Severity: ptr("major"), Outcome: "isolate", AutomationMode: 3, ActionClass: "protective", Owner: "TEC-01", ApprovedBy: ptr("HQC-01"), ThresholdBP: ptr(6000)},
		{RuleID: "R-12", Title: "Параметр спецпроцесса вне уставки", Trigger: "Ток сварки вне 160 ± 10 А → сигнал и область риска по оборудованию", Outcome: "question_to_technologist", AutomationMode: 3, ActionClass: "protective", Owner: "TEC-01", ApprovedBy: ptr("HQC-01")},
		{RuleID: "R-15", Title: "Критичный признак сборки", Trigger: "Нет крепежа или метки затяжки → изоляция до решения", DefectTypeCode: ptr("missing_torque_mark"), Severity: ptr("major"), Outcome: "manual_review", AutomationMode: 2, ActionClass: "protective", Owner: "TEC-01", ThresholdBP: ptr(5000)},
		{RuleID: "R-21", Title: "Уплотнение вне склада дольше 4 ч", Trigger: "Уплотнение выдано и не установлено за 4 ч → блок к установке", Outcome: "manual_review", AutomationMode: 2, ActionClass: "protective", Owner: "TEC-01"},
	}}
}
