package analysis

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
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
		ConclusionIsCategorical: a.Categorical, BasisSeq: max(iv.BasisSeq, n.Seq, s.incidentBasis(ctx, n))}
	if r := a.Operation; r != nil {
		out.Operation = &OperationSpan{OperationRunID: r.RunID, Label: operationLabel(*r), StartedAt: r.Started, FinishedAt: r.Finished}
	}
	if w := a.Window; w != nil {
		out.Window = &CausalWindow{Start: w.Start, End: w.End, LowerBoundEventID: strp(w.LowerBound), UpperBoundEventID: strp(w.UpperBound)}
	}
	for _, mk := range a.Records {
		out.Records = append(out.Records, circumstanceRecord(mk, a.Records))
	}
	out.Lanes = laneQualities(a)
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
		ConclusionIsCategorical: a.Categorical, SimilarCases: similar, BasisSeq: max(iv.BasisSeq, n.Seq, s.incidentBasis(ctx, n))}
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
	var inc *dom.IncidentRecord
	for _, id := range n.IncidentIDs {
		if v, err := s.incident(ctx, id); err == nil {
			inc = &v
			break
		}
	}
	branch := "why_made"
	for _, h := range a.Hypotheses {
		x := Hypothesis{HypothesisID: h.ID, Category: h.Category, Branch: &branch, Statement: strp(h.Statement),
			Status: status(h.ID, h.Category, "proposed_by_system"), ConfidenceBP: h.ConfidenceBP,
			Supporting: refs.of(h.Supporting), Contradicting: refs.of(h.Contradicting), MeasurementHint: strp(h.MeasurementHint),
			History: hypothesisHistory(n, h.ID, h.Category), NextCheck: nextCheck(h, inc)}
		// Результат измерения: уверенность пересчитана, проверка выполнена.
		if r := measured(n, h.ID); r != nil {
			prev := 0
			if h.ConfidenceBP != nil {
				prev = *h.ConfidenceBP
			}
			if bp, ok := resultConfidence(r.Outcome, prev); ok {
				x.ConfidenceBP = &bp
				x.NextCheck = nil
			}
		}
		out.Hypotheses = append(out.Hypotheses, x)
	}
	for _, h := range n.Hypotheses {
		if h.Verdict == "rejected" {
			continue
		}
		b := h.Branch
		x := Hypothesis{HypothesisID: h.HypothesisID, Category: h.Category, Branch: strp(b), Statement: strp(h.Statement),
			Status: status(h.HypothesisID, h.Category, "recorded"), Supporting: refs.of(h.Supporting), Contradicting: []JournalRecordRef{},
			History: []HypothesisChange{{At: h.At, EventID: strp(h.EventID), Text: "Записана человеком (" + h.Actor + ")"}}}
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
		row := NcGroup{GroupKey: k, DefectType: defect, Operation: op, Equipment: eq, NCCount: len(g), NCIDs: []string{}, LastFoundAt: g[0].At}
		for _, p := range g {
			row.NCIDs = append(row.NCIDs, p.NCID)
			if p.At.After(row.LastFoundAt) {
				row.LastFoundAt = p.At
			}
		}
		row.Investigation, err = s.investigation(ctx, g, ncs)
		if err != nil {
			return NcGroupList{}, err
		}
		row.DefectTypeLabel = s.name(func(n Names) (string, bool) { return n.DefectLabel(ctx, defect) })
		row.OperationLabel = s.name(func(n Names) (string, bool) { return n.StepName(ctx, op) })
		row.EquipmentLabel = s.name(func(n Names) (string, bool) { return n.FactorLabel(ctx, dom.FactorMachine, eq) })
		row.IncidentID = s.groupIncident(ctx, g, ncs)
		out.Items = append(out.Items, row)
	}
	return out, nil
}

// groupIncident — расследование группы: инцидент, к которому решениями людей
// отнесено несоответствие группы, иначе инцидент, в область которого входит
// изделие несоответствия.
func (s *Service) groupIncident(ctx context.Context, g []dom.Profile, ncs map[string]dom.NCRecord) *string {
	for _, p := range g {
		if ids := ncs[p.NCID].IncidentIDs; len(ids) > 0 {
			return strp(ids[0])
		}
	}
	ids, err := s.list(ctx, ProjectionIncident)
	if err != nil {
		return nil
	}
	for _, id := range ids {
		v, err := s.incident(ctx, id)
		if err != nil {
			continue
		}
		for _, p := range g {
			if _, ok := v.Members[ncs[p.NCID].ItemID]; ok && p.ItemID != "" {
				return strp(id)
			}
		}
	}
	return nil
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
		f, v := r.Factor, r.Value
		out.Rows = append(out.Rows, CommonFactorRow{Factor: f, Value: strp(v), Matches: r.Matches, DistinctValues: r.Distinct,
			ValueLabel: s.name(func(n Names) (string, bool) { return n.FactorLabel(ctx, f, v) })})
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
	idx, err := s.ncIndex(ctx)
	if err != nil {
		return IncidentList{}, err
	}
	for _, id := range ids {
		v, err := s.incident(ctx, id)
		if err != nil {
			continue
		}
		if m.AsOf != nil && v.OpenedAt.After(*m.AsOf) {
			continue
		}
		versions := versionsAt(v, m)
		x := IncidentSummary{IncidentID: id, Label: v.Label, CommonFactor: s.labeled(ctx, factorRef(v)), InitialSize: v.InitialSize, Status: "open", OpenedAt: v.OpenedAt}
		if n := len(versions); n > 0 {
			x.Size, x.ScopeVersion = versions[n-1].Size, versions[n-1].Version
		}
		if v.Closed && (m.AsOf == nil || v.ClosedAt == nil || !v.ClosedAt.After(*m.AsOf)) {
			x.Status = "closed"
		}
		x.IncidentLink = idx.link(v)
		x.InvestigationState = investigationState(v, s.facts(ctx, v, x.PrimaryNCID))
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
	out := RiskScope{IncidentID: incidentID, IncidentLabel: v.Label, CommonFactor: s.labeled(ctx, factorRef(v)), Versions: []ScopeVersion{}, Items: []ScopeItem{}, BasisSeq: v.BasisSeq}
	idx, err := s.ncIndex(ctx)
	if err != nil {
		return RiskScope{}, err
	}
	out.IncidentLink = idx.link(v)
	if v.WindowStart != nil && v.WindowEnd != nil {
		out.Window = &TimeWindow{Start: *v.WindowStart, End: *v.WindowEnd}
		if v.KnownGoodItem != "" {
			out.LastKnownGood = &KnownGood{Label: s.itemLabel(ctx, v.KnownGoodItem) + " — последняя подтверждённо годная деталь", At: *v.WindowStart}
		}
	}
	for _, x := range versionsAt(v, m) {
		sv := ScopeVersion{ScopeVersion: x.Version, Change: x.Change, Size: x.Size, RecordedAt: x.RecordedAt, Author: strp(x.Author),
			EvidenceEventIDs: append([]string{}, x.Evidence...), Breakdown: ScopeBreakdown(x.Breakdown), ScopeVersionDiff: s.versionDiff(ctx, v, x)}
		if x.Reason != nil {
			sv.Reason = &Reason{Code: strp(x.Reason.Code), Text: x.Reason.Text}
		}
		out.Versions = append(out.Versions, sv)
	}
	current := 0
	if n := len(out.Versions); n > 0 {
		current = out.Versions[n-1].ScopeVersion
	}
	var views []ItemView
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
		out.Items = append(out.Items, ScopeItem{ItemID: id, Label: labelOf(iv), Known: mem.Status, Action: mem.Action,
			Location: dom.Location(iv.State, v.StepKey)})
		if mem.Status != dom.StatusExcluded && mem.Status != dom.StatusConfirmed {
			views = append(views, iv)
		}
	}
	out.NarrowOptions = s.narrowOptions(ctx, v, views)
	return out, nil
}

// driftOption — сужение по времени выхода режима (FR-61, опоздавшие данные):
// журнал оборудования инцидента показал первое отклонение; изделия области
// «под подозрением», выполненные на нём и законченные раньше, с записями
// журнала за выполнение и без отклонений, — «исключить выполненные до выхода
// тока из уставки». Основание — запись первого отклонения и записи журнала
// этих выполнений. Отклонений нет — предложения нет.
func (s *Service) driftOption(ctx context.Context, v dom.IncidentRecord, views []ItemView) []NarrowOption {
	var first *dom.EquipmentEvent
	for _, iv := range views {
		for _, e := range append(slices.Clone(iv.Equipment), iv.State.Equipment...) {
			// Первое отклонение; в ту же минуту — запись отклонения, а не сводка цикла.
			if e.EquipmentID == v.FactorValue && e.Deviation && (first == nil || e.OccurredAt.Before(first.OccurredAt) ||
				e.OccurredAt.Equal(first.OccurredAt) && e.EventType == string(catalog.EquipmentDeviationDetected)) {
				x := e
				first = &x
			}
		}
	}
	if first == nil {
		return nil
	}
	o := NarrowOption{Evidence: []JournalRecordRef{markRef(dom.Mark{EventID: first.EventID, EventType: first.EventType, Variant: first.Variant,
		OccurredAt: first.OccurredAt, Params: first.Params})}}
	seen := map[string]bool{first.EventID: true}
	for _, iv := range views {
		var run *dom.Run
		for i := range iv.State.Runs {
			if r := &iv.State.Runs[i]; r.StepKey == v.StepKey && r.Equipment != "" {
				run = r
			}
		}
		if run == nil || run.Equipment != v.FactorValue || run.Finished == nil || !run.Finished.Before(first.OccurredAt) {
			continue
		}
		var ev []JournalRecordRef
		ok := false
		for _, e := range append(slices.Clone(iv.Equipment), iv.State.Equipment...) {
			if e.EquipmentID != run.Equipment || e.OccurredAt.After(*run.Finished) || (e.EndedAt != nil && e.EndedAt.Before(run.Started)) ||
				(e.EndedAt == nil && e.OccurredAt.Before(run.Started)) {
				continue
			}
			if e.Deviation {
				ok = false
				break
			}
			ok = true
			if !seen[e.EventID] {
				seen[e.EventID] = true
				ev = append(ev, markRef(dom.Mark{EventID: e.EventID, EventType: e.EventType, Variant: e.Variant, OccurredAt: e.OccurredAt, Params: e.Params}))
			}
		}
		if ok {
			o.ItemIDs = append(o.ItemIDs, iv.ItemID)
			o.Evidence = append(o.Evidence, ev...)
		}
	}
	if len(o.ItemIDs) == 0 {
		return nil
	}
	name := v.FactorValue
	if s.cfg.Names != nil {
		if l, ok := s.cfg.Names.FactorLabel(ctx, dom.FactorMachine, name); ok {
			name = l
		}
	}
	at := first.OccurredAt.UTC().Format("02.01.2006 15:04 UTC")
	o.Label = fmt.Sprintf("Исключить выполненные на %s до выхода режима из уставки (%d) — журнал %s", name, len(o.ItemIDs), name)
	o.ReasonText = fmt.Sprintf("Журнал %s: режим впервые вне уставки %s; выполнения на %s до этого — в уставке", name, at, name)
	return []NarrowOption{o}
}

// incidentBasis — basis_seq потоков инцидентов несоответствия: команды из
// разбора и гипотез (запрос измерения, вывод о причине) проверяются по потоку
// инцидента (AD-39), поэтому чтение, с которого их шлют, отдаёт и его seq —
// иначе после сужения области команда получает journal.stale_state.
func (s *Service) incidentBasis(ctx context.Context, n dom.NCRecord) int64 {
	var out int64
	for _, id := range n.IncidentIDs {
		if v, err := s.incident(ctx, id); err == nil {
			out = max(out, v.BasisSeq)
		}
	}
	// Инцидент команды (тот же выбор, что у RequestMeasurement).
	if v, err := s.ncIncident(ctx, n); err == nil {
		out = max(out, v.BasisSeq)
	}
	return out
}

// labelOf — метка изделия из проекции разбора; нет — номер из id.
func labelOf(iv ItemView) string {
	if iv.Label != "" {
		return iv.Label
	}
	return dom.LocalID(iv.ItemID)
}

// itemLabel — метка изделия по id (проекция разбора; нет — номер из id).
func (s *Service) itemLabel(ctx context.Context, id string) string {
	iv, err := s.item(ctx, id)
	if err != nil {
		return dom.LocalID(id)
	}
	iv.ItemID = id
	return labelOf(iv)
}

// narrowOptions — сужение по оборудованию (FR-61): изделия области «под
// подозрением», выполненные на шаге инцидента другим оборудованием, чей
// журнал за эти выполнения есть и весь в уставке, — «исключить сваренные на
// ИС-1». Основание — те же записи журнала; без журнала или с отклонением
// предложения нет (гард incident.basis_required).
func (s *Service) narrowOptions(ctx context.Context, v dom.IncidentRecord, views []ItemView) []NarrowOption {
	if v.Factor != dom.FactorMachine || v.FactorValue == "" {
		return nil
	}
	type group struct {
		items    []string
		evidence []JournalRecordRef
		seen     map[string]bool
		bad      bool
	}
	groups := map[string]*group{}
	for _, iv := range views {
		var run *dom.Run
		for i := range iv.State.Runs {
			if r := &iv.State.Runs[i]; r.StepKey == v.StepKey && r.Equipment != "" {
				run = r
			}
		}
		if run == nil || run.Equipment == v.FactorValue {
			continue
		}
		g := groups[run.Equipment]
		if g == nil {
			g = &group{seen: map[string]bool{}}
			groups[run.Equipment] = g
		}
		g.items = append(g.items, iv.ItemID)
		end := run.Started.Add(24 * time.Hour)
		if run.Finished != nil {
			end = *run.Finished
		}
		found := false
		for _, e := range append(slices.Clone(iv.Equipment), iv.State.Equipment...) {
			if e.EquipmentID != run.Equipment || e.OccurredAt.After(end) || (e.EndedAt != nil && e.EndedAt.Before(run.Started)) ||
				(e.EndedAt == nil && e.OccurredAt.Before(run.Started)) {
				continue
			}
			found = true
			g.bad = g.bad || e.Deviation
			if !g.seen[e.EventID] {
				g.seen[e.EventID] = true
				g.evidence = append(g.evidence, markRef(dom.Mark{EventID: e.EventID, EventType: e.EventType, Variant: e.Variant, OccurredAt: e.OccurredAt, Params: e.Params}))
			}
		}
		g.bad = g.bad || !found
	}
	out := s.driftOption(ctx, v, views)
	for _, eq := range slices.Sorted(maps.Keys(groups)) {
		g := groups[eq]
		if g.bad || len(g.items) == 0 {
			continue
		}
		name, common := eq, v.FactorValue
		if s.cfg.Names != nil {
			if l, ok := s.cfg.Names.FactorLabel(ctx, dom.FactorMachine, eq); ok {
				name = l
			}
			if l, ok := s.cfg.Names.FactorLabel(ctx, dom.FactorMachine, common); ok {
				common = l
			}
		}
		slices.SortFunc(g.evidence, func(a, b JournalRecordRef) int { return a.OccurredAt.Compare(b.OccurredAt) })
		out = append(out, NarrowOption{
			Label:      fmt.Sprintf("Исключить выполненные на %s (%d) — журнал в уставке", name, len(g.items)),
			ItemIDs:    g.items,
			Evidence:   g.evidence,
			ReasonText: fmt.Sprintf("Журнал %s за выполнения этих изделий непрерывный и в уставке; общий фактор — %s", name, common),
		})
	}
	return out
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
	r := CircumstanceRecord{JournalRecordRef: markRef(mk), Lane: mk.Lane, EndedAt: mk.EndedAt, EvidenceRefs: mk.Evidence, SourceKind: strp(mk.SourceKind),
		ReceivedAt: mk.ReceivedAt}
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
	return JournalRecordRef{EventID: mk.EventID, EventType: mk.EventType, Variant: strp(mk.Variant), OccurredAt: mk.OccurredAt, Params: mk.Params, Text: strp(markText(mk))}
}

// refs — ссылки на записи журнала для доводов гипотез.
type refs map[string]JournalRecordRef

func refIndex(st dom.State, a dom.Analysis) refs {
	out := refs{}
	for _, mk := range st.Marks {
		out[mk.EventID] = markRef(mk)
	}
	for _, e := range st.Equipment {
		out[e.EventID] = markRef(dom.Mark{EventID: e.EventID, EventType: e.EventType, Variant: e.Variant, OccurredAt: e.OccurredAt, Params: e.Params})
	}
	for _, mk := range a.Records {
		out[mk.EventID] = markRef(mk)
	}
	for _, c := range st.Cases {
		out[c.EventID] = JournalRecordRef{EventID: c.EventID, EventType: "decision.nonconformity.confirmed", OccurredAt: c.At, Text: strp("Несоответствие подтверждено контролёром")}
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
