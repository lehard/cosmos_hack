package analysis

import (
	"context"
	"fmt"
	"slices"
	"time"

	app "ant/internal/application/analysis"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
)

// Сессионное наложение стола технолога (FR-129, FR-60…FR-64): назначенная
// мера, сужение и расширение области риска, вывод о причине и закрытие мир
// заготовок не меняют, но стенд показывает их до сброса прогона — мера
// появляется в списке мер со статусом «назначена», область риска получает
// новую ступень с основанием и доказательствами, стадия инцидента двигается.

// kindIncident — вид объекта фактов инцидента.
const kindIncident = "incident"

// record — команда над инцидентом: квитанция и факт сессии.
func record(ctx context.Context, op, id string, meta platform.CommandMeta, body any) (platform.Receipt, error) {
	rt, err := loader.Default()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Record(ctx, op, loader.ObjectRef{Kind: kindIncident, ID: id}, meta, body,
		loader.Change{Entity: string(platform.EntityIncident), ID: "global"}, loader.Change{Entity: string(platform.EntityItem), ID: "global"})
}

// recordNC — команда над гипотезами несоответствия: квитанция и факт сессии.
func recordNC(ctx context.Context, op, ncID string, meta platform.CommandMeta, body any) (platform.Receipt, error) {
	rt, err := loader.Default()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Record(ctx, op, loader.ObjectRef{Kind: string(platform.EntityNonconformity), ID: ncID}, meta, body,
		loader.Change{Entity: string(platform.EntityIncident), ID: "global"})
}

// incidentFacts — факты инцидентов сессии на момент m: id (без префикса) → факты.
func incidentFacts(ctx context.Context, m platform.Moment) (map[string][]loader.Fact, *loader.Runtime) {
	rt, err := loader.Default()
	if err != nil {
		return nil, nil
	}
	out := map[string][]loader.Fact{}
	for _, f := range rt.Facts(ctx, &m, kindIncident) {
		out[f.ID] = append(out[f.ID], f)
	}
	return out, rt
}

// withActions — список мер с мерами сессии: новые сверху, сводка пересчитана.
func withActions(ctx context.Context, l app.CorrectiveActionList, m platform.Moment) app.CorrectiveActionList {
	fs, rt := incidentFacts(ctx, m)
	if len(fs) == 0 {
		return l
	}
	var mine []app.CorrectiveActionView
	for id, list := range fs {
		for _, f := range list {
			in, ok := f.Body.(app.AssignAction)
			if !ok {
				continue
			}
			mine = append(mine, actionOf(rt.Local(ctx, id), in, f))
		}
	}
	slices.SortFunc(mine, func(a, b app.CorrectiveActionView) int { return int(b.BasisSeq - a.BasisSeq) })
	l.Items = append(mine, l.Items...)
	l.Summary.Open += len(mine)
	return l
}

// actionOf — мера сессии: назначена, первая попытка, план проверки эффективности.
func actionOf(incidentID string, in app.AssignAction, f loader.Fact) app.CorrectiveActionView {
	actor := f.Actor
	evt := fmt.Sprintf("EV-SESSION-%d", f.Seq)
	v := app.CorrectiveActionView{ActionID: fmt.Sprintf("CA-S-%d", f.Seq), IncidentID: incidentID, ActionType: in.ActionType, Direction: in.Direction,
		Owner: in.OwnerID, Status: "assigned", Cycle: 1, AssignedAt: f.At, Flags: []string{}, BasisSeq: f.Seq,
		Plan: app.EffectivenessPlanView{Metric: in.EffectivenessPlan.Metric, Baseline: in.EffectivenessPlan.Baseline, WindowDays: in.EffectivenessPlan.WindowDays,
			SuccessCriterion: in.EffectivenessPlan.SuccessCriterion},
		History: []app.ActionHistory{{Type: "assigned", Actor: &actor, At: f.At, Cycle: 1, EventID: evt}}}
	if in.EffectivenessPlan.EnhancedControl != "" {
		v.Plan.EnhancedControl = &in.EffectivenessPlan.EnhancedControl
	}
	if in.Title != "" {
		v.Title = &in.Title
	}
	if in.SuggestionID != "" {
		v.SuggestionID = &in.SuggestionID
	}
	if t, err := time.Parse(time.RFC3339, in.DueAt); err == nil {
		v.DueAt = &t
	}
	return v
}

// scopeChange — ступень области риска сессии.
type scopeChange struct {
	Narrow bool
	In     app.ChangeScope
}

// withScope — область риска со ступенями сессии: изделия исключены или
// добавлены, новая версия с основанием, автором и доказательствами.
func withScope(ctx context.Context, rs app.RiskScope, m platform.Moment) app.RiskScope {
	fs, rt := incidentFacts(ctx, m)
	list := fs[rt.Local(ctx, rs.IncidentID)]
	if len(list) == 0 {
		return rs
	}
	rs.Versions, rs.Items = slices.Clone(rs.Versions), slices.Clone(rs.Items)
	for _, f := range list {
		ch, ok := f.Body.(scopeChange)
		if !ok {
			continue
		}
		var added, removed []string
		for _, id := range ch.In.ItemIDs {
			i := slices.IndexFunc(rs.Items, func(x app.ScopeItem) bool { return x.ItemID == id || rt.Local(ctx, x.ItemID) == rt.Local(ctx, id) })
			switch {
			case ch.Narrow && i >= 0:
				removed = append(removed, rs.Items[i].ItemID)
				rs.Items = slices.Delete(rs.Items, i, i+1)
			case !ch.Narrow && i < 0:
				added = append(added, id)
				rs.Items = append(rs.Items, app.ScopeItem{ItemID: id, Label: labelOf(rt.Local(ctx, id)), Known: "suspect", Action: "check", Location: "in_production"})
			}
		}
		actor := f.Actor
		change := "expanded"
		if ch.Narrow {
			change = "narrowed"
		}
		reason := app.Reason{Text: ch.In.Reason.Text, Code: ch.In.Reason.Code}
		v := app.ScopeVersion{ScopeVersion: len(rs.Versions) + 1, Change: change, Size: len(rs.Items), RecordedAt: f.At, Author: &actor, Reason: &reason,
			EvidenceEventIDs: append([]string{}, ch.In.EvidenceEventIDs...), Breakdown: breakdown(rs.Items)}
		v.ItemsAdded, v.ItemsRemoved = nonNil(added), nonNil(removed)
		v.Evidence = []app.JournalRecordRef{}
		for _, id := range ch.In.EvidenceEventIDs {
			v.Evidence = append(v.Evidence, app.JournalRecordRef{EventID: id, EventType: "journal.record", OccurredAt: f.At})
		}
		if ch.Narrow {
			v.SignedBy, v.KeyClass = &actor, ptr("personal")
		}
		rs.Versions = append(rs.Versions, v)
	}
	return rs
}

// withIncidents — инциденты со ступенями области и стадией расследования сессии.
func withIncidents(ctx context.Context, l app.IncidentList, m platform.Moment) app.IncidentList {
	fs, rt := incidentFacts(ctx, m)
	if len(fs) == 0 {
		return l
	}
	for i, x := range l.Items {
		list := fs[rt.Local(ctx, x.IncidentID)]
		if len(list) == 0 {
			continue
		}
		if rs, err := respond[app.RiskScope](ctx, "analysis.risk_scope.read", incident(x.IncidentID), &m); err == nil {
			rs = withScope(ctx, rs, m)
			x.Size, x.ScopeVersion = len(rs.Items), len(rs.Versions)
			x.Counts = counts(rs.Items)
		}
		for _, f := range list {
			at := f.At
			x.LastEventAt = &at
			switch b := f.Body.(type) {
			case app.ConcludeCause:
				if b.Conclusion == "confirmed" && stageRank(x.Stage) < stageRank("cause_confirmed") {
					x.Stage = "cause_confirmed"
				}
			case app.AssignAction:
				if stageRank(x.Stage) >= stageRank("cause_confirmed") && stageRank(x.Stage) < stageRank("action_assigned") {
					x.Stage = "action_assigned"
				}
			case app.CloseIncident:
				if b.Scope == "investigation" {
					x.Stage, x.CloseBlockers = "closed", []app.CloseBlocker{}
				} else {
					x.Status = "closed"
				}
			}
		}
		l.Items[i] = x
	}
	return l
}

var stages = []string{"scope_defined", "hypothesis", "cause_confirmed", "action_assigned", "effectiveness_check", "closed"}

func stageRank(s string) int { return slices.Index(stages, s) }

func counts(items []app.ScopeItem) app.KnownCountsView {
	var c app.KnownCountsView
	for _, x := range items {
		switch x.Known {
		case "confirmed":
			c.Confirmed++
		case "suspect":
			c.Suspect++
		case "excluded":
			c.Excluded++
		default:
			c.Unknown++
		}
	}
	return c
}

func breakdown(items []app.ScopeItem) app.ScopeBreakdown {
	var b app.ScopeBreakdown
	for _, x := range items {
		switch x.Location {
		case "moved_on":
			b.MovedOn++
		case "assembled":
			b.Assembled++
		case "shipped":
			b.Shipped++
		default:
			b.InProduction++
		}
	}
	return b
}

// labelOf — номер детали для людей по id изделия: «ENT01:F-017» → «Ф-017».
func labelOf(id string) string {
	if i := len(id) - 5; i >= 0 && id[i:i+2] == "F-" {
		return "Ф-" + id[i+2:]
	}
	return id
}

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

func ptr[T any](v T) *T { return &v }

// withHypotheses — гипотезы несоответствия с решениями сессии: записанная
// гипотеза появляется «записана», отклонённая — «отклонена», запрос
// измерения и вывод о причине — строкой истории гипотезы.
func withHypotheses(ctx context.Context, h app.Hypotheses, m platform.Moment) app.Hypotheses {
	rt, err := loader.Default()
	if err != nil {
		return h
	}
	fs := rt.FactsOf(ctx, &m, string(platform.EntityNonconformity), h.NCID)
	all, _ := incidentFacts(ctx, m)
	for _, list := range all {
		for _, f := range list {
			if c, ok := f.Body.(app.ConcludeCause); ok && c.HypothesisID != "" {
				fs = append(fs, f)
			}
		}
	}
	slices.SortStableFunc(fs, func(a, b loader.Fact) int { return int(a.Seq - b.Seq) })
	if len(fs) == 0 {
		return h
	}
	h.Hypotheses = slices.Clone(h.Hypotheses)
	find := func(id string) *app.Hypothesis {
		for i := range h.Hypotheses {
			if h.Hypotheses[i].HypothesisID == id || rt.Local(ctx, h.Hypotheses[i].HypothesisID) == rt.Local(ctx, id) {
				h.Hypotheses[i].History = slices.Clone(h.Hypotheses[i].History)
				return &h.Hypotheses[i]
			}
		}
		return nil
	}
	for _, f := range fs {
		evt := fmt.Sprintf("EV-SESSION-%d", f.Seq)
		switch b := f.Body.(type) {
		case app.RecordHypothesis:
			branch, st := b.Branch, b.Statement
			h.Hypotheses = append(h.Hypotheses, app.Hypothesis{HypothesisID: fmt.Sprintf("H-S-%d", f.Seq), Category: b.Category, Branch: &branch, Statement: &st,
				Status: "recorded", Supporting: []app.JournalRecordRef{}, Contradicting: []app.JournalRecordRef{},
				History: []app.HypothesisChange{{At: f.At, EventID: &evt, Text: "Гипотеза записана: " + f.Actor}}})
		case app.RejectHypothesis:
			if x := find(b.HypothesisID); x != nil {
				x.Status = "rejected"
				x.History = append(x.History, app.HypothesisChange{At: f.At, EventID: &evt, Text: "Отклонена: " + b.Reason.Text})
			}
		case app.RequestMeasurement:
			if x := find(b.HypothesisID); x != nil {
				x.History = append(x.History, app.HypothesisChange{At: f.At, EventID: &evt, Text: "Запрошено: " + b.What})
			}
		case app.ConcludeCause:
			if x := find(b.HypothesisID); x != nil && b.Conclusion == "confirmed" {
				x.Status = "confirmed"
				x.History = append(x.History, app.HypothesisChange{At: f.At, EventID: &evt, Text: "Причина подтверждена: " + b.Verification})
			}
		}
	}
	return h
}
