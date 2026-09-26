package world

import (
	"fmt"
	"slices"
	"strings"
	"time"

	analysisapp "ant/internal/application/analysis"
	dom "ant/internal/domain/analysis"
)

// Расследование на столе технолога в мире заготовок (эпик 12, «Бэкенд для
// интерфейса 4»): связь инцидента с разбором, стадия и «что дальше» (те же
// доменные функции, что у live), повод и разница ступеней области, вторая
// причина why_missed, история уверенности гипотез, «что проверить
// следующим», качество данных дорожек.

// whyMissed — вторая причина инцидента в журнале мира: гипотеза, доводы и вывод.
type whyMissed struct {
	Spec       *WhyMissedSpec
	Incident   *Incident
	Primary    *NC
	Supporting []*Event
	Proposed   *Event
	Concluded  *Event
}

// buildWhyMissed — записи ветки why_missed (гипотеза человека и вывод о
// причине). Пишутся после всех остальных записей мира — нумерация прежних
// записей не сдвигается.
func (m *Model) buildWhyMissed() {
	for _, in := range m.Incidents {
		s := in.Spec.WhyMissed
		if s == nil {
			continue
		}
		w := &whyMissed{Spec: s, Incident: in, Primary: m.ncByID[in.Spec.Trigger]}
		for _, key := range s.Supporting {
			if e := m.supportEvent(key, in); e != nil {
				w.Supporting = append(w.Supporting, e)
			}
		}
		opts := []evOpt{entity("incident", in.Spec.ID), withParams("incident_id", in.Spec.ID, "branch", dom.BranchWhyMissed, "category", s.Category)}
		w.Proposed = m.ev("incident.hypothesis.recorded", "decision", s.Proposed.Time(), "Гипотеза «почему не остановили раньше»: "+s.Statement,
			append(slices.Clone(opts), withAuthor(s.By))...)
		if !s.Confirmed.IsZero() {
			w.Concluded = m.ev("incident.cause.concluded", "decision", s.Confirmed.Time(), "Причина «почему не остановили раньше» подтверждена — "+s.Verification,
				append(slices.Clone(opts), withAuthor(s.By))...)
		}
		m.WhyMissed = append(m.WhyMissed, w)
	}
}

// supportEvent — запись-довод по ключу world.yaml (why_missed.supporting).
func (m *Model) supportEvent(key string, in *Incident) *Event {
	for _, e := range m.Events {
		switch key {
		case "loss_alert":
			if e.Type == "ingest.source.loss_suspected" {
				return e
			}
		case "zt3_incomplete":
			if e.Type == "decision.presentation.resolved" && e.Params["outcome"] == "accepted_incomplete_data" {
				return e
			}
		case "camera":
			if e.Type == "inspection.result.recorded" && e.StepKey == "welding.kt3_camera" && e.Item != nil && e.Item.ID == "F-015" {
				return e
			}
		case "lot_accept":
			if e.Type == "decision.lot.resolved" && slices.Contains(in.Spec.Factors, e.Params["lot_id"]) {
				return e
			}
		}
	}
	return nil
}

// whyMissedOf — вторая причина инцидента (nil — нет в мире).
func (m *Model) whyMissedOf(id string) *whyMissed {
	for _, w := range m.WhyMissed {
		if w.Incident.Spec.ID == id {
			return w
		}
	}
	return nil
}

// visibleNCs — несоответствия инцидента, подтверждённые к часам шага.
func (c *Ctx) incidentLink(in *Incident) analysisapp.IncidentLink {
	out := analysisapp.IncidentLink{NCIDs: []string{}}
	for _, id := range c.incidentNCs(in) {
		if n := c.M.ncByID[id]; n != nil && !n.ConfirmedAt.After(c.T) {
			out.NCIDs = append(out.NCIDs, id)
		}
	}
	p := c.M.ncByID[in.Spec.Trigger]
	if p == nil || p.ConfirmedAt.After(c.T) {
		return out
	}
	out.PrimaryNCID = ptr(p.ID)
	for _, g := range c.groups() {
		if slices.Contains(g.row.NCIDs, p.ID) {
			out.GroupKey = ptr(g.row.GroupKey)
		}
	}
	return out
}

// investigationFacts — факты расследования к часам шага.
func (c *Ctx) investigationFacts(in *Incident, v *ScopeState) dom.InvestigationFacts {
	f := dom.InvestigationFacts{IncidentID: in.Spec.ID, Causes: map[string]string{}}
	f.ScopeClosed = !in.Spec.Closed.IsZero() && !in.Spec.Closed.Time().After(c.T)
	for _, x := range v.Status {
		f.Counts.Add(x)
	}
	for i := range in.Versions {
		sv := &in.Versions[i]
		if i > 0 && !sv.Spec.At.Time().After(c.T) && sv.Spec.Rule != "confirm_all" {
			f.Narrowed = true
		}
	}
	if p := c.M.ncByID[in.Spec.Trigger]; p != nil && !p.SignalAt.After(c.T) {
		best := -1
		for _, h := range c.hypotheses(p).Hypotheses {
			if h.Status == "rejected" {
				continue
			}
			f.Hypotheses++
			if h.Status == "confirmed" || h.NextCheck == nil {
				continue
			}
			if conf := intOr0(h.ConfidenceBP); conf > best {
				best, f.NextCheck = conf, h.NextCheck.Text
			}
		}
		if cs := p.Spec.Cause; cs != nil && !cs.At.Time().After(c.T) {
			f.Causes[dom.BranchWhyMade] = "confirmed"
		}
	}
	if w := c.M.whyMissedOf(in.Spec.ID); w != nil && w.Concluded != nil && w.Concluded.Step <= c.N {
		f.Causes[dom.BranchWhyMissed] = "confirmed"
	}
	for _, la := range c.M.Loop.Actions {
		if la.Incident == in.Spec.ID && !la.Assigned.After(c.T) {
			f.Actions = append(f.Actions, c.actionAt(la).Status)
		}
	}
	return f
}

func intOr0(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// investigationState — шапка расследования на часах шага.
func (c *Ctx) investigationState(in *Incident, v *ScopeState) analysisapp.InvestigationState {
	f := c.investigationFacts(in, v)
	st := analysisapp.InvestigationState{Stage: dom.InvestigationStage(f), Counts: analysisapp.KnownCountsView(f.Counts),
		NextStep: optStr(dom.InvestigationNextStep(f)), CloseBlockers: []analysisapp.CloseBlocker{}}
	for _, b := range dom.CloseBlockers(f) {
		st.CloseBlockers = append(st.CloseBlockers, analysisapp.CloseBlocker{Code: string(b.Code), Text: b.Text})
	}
	var last time.Time
	primary := in.Spec.Trigger
	for _, e := range c.Visible() {
		if e.Entity.ID == in.Spec.ID || e.Params["incident_id"] == in.Spec.ID || e.Entity.ID == primary && strings.HasPrefix(e.Type, "incident.") {
			if e.Recorded.After(last) {
				last = e.Recorded
			}
		}
	}
	if !last.IsZero() {
		st.LastEventAt = &last
	}
	return st
}

// versionDiff — повод и разница ступени области i (FR-61, FR-144).
func (c *Ctx) versionDiff(in *Incident, i int) analysisapp.ScopeVersionDiff {
	sv := &in.Versions[i]
	d := analysisapp.ScopeVersionDiff{ItemsAdded: []string{}, ItemsRemoved: []string{}, Evidence: []analysisapp.JournalRecordRef{}}
	prev := map[string]string{}
	if i > 0 {
		prev = in.Versions[i-1].Status
	}
	for _, id := range sortedKeys(sv.Status) {
		was, now := prev[id], sv.Status[id]
		switch {
		case now != "excluded" && (was == "" || was == "excluded"):
			d.ItemsAdded = append(d.ItemsAdded, FullID(id))
		case now == "excluded" && was != "" && was != "excluded":
			d.ItemsRemoved = append(d.ItemsRemoved, FullID(id))
		}
	}
	at := sv.Spec.At.Time()
	var trig *Event
	switch {
	case sv.Spec.LateEvent != "":
		for _, le := range c.M.Spec.LateEvents {
			if le.ID != sv.Spec.LateEvent {
				continue
			}
			occ, rec := le.Occurred.Time(), le.Received.Time()
			d.Trigger = &analysisapp.ScopeTrigger{Kind: "late_event", Label: upperFirst(c.M.lateArrival(le.ID)), ReceivedAt: &rec, OccurredAt: &occ}
			for _, e := range c.M.Events {
				if e.Type == "equipment.deviation.detected" && e.Params["record"] == le.ID {
					trig = e
				}
			}
		}
	case i == 0:
		label := "Правило системы: " + sv.Spec.Basis
		if n := c.M.ncByID[in.Spec.Trigger]; n != nil {
			label = fmt.Sprintf("Подтверждено %s: область по общим факторам — правило системы", n.Number)
			for _, e := range c.M.Events {
				if e.Entity.ID == n.ID && (e.Type == "decision.nonconformity.confirmed" || e.Type == "decision.nonconformity.registered") {
					trig = e
				}
			}
		}
		d.Trigger = &analysisapp.ScopeTrigger{Kind: "computed", Label: label, ReceivedAt: &at}
	default:
		who := c.M.personName(sv.Spec.By)
		d.Trigger = &analysisapp.ScopeTrigger{Kind: "human", Label: "Решение: " + who + " — " + firstClause(sv.Spec.Basis), ReceivedAt: &at, OccurredAt: &at}
	}
	if trig != nil {
		d.Trigger.EventID = ptr(trig.ID)
		if trig.Step <= c.N {
			d.Evidence = append(d.Evidence, c.jref(trig))
		}
	}
	if i > 0 && sv.Spec.By != "" && sv.Spec.By != "system" {
		d.AuthorName, d.SignedBy, d.KeyClass = ptr(c.M.personName(sv.Spec.By)), ptr(sv.Spec.By), ptr("personal")
	}
	for _, e := range c.M.Events {
		if e.Entity.ID == in.Spec.ID && e.Params["scope_version"] == fmt.Sprint(sv.Spec.V) && e != trig {
			d.Evidence = append(d.Evidence, c.jref(e))
		}
	}
	return d
}

// firstClause — первая часть основания до двоеточия или точки с запятой.
func firstClause(s string) string {
	if i := strings.IndexAny(s, ":;"); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// sourceLabel — источник записи словами: человек, журнал оборудования, камера.
func (m *Model) sourceLabel(e *Event) string {
	if e.Author != "" {
		return m.personName(e.Author)
	}
	for _, s := range m.Spec.Sources {
		if s.ID == e.Source {
			if name := nameOf(m.names.Equipment, s.Equipment); name != nil {
				return "журнал «" + *name + "»"
			}
		}
	}
	switch e.Source {
	case "edge-weld-1", "edge-weld-2":
		eq := map[string]string{"edge-weld-1": "IS-1", "edge-weld-2": "IS-2"}[e.Source]
		if name := nameOf(m.names.Equipment, eq); name != nil {
			return "журнал «" + *name + "»"
		}
	case "edge-kt2", "edge-kt3", "edge-kt3-2", "edge-kt4", "edge-kt5":
		return "камера " + map[string]string{"edge-kt2": "КТ-2", "edge-kt3": "КТ-3", "edge-kt3-2": "КТ-3 (ракурс 2)", "edge-kt4": "КТ-4", "edge-kt5": "КТ-5"}[e.Source]
	case "onec":
		return "1С"
	case "ant", "":
		return "система"
	}
	return e.Source
}

// hypothesisHistory — что меняло уверенность системной гипотезы категории cat:
// версии вывода (incident.hypothesis.computed) и подтверждение причины.
func (c *Ctx) hypothesisHistory(n *NC, cat string) []analysisapp.HypothesisChange {
	out := []analysisapp.HypothesisChange{}
	prev := ""
	for i := range n.Hyps {
		h := &n.Hyps[i]
		if h.At.After(c.T) {
			break
		}
		strength := map[string]string{"equipment": h.Equipment, "performer": h.Performer, "incoming": h.Incoming}[cat]
		if strength == "" || strength == prev {
			continue
		}
		x := analysisapp.HypothesisChange{At: h.At}
		if strength != "not_assessable" {
			x.ConfidenceBP = ptr(strengthBP[strength])
		}
		switch {
		case h.Categorical:
			x.Text = "Причина подтверждена решением человека"
			if cs := n.Spec.Cause; cs != nil {
				for _, e := range c.M.Events {
					if e.Type == "incident.cause.concluded" && e.Entity.ID == n.ID {
						x.EventID = ptr(e.ID)
						x.Text = e.Summary
					}
				}
			}
		default:
			x.Text = fmt.Sprintf("Версия вывода %d: %s", h.V, strengthTitle(strength))
			if prev != "" {
				x.Text += " (было: " + strengthTitle(prev) + ")"
			}
			if h.RevisedDueTo != "" {
				x.Text += " — " + c.revisedText(h.RevisedDueTo)
			}
			for _, e := range c.M.Events {
				if e.Type == "incident.hypothesis.computed" && e.Entity.ID == n.ID && e.Params["version"] == fmt.Sprint(h.V) {
					x.EventID = ptr(e.ID)
				}
			}
		}
		out = append(out, x)
		prev = strength
	}
	return out
}

var strengthBP = map[string]int{"not_assessable": 0, "weak": 2000, "possible": 5000, "strong": 8500, "confirmed": 9500}

// revisedText — что пересмотрело вывод словами.
func (c *Ctx) revisedText(due string) string {
	var parts []string
	for _, id := range strings.Split(due, ",") {
		if n := c.M.ncByID[id]; n != nil {
			parts = append(parts, "рентген "+n.Number)
			continue
		}
		parts = append(parts, c.M.lateArrival(id))
	}
	return strings.Join(parts, ", ")
}

// nextCheck — «что проверить следующим» для гипотезы категории cat: до
// опоздавшего журнала — сам журнал; после — контрольный образец. Оценка
// исключаемых — изделия, которые исключит ближайшее сужение области.
func (c *Ctx) nextCheck(n *NC, cat string) *analysisapp.NextCheck {
	in := c.incidentOfNC(n)
	x := &analysisapp.NextCheck{}
	if in != nil {
		if v := in.VersionAt(c.T); v != nil {
			x.ScopeSize = v.Size()
			x.CouldExclude = c.nextNarrowing(in, v)
		}
	}
	switch cat {
	case "equipment":
		logMissing := false
		for _, it := range n.Items {
			if w := c.weldBefore(it, n.SignalAt); w != nil && (w.LogLost || w.LogReceived.After(c.T)) {
				logMissing = true
			}
		}
		if logMissing {
			x.Text, x.MeasurementKind = "Получить журнал ИС-2 за Вт–Ср (шлюз копит записи) или снять его с источника вручную: ток по каждой сварке против уставки 160 ± 10 А", "equipment_log"
			x.UnlocksText = "Граница окна нарушения режима: сварки на ИС-2 до первого выхода тока из уставки исключаются из области по основанию"
			return x
		}
		x.Text, x.MeasurementKind = "Контрольный образец на ИС-2 при уставке 160 А", "control_sample"
		x.UnlocksText = "Подтверждение причины «оборудование» (ветка «почему возник»): после неё — меры и возврат ИС-2 в работу"
		x.CouldExclude = 0
	case "performer":
		x.Text, x.MeasurementKind = "Сравнить сварки С-02 и С-03 на ИС-2 в окне: дефект у обоих — не исполнитель; объяснение работника — только после подтверждения причины", "document_check"
		x.UnlocksText = "Отклонить гипотезу об отклонении от процедуры — ошибка сварщика не приписывается"
		x.CouldExclude = 0
	case "incoming":
		x.Text, x.MeasurementKind = "Выборочный рентген колец партии П-117 со склада", "sample_inspection"
		x.UnlocksText = "Подтвердить входной брак партии; изделия с кольцами партии, где рентген чистый, исключаются"
	default:
		return nil
	}
	return x
}

// incidentOfNC — инцидент, открытый по несоответствию или содержащий его изделие.
func (c *Ctx) incidentOfNC(n *NC) *Incident {
	for _, in := range c.M.Incidents {
		if in.Spec.Trigger == n.ID {
			return in
		}
	}
	for _, in := range c.M.Incidents {
		if slices.Contains(c.incidentNCs(in), n.ID) {
			return in
		}
	}
	return nil
}

// nextNarrowing — сколько изделий исключит ближайшее сужение после текущей версии.
func (c *Ctx) nextNarrowing(in *Incident, v *ScopeState) int {
	for i := range in.Versions {
		nv := &in.Versions[i]
		if !nv.Spec.At.Time().After(c.T) {
			continue
		}
		n := 0
		for id, x := range nv.Status {
			if x == "excluded" && v.Status[id] != "excluded" {
				n++
			}
		}
		return n
	}
	return 0
}

// whyMissedHypothesis — гипотеза ветки why_missed у ведущего несоответствия.
func (c *Ctx) whyMissedHypothesis(n *NC) *analysisapp.Hypothesis {
	var w *whyMissed
	for _, x := range c.M.WhyMissed {
		if x.Primary == n {
			w = x
		}
	}
	if w == nil || w.Proposed.Step > c.N {
		return nil
	}
	s := w.Spec
	h := &analysisapp.Hypothesis{HypothesisID: fmt.Sprintf("HYP-%s-why_missed", n.ID), Category: s.Category, Branch: ptr(dom.BranchWhyMissed),
		Statement: ptr(s.Statement), Status: "recorded", ConfidenceBP: ptr(s.ConfidenceBP), Supporting: []analysisapp.JournalRecordRef{},
		Contradicting: []analysisapp.JournalRecordRef{}}
	for _, e := range w.Supporting {
		if e.Step <= c.N {
			h.Supporting = append(h.Supporting, c.jref(e))
		}
	}
	h.History = []analysisapp.HypothesisChange{{At: w.Proposed.Recorded, ConfidenceBP: ptr(s.ConfidenceBP), EventID: ptr(w.Proposed.ID),
		Text: "Гипотеза «почему не остановили раньше» записана: " + c.M.personName(s.By) + ", уверенность " + pct(s.ConfidenceBP)}}
	if w.Concluded != nil && w.Concluded.Step <= c.N {
		h.Status, h.ConfidenceBP = "confirmed", ptr(9500)
		h.History = append(h.History, analysisapp.HypothesisChange{At: w.Concluded.Recorded, ConfidenceBP: ptr(9500), EventID: ptr(w.Concluded.ID), Text: w.Concluded.Summary})
		return h
	}
	h.MeasurementHint = ptr(s.Check)
	x := &analysisapp.NextCheck{Text: s.Check, MeasurementKind: "document_check", UnlocksText: s.Unlocks}
	if v := w.Incident.VersionAt(c.T); v != nil {
		x.ScopeSize = v.Size()
	}
	h.NextCheck = x
	return h
}

// laneQualities — качество данных дорожек разбора (опоздания и пропуски) по
// изделию несоответствия на часах шага.
func (c *Ctx) laneQualities(n *NC, it *Item) *analysisapp.LaneQualities {
	q := &analysisapp.LaneQualities{Item: analysisapp.LaneQuality{Gaps: []analysisapp.DataGap{}}, Person: analysisapp.LaneQuality{Gaps: []analysisapp.DataGap{}},
		Equipment: analysisapp.LaneQuality{Gaps: []analysisapp.DataGap{}}}
	for _, e := range c.itemEvents(it) {
		delay := e.Recorded.Sub(e.Occurred)
		if delay < 5*time.Minute {
			continue
		}
		var l *analysisapp.LaneQuality
		switch {
		case strings.HasPrefix(e.Type, "equipment.") || e.Type == "quality.inspection.missing":
			l = &q.Equipment
		case strings.HasPrefix(e.Type, "operation.run") || strings.HasPrefix(e.Type, "operator."):
			l = &q.Person
		case e.Type == "inspection.result.recorded" || strings.HasPrefix(e.Type, "operation.movement"):
			l = &q.Item
		default:
			continue
		}
		l.LateCount++
		l.MaxDelayMin = max(l.MaxDelayMin, int(delay.Minutes()))
	}
	for _, r := range it.Runs {
		if r.Kind != "welding" || r.From.After(n.SignalAt) {
			continue
		}
		src := map[string]string{"IS-1": "edge-weld-1", "IS-2": "edge-weld-2"}[r.Equipment]
		for _, s := range c.M.Spec.Sources {
			if s.ID != src {
				continue
			}
			lw := s.Batch.LostWindow
			if r.LogLost && !lw.IsZero() && !r.LogReceived.After(c.T) {
				q.Equipment.Gaps = append(q.Equipment.Gaps, analysisapp.DataGap{From: lw.From.Time(), To: lw.To.Time(), Source: s.ID,
					Text: fmt.Sprintf("Записи %s потеряны у шлюза (%d записей, разрыв %s) — исключать нельзя", r.Equipment, s.Batch.Lost, s.Batch.Gap)})
			}
			if !r.LogLost && r.LogReceived.After(c.T) {
				to := c.T
				q.Equipment.Gaps = append(q.Equipment.Gaps, analysisapp.DataGap{From: s.Lost.Time(), To: to, Source: s.ID,
					Text: fmt.Sprintf("Журнал %s за время сварки %s ещё не пришёл: шлюз без связи — исключать нельзя", r.Equipment, r.Label)})
			}
		}
	}
	return q
}
