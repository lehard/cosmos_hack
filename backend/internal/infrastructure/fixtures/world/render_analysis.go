package world

import (
	"fmt"
	"strings"
	"time"

	analysisapp "ant/internal/application/analysis"
	"ant/internal/infrastructure/fixtures/loader"
)

// «Возможные обстоятельства», не «причина»; уверенность ≠ вероятность вины;
// «под подозрением» ≠ брак (NFR-UI-4, FR-58…FR-62).

func jref(e *Event) analysisapp.JournalRecordRef {
	r := analysisapp.JournalRecordRef{EventID: e.ID, EventType: e.Type, OccurredAt: e.Occurred}
	if v := e.Params["outcome"]; v != "" {
		r.Variant = ptr(v)
	}
	if len(e.Params) > 0 {
		r.Params = e.Params
	}
	return r
}

var catOf = map[string]string{"equipment": "equipment", "incoming": "incoming"}

// renderAnalysis — разбор обстоятельств, гипотезы, похожие, группы, общие факторы, инциденты, области риска.
func renderAnalysis(c *Ctx) []loader.Response {
	var out []loader.Response
	for _, n := range c.M.NCs {
		if n.SignalAt.After(c.T) {
			continue
		}
		out = append(out,
			resp("analysis.circumstances.read", c.circumstances(n), "nc_id", n.ID),
			resp("analysis.hypothesis.list", c.hypotheses(n), "nc_id", n.ID),
			resp("analysis.similar.list", analysisapp.SimilarCaseList{Items: c.similar(n)}, "nc_id", n.ID))
	}
	groups := analysisapp.NcGroupList{Items: []analysisapp.NcGroup{}}
	for _, g := range c.groups() {
		groups.Items = append(groups.Items, g.row)
		out = append(out, resp("analysis.common_factors.read", g.factors, "group_key", g.row.GroupKey))
	}
	out = append(out, resp("analysis.group.list", groups))
	incs := analysisapp.IncidentList{Items: []analysisapp.IncidentSummary{}}
	for _, in := range c.M.Incidents {
		v := in.VersionAt(c.T)
		if v == nil {
			continue
		}
		st := "open"
		if !in.Spec.Closed.IsZero() && !in.Spec.Closed.Time().After(c.T) {
			st = "closed"
		}
		incs.Items = append(incs.Items, analysisapp.IncidentSummary{IncidentID: in.Spec.ID, Label: in.Spec.Label, CommonFactor: factorRef(in), Size: v.Size(), InitialSize: in.Versions[0].Size(), ScopeVersion: v.Spec.V, Status: st, OpenedAt: in.Spec.Opened.Time()})
		out = append(out, resp("analysis.risk_scope.read", c.riskScope(in, v), "incident_id", in.Spec.ID))
	}
	out = append(out, resp("analysis.incident.list", incs))
	return out
}

func factorRef(in *Incident) *analysisapp.FactorRef {
	switch {
	case strings.HasPrefix(in.Spec.Factors[len(in.Spec.Factors)-1], "LOT-") && len(in.Spec.Factors) == 1:
		return &analysisapp.FactorRef{Factor: "material_batch", Value: in.Spec.Factors[0]}
	default:
		for _, f := range in.Spec.Factors {
			if strings.HasPrefix(f, "IS-") {
				return &analysisapp.FactorRef{Factor: "machine", Value: f}
			}
		}
	}
	return nil
}

func (c *Ctx) circumstances(n *NC) analysisapp.Circumstances {
	ci := analysisapp.Circumstances{NCID: n.ID, Records: []analysisapp.CircumstanceRecord{}, MissingInformation: []string{}, BasisSeq: c.EntitySeq(n.ID, n.Items...)}
	if len(n.Items) == 0 || len(n.Spec.Items) > 0 {
		ci.MissingInformation = append(ci.MissingInformation, "other")
		return ci
	}
	it := n.Items[0]
	var run *OpRun
	for _, r := range it.Runs {
		if r.Kind == "welding" && !r.From.After(n.SignalAt) {
			run = r
		}
	}
	if run != nil {
		ci.Operation = &analysisapp.OperationSpan{OperationRunID: run.ID, Label: run.Label + " — сварка фланца с кольцом", StartedAt: run.From, FinishedAt: tptr(run.To)}
	}
	var lastClean, first *Event
	for _, e := range c.itemEvents(it) {
		lane := "item"
		switch {
		case strings.HasPrefix(e.Type, "equipment."):
			lane = "equipment"
		case strings.HasPrefix(e.Type, "operation.run") || strings.HasPrefix(e.Type, "operator."):
			lane = "person"
		case e.Type == "inspection.result.recorded", strings.HasPrefix(e.Type, "operation.movement"):
		default:
			continue
		}
		rec := analysisapp.CircumstanceRecord{JournalRecordRef: jref(e), Lane: lane, JournalSeq: ptr(e.Seq)}
		if sk := sourceKindOf(e); sk != "" {
			rec.SourceKind = ptr(sk)
		}
		ci.Records = append(ci.Records, rec)
		if e.Type == "inspection.result.recorded" {
			switch {
			case e.Params["outcome"] == "no_defect_indicated" && !e.Occurred.After(n.SignalAt) && (first == nil):
				lastClean = e
			case e.Params["outcome"] == "defect_indicated" && first == nil && !e.Occurred.After(n.SignalAt):
				first = e
			}
		}
	}
	if n.ID == "NC-01" {
		ci.Records = append(ci.Records, analysisapp.CircumstanceRecord{JournalRecordRef: analysisapp.JournalRecordRef{EventID: c.M.eventID("override/F-017"), EventType: "operator.override.performed",
			Variant: ptr("manual_override"), OccurredAt: c.M.clk.at(23, 10, 50), Params: map[string]string{"note": "деталь переставлена в приспособлении"}}, Lane: "person", SourceKind: ptr("manual_entry")})
	}
	if lastClean != nil && first != nil {
		ci.Window = &analysisapp.CausalWindow{Start: lastClean.Occurred, End: first.Occurred, LowerBoundEventID: ptr(lastClean.ID), UpperBoundEventID: ptr(first.ID)}
	}
	if run != nil && (run.LogLost || run.LogReceived.After(c.T)) {
		ci.MissingInformation = append(ci.MissingInformation, "equipment_log_missing")
	}
	if h := c.hypAt(n); h != nil {
		ci.ConclusionIsCategorical = h.Categorical
	}
	return ci
}

func (c *Ctx) hypotheses(n *NC) analysisapp.Hypotheses {
	hs := analysisapp.Hypotheses{NCID: n.ID, Hypotheses: []analysisapp.Hypothesis{}, MissingInformation: []string{}, SimilarCases: c.similar(n), BasisSeq: c.EntitySeq(n.ID, n.Items...)}
	h := c.hypAt(n)
	if h == nil {
		return hs
	}
	hs.Version, hs.ConclusionIsCategorical = h.V, h.Categorical
	hs.MissingInformation = append(hs.MissingInformation, h.Missing...)
	sup := func(pred func(*Event) bool) []analysisapp.JournalRecordRef {
		out := []analysisapp.JournalRecordRef{}
		for _, e := range c.Visible() {
			if pred(e) && len(out) < 6 {
				out = append(out, jref(e))
			}
		}
		return out
	}
	add := func(cat, strength, stmt, hint string, sp, con []analysisapp.JournalRecordRef) {
		if strength == "" {
			return
		}
		status := "proposed_by_system"
		conf := map[string]int{"not_assessable": 0, "weak": 2000, "possible": 5000, "strong": 8500, "confirmed": 9500}[strength]
		switch strength {
		case "confirmed":
			status = "confirmed"
		case "weak":
			if h.Categorical {
				status = "rejected"
			}
		}
		x := analysisapp.Hypothesis{HypothesisID: fmt.Sprintf("HYP-%s-%s", n.ID, cat), Category: cat, Branch: ptr("why_made"), Statement: ptr(stmt), Status: status, Supporting: sp, Contradicting: con}
		if strength != "not_assessable" {
			x.ConfidenceBP = ptr(conf)
		}
		if hint != "" {
			x.MeasurementHint = ptr(hint)
		}
		hs.Hypotheses = append(hs.Hypotheses, x)
	}
	isDev := func(e *Event) bool {
		return e.Type == "equipment.deviation.detected" || e.Type == "equipment.cycle.summarized" && strings.HasPrefix(e.Params["value"], "17")
	}
	add("equipment", h.Equipment, "Оборудование: режим ИС-2 — ток сварки выше уставки 160 ± 10 А", "Контрольный образец на ИС-2 при уставке 160 А", sup(isDev), []analysisapp.JournalRecordRef{})
	add("performer", h.Performer, "Отклонение от процедуры: ручная перестановка детали рядом по времени со сваркой У2 (обстоятельство, не вина)", "Письменное объяснение работника — только после подтверждения причины",
		[]analysisapp.JournalRecordRef{}, sup(func(e *Event) bool {
			return e.Type == "decision.nonconformity.confirmed" && (e.Entity.ID == "NC-02" || e.Entity.ID == "NC-03")
		}))
	add("incoming", h.Incoming, "Входной брак: пора в теле кольца партии П-117, зону не затрагивала ни одна операция", "Выборочный рентген колец партии со склада",
		sup(func(e *Event) bool {
			return e.Type == "inspection.result.recorded" && strings.Contains(e.Summary, "теле кольца")
		}), []analysisapp.JournalRecordRef{})
	return hs
}

func (c *Ctx) similar(n *NC) []analysisapp.SimilarCase {
	out := []analysisapp.SimilarCase{}
	if len(n.Spec.Defects) == 0 {
		return out
	}
	for _, o := range c.M.NCs {
		if o == n || len(o.Spec.Defects) == 0 || o.Spec.Defects[0].Kind != n.Spec.Defects[0].Kind || o.ConfirmedAt.After(c.T) {
			continue
		}
		sc := analysisapp.SimilarCase{NCID: o.ID, Number: o.Number}
		if o.Spec.Cause != nil && !o.Spec.Cause.At.Time().After(c.T) {
			sc.CauseCategory, sc.CauseConfirmed = ptr(catOf[o.Spec.Cause.Category]), true
			sc.Measure, sc.Result = ptr("Ремонт регулятора тока ИС-2 и проверка контрольным образцом"), ptr("assigned")
		}
		out = append(out, sc)
	}
	return out
}

type ncGroup struct {
	row     analysisapp.NcGroup
	factors analysisapp.CommonFactors
}

// groups — группы несоответствий «вид дефекта × операция × оборудование» и общие факторы (FR-135).
func (c *Ctx) groups() []ncGroup {
	type acc struct {
		key, defect, op, eq string
		ncs                 []*NC
	}
	var order []string
	by := map[string]*acc{}
	for _, n := range c.M.NCs {
		if len(n.Spec.Items) > 0 || len(n.Spec.Defects) == 0 || n.ConfirmedAt.After(c.T) {
			continue
		}
		d := n.Spec.Defects[0].Kind
		op, eq := "welding.weld", ""
		if len(n.Items) > 0 {
			if w := c.weldBefore(n.Items[0], n.SignalAt); w != nil {
				eq = w.Equipment
			}
		}
		if n.Spec.Cause != nil && n.Spec.Cause.Category == "incoming" || n.Spec.Component != "" {
			op, eq = "incoming.zt1_lot_acceptance", "LOT-R-117"
		}
		k := d + "|" + op + "|" + eq
		if by[k] == nil {
			by[k] = &acc{key: k, defect: d, op: op, eq: eq}
			order = append(order, k)
		}
		by[k].ncs = append(by[k].ncs, n)
	}
	var out []ncGroup
	for _, k := range order {
		a := by[k]
		inv := "hypothesis_only"
		last := a.ncs[0].SignalAt
		allCause := true
		for _, n := range a.ncs {
			if n.SignalAt.After(last) {
				last = n.SignalAt
			}
			if n.Spec.Cause == nil || n.Spec.Cause.At.Time().After(c.T) {
				allCause = false
			}
		}
		if allCause {
			inv = "cause_confirmed"
		}
		g := ncGroup{row: analysisapp.NcGroup{GroupKey: strings.ReplaceAll(k, "|", "."), DefectType: a.defect, Operation: a.op, Equipment: a.eq, NCCount: len(a.ncs), NCIDs: []string{}, Investigation: inv, LastFoundAt: last}}
		for _, n := range a.ncs {
			g.row.NCIDs = append(g.row.NCIDs, n.ID)
		}
		g.factors = analysisapp.CommonFactors{GroupKey: g.row.GroupKey, GroupLabel: fmt.Sprintf("%s × %s × %s", defectTitle(a.defect), a.op, a.eq), NCCount: len(a.ncs), Rows: c.factorRows(a.ncs)}
		out = append(out, g)
	}
	return out
}

// weldBefore — последнее выполнение сварки изделия, начатое не позже t.
func (c *Ctx) weldBefore(it *Item, t time.Time) *OpRun {
	var out *OpRun
	for _, r := range it.Runs {
		if r.Kind == "welding" && !r.From.After(t) {
			out = r
		}
	}
	return out
}

func (c *Ctx) factorRows(ncs []*NC) []analysisapp.CommonFactorRow {
	vals := map[string]map[string]int{}
	add := func(f, v string) {
		if vals[f] == nil {
			vals[f] = map[string]int{}
		}
		vals[f][v]++
	}
	for _, n := range ncs {
		var w *OpRun
		if len(n.Items) > 0 {
			for _, r := range n.Items[0].Runs {
				if r.Kind == "welding" && !r.From.After(n.SignalAt) {
					w = r
				}
			}
		}
		if w == nil {
			add("material_batch", "LOT-R-117")
			add("machine", "")
			add("performer", "")
			continue
		}
		add("machine", w.Equipment)
		add("program", w.Program)
		add("performer", w.Performer)
		add("material_batch", "LOT-W-88")
		add("tool", "")
		add("fixture", "")
	}
	rows := []analysisapp.CommonFactorRow{}
	for _, f := range []string{"machine", "tool", "fixture", "program", "performer", "material_batch"} {
		m := vals[f]
		if m == nil {
			continue
		}
		best, bn := "", 0
		for _, v := range sortedKeys(m) {
			if m[v] > bn {
				best, bn = v, m[v]
			}
		}
		row := analysisapp.CommonFactorRow{Factor: f, Matches: bn, DistinctValues: len(m)}
		if best != "" {
			row.Value = ptr(best)
		} else {
			row.Matches = 0
		}
		rows = append(rows, row)
	}
	return rows
}

// riskScope — область риска с версиями (FR-61, FR-62): «тающая» область 34 → 13 → 6.
func (c *Ctx) riskScope(in *Incident, v *ScopeState) analysisapp.RiskScope {
	rs := analysisapp.RiskScope{IncidentID: in.Spec.ID, IncidentLabel: in.Spec.Label, CommonFactor: factorRef(in), Versions: []analysisapp.ScopeVersion{}, Items: []analysisapp.ScopeItem{}, BasisSeq: c.EntitySeq(in.Spec.ID)}
	if c.M.anchor != nil && in.Spec.ID == "RS-01" {
		w := firstWeld(c.M.anchor)
		rs.LastKnownGood = &analysisapp.KnownGood{Label: c.M.anchor.Label + " — последняя подтверждённо годная сварка", At: w.To}
		rs.Window = &analysisapp.TimeWindow{Start: w.To, End: in.Spec.Opened.Time()}
	}
	for i := range in.Versions {
		sv := &in.Versions[i]
		if sv.Spec.At.Time().After(c.T) {
			break
		}
		x := analysisapp.ScopeVersion{ScopeVersion: sv.Spec.V, Change: "narrowed", Size: sv.Size(), RecordedAt: sv.Spec.At.Time(), EvidenceEventIDs: []string{}, Reason: &analysisapp.Reason{Text: sv.Spec.Basis}}
		if i == 0 {
			x.Change = "computed"
		} else {
			x.Author = ptr(sv.Spec.By)
		}
		if sv.Spec.LateEvent != "" {
			x.Reason.Code = ptr("late_event:" + sv.Spec.LateEvent)
		}
		for _, e := range c.M.Events {
			if e.Entity.ID == in.Spec.ID && e.Params["scope_version"] == fmt.Sprint(sv.Spec.V) {
				x.EvidenceEventIDs = append(x.EvidenceEventIDs, e.ID)
			}
		}
		x.Breakdown = c.breakdown(sv, sv.Spec.At.Time())
		rs.Versions = append(rs.Versions, x)
	}
	for _, id := range sortedKeys(v.Status) {
		it := c.M.itemByID[id]
		known := v.Status[id]
		act := map[string]string{"confirmed": "block", "suspect": "check", "unknown": "check", "excluded": "release"}[known]
		rs.Items = append(rs.Items, analysisapp.ScopeItem{ItemID: FullID(id), Label: it.Label, Known: known, Action: act, Location: c.locationClass(it, c.T)})
	}
	return rs
}

func (c *Ctx) locationClass(it *Item, t time.Time) string {
	st := it.State(t)
	switch {
	case st.Position == "completed":
		return "shipped"
	case strings.HasPrefix(st.Step, "assembly") || strings.HasPrefix(st.Step, "testing") || strings.HasPrefix(st.Step, "final"):
		return "assembled"
	case st.Location == "WH-WC":
		return "moved_on"
	}
	return "in_production"
}

func (c *Ctx) breakdown(sv *ScopeState, at time.Time) analysisapp.ScopeBreakdown {
	var b analysisapp.ScopeBreakdown
	for id, s := range sv.Status {
		if s == "excluded" {
			continue
		}
		switch c.locationClass(c.M.itemByID[id], at) {
		case "shipped":
			b.Shipped++
		case "assembled":
			b.Assembled++
		case "moved_on":
			b.MovedOn++
		default:
			b.InProduction++
		}
	}
	return b
}
