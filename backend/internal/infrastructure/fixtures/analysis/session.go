package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	app "ant/internal/application/analysis"
	"ant/internal/application/platform"
	dom "ant/internal/domain/analysis"
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

// withScope — область риска со ступенями сессии. Ответ мира несёт все
// изделия, когда-либо входившие в область (исключённые — known = excluded),
// поэтому текущая область — изделия не «исключено»: сужение исключает их с
// основанием, расширение возвращает или добавляет; новая версия — размер
// текущей области, исключённые и добавленные изделия, доказательства словами,
// автор с именем.
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
		rs = applyScope(ctx, rt, rs, ch, f, m)
	}
	return rs
}

// applyScope — одна ступень сужения или расширения поверх области rs.
func applyScope(ctx context.Context, rt *loader.Runtime, rs app.RiskScope, ch scopeChange, f loader.Fact, m platform.Moment) app.RiskScope {
	var added, removed []string
	for _, id := range ch.In.ItemIDs {
		i := slices.IndexFunc(rs.Items, func(x app.ScopeItem) bool { return rt.Local(ctx, x.ItemID) == rt.Local(ctx, id) })
		switch {
		case ch.Narrow && i >= 0 && rs.Items[i].Known != "excluded":
			rs.Items[i].Known, rs.Items[i].Action = "excluded", "release"
			removed = append(removed, rs.Items[i].ItemID)
		case !ch.Narrow && i >= 0 && rs.Items[i].Known == "excluded":
			rs.Items[i].Known, rs.Items[i].Action = "suspect", "check"
			added = append(added, rs.Items[i].ItemID)
		case !ch.Narrow && i < 0:
			rs.Items = append(rs.Items, app.ScopeItem{ItemID: id, Label: labelOf(rt.Local(ctx, id)), Known: "suspect", Action: "check", Location: "in_production"})
			added = append(added, id)
		}
	}
	actor := f.Actor
	change := "expanded"
	if ch.Narrow {
		change = "narrowed"
	}
	reason := app.Reason{Text: ch.In.Reason.Text, Code: ch.In.Reason.Code}
	cur := inScope(rs.Items)
	v := app.ScopeVersion{ScopeVersion: len(rs.Versions) + 1, Change: change, Size: len(cur), RecordedAt: f.At, Author: &actor, Reason: &reason,
		EvidenceEventIDs: append([]string{}, ch.In.EvidenceEventIDs...), Breakdown: breakdown(cur)}
	v.ItemsAdded, v.ItemsRemoved = nonNil(added), nonNil(removed)
	v.Evidence = evidenceOf(ctx, rt, rs, ch.In.EvidenceEventIDs, f, m)
	if name := personName(ctx, rt, actor); name != "" {
		v.AuthorName = &name
	}
	if ch.Narrow {
		v.SignedBy, v.KeyClass = &actor, ptr("personal")
	}
	at := f.At
	v.Trigger = &app.ScopeTrigger{Kind: "human", Label: "Решение: " + nameOr(personName(ctx, rt, actor), actor) + " — " + reason.Text, ReceivedAt: &at, OccurredAt: &at}
	rs.Versions = append(rs.Versions, v)
	return rs
}

// inScope — изделия текущей области (не исключённые).
func inScope(items []app.ScopeItem) []app.ScopeItem {
	var out []app.ScopeItem
	for _, x := range items {
		if x.Known != "excluded" {
			out = append(out, x)
		}
	}
	return out
}

// evidenceOf — доказательства ступени словами: те же записи, что в
// обстоятельствах несоответствий инцидента и в ступенях мира (тот же
// построитель — генератор мира); запись не найдена — только её id.
func evidenceOf(ctx context.Context, rt *loader.Runtime, rs app.RiskScope, ids []string, f loader.Fact, m platform.Moment) []app.JournalRecordRef {
	out := []app.JournalRecordRef{}
	if len(ids) == 0 {
		return out
	}
	var pool []any
	for _, v := range rs.Versions {
		for _, e := range v.Evidence {
			pool = append(pool, e)
		}
	}
	for _, nc := range rs.NCIDs {
		var c any
		if err := rt.Respond(ctx, "analysis.circumstances.read", map[string]string{"nc_id": nc}, &m, &c); err == nil {
			pool = append(pool, c)
		}
	}
	for _, id := range ids {
		ref := app.JournalRecordRef{EventID: id, EventType: "journal.record", OccurredAt: f.At}
		if found, ok := findRecord(pool, id); ok {
			ref = found
		}
		out = append(out, ref)
	}
	return out
}

// findRecord — запись журнала с event_id в любом месте ответов pool; из
// нескольких — с текстом.
func findRecord(pool []any, id string) (app.JournalRecordRef, bool) {
	var best app.JournalRecordRef
	found := false
	var walk func(any)
	walk = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if v["event_id"] == id && v["event_type"] != nil {
				var r app.JournalRecordRef
				if b, err := json.Marshal(v); err == nil && json.Unmarshal(b, &r) == nil && (!found || (best.Text == nil && r.Text != nil)) {
					best, found = r, true
				}
			}
			for _, y := range v {
				walk(y)
			}
		case []any:
			for _, y := range v {
				walk(y)
			}
		}
	}
	for _, p := range pool {
		b, err := json.Marshal(p)
		if err != nil {
			continue
		}
		var g any
		if json.Unmarshal(b, &g) == nil {
			walk(g)
		}
	}
	return best, found
}

// personName — имя демо-персоны по псевдониму (access.persona.list мира).
func personName(ctx context.Context, rt *loader.Runtime, id string) string {
	var l struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if rt.Respond(ctx, "access.persona.list", nil, nil, &l) == nil {
		for _, p := range l.Items {
			if p.ID == id {
				return p.Name
			}
		}
	}
	return ""
}

func nameOr(name, id string) string {
	if name != "" {
		return name
	}
	return id
}

// checkScope — гарды сужения и расширения те же, что у live
// (domain/analysis.GuardNarrow, GuardExpand; AD-27): сужение — только с
// доказательствами и текстом основания и только изделий текущей области,
// расширение — с основанием и хотя бы одно изделие вне области; закрытый
// инцидент — отказ.
func checkScope(ctx context.Context, incidentID string, narrow bool, in app.ChangeScope) error {
	rs, err := Adapter{}.RiskScope(ctx, incidentID, platform.Moment{})
	if err != nil {
		return err
	}
	rt, err := loader.Default()
	if err != nil {
		return err
	}
	v := dom.IncidentRecord{IncidentID: incidentID, Members: map[string]dom.MemberRecord{}}
	for _, x := range rs.Items {
		v.Members[rt.Local(ctx, x.ItemID)] = dom.MemberRecord{Status: x.Known, Action: x.Action}
	}
	if l, err := (Adapter{}).Incidents(ctx, platform.Moment{}, platform.Page{}); err == nil {
		for _, x := range l.Items {
			if x.IncidentID == incidentID && x.Status == "closed" {
				v.Closed = true
			}
		}
	}
	items := make([]string, len(in.ItemIDs))
	for i, id := range in.ItemIDs {
		items[i] = rt.Local(ctx, id)
	}
	if narrow {
		err = dom.GuardNarrow(v, items, in.EvidenceEventIDs, in.Reason.Text)
	} else {
		err = dom.GuardExpand(v, items, in.Reason.Text)
	}
	if err == nil {
		return nil
	}
	if pe, ok := platform.AsError(err); ok {
		return pe
	}
	return err
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
			x.Size, x.ScopeVersion = len(inScope(rs.Items)), len(rs.Versions)
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
