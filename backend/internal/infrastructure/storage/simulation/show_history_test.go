package simulation

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	appanalysis "ant/internal/application/analysis"
	"ant/internal/application/analysis/analysistest"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	sim "ant/internal/domain/simulation"
)

// TestShowOldInvestigation — завершённое расследование ИС-2 позапрошлой недели
// в истории SHOW-IS2 (стол технолога не пуст): сварки Ф-301…Ф-303 на ИС-2,
// журнал тока, КТ-3 и рентген — из генератора; решения — как в карточке
// (сужения, гипотеза «почему не обнаружили», проверка контрольным образцом,
// причины по обеим веткам, закрытие области, меры, внедрение одной). Итог:
// область 3 → 2 → 1 с основаниями, обе гипотезы подтверждены с историей,
// стадия «проверка эффективности», инцидент не открыт (живой откроет свой).
func TestShowOldInvestigation(t *testing.T) {
	ctx := context.Background()
	f := NewFiles(filepath.Join(repo, "scenarios"))
	b, err := f.Bundle(ctx, "SHOW-IS2")
	if err != nil {
		t.Fatal(err)
	}
	p, err := sim.Generate(b, sim.Params{RunID: GoldenRunID("SHOW-IS2", b.Run.Seed)})
	if err != nil {
		t.Fatal(err)
	}
	j := enginemem.New(nil)
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}
	w := analysistest.New(j, codec, "show-old")
	item := func(id string) string { return "ENT01:" + id }
	local := func(x string) string { v, _ := p.IDs.ExpandString("{local:"+x+"}", false); return v }
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	old := func(id string) bool { return id == "F-301" || id == "F-302" || id == "F-303" }
	eq := map[string]string{"WS-1": "IS-1", "WS-2": "IS-2"}
	type rec struct {
		tp     catalog.Type
		it, st string
		at     time.Time
		data   any
		label  string
	}
	var queue []rec
	for _, x := range p.Truth.Welds {
		if !old(x.Item) {
			continue
		}
		queue = append(queue,
			rec{tp: catalog.OperationRunStarted, it: item(x.Item), at: x.Start, data: map[string]any{"operation_run_id": local(x.Run),
				"step_key": analysistest.StepWeld, "operation_code": "030", "equipment_id": eq[x.Station], "operator_id": x.Welder, "program_ref": analysistest.Program}},
			rec{tp: catalog.OperationRunFinished, it: item(x.Item), at: x.End, data: map[string]any{"operation_run_id": local(x.Run), "completion": "completed"}})
	}
	confirmAt := time.Date(2026, 9, 8, 9, 0, 0, 0, analysistest.MSK).UTC()
	for _, e := range p.Emissions {
		if !e.OccurredAt.Before(confirmAt) {
			continue
		}
		var env struct {
			Data map[string]any `json:"data"`
		}
		must(json.Unmarshal(e.Event, &env))
		switch {
		case strings.HasPrefix(e.EventType, "equipment."):
			queue = append(queue, rec{tp: catalog.Type(e.EventType), st: "equipment:" + env.Data["equipment_id"].(string), at: e.OccurredAt, data: env.Data, label: e.Label})
		case e.EventType == string(catalog.InspectionResultRecorded) && old(e.Item):
			queue = append(queue, rec{tp: catalog.InspectionResultRecorded, it: item(e.Item), at: e.OccurredAt, data: env.Data, label: e.Label})
		}
	}
	slices.SortStableFunc(queue, func(a, b rec) int { return a.at.Compare(b.at) })
	ev := map[string]string{}
	for _, r := range queue {
		id, err := w.Fact(ctx, r.tp, r.it, r.st, r.at, r.data)
		must(err)
		if r.label != "" {
			ev[r.label] = id
		}
	}
	for _, l := range []string{"F-301/weld/0907-0510", "F-303/xray", "F-302/xray", "F-302/kt3"} {
		if ev[l] == "" {
			t.Fatalf("нет записи %s в истории генератора", l)
		}
	}
	const nc = "NC-F302"
	_, err = w.Fact(ctx, catalog.DecisionNonconformityConfirmed, item("F-302"), "", confirmAt, map[string]any{"nc_id": nc,
		"defect_type_code": "W-BURNTHRU", "severity": "critical", "signal_ids": []string{"SIG-F302"}, "reason": map[string]any{"text": "РК-0907-02: прожог корня У2"}})
	must(err)
	must(w.Settle(ctx))

	at := func(h, m int) time.Time { return time.Date(2026, 9, 8, h, m, 0, 0, analysistest.MSK).UTC() }
	svc := func(now time.Time) *appanalysis.Service { return w.Service(j, now) }
	m := platform.Moment{Axis: platform.AxisOccurred}
	tec := analysistest.Principal(ctx, "TEC-01")
	hdr := func(basis int64) platform.CommandHeader {
		return platform.CommandHeader{CommandID: w.ID("command"), BasisSeq: basis}
	}
	incs, err := svc(at(9, 5)).Incidents(ctx, m, platform.Page{})
	if err != nil || len(incs.Items) != 1 {
		t.Fatalf("инциденты: %+v %v", incs.Items, err)
	}
	inc := incs.Items[0].IncidentID
	if incs.Items[0].PrimaryNCID == nil || *incs.Items[0].PrimaryNCID != nc {
		t.Fatalf("ссылка карточки INC-F302 ищет инцидент по primary_nc_id: %+v", incs.Items[0])
	}
	scope := func(now time.Time) appanalysis.RiskScope {
		t.Helper()
		rs, err := svc(now).RiskScope(ctx, inc, m)
		must(err)
		return rs
	}
	narrow := func(now time.Time, it, evidence, reason string) {
		t.Helper()
		rs := scope(now)
		_, err := svc(now).NarrowScope(tec, inc, appanalysis.ChangeScope{CommandHeader: hdr(rs.BasisSeq), ItemIDs: []string{item(it)},
			EvidenceEventIDs: []string{ev[evidence]}, ReleaseContainment: true, Reason: appanalysis.Reason{Text: reason}})
		must(err)
		must(w.Settle(ctx))
	}
	narrow(at(9, 30), "F-301", "F-301/weld/0907-0510", "Журнал ИС-2 за сварку Ф-301 в уставке")
	narrow(at(9, 35), "F-303", "F-303/xray", "Рентген РК-0907-03 без признаков")

	hs, err := svc(at(9, 40)).Hypotheses(ctx, nc, m)
	must(err)
	var hypEq string
	for _, h := range hs.Hypotheses {
		if h.Category == "equipment" && hypEq == "" {
			hypEq = h.HypothesisID
		}
	}
	if hypEq == "" {
		t.Fatalf("нет гипотезы «оборудование» (ссылка HYP-F302-EQ): %+v", hs.Hypotheses)
	}
	_, err = svc(at(10, 0)).RequestMeasurement(tec, nc, appanalysis.RequestMeasurement{CommandHeader: hdr(0), HypothesisID: hypEq,
		What: "Контрольный образец на ИС-2 при уставке 160 А"})
	must(err)
	_, err = svc(at(10, 5)).RecordHypothesis(tec, nc, appanalysis.RecordHypothesis{CommandHeader: hdr(0), IncidentID: inc, Branch: "why_missed",
		Category: "documentation", Statement: "Камера КТ-3 снимает лицевую сторону шва: прожог корня видит только рентген",
		SupportingEventIDs: []string{ev["F-302/kt3"], ev["F-302/xray"]}})
	must(err)
	must(w.Settle(ctx))
	lab := analysistest.Principal(ctx, "INS-02")
	_, err = svc(at(11, 30)).RecordMeasurement(lab, nc, appanalysis.RecordMeasurement{CommandHeader: hdr(0), HypothesisID: hypEq, Outcome: "supports",
		Result: "КО-0908 на ИС-2 при уставке 160 А: по журналу 179–181 А; прожог корня повторён"})
	must(err)
	must(w.Settle(ctx))
	// Показ, шаг G: после результата контрольного образца — «Подтвердить причину».
	incs, err = svc(at(11, 35)).Incidents(ctx, m, platform.Page{})
	must(err)
	if st := incs.Items[0].InvestigationState; st.NextStep == nil || !strings.HasPrefix(*st.NextStep, "Подтвердить причину") {
		t.Fatalf("что дальше после результата проверки: %+v", incs.Items[0].InvestigationState)
	}
	hs, err = svc(at(11, 35)).Hypotheses(ctx, nc, m)
	must(err)
	var hypMiss string
	for _, h := range hs.Hypotheses {
		if h.Branch != nil && *h.Branch == "why_missed" {
			hypMiss = h.HypothesisID
		}
		if h.HypothesisID == hypEq && !slices.ContainsFunc(h.History, func(c appanalysis.HypothesisChange) bool {
			return strings.HasPrefix(c.Text, "Результат проверки «Контрольный образец")
		}) {
			t.Fatalf("история гипотезы «оборудование» без результата проверки: %+v", h.History)
		}
	}
	if hypMiss == "" {
		t.Fatalf("нет гипотезы «почему не обнаружили» (ссылка HYP-F302-MISS): %+v", hs.Hypotheses)
	}
	conclude := func(now time.Time, hyp, branch, category string) {
		t.Helper()
		_, err := svc(now).ConcludeCause(tec, inc, appanalysis.ConcludeCause{CommandHeader: hdr(0), HypothesisID: hyp, Branch: branch, NCIDs: []string{nc},
			Conclusion: "confirmed", Category: category, Verification: "КО-0908, журнал ИС-2, РК-0907-02", Reason: appanalysis.Reason{Text: "Решение технолога"}})
		must(err)
		must(w.Settle(ctx))
	}
	conclude(at(12, 0), hypMiss, "why_missed", "documentation")
	conclude(at(12, 5), hypEq, "why_made", "equipment")
	_, err = svc(at(13, 0)).CloseIncident(tec, inc, appanalysis.CloseIncident{CommandHeader: hdr(0), Scope: "risk_scope", Summary: "Область закрыта: было 3, подтверждено 1"})
	must(err)
	assign := func(now time.Time, direction, title string) {
		t.Helper()
		_, err := svc(now).AssignAction(tec, inc, appanalysis.AssignAction{CommandHeader: hdr(0), ActionType: "corrective_action", Direction: direction,
			OwnerID: "TEC-01", Title: title, EffectivenessPlan: appanalysis.EffectivenessPlanInput{Metric: "сварки ИС-2 вне уставки",
				Baseline: "2 из 3", WindowDays: 21, SuccessCriterion: "ни одной сварки вне уставки"}})
		must(err)
		must(w.Settle(ctx))
	}
	assign(at(13, 10), "prevent_occurrence", "Ремонт регулятора тока ИС-2")
	assign(at(13, 15), "improve_detection", "Проверка корня на КТ-3")
	acts, err := svc(at(13, 20)).CorrectiveActions(ctx, m)
	must(err)
	var actEq string
	for _, a := range acts.Items {
		if a.IncidentID == inc && a.Direction == "prevent_occurrence" {
			actEq = a.ActionID
		}
	}
	if actEq == "" {
		t.Fatalf("нет меры по оборудованию (ссылка ACT-F302-EQ): %+v", acts.Items)
	}
	_, err = svc(time.Date(2026, 9, 10, 16, 0, 0, 0, analysistest.MSK)).ImplementAction(tec, inc, actEq, appanalysis.ImplementAction{CommandHeader: hdr(0), Note: "Регулятор отремонтирован"})
	must(err)
	must(w.Settle(ctx))

	// Итог — как увидит стол технолога в показе (Пн 21.09, чтение без run_id).
	now := time.Date(2026, 9, 21, 8, 0, 0, 0, analysistest.MSK)
	incs, err = svc(now).Incidents(ctx, m, platform.Page{})
	must(err)
	x := incs.Items[0]
	if x.Status != "closed" || x.InvestigationState.Stage != "effectiveness_check" || x.InvestigationState.NextStep == nil {
		t.Fatalf("расследование позапрошлой недели: %+v %+v", x, x.InvestigationState)
	}
	var sizes []int
	for _, v := range scope(now).Versions {
		sizes = append(sizes, v.Size)
		if v.Reason == nil || v.Reason.Text == "" || len(v.EvidenceEventIDs) == 0 {
			t.Fatalf("версия без основания: %+v", v)
		}
	}
	if !slices.Equal(sizes, []int{3, 2, 1}) {
		t.Fatalf("область %v, ждали 3 → 2 → 1", sizes)
	}
	hs, err = svc(now).Hypotheses(ctx, nc, m)
	must(err)
	for _, id := range []string{hypEq, hypMiss} {
		i := slices.IndexFunc(hs.Hypotheses, func(h appanalysis.Hypothesis) bool { return h.HypothesisID == id })
		if i < 0 || hs.Hypotheses[i].Status != "confirmed" || len(hs.Hypotheses[i].History) == 0 {
			t.Fatalf("гипотеза %s: %+v", id, hs.Hypotheses)
		}
	}
}
