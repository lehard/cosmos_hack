package world

import (
	"fmt"
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
		s := ncapp.NCSummary{NCID: n.ID, Number: n.Number, Status: n.Status(c.M, c.T), ItemID: id, ItemLabel: label, Severity: "major", StepKey: n.StepKey, Disposition: "none", FoundAt: n.SignalAt}
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
		if rv.LateEvent != "" {
			row.SourceEventID = ptr(rv.LateEvent)
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
		HumanDecisions: []ncapp.NCRecordRef{}, ToDecide: ncapp.NCToDecide{Decisions: []string{}}, BasisSeq: c.EntitySeq(n.ID, n.Items...)}
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
