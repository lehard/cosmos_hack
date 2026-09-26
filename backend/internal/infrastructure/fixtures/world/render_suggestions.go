package world

import (
	"slices"

	analysisapp "ant/internal/application/analysis"
	dom "ant/internal/domain/analysis"
	"ant/internal/infrastructure/fixtures/loader"
)

// Предложения, меры и карта дефицита данных на шаге (эпик 42; FR-63, FR-64,
// FR-138, FR-143): тела analysis.suggestion.list, analysis.action.list и
// analysis.data_deficit.read из контура улучшений мира (quality_loop.go).
// «Принято» ничего не применяет само; «внедрено» ≠ «эффективно».

func renderSuggestions(c *Ctx) []loader.Response {
	return []loader.Response{
		resp("analysis.suggestion.list", c.suggestions()),
		resp("analysis.action.list", c.correctiveActions()),
		resp("analysis.data_deficit.read", c.dataDeficit()),
	}
}

func (c *Ctx) suggestions() analysisapp.SuggestionList {
	out := analysisapp.SuggestionList{Items: []analysisapp.Suggestion{}, BasisSeq: c.Seq()}
	for _, g := range analysisapp.RuleGenerators() {
		out.Generators = append(out.Generators, analysisapp.GeneratorInfo{ID: g.ID(), Kind: g.Kind(), Connected: true})
	}
	for _, s := range c.M.Loop.Suggestions {
		if s.At.After(c.T) {
			continue
		}
		v := analysisapp.Suggestion{SuggestionID: s.ID, Generator: s.Generator, Kind: s.Kind, Title: s.Title, Statement: s.Statement,
			Estimate: optStr(s.Estimate), ResponsibleRole: optStr(s.Role), StepKey: optStr(s.Step), IncidentID: optStr(s.Incident), MissingKind: optStr(s.Missing),
			Basis: slices.Clone(s.Basis), Status: dom.SuggestionNew, RecordedAt: s.At, EventID: s.EventID, BasisSeq: c.EntitySeq(s.ID),
			History: []analysisapp.SuggestionHistory{{Type: "recorded", At: s.At, EventID: s.EventID}}}
		if f := s.Forward; f != nil && !f.At.After(c.T) {
			v.Status, v.ResponsibleID, v.ResponsibleRole = dom.SuggestionForwarded, ptr(f.To), ptr(f.Role)
			v.History = append(v.History, analysisapp.SuggestionHistory{Type: "forwarded", Actor: ptr(f.By), At: f.At, ResponsibleID: ptr(f.To), ResponsibleRole: ptr(f.Role), Text: ptr(f.Text), EventID: f.EventID})
		}
		if r := s.Resolve; r != nil && !r.At.After(c.T) {
			v.Status = r.Resolution
			v.History = append(v.History, analysisapp.SuggestionHistory{Type: r.Resolution, Actor: ptr(r.By), At: r.At, Text: ptr(r.Text), EventID: r.EventID})
		}
		out.Items = append(out.Items, v)
	}
	// Новые сверху, как у live.
	slices.SortStableFunc(out.Items, func(a, b analysisapp.Suggestion) int { return b.RecordedAt.Compare(a.RecordedAt) })
	return out
}

func optStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// actionAt — мера в проекции на часы шага: статус, цикл и история до c.T.
func (c *Ctx) actionAt(a *loopAction) dom.CorrectiveAction {
	x := dom.CorrectiveAction{ActionID: a.ID, IncidentID: a.Incident, ActionType: a.Type, Direction: a.Direction, Owner: a.Owner, Title: a.Title,
		SuggestionID: a.Suggestion, DueAt: tptr(a.Due), Plan: a.Plan, Status: dom.ActionAssigned, Cycle: 1, AssignedAt: a.Assigned, AssignedBy: a.By,
		EventID: a.EventID, BasisSeq: c.EntitySeq(a.Incident)}
	for _, h := range a.History {
		if h.At.After(c.T) {
			break
		}
		x.History = append(x.History, dom.ActionEvaluation{Type: h.Type, Result: h.Result, Text: h.Text, Actor: h.By, At: h.At, EventID: h.EventID, Cycle: x.Cycle, Evidence: h.Evidence})
		switch {
		case h.Type == "implemented":
			x.Status, x.ImplementedAt = dom.ActionImplemented, tptr(h.At)
		case h.Result == "effective":
			x.Status = dom.ActionEffective
		case h.Result == "failed":
			x.Status, x.ImplementedAt = dom.ActionReopened, nil
			x.Cycle++
		}
	}
	return x
}

// incidentCause — категория подтверждённой причины инцидента к часам шага.
func (c *Ctx) incidentCause(id string) string {
	for _, in := range c.M.Incidents {
		if in.Spec.ID != id || in.Spec.CauseConfirmed.IsZero() || in.Spec.CauseConfirmed.Time().After(c.T) {
			continue
		}
		if n := c.M.ncByID[in.Spec.Trigger]; n != nil && n.Spec.Cause != nil {
			return n.Spec.Cause.Category
		}
	}
	return ""
}

func (c *Ctx) incidentFactor(id string) string {
	for _, in := range c.M.Incidents {
		if in.Spec.ID == id {
			if f := factorRef(in); f != nil {
				return f.Value
			}
		}
	}
	return ""
}

func (c *Ctx) correctiveActions() analysisapp.CorrectiveActionList {
	out := analysisapp.CorrectiveActionList{Items: []analysisapp.CorrectiveActionView{}, Recurring: []analysisapp.RecurringProblem{}, Memory: []analysisapp.MemoryEntry{}, AsOf: c.T}
	withAction := map[string]bool{}
	for _, la := range c.M.Loop.Actions {
		if la.Assigned.After(c.T) {
			continue
		}
		a := c.actionAt(la)
		withAction[a.IncidentID] = true
		v := analysisapp.CorrectiveActionView{ActionID: a.ActionID, IncidentID: a.IncidentID, ActionType: a.ActionType, Direction: a.Direction, Owner: a.Owner,
			Title: optStr(a.Title), SuggestionID: optStr(a.SuggestionID), DueAt: a.DueAt, Status: a.Status, Cycle: a.Cycle, AssignedAt: a.AssignedAt,
			ImplementedAt: a.ImplementedAt, EvaluationDueAt: a.EvaluationDueAt(), Flags: dom.ActionFlags(a, c.T), History: []analysisapp.ActionHistory{}, BasisSeq: a.BasisSeq,
			Plan: analysisapp.EffectivenessPlanView{Metric: a.Plan.Metric, Baseline: a.Plan.Baseline, WindowDays: a.Plan.WindowDays, SuccessCriterion: a.Plan.SuccessCriterion,
				EnhancedControl: optStr(a.Plan.EnhancedControl)},
			CauseCategory: optStr(c.incidentCause(a.IncidentID)), Factor: optStr(c.incidentFactor(a.IncidentID))}
		for _, h := range a.History {
			v.History = append(v.History, analysisapp.ActionHistory{Type: h.Type, Result: optStr(h.Result), Text: optStr(h.Text), Evidence: optStr(h.Evidence),
				Actor: optStr(h.Actor), At: h.At, Cycle: h.Cycle, EventID: h.EventID})
		}
		out.Items = append(out.Items, v)
		out.BasisSeq = max(out.BasisSeq, a.BasisSeq)
		if a.Status != dom.ActionEffective {
			out.Summary.Open++
		}
		for _, f := range v.Flags {
			switch f {
			case dom.FlagOverdue:
				out.Summary.Overdue++
			case dom.FlagIneffective:
				out.Summary.Ineffective++
			case dom.FlagHangingControl:
				out.Summary.HangingTemporary++
			case dom.FlagEvaluationDue:
				out.Summary.EvaluationDue++
			}
		}
		// Организационная память: меры, по которым уже что-то сделано (FR-138).
		if len(a.History) > 0 {
			outcome := "in_progress"
			switch {
			case a.Status == dom.ActionEffective && a.Failures() == 0:
				outcome = "helped"
			case a.Status == dom.ActionEffective:
				outcome = "helped_after_retry"
			case a.Failures() > 0:
				outcome = "not_helped"
			}
			out.Memory = append(out.Memory, analysisapp.MemoryEntry{ActionID: a.ActionID, IncidentID: a.IncidentID, Title: a.Title, ActionType: a.ActionType,
				Direction: a.Direction, CauseCategory: v.CauseCategory, Factor: v.Factor, Outcome: outcome, Cycles: a.Cycle})
		}
	}
	if out.BasisSeq == 0 {
		out.BasisSeq = c.Seq()
	}
	slices.SortStableFunc(out.Items, func(a, b analysisapp.CorrectiveActionView) int { return b.AssignedAt.Compare(a.AssignedAt) })
	out.Recurring = c.recurring(withAction)
	out.Summary.Recurring = len(out.Recurring)
	return out
}

// recurring — повторяющиеся проблемы (FR-138): вид дефекта × узел, где он
// возник, от двух подтверждённых несоответствий.
func (c *Ctx) recurring(withAction map[string]bool) []analysisapp.RecurringProblem {
	type acc struct {
		defect, step string
		ncs          []string
		action       bool
	}
	var order []string
	by := map[string]*acc{}
	for _, n := range c.M.NCs {
		if len(n.Spec.Defects) == 0 || n.ConfirmedAt.After(c.T) {
			continue
		}
		step := "welding.weld"
		if n.Spec.Component != "" {
			step = "incoming.zt1_lot_acceptance"
		}
		k := n.Spec.Defects[0].Kind + "|" + step
		if by[k] == nil {
			by[k] = &acc{defect: n.Spec.Defects[0].Kind, step: step}
			order = append(order, k)
		}
		by[k].ncs = append(by[k].ncs, n.ID)
		for _, in := range c.M.Incidents {
			if withAction[in.Spec.ID] && slices.Contains(c.incidentNCs(in), n.ID) {
				by[k].action = true
			}
		}
	}
	out := []analysisapp.RecurringProblem{}
	for _, k := range order {
		if a := by[k]; len(a.ncs) >= 2 {
			out = append(out, analysisapp.RecurringProblem{DefectType: a.defect, StepKey: a.step, Count: len(a.ncs), NCIDs: a.ncs, WithAction: a.action})
		}
	}
	return out
}

// incidentNCs — несоответствия, связанные с инцидентом: изделие входило в его
// область или несоответствие по той же партии.
func (c *Ctx) incidentNCs(in *Incident) []string {
	var out []string
	for _, n := range c.M.NCs {
		for _, id := range n.Spec.AllItems() {
			for _, v := range in.Versions {
				if _, ok := v.Status[id]; ok && !slices.Contains(out, n.ID) {
					out = append(out, n.ID)
				}
			}
		}
		if n.Spec.Lot != "" && slices.Contains(in.Spec.Factors, n.Spec.Lot) && !slices.Contains(out, n.ID) {
			out = append(out, n.ID)
		}
	}
	return out
}

// deficitRule — вид недостающих сведений в расследованиях главной истории.
type deficitRule struct {
	kind, place, incident string
	ncs                   []string
}

// Где не хватало сведений (как в разборах мира): журнала ИС-2 — у Ф-017
// (шлюз копил журнал) и Ф-019 (записи потеряны); наблюдения корня шва после
// сварки — у Ф-021, Ф-023 (камера КТ-3 видит только лицевую сторону);
// наблюдения тела кольца до сварки — у НС-04, НС-05 (ЗТ-1 без рентгена).
var deficitRules = []deficitRule{
	{kind: dom.MissingAfter, place: "welding.kt3_camera", incident: "RS-01", ncs: []string{"NC-02", "NC-03"}},
	{kind: dom.MissingBefore, place: "incoming.zt1_lot_acceptance", incident: "RS-02", ncs: []string{"NC-04", "NC-05"}},
	{kind: dom.MissingEquipmentLog, place: "IS-2", incident: "RS-01", ncs: []string{"NC-01", "NC-04"}},
}

func (c *Ctx) dataDeficit() analysisapp.DataDeficitMap {
	out := analysisapp.DataDeficitMap{Rows: []analysisapp.DeficitRow{}, BasisSeq: c.Seq()}
	for _, n := range c.M.NCs {
		if !n.Spec.Signal.IsZero() && !n.SignalAt.After(c.T) {
			out.Investigations++
		}
	}
	visible := func(id string) bool {
		n := c.M.ncByID[id]
		return n != nil && !n.SignalAt.After(c.T)
	}
	// Сначала — по порядку видов контракта (dom.MissingKinds).
	for _, kind := range dom.MissingKinds {
		for _, r := range deficitRules {
			if r.kind != kind {
				continue
			}
			row := analysisapp.DeficitRow{Kind: r.kind, Places: []analysisapp.DeficitPlace{}, NCIDs: []string{}, IncidentIDs: []string{}, Digitization: dom.Digitization(r.kind)}
			for _, id := range r.ncs {
				if visible(id) {
					row.NCIDs = append(row.NCIDs, id)
				}
			}
			if len(row.NCIDs) == 0 {
				continue
			}
			row.Investigations = len(row.NCIDs)
			row.Places = append(row.Places, analysisapp.DeficitPlace{Place: r.place, Count: len(row.NCIDs)})
			for _, in := range c.M.Incidents {
				v := in.VersionAt(c.T)
				if in.Spec.ID != r.incident || v == nil {
					continue
				}
				row.IncidentIDs = append(row.IncidentIDs, in.Spec.ID)
				if from, to := in.Versions[0].Size(), v.Size(); to < from {
					row.Estimate = &analysisapp.DeficitEstimate{FromTenths: from * 10, ToTenths: to * 10, Incidents: 1,
						Text: "в среднем с " + dom.TenthsText(from*10) + " до " + dom.TenthsText(to*10) + " деталей"}
				}
			}
			for _, s := range c.M.Loop.Suggestions {
				if s.Missing == r.kind && !s.At.After(c.T) {
					row.SuggestionID = ptr(s.ID)
				}
			}
			out.Rows = append(out.Rows, row)
		}
	}
	return out
}
