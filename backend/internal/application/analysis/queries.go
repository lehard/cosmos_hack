package analysis

import (
	"context"
	"slices"
	"strings"

	"ant/internal/application/platform"
	dom "ant/internal/domain/analysis"
)

// Чтение модуля analysis в режиме live (AD-36): проекции analysis.* и
// доменные функции разбора. Формулировки — «возможные обстоятельства», не
// «причина» (FR-153, NFR-UI-4). Момент as_of: версии области риска
// отбираются по времени записи; прочие ответы — по текущему состоянию
// проекций (пересвёртка на момент — через движок, AD-22, в работе).

// Circumstances — разбор обстоятельств (analysis.circumstances.read, FR-58, FR-153).
func (s *Service) Circumstances(ctx context.Context, ncID string, m platform.Moment) (Circumstances, error) {
	if !s.live() {
		return s.Unimplemented.Circumstances(ctx, ncID, m)
	}
	n, err := s.nc(ctx, ncID)
	if err != nil {
		return Circumstances{}, err
	}
	a, iv, err := s.analyze(ctx, n)
	if err != nil {
		return Circumstances{}, err
	}
	out := Circumstances{NCID: ncID, Records: []CircumstanceRecord{}, MissingInformation: append([]string{}, a.Missing...),
		ConclusionIsCategorical: a.Categorical, BasisSeq: max(iv.BasisSeq, n.Seq)}
	if r := a.Operation; r != nil {
		out.Operation = &OperationSpan{OperationRunID: r.RunID, Label: operationLabel(*r), StartedAt: r.Started, FinishedAt: r.Finished}
	}
	if w := a.Window; w != nil {
		out.Window = &CausalWindow{Start: w.Start, End: w.End, LowerBoundEventID: strp(w.LowerBound), UpperBoundEventID: strp(w.UpperBound)}
	}
	for _, mk := range a.Records {
		out.Records = append(out.Records, circumstanceRecord(mk, a.Records))
	}
	return out, nil
}

// Hypotheses — гипотезы по несоответствию (analysis.hypothesis.list, FR-59, FR-60):
// вывод системы («предложена системой») и записи людей; «подтверждена» —
// только решением человека incident.cause.concluded.
func (s *Service) Hypotheses(ctx context.Context, ncID string, m platform.Moment) (Hypotheses, error) {
	if !s.live() {
		return s.Unimplemented.Hypotheses(ctx, ncID, m)
	}
	n, err := s.nc(ctx, ncID)
	if err != nil {
		return Hypotheses{}, err
	}
	a, iv, err := s.analyze(ctx, n)
	if err != nil {
		return Hypotheses{}, err
	}
	similar, err := s.similar(ctx, ncID)
	if err != nil {
		return Hypotheses{}, err
	}
	out := Hypotheses{NCID: ncID, Version: n.Versions, Hypotheses: []Hypothesis{}, MissingInformation: append([]string{}, a.Missing...),
		ConclusionIsCategorical: a.Categorical, SimilarCases: similar, BasisSeq: max(iv.BasisSeq, n.Seq)}
	refs := refIndex(iv.State, a)
	rejected := map[string]*dom.ReasonRecord{}
	for _, h := range n.Hypotheses {
		if h.Verdict == "rejected" {
			rejected[h.HypothesisID] = h.Reason
		}
	}
	status := func(id, category, def string) string {
		if c := n.Cause; c != nil && c.Conclusion == "confirmed" && c.Category == category {
			return "confirmed"
		}
		if _, ok := rejected[id]; ok {
			return "rejected"
		}
		return def
	}
	branch := "why_made"
	for _, h := range a.Hypotheses {
		x := Hypothesis{HypothesisID: h.ID, Category: h.Category, Branch: &branch, Statement: strp(h.Statement),
			Status: status(h.ID, h.Category, "proposed_by_system"), ConfidenceBP: h.ConfidenceBP,
			Supporting: refs.of(h.Supporting), Contradicting: refs.of(h.Contradicting), MeasurementHint: strp(h.MeasurementHint)}
		out.Hypotheses = append(out.Hypotheses, x)
	}
	for _, h := range n.Hypotheses {
		if h.Verdict == "rejected" {
			continue
		}
		b := h.Branch
		x := Hypothesis{HypothesisID: h.HypothesisID, Category: h.Category, Branch: strp(b), Statement: strp(h.Statement),
			Status: status(h.HypothesisID, h.Category, "recorded"), Supporting: refs.of(h.Supporting), Contradicting: []JournalRecordRef{}}
		out.Hypotheses = append(out.Hypotheses, x)
	}
	return out, nil
}

// Similar — похожие случаи (analysis.similar.list, FR-60).
func (s *Service) Similar(ctx context.Context, ncID string, m platform.Moment) (SimilarCaseList, error) {
	if !s.live() {
		return s.Unimplemented.Similar(ctx, ncID, m)
	}
	if _, err := s.nc(ctx, ncID); err != nil {
		return SimilarCaseList{}, err
	}
	items, err := s.similar(ctx, ncID)
	return SimilarCaseList{Items: items}, err
}

func (s *Service) similar(ctx context.Context, ncID string) ([]SimilarCase, error) {
	ps, ncs, err := s.profiles(ctx)
	if err != nil {
		return nil, err
	}
	out := []SimilarCase{}
	i := slices.IndexFunc(ps, func(p dom.Profile) bool { return p.NCID == ncID })
	if i < 0 {
		return out, nil
	}
	incidents := map[string]dom.IncidentRecord{}
	for _, sc := range dom.Similar(ps[i], ps) {
		n := ncs[sc.NCID]
		x := SimilarCase{NCID: sc.NCID, Number: sc.NCID}
		if c := n.Cause; c != nil {
			cat := c.Category
			if c.Conclusion == "not_established" {
				cat = dom.CatNotEstablished
			}
			x.CauseCategory, x.CauseConfirmed = strp(cat), c.Conclusion == "confirmed"
			inc, ok := incidents[c.IncidentID]
			if !ok {
				inc, _ = s.incident(ctx, c.IncidentID)
				incidents[c.IncidentID] = inc
			}
			if len(inc.Actions) > 0 {
				act := inc.Actions[len(inc.Actions)-1]
				x.Measure, x.Result = strp(measureLabel(act)), strp(act.Status)
			}
		}
		out = append(out, x)
	}
	return out, nil
}

// Groups — группы несоответствий «вид дефекта × операция × оборудование».
func (s *Service) Groups(ctx context.Context, m platform.Moment) (NcGroupList, error) {
	if !s.live() {
		return s.Unimplemented.Groups(ctx, m)
	}
	ps, ncs, err := s.profiles(ctx)
	if err != nil {
		return NcGroupList{}, err
	}
	out := NcGroupList{Items: []NcGroup{}}
	var keys []string
	by := map[string][]dom.Profile{}
	for _, p := range ps {
		k := dom.GroupKey(p)
		if _, ok := by[k]; !ok {
			keys = append(keys, k)
		}
		by[k] = append(by[k], p)
	}
	for _, k := range keys {
		g := by[k]
		defect, op, eq := dom.SplitGroupKey(k)
		row := NcGroup{GroupKey: k, DefectType: defect, Operation: op, Equipment: eq, NCCount: len(g), LastFoundAt: g[0].At}
		for _, p := range g {
			if p.At.After(row.LastFoundAt) {
				row.LastFoundAt = p.At
			}
		}
		row.Investigation, err = s.investigation(ctx, g, ncs)
		if err != nil {
			return NcGroupList{}, err
		}
		out.Items = append(out.Items, row)
	}
	return out, nil
}

// investigation — статус расследования группы: от мер к гипотезам.
func (s *Service) investigation(ctx context.Context, g []dom.Profile, ncs map[string]dom.NCRecord) (string, error) {
	rank := map[string]int{"hypothesis_only": 1, "in_progress": 2, "cause_not_established": 3, "cause_confirmed": 4,
		"measures_assigned": 5, "effectiveness_check": 6}
	best := "hypothesis_only"
	set := func(st string) {
		if rank[st] > rank[best] {
			best = st
		}
	}
	for _, p := range g {
		n := ncs[p.NCID]
		if len(n.Hypotheses) > 0 {
			set("in_progress")
		}
		c := n.Cause
		if c == nil {
			continue
		}
		if c.Conclusion == "confirmed" {
			set("cause_confirmed")
		} else {
			set("cause_not_established")
		}
		inc, err := s.incident(ctx, c.IncidentID)
		if err != nil {
			continue
		}
		for _, a := range inc.Actions {
			if a.Status == "assigned" {
				set("measures_assigned")
			} else {
				set("effectiveness_check")
			}
		}
	}
	return best, nil
}

// CommonFactors — общие факторы группы (analysis.common_factors.read, FR-135).
func (s *Service) CommonFactors(ctx context.Context, groupKey string, m platform.Moment) (CommonFactors, error) {
	if !s.live() {
		return s.Unimplemented.CommonFactors(ctx, groupKey, m)
	}
	ps, _, err := s.profiles(ctx)
	if err != nil {
		return CommonFactors{}, err
	}
	var g []dom.Profile
	for _, p := range ps {
		if dom.GroupKey(p) == groupKey {
			g = append(g, p)
		}
	}
	if len(g) == 0 {
		return CommonFactors{}, notFound("Группа несоответствий", groupKey)
	}
	defect, op, eq := dom.SplitGroupKey(groupKey)
	out := CommonFactors{GroupKey: groupKey, GroupLabel: defect + " × " + op + " × " + eq, NCCount: len(g), Rows: []CommonFactorRow{}}
	for _, r := range dom.CommonFactors(g) {
		out.Rows = append(out.Rows, CommonFactorRow{Factor: r.Factor, Value: strp(r.Value), Matches: r.Matches, DistinctValues: r.Distinct})
	}
	return out, nil
}

// Incidents — инциденты (analysis.incident.list): все, по порядку открытия.
func (s *Service) Incidents(ctx context.Context, m platform.Moment, p platform.Page) (IncidentList, error) {
	if !s.live() {
		return s.Unimplemented.Incidents(ctx, m, p)
	}
	ids, err := s.list(ctx, ProjectionIncident)
	if err != nil {
		return IncidentList{}, err
	}
	out := IncidentList{Items: []IncidentSummary{}}
	for _, id := range ids {
		v, err := s.incident(ctx, id)
		if err != nil {
			continue
		}
		if m.AsOf != nil && v.OpenedAt.After(*m.AsOf) {
			continue
		}
		versions := versionsAt(v, m)
		x := IncidentSummary{IncidentID: id, Label: v.Label, CommonFactor: factorRef(v), InitialSize: v.InitialSize, Status: "open", OpenedAt: v.OpenedAt}
		if n := len(versions); n > 0 {
			x.Size, x.ScopeVersion = versions[n-1].Size, versions[n-1].Version
		}
		if v.Closed && (m.AsOf == nil || v.ClosedAt == nil || !v.ClosedAt.After(*m.AsOf)) {
			x.Status = "closed"
		}
		out.Items = append(out.Items, x)
	}
	return out, nil
}

// RiskScope — область риска с версиями (analysis.risk_scope.read, FR-61, FR-62):
// «тающая» область, у каждой версии — основание, автор, время, доказательства.
func (s *Service) RiskScope(ctx context.Context, incidentID string, m platform.Moment) (RiskScope, error) {
	if !s.live() {
		return s.Unimplemented.RiskScope(ctx, incidentID, m)
	}
	v, err := s.incident(ctx, incidentID)
	if err != nil {
		return RiskScope{}, err
	}
	out := RiskScope{IncidentID: incidentID, IncidentLabel: v.Label, CommonFactor: factorRef(v), Versions: []ScopeVersion{}, Items: []ScopeItem{}, BasisSeq: v.BasisSeq}
	if v.WindowStart != nil && v.WindowEnd != nil {
		out.Window = &TimeWindow{Start: *v.WindowStart, End: *v.WindowEnd}
		if v.KnownGoodItem != "" {
			out.LastKnownGood = &KnownGood{Label: dom.LocalID(v.KnownGoodItem) + " — последняя подтверждённо годная деталь", At: *v.WindowStart}
		}
	}
	for _, x := range versionsAt(v, m) {
		sv := ScopeVersion{ScopeVersion: x.Version, Change: x.Change, Size: x.Size, RecordedAt: x.RecordedAt, Author: strp(x.Author),
			EvidenceEventIDs: append([]string{}, x.Evidence...), Breakdown: ScopeBreakdown(x.Breakdown)}
		if x.Reason != nil {
			sv.Reason = &Reason{Code: strp(x.Reason.Code), Text: x.Reason.Text}
		}
		out.Versions = append(out.Versions, sv)
	}
	current := 0
	if n := len(out.Versions); n > 0 {
		current = out.Versions[n-1].ScopeVersion
	}
	ids := make([]string, 0, len(v.Members))
	for id := range v.Members {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		mem := v.Members[id]
		if mem.Version > current && m.AsOf != nil {
			continue
		}
		iv, err := s.item(ctx, id)
		if err != nil {
			return RiskScope{}, err
		}
		out.Items = append(out.Items, ScopeItem{ItemID: id, Label: dom.LocalID(id), Known: mem.Status, Action: mem.Action,
			Location: dom.Location(iv.State, v.StepKey)})
	}
	return out, nil
}

// versionsAt — версии области на момент: «что мы знали» — по времени записи версии.
func versionsAt(v dom.IncidentRecord, m platform.Moment) []dom.VersionRecord {
	if m.AsOf == nil {
		return v.Versions
	}
	var out []dom.VersionRecord
	for _, x := range v.Versions {
		if !x.RecordedAt.After(*m.AsOf) {
			out = append(out, x)
		}
	}
	return out
}

func factorRef(v dom.IncidentRecord) *FactorRef {
	if v.Factor == "" || v.FactorValue == "" {
		return nil
	}
	return &FactorRef{Factor: v.Factor, Value: v.FactorValue}
}

// operationLabel — подпись выполнения: шаг процесса и оборудование.
func operationLabel(r dom.Run) string {
	parts := []string{r.StepKey}
	if r.Equipment != "" {
		parts = append(parts, r.Equipment)
	}
	return strings.Join(parts, " · ")
}

func measureLabel(a dom.ActionRecord) string {
	kind := map[string]string{"correction": "Коррекция", "corrective_action": "Корректирующее действие", "preventive_action": "Предупреждающее действие"}[a.ActionType]
	if kind == "" {
		kind = "Мера"
	}
	if a.Owner != "" {
		kind += " (владелец " + a.Owner + ")"
	}
	return kind
}

// circumstanceRecord — строка дорожки разбора с переходом к записи журнала и
// связанными записями (то же выполнение операции).
func circumstanceRecord(mk dom.Mark, all []dom.Mark) CircumstanceRecord {
	r := CircumstanceRecord{JournalRecordRef: markRef(mk), Lane: mk.Lane, EndedAt: mk.EndedAt, EvidenceRefs: mk.Evidence, SourceKind: strp(mk.SourceKind)}
	if mk.Seq > 0 {
		seq := mk.Seq
		r.JournalSeq = &seq
	}
	if mk.RunID != "" {
		for _, o := range all {
			if o.EventID != mk.EventID && o.RunID == mk.RunID {
				r.RelatedEventIDs = append(r.RelatedEventIDs, o.EventID)
			}
		}
	}
	return r
}

func markRef(mk dom.Mark) JournalRecordRef {
	return JournalRecordRef{EventID: mk.EventID, EventType: mk.EventType, Variant: strp(mk.Variant), OccurredAt: mk.OccurredAt, Params: mk.Params}
}

// refs — ссылки на записи журнала для доводов гипотез.
type refs map[string]JournalRecordRef

func refIndex(st dom.State, a dom.Analysis) refs {
	out := refs{}
	for _, mk := range st.Marks {
		out[mk.EventID] = markRef(mk)
	}
	for _, e := range st.Equipment {
		out[e.EventID] = JournalRecordRef{EventID: e.EventID, EventType: e.EventType, Variant: strp(e.Variant), OccurredAt: e.OccurredAt, Params: e.Params}
	}
	for _, mk := range a.Records {
		out[mk.EventID] = markRef(mk)
	}
	for _, c := range st.Cases {
		out[c.EventID] = JournalRecordRef{EventID: c.EventID, EventType: "decision.nonconformity.confirmed", OccurredAt: c.At}
	}
	return out
}

func (r refs) of(ids []string) []JournalRecordRef {
	out := []JournalRecordRef{}
	for _, id := range ids {
		if x, ok := r[id]; ok {
			out = append(out, x)
		}
	}
	return out
}

func strp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
