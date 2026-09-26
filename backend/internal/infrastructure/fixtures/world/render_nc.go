package world

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	ncapp "ant/internal/application/nonconformity"
	"ant/internal/infrastructure/fixtures/loader"
)

func (c *Ctx) axes(it *Item) ncapp.NCItemAxes {
	st := c.S(it)
	return ncapp.NCItemAxes{Position: st.Position, Quality: st.Quality, Disposition: st.Disposition, Containment: st.Containment, ErpAccounting: st.ERP}
}

func recRef(e *Event) ncapp.NCRecordRef {
	r := ncapp.NCRecordRef{EventID: e.ID, EventType: e.Type, Kind: e.Kind, Seq: ptr(e.Seq), OccurredAt: e.Occurred, Summary: e.Summary}
	if sk := sourceKindOf(e); sk != "" {
		r.SourceKind = ptr(sk)
	}
	if e.Author != "" {
		r.Author = ptr(e.Author)
	}
	if len(e.Params) > 0 {
		r.Params = e.Params
	}
	r.Reading = e.Reading
	return r
}

// ncItem — изделие карточки: первое изделие; у несоответствия партии — компонент.
func ncItem(n *NC) (string, string) {
	if len(n.Items) > 0 {
		return FullID(n.Items[0].ID), n.Items[0].Label
	}
	return FullID(n.Spec.Component), labelOf(n.Spec.Component)
}

// renderNonconformity — очередь «Ждут моего решения», карточки, список, разрешения (FR-51…FR-54).
func renderNonconformity(c *Ctx) []loader.Response {
	var out []loader.Response
	list := ncapp.NCList{Items: []ncapp.NCSummary{}}
	for _, n := range c.M.NCs {
		if n.SignalAt.After(c.T) {
			continue
		}
		id, label := ncItem(n)
		s := ncapp.NCSummary{NCID: n.ID, Number: n.Number, Status: n.Status(c.M, c.T), ItemID: id, ItemLabel: label, Severity: "major", StepKey: n.StepKey, Disposition: "none", FoundAt: n.SignalAt,
			InvestigationStatus: c.investigation(n)}
		if len(n.Spec.Defects) > 0 {
			s.DefectTypeCode = ptr(n.Spec.Defects[0].Kind)
			s.DefectTypeLabel = defectLabel(n.Spec.Defects[0].Kind)
		}
		if d := n.DispositionAt(c.M); d != nil && !d.After(c.T) {
			s.Disposition = n.Disposition(c.M)
		}
		list.Items = append(list.Items, s)
		out = append(out, resp("nonconformity.card.read", c.card(n), "nc_id", n.ID))
	}
	out = append(out, resp("nonconformity.nonconformity.list", list))
	q := c.queue()
	out = append(out, resp("nonconformity.queue.list", q))
	out = append(out, c.presentations(q)...)
	for _, role := range []string{"quality_inspector", "head_of_qc"} {
		out = append(out, resp("nonconformity.queue.list", q, "role", role))
	}
	tech := ncapp.DecisionQueue{Items: []ncapp.DecisionQueueRow{}}
	for _, r := range q.Items {
		if r.Kind == "isolated" {
			tech.Items = append(tech.Items, r)
		}
	}
	for _, role := range []string{"technologist", "chief_welder"} {
		out = append(out, resp("nonconformity.queue.list", tech, "role", role))
	}
	if c.N == 0 {
		out = append(out, resp("nonconformity.concession.list", ncapp.ConcessionList{Items: []ncapp.Concession{}}))
	}
	return out
}

// queue — очередь контролёра: сигналы на рассмотрение, пересмотры, изолированные со сроком, точки предъявления.
func (c *Ctx) queue() ncapp.DecisionQueue {
	q := ncapp.DecisionQueue{Items: []ncapp.DecisionQueueRow{}}
	for _, s := range c.M.Signals() {
		if s.At.After(c.T) || !s.Closed.After(c.T) {
			continue
		}
		it := c.M.itemByID[s.Item]
		row := ncapp.DecisionQueueRow{Kind: "signal", ObjectID: s.ID, ItemID: FullID(it.ID), ItemLabel: it.Label, StepKey: s.Step,
			Title: fmt.Sprintf("Сигнал: %s %s", defectTitle(s.Kind), zoneTitle(s.Zone)), Severity: "major", BasisSeq: c.ItemSeq(it)}
		if s.NC != nil {
			row.NCID = ptr(s.NC.ID)
			row.Reason = c.M.ncEssence(s.NC)
		}
		q.Items = append(q.Items, row)
	}
	for _, rv := range c.M.Spec.Reviews {
		if rv.Flagged.Time().After(c.T) || !rv.Completed.Time().After(c.T) {
			continue
		}
		it := c.M.itemByID[rv.Item]
		// Пересмотр (AD-3): отдельный вид строки; что пришло — словами, id записи — полем.
		row := ncapp.DecisionQueueRow{Kind: "review", ObjectID: "REVIEW-" + rv.Gate + "-" + it.ID, ItemID: FullID(it.ID), ItemLabel: it.Label,
			StepKey: "welding.zt3_acceptance", Title: "Решение ЗТ-3 принято до новых данных — пересмотрите: " + c.M.lateArrival(rv.LateEvent), Severity: "major", PresentationN: ptr(1), BasisSeq: c.ItemSeq(it),
			ReviewSince: tptr(rv.Flagged.Time())}
		if e := c.M.lateRecord(rv.LateEvent); e != nil {
			row.SourceEventID = ptr(e.ID)
		}
		q.Items = append(q.Items, row)
	}
	for _, n := range c.M.NCs {
		if len(n.Spec.Items) > 0 || n.ConfirmedAt.After(c.T) || len(n.Items) == 0 {
			continue
		}
		if d := n.DispositionAt(c.M); d != nil && !d.After(c.T) {
			continue
		}
		it := n.Items[0]
		due := workingDaysAfter(c.M, n.ConfirmedAt, 3)
		// Суть — в заголовке и полем reason (вид дефекта и зона по справочникам).
		title := n.Number + ": ждёт решения по изделию"
		why := c.M.ncEssence(n)
		if why != nil {
			title = n.Number + ": " + *why + " — ждёт решения по изделию"
		}
		q.Items = append(q.Items, ncapp.DecisionQueueRow{Kind: "isolated", ObjectID: n.ID, NCID: ptr(n.ID), ItemID: FullID(it.ID), ItemLabel: it.Label, StepKey: n.StepKey,
			Title: title, Reason: why, Severity: "major", DueAt: tptr(due), Overdue: due.Before(c.T), BasisSeq: c.ItemSeq(it)})
	}
	for _, it := range c.Existing() {
		st := c.S(it)
		if st.Position != "at_presentation_point" || !strings.HasSuffix(st.Step, "_acceptance") && st.Step != "testing.zt5_protocol" {
			continue
		}
		n := 1
		if st.ReworkDone || c.hasRework(it) {
			n = 2
		}
		q.Items = append(q.Items, ncapp.DecisionQueueRow{Kind: "presentation", ObjectID: "PRES-" + it.ID + "-" + st.Step, ItemID: FullID(it.ID), ItemLabel: it.Label, StepKey: st.Step,
			Title: stepTitle[st.Step] + ": предъявлено", Severity: "minor", PresentationN: ptr(n), BasisSeq: c.ItemSeq(it)})
	}
	for i := range q.Items {
		q.Items[i].RiskRank = i
	}
	return q
}

func (c *Ctx) hasRework(it *Item) bool {
	for _, r := range it.Runs {
		if r.ReworkOf != "" && !r.From.After(c.T) {
			return true
		}
	}
	return false
}

// workingDaysAfter — срок по производственному календарю: n рабочих дней (сб, вс — выходные).
func workingDaysAfter(m *Model, t time.Time, n int) time.Time {
	d := t
	for n > 0 {
		d = d.Add(24 * time.Hour)
		if wd := d.In(m.clk.loc).Weekday(); wd != time.Saturday && wd != time.Sunday {
			n--
		}
	}
	return d
}

// card — карточка несоответствия (FR-51): что произошло, доказательства, анализ системы, решения людей.
func (c *Ctx) card(n *NC) ncapp.NCCard {
	id, label := ncItem(n)
	card := ncapp.NCCard{NCID: n.ID, Number: n.Number, Status: n.Status(c.M, c.T), ItemID: id, ItemLabel: label,
		Happened:       ncapp.NCHappened{Before: []ncapp.NCRecordRef{}, During: []ncapp.NCRecordRef{}, After: []ncapp.NCRecordRef{}},
		Evidence:       ncapp.NCEvidence{Signals: []ncapp.NCSourceSignal{}, ZoneHistory: []ncapp.NCRecordRef{}},
		SystemAnalysis: ncapp.NCSystemAnalysis{Versions: []ncapp.NCConclusionVersion{}, Why: []string{}, Alternatives: []string{}, MissingInformation: []string{}},
		HumanDecisions: []ncapp.NCRecordRef{}, ToDecide: ncapp.NCToDecide{Decisions: []string{}}, BasisSeq: c.EntitySeq(n.ID, n.Items...),
		InvestigationStatus: c.investigation(n)}
	if len(n.Items) > 0 {
		card.Axes = c.axes(n.Items[0])
	} else {
		card.Axes = ncapp.NCItemAxes{Position: "in_storage", Quality: "nonconforming", Disposition: "none", Containment: "lot_hold", ErpAccounting: "not_sent"}
		if d := n.DispositionAt(c.M); d != nil && !d.After(c.T) {
			card.Axes.Disposition = n.Disposition(c.M)
		}
	}
	var it *Item
	if len(n.Items) > 0 {
		it = n.Items[0]
	}
	var run *OpRun
	if it != nil {
		for _, r := range it.Runs {
			if r.Kind == "welding" && !r.From.After(n.SignalAt) {
				run = r
			}
		}
	}
	if run != nil && len(n.Spec.Items) == 0 && n.Spec.Component == "" {
		card.Happened.Operation = &ncapp.NCOperationContext{OperationRunID: run.ID, StepKey: run.StepKey, Label: run.Label + " — сварка фланца с кольцом", EquipmentID: ptr(run.Equipment), EquipmentLabel: nameOf(c.M.names.Equipment, run.Equipment),
			ProgramRef: ptr(run.Program), PerformerID: ptr(run.Performer), StartedAt: tptr(run.From), FinishedAt: tptr(run.To)}
		for _, e := range c.itemEvents(it) {
			switch {
			case e.Occurred.Before(run.From) && e.Type == "inspection.result.recorded":
				card.Happened.Before = append(card.Happened.Before, recRef(e))
				card.Evidence.ZoneHistory = append(card.Evidence.ZoneHistory, recRef(e))
			case !e.Occurred.Before(run.From) && !e.Occurred.After(run.To) && (strings.HasPrefix(e.Type, "equipment.") || strings.HasPrefix(e.Type, "operat")):
				card.Happened.During = append(card.Happened.During, recRef(e))
			case e.Occurred.After(run.To) && e.Type == "inspection.result.recorded":
				card.Happened.After = append(card.Happened.After, recRef(e))
				card.Evidence.ZoneHistory = append(card.Evidence.ZoneHistory, recRef(e))
			}
		}
		if n.ID == "NC-01" {
			card.Happened.During = append(card.Happened.During, ncapp.NCRecordRef{EventID: c.M.eventID("override/F-017"), EventType: "operator.override.performed", Kind: "fact",
				OccurredAt: c.M.clk.at(23, 10, 50), SourceKind: ptr("manual_entry"), Author: ptr("W21"), Summary: "Пауза и ручное вмешательство: деталь переставлена в приспособлении (рядом по времени со сваркой У2)"})
		}
	}
	for _, s := range c.M.Signals() {
		if s.NC != n || s.At.After(c.T) {
			continue
		}
		v := c.signalView(s)
		src := ncapp.NCSourceSignal{SignalID: v.SignalID, BasisKind: v.BasisKind, DefectTypeCode: v.DefectTypeCode, DefectTypeKnown: v.DefectTypeKnown, ZoneID: v.ZoneID,
			Severity: v.Severity, AnalyzerConfidenceBP: v.AnalyzerConfidenceBP, ObservationQualityBP: v.ObservationQualityBP, Stages: []ncapp.NCAnalyzerStage{}, Versions: v.Versions, EvidenceRefs: v.EvidenceRefs}
		if v.DefectTypeCode != nil {
			src.DefectTypeLabel = defectLabel(*v.DefectTypeCode)
		}
		if v.ZoneID != nil {
			src.ZoneLabel = nameOf(c.M.names.Zones, *v.ZoneID)
		}
		for _, st := range v.Stages {
			src.Stages = append(src.Stages, ncapp.NCAnalyzerStage{Stage: st.Stage, Version: st.Version, ConfidenceBP: st.ConfidenceBP, OutputNote: st.OutputNote})
		}
		for _, e := range c.M.Events {
			if e.Type == "quality.signal.raised" && e.Params["signal_id"] == s.ID {
				src.Record = recRef(e)
			}
		}
		card.Evidence.Signals = append(card.Evidence.Signals, src)
	}
	card.Evidence.Requirement = &ncapp.NCRequirement{Characteristic: "Шов W-1 без прожогов, подрезов и пор (ГОСТ Р ИСО 5817, уровень B)", KDRef: ptr("ФЛ-100.00.000 СБ, ревизия Б")}
	if n.Spec.Component != "" {
		card.Evidence.Requirement = &ncapp.NCRequirement{Characteristic: "Кольцо ФЛ-100.01.002: сплошность основного металла", KDRef: ptr("ФЛ-100.01.002, ревизия А; сертификат партии")}
	}
	for _, o := range c.M.NCs {
		if o != n && len(o.Spec.Defects) > 0 && len(n.Spec.Defects) > 0 && o.Spec.Defects[0].Kind == n.Spec.Defects[0].Kind && !o.ConfirmedAt.After(c.T) {
			card.Evidence.SimilarCount++
		}
	}
	// Анализ системы: версии вывода по слоту реакции (AD-3) и гипотезы.
	for _, e := range c.M.Events {
		if e.Step > c.N || e.Entity.ID != n.ID {
			continue
		}
		switch e.Type {
		case "decision.nonconformity.drafted":
			card.SystemAnalysis.Versions = append(card.SystemAnalysis.Versions, ncapp.NCConclusionVersion{Version: len(card.SystemAnalysis.Versions) + 1, EventID: e.ID, RuleID: "R-07",
				AutomationMode: 3, Outcome: "isolate", RecordedAt: e.Recorded, Causes: []ncapp.NCRecordRef{}})
		case "incident.hypothesis.computed":
			v := ncapp.NCConclusionVersion{Version: len(card.SystemAnalysis.Versions) + 1, EventID: e.ID, RuleID: "R-12", AutomationMode: 3, Outcome: "question_to_technologist", RecordedAt: e.Recorded, Causes: []ncapp.NCRecordRef{}}
			if strings.Contains(e.Summary, "пересмотрено") {
				v.RevisedDueTo = ptr(lateEventID(c.M, e))
			}
			card.SystemAnalysis.Versions = append(card.SystemAnalysis.Versions, v)
		case "decision.nonconformity.confirmed", "decision.nonconformity.registered", "decision.item.isolated", "decision.disposition.set", "incident.cause.concluded", "decision.disposition.verified":
			card.HumanDecisions = append(card.HumanDecisions, recRef(e))
		}
	}
	h := c.hypAt(n)
	if h != nil {
		card.SystemAnalysis.Why = append(card.SystemAnalysis.Why, fmt.Sprintf("Гипотезы, версия %d: оборудование — %s; отклонение от процедуры — %s", h.V, strengthTitle(h.Equipment), strengthTitle(h.Performer)))
		for _, mi := range h.Missing {
			card.SystemAnalysis.MissingInformation = append(card.SystemAnalysis.MissingInformation, missingTitle[mi])
		}
	}
	if n.ID == "NC-01" {
		card.SystemAnalysis.Alternatives = []string{"Подготовка кромок: окно 1 ч 50 мин — в норме (слабая)", "Скрытый дефект материала: камера КТ-2 видит только поверхность (слабая)"}
		card.SystemAnalysis.Why = append(card.SystemAnalysis.Why, "Правило R-07: прожог на шве → карантин и черновик карточки")
	}
	if n.Spec.Cause != nil && n.Spec.Cause.Category == "incoming" {
		card.SystemAnalysis.Why = append(card.SystemAnalysis.Why, "Зону «тело кольца» не затрагивала ни одна операция; кольцо — партия П-117; камера КТ-1 внутренние поры не видит")
		card.SystemAnalysis.Alternatives = []string{}
	}
	switch card.Status {
	case "draft":
		card.ToDecide.Decisions = []string{"nonconformity.nonconformity.confirm", "nonconformity.signal.reject", "nonconformity.recheck.request"}
	case "confirmed":
		card.ToDecide.Decisions = []string{"nonconformity.disposition.set", "nonconformity.item.isolate", "nonconformity.containment.set"}
		card.ToDecide.DecisionDueAt = tptr(workingDaysAfter(c.M, n.ConfirmedAt, 3))
		card.ToDecide.ConcessionRequired = true
	case "disposition_set":
		card.ToDecide.Decisions = []string{"nonconformity.disposition.verify"}
	}
	// Интерфейс 7: решения с последствиями и исполнение решения по изделию.
	c.decisionActions(n, it, run, &card)
	card.Handoff = c.handoff(n, it, run, &card)
	return card
}

// ncEssence — суть несоответствия: виды дефектов и зоны («Прожог · Шов W-1,
// участок У2 (40–80 мм)»; несколько дефектов — через «; »); зоны нет в
// справочнике — только вид.
func (m *Model) ncEssence(n *NC) *string {
	var parts []string
	for _, d := range n.Spec.Defects {
		l := defectLabel(d.Kind)
		if l == nil {
			continue
		}
		p := *l
		if z := nameOf(m.names.Zones, d.Zone); z != nil {
			p += " · " + *z
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		return nil
	}
	s := strings.Join(parts, "; ")
	return &s
}

// investigation — второй статус несоответствия «системное расследование»
// (FR-51) по области риска (инциденту) модуля analysis: инцидент, начатый
// этим несоответствием или с его причиной среди общих факторов; до открытия
// инцидента — none, открыт — open, закрыт (incident.incident.closed, как у
// live) — closed; инцидента нет — none.
func (c *Ctx) investigation(n *NC) string {
	for _, in := range c.M.Spec.Incidents {
		linked := in.Trigger == n.ID
		if n.Spec.Cause != nil && n.Spec.Cause.Ref != "" && slices.Contains(in.Factors, n.Spec.Cause.Ref) {
			linked = true
		}
		switch {
		case !linked || in.Opened.Time().After(c.T):
			continue
		case !in.Closed.IsZero() && !in.Closed.Time().After(c.T):
			return "closed"
		default:
			return "open"
		}
	}
	return "none"
}

// lateSource — источник опоздавшей записи id по спецификации мира; нет — "".
func (m *Model) lateSource(id string) string {
	for _, le := range m.Spec.LateEvents {
		if le.ID == id {
			return le.Source
		}
	}
	return ""
}

// lateRecord — запись журнала мира об опоздавшей записи источника id; нет — nil.
func (m *Model) lateRecord(id string) *Event {
	for _, e := range m.Events {
		if id != "" && e.Type == "equipment.deviation.detected" && e.Params["record"] == id {
			return e
		}
	}
	return nil
}

// presentations — точки предъявления строк очереди (nonconformity.presentation.read,
// FR-19): предъявление и пересмотр (AD-3) — всё, что нужно команде решения.
func (c *Ctx) presentations(q ncapp.DecisionQueue) []loader.Response {
	var out []loader.Response
	for _, r := range q.Items {
		if r.Kind != "presentation" && r.Kind != "review" {
			continue
		}
		it := c.M.itemByID[strings.TrimPrefix(r.ItemID, enterprise+":")]
		if it == nil {
			continue
		}
		v := c.presentationView(it, r)
		out = append(out, resp("nonconformity.presentation.read", v, "item_id", r.ItemID))
	}
	return out
}

// presentationView — точка предъявления изделия: шаг, ЗТ, результаты методов
// этапа до точки; у пересмотра — прежнее решение, его основание и опоздавшая запись.
func (c *Ctx) presentationView(it *Item, r ncapp.DecisionQueueRow) ncapp.NCPresentationView {
	n := 1
	if r.PresentationN != nil {
		n = *r.PresentationN
	}
	p := ncapp.NCPresentationPoint{EventID: r.ObjectID, StepKey: r.StepKey, PresentationNo: n, MethodEventIDs: []string{},
		AllowedResolutions: []string{"accept", "reject", "insufficient_data"}}
	if nd := c.M.Bpmn[r.StepKey]; nd != nil {
		p.StepLabel = c.nodeName(r.StepKey)
		p.ClosingPoint = nd.Props["closingPoint"]
		if p.ClosingPoint != "" {
			p.ClosingPointLabel = c.nodeName(r.StepKey)
		}
		p.NextStepLabel = c.M.nextStepName(nd)
	}
	v := ncapp.NCPresentationView{ItemID: FullID(it.ID), ItemLabel: it.Label, MethodResults: []ncapp.NCRecordRef{}, BasisSeq: c.ItemSeq(it)}
	stage, _, _ := strings.Cut(r.StepKey, ".")
	var decision *Event
	if r.Kind == "review" {
		for _, e := range c.itemEvents(it) {
			if e.Type == "decision.presentation.resolved" && e.StepKey == r.StepKey && e.Params["outcome"] == "accepted_incomplete_data" {
				decision = e
			}
		}
	}
	upTo := c.T
	if decision != nil {
		upTo = decision.Occurred
	}
	latest := map[string]*Event{}
	var order []string
	for _, e := range c.itemEvents(it) {
		if e.Type != "inspection.result.recorded" || e.Kind != "fact" || e.Occurred.After(upTo) || !strings.HasPrefix(e.StepKey, stage+".") {
			continue
		}
		if _, ok := latest[e.StepKey]; !ok {
			order = append(order, e.StepKey)
		}
		latest[e.StepKey] = e
	}
	for _, k := range order {
		e := latest[k]
		p.MethodEventIDs = append(p.MethodEventIDs, e.ID)
		v.MethodResults = append(v.MethodResults, recRef(e))
	}
	if decision != nil {
		rv := &ncapp.NCPresentationReview{Decision: recRef(decision), KnownAtDecision: slices.Clone(v.MethodResults), NewFacts: []ncapp.NCRecordRef{}}
		for _, x := range c.M.Spec.Reviews {
			if e := c.M.lateRecord(x.LateEvent); x.Item == it.ID && e != nil && !e.Recorded.After(c.T) {
				// Новый факт словами по источнику, как в очереди контролёра; код записи — в event_id.
				nf := recRef(e)
				if v, sp := e.Params["value"], e.Params["setpoint"]; v != "" && sp != "" {
					nf.Summary = fmt.Sprintf("%s: ток %s А при уставке %s — первое отклонение", c.M.sourceLabel(e), v, sp)
				}
				rv.NewFacts = append(rv.NewFacts, nf)
				c.reviewBasis(it, x, decision, e, rv, &v)
			}
		}
		v.Review = rv
	} else {
		// Решения на ждущем предъявлении (Д-81): те же тексты, что у live.
		gate := first(derefOr(p.ClosingPointLabel), p.ClosingPoint)
		next := "следующий шаг процесса"
		if p.NextStepLabel != nil {
			next = "«" + *p.NextStepLabel + "»"
		}
		for _, r := range []string{"accept", "reject", "insufficient_data"} {
			res := r
			a := ncapp.NCPresentationAction{Operation: "nonconformity.presentation.resolve", Resolution: &res, Allowed: slices.Contains(p.AllowedResolutions, r)}
			var cons []string
			a.Label, a.WhyAvailable, cons = ncapp.ResolveTexts(r, gate, next, "", p.PresentationNo)
			a.Consequences, a.TechnicalConsequences = ncapp.SplitConsequences(cons)
			v.Actions = append(v.Actions, a)
		}
		v.Recommendation = &ncapp.NCRecommendation{Outcome: "accept", Why: []string{"Методы контроля признаков дефекта не нашли; блока и открытых несоответствий нет"}}
		for _, m := range v.MethodResults {
			if strings.Contains(m.Summary, "оценка невозможна") || strings.Contains(m.Summary, "качество 0,3") {
				v.Recommendation = &ncapp.NCRecommendation{Outcome: "insufficient_data", Why: []string{m.Summary}}
			}
		}
	}
	v.Presentation = p
	return v
}

func derefOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func first(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// reviewBasis — пересмотр в заготовках (Д-81): «данных не было» по источнику
// опоздавшей записи (запись о потере связи), почему запись значима,
// рекомендация и решения с последствиями — теми же текстами, что у live.
func (c *Ctx) reviewBasis(it *Item, x ReviewSpec, decision, late *Event, rv *ncapp.NCPresentationReview, v *ncapp.NCPresentationView) {
	gate := first(derefOr(c.nodeName(decision.StepKey)), x.Gate)
	var src *SourceSpec
	for i := range c.M.Spec.Sources {
		if s := &c.M.Spec.Sources[i]; late.Params["record"] != "" && s.ID == c.M.lateSource(x.LateEvent) {
			src = s
		}
	}
	if src != nil {
		for _, e := range c.M.Events {
			if e.Type == "ingest.source.loss_suspected" && e.Params["source_id"] == src.ID && !e.Occurred.After(decision.Occurred) {
				a := recRef(e)
				a.Absent, a.Summary = true, "Нет данных: журнала режима «"+srcEquipment(src)+"» на момент решения не было — источник без связи"
				rv.KnownAtDecision = append(rv.KnownAtDecision, a)
				break
			}
		}
	}
	op := first(derefOr(c.nodeName("welding.weld")), "сварка")
	why := []string{
		"Отклонение режима «" + first(srcEquipment(src), "оборудования") + "»: возникло за " + ncapp.Span(decision.Occurred.Sub(late.Occurred)) +
			" до решения, стало известно через " + ncapp.Span(late.Recorded.Sub(decision.Occurred)) + " после него — решение принималось без этой записи",
		"Относится к операции «" + op + "» " + it.Label + " до приёмки на точке «" + gate + "»",
	}
	significant := []string{}
	if line := ncapp.OutOfSetpoint(late.Reading); line != "" {
		why = append(why, line)
		significant = append(significant, line)
	}
	if nd := c.M.Bpmn["welding.weld"]; nd != nil && nd.Props["specialProcess"] == "true" {
		line := "«" + op + "» — специальный процесс: нарушение режима само по себе — несоответствие, даже если контроль дефекта не нашёл (FR-151)"
		why = append(why, line)
		significant = append(significant, line)
	}
	if src != nil {
		why = append(why, "При подписи данных «"+srcEquipment(src)+"» не было: приёмка стояла только на результатах методов контроля")
	}
	rv.WhySignificant = why
	v.Recommendation = &ncapp.NCRecommendation{Outcome: "revoked", Why: significant}

	st := c.S(it)
	var incidents []string
	for _, id := range slices.Sorted(maps.Keys(st.Incidents)) {
		incidents = append(incidents, id)
	}
	where := "текущем шаге"
	if n := c.nodeName(st.Step); n != nil {
		where = "«" + *n + "»"
	}
	revoked, upheld := ncapp.ReviewConsequences(gate, "accept", where, incidents)
	revoked, revokedTech := ncapp.SplitConsequences(revoked)
	upheld, upheldTech := ncapp.SplitConsequences(upheld)
	policy := ptr("полномочие точки ЗТ-3; вторая подпись по политике не требуется (Д-81)")
	rvk, uph := "revoked", "upheld"
	blocked := st.Containment == "item_hold" || st.Containment == "lot_hold" || len(incidents) > 0
	up := ncapp.NCPresentationAction{Operation: "nonconformity.presentation.review", Outcome: &uph, Label: ncapp.LabelUphold, Allowed: !blocked,
		WhyAvailable: ncapp.ReviewWhyAllowed(uph, gate), Consequences: upheld, TechnicalConsequences: upheldTech, PolicyRef: policy}
	if blocked {
		up.WhyAvailable = "Изделие заблокировано — операция запрещена до решения: оставить приёмку в силе нельзя, путь — несоответствие и разрешение на отклонение"
	}
	v.Actions = []ncapp.NCPresentationAction{
		{Operation: "nonconformity.presentation.review", Outcome: &rvk, Label: ncapp.LabelRevoke, Allowed: true,
			WhyAvailable: ncapp.ReviewWhyAllowed(rvk, gate), Consequences: revoked, TechnicalConsequences: revokedTech, PolicyRef: policy},
		up,
	}
}

// srcEquipment — оборудование источника для людей («Сварочный источник ИС-2»).
func srcEquipment(s *SourceSpec) string {
	if s == nil {
		return ""
	}
	if t := equipmentTitle[s.Equipment][0]; t != "" {
		return t
	}
	return s.Equipment
}

// nextStepName — имя следующего шага при «Принять» (не шлюз, с именем); на
// развилке — ветка «годно» (decision == 'accept' или test.result == 'tight'), если она есть.
func (m *Model) nextStepName(n *BpmnNode) *string {
	byID := map[string]*BpmnNode{}
	for _, x := range m.Bpmn {
		byID[x.ID] = x
	}
	seen := map[string]bool{n.ID: true}
	next := func(x *BpmnNode) []string {
		for id, c := range x.Cond {
			if strings.Contains(c, "'accept'") || strings.Contains(c, "'tight'") {
				return []string{id}
			}
		}
		return slices.Clone(x.Next)
	}
	queue := next(n)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		x := byID[id]
		if x == nil || seen[id] {
			continue
		}
		seen[id] = true
		if !strings.HasSuffix(x.Type, "Gateway") && !strings.HasSuffix(x.Type, "Event") && x.Name != "" {
			return ptr(x.Name)
		}
		queue = append(queue, next(x)...)
	}
	return nil
}

// lateArrival — что пришло после решения, словами: журнал оборудования
// (название — из справочника оборудования) и значение против уставки.
func (m *Model) lateArrival(id string) string {
	for _, le := range m.Spec.LateEvents {
		if le.ID != id {
			continue
		}
		what := "пришёл опоздавший журнал оборудования"
		for _, s := range m.Spec.Sources {
			if s.ID != le.Source {
				continue
			}
			if name := nameOf(m.names.Equipment, s.Equipment); name != nil {
				what = "пришёл журнал «" + *name + "»"
			}
		}
		if le.CurrentA > 0 && le.Setpoint != "" {
			what += fmt.Sprintf(": ток %d А при уставке %s", le.CurrentA, le.Setpoint)
		}
		return what
	}
	return "пришли новые данные"
}

var missingTitle = map[string]string{"equipment_log_missing": "Журнал параметров ИС-2 за время сварки не пришёл", "no_observation_after_operation": "Нет рентгена после сварки", "other": "Проверки других колец партии"}

func lateEventID(m *Model, e *Event) string {
	for _, le := range m.Spec.LateEvents {
		if strings.Contains(e.Summary, le.ID) {
			return le.ID
		}
	}
	for _, x := range m.Events {
		if x.Type == "decision.nonconformity.confirmed" && x.Entity.ID == "NC-02" {
			return x.ID
		}
	}
	return ""
}

// hypAt — последняя видимая версия гипотез несоответствия.
func (c *Ctx) hypAt(n *NC) *HypVersion {
	var out *HypVersion
	for i := range n.Hyps {
		if !n.Hyps[i].At.After(c.T) {
			out = &n.Hyps[i]
		}
	}
	return out
}
