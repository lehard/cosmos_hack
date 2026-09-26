package simulation

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"strconv"
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

// TestShowIncidentScope — область инцидента ИС-2 показа SHOW-IS2 по данным
// генератора (вариант «а» «Процесса»): сварки истории и живой партии,
// журналы источников (вовремя и опоздавшая пачка ИС-2) — через межизделийную
// стадию и проекции analysis на журнале в памяти. Подтверждение НС Ф-003 —
// v1 = 34 (всё после последней годной на ИС-2 — Ф-202); сужение по
// предложению «исключить выполненные на ИС-1» — 13; опоздавший журнал ИС-2 и
// сужение до выхода тока из уставки (как в карточке) — 6: Ф-128…Ф-131, Ф-002, Ф-003.
func TestShowIncidentScope(t *testing.T) {
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
	w := analysistest.New(j, codec, "show")
	item := func(id string) string { return "ENT01:" + id }
	local := func(x string) string { v, _ := p.IDs.ExpandString("{local:"+x+"}", false); return v }
	type rec struct {
		tp         catalog.Type
		it, stream string
		at         time.Time
		data       any
		label      string
	}
	var queue []rec
	fact := func(tp catalog.Type, it, stream string, at time.Time, data any) string {
		t.Helper()
		id, err := w.Fact(ctx, tp, it, stream, at, data)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	later := func(tp catalog.Type, it, stream string, at time.Time, data any) {
		queue = append(queue, rec{tp: tp, it: it, stream: stream, at: at, data: data})
	}
	// Сварки до подтверждения НС: история (Ф-201, Ф-202, Ф-101…Ф-131) и живая партия Ф-001…Ф-003.
	var f3 sim.WeldTruth
	for _, x := range p.Truth.Welds {
		if x.Item == "F-003" && x.Rework == "" {
			f3 = x
		}
	}
	confirmAt := f3.End.Add(15 * time.Minute)
	welds := slices.Clone(p.Truth.Welds)
	slices.SortFunc(welds, func(a, b sim.WeldTruth) int { return a.Start.Compare(b.Start) })
	eq := map[string]string{"WS-1": "IS-1", "WS-2": "IS-2"}
	for _, x := range welds {
		if x.Start.After(confirmAt) || x.Rework != "" {
			continue
		}
		later(catalog.OperationRunStarted, item(x.Item), "", x.Start, map[string]any{"operation_run_id": local(x.Run), "step_key": analysistest.StepWeld,
			"operation_code": "030", "equipment_id": eq[x.Station], "operator_id": x.Welder, "program_ref": analysistest.Program})
		later(catalog.OperationRunFinished, item(x.Item), "", x.End, map[string]any{"operation_run_id": local(x.Run), "completion": "completed"})
		switch x.Item {
		case "F-201", "F-202":
			later(catalog.ItemReleaseRecorded, item(x.Item), "", x.End.Add(24*time.Hour), map[string]any{"received_by": "STK-51", "warehouse_id": "WH-FG", "after_rework": false})
		case "F-001":
			later(catalog.DecisionPresentationResolved, item(x.Item), "", x.End.Add(17*time.Minute), map[string]any{"step_key": "welding.zt3_acceptance",
				"closing_point": "ZT-3", "resolution": "accept", "presentation_no": 1, "method_event_ids": []string{}})
		}
	}
	// Журналы источников: вовремя — сразу; опоздавшая пачка ИС-2 — после первого сужения.
	type envelope struct {
		Data json.RawMessage `json:"data"`
	}
	var late []sim.Emission
	byLabel := map[string]string{}
	emit := func(e sim.Emission) {
		var env envelope
		if err := json.Unmarshal(e.Event, &env); err != nil {
			t.Fatal(err)
		}
		var d map[string]any
		_ = json.Unmarshal(env.Data, &d)
		queue = append(queue, rec{tp: catalog.Type(e.EventType), stream: "equipment:" + d["equipment_id"].(string), at: e.OccurredAt, data: d, label: e.Label})
	}
	flush := func() {
		slices.SortStableFunc(queue, func(a, b rec) int { return a.at.Compare(b.at) })
		for _, r := range queue {
			id := fact(r.tp, r.it, r.stream, r.at, r.data)
			if r.label != "" {
				byLabel[r.label] = id
			}
		}
		queue = nil
	}
	for _, e := range p.Emissions {
		if !strings.HasPrefix(e.EventType, "equipment.") || e.OccurredAt.After(confirmAt) {
			continue
		}
		if e.Delivery == "late" {
			late = append(late, e)
			continue
		}
		emit(e)
	}
	flush()
	fact(catalog.DecisionNonconformityConfirmed, item("F-003"), "", confirmAt, map[string]any{"nc_id": "NC-F003",
		"defect_type_code": "W-BURNTHRU", "severity": "critical", "signal_ids": []string{"SIG-F003"}, "reason": map[string]any{"text": "Прожог У2"}})
	if err := w.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	now := confirmAt.Add(10 * time.Minute)
	svc := w.Service(j, now)
	m := platform.Moment{Axis: platform.AxisOccurred}
	incs, err := svc.Incidents(ctx, m, platform.Page{})
	if err != nil || len(incs.Items) != 1 {
		t.Fatalf("инциденты: %+v %v", incs.Items, err)
	}
	inc := incs.Items[0].IncidentID
	scope := func() appanalysis.RiskScope {
		t.Helper()
		rs, err := svc.RiskScope(ctx, inc, m)
		if err != nil {
			t.Fatal(err)
		}
		return rs
	}
	size := func(rs appanalysis.RiskScope) int { return rs.Versions[len(rs.Versions)-1].Size }
	rs := scope()
	if size(rs) != 34 || rs.LastKnownGood == nil || !strings.Contains(rs.LastKnownGood.Label, "F-202") {
		t.Fatalf("v1: %d изделий, отсчёт %+v", size(rs), rs.LastKnownGood)
	}
	opt := func(rs appanalysis.RiskScope, what string) *appanalysis.NarrowOption {
		for i := range rs.NarrowOptions {
			if strings.Contains(rs.NarrowOptions[i].Label, what) {
				return &rs.NarrowOptions[i]
			}
		}
		return nil
	}
	is1 := opt(rs, "на IS-1")
	if is1 == nil || len(is1.ItemIDs) != 21 || !slices.Contains(is1.ItemIDs, item("F-001")) {
		t.Fatalf("предложение сужения по ИС-1: %+v", is1)
	}
	tec := analysistest.Principal(ctx, "TEC-01")
	narrow := func(rs appanalysis.RiskScope, items, evidence []string, reason string) {
		t.Helper()
		if _, err := svc.NarrowScope(tec, inc, appanalysis.ChangeScope{CommandHeader: platform.CommandHeader{CommandID: w.ID("command"), BasisSeq: rs.BasisSeq},
			ItemIDs: items, EvidenceEventIDs: evidence, Reason: appanalysis.Reason{Text: reason}}); err != nil {
			t.Fatal(err)
		}
		if err := w.Settle(ctx); err != nil {
			t.Fatal(err)
		}
	}
	o := *is1
	var ev []string
	for _, e := range o.Evidence {
		ev = append(ev, e.EventID)
	}
	narrow(rs, o.ItemIDs, ev, o.ReasonText)
	if rs = scope(); size(rs) != 13 {
		t.Fatalf("v2: %d изделий", size(rs))
	}
	// Опоздавший журнал ИС-2 и сужение технолога до выхода тока из уставки.
	for _, e := range late {
		emit(e)
	}
	flush()
	if err := w.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	// Предложение второго сужения — по опоздавшему журналу (основание — EV-WS2-0918).
	var pre []string
	for i := 121; i <= 127; i++ {
		pre = append(pre, item("F-"+strconv.Itoa(i)))
	}
	rs = scope()
	drift := opt(rs, "до выхода")
	if drift == nil || !slices.Equal(drift.ItemIDs, pre) || drift.Evidence[0].EventID != byLabel["EV-WS2-0918"] {
		t.Fatalf("предложение сужения по опоздавшему журналу: %+v", rs.NarrowOptions)
	}
	ev = nil
	for _, e := range drift.Evidence {
		ev = append(ev, e.EventID)
	}
	narrow(rs, drift.ItemIDs, ev, drift.ReasonText)
	rs = scope()
	var left []string
	for _, it := range rs.Items {
		if it.Known != "excluded" {
			left = append(left, strings.TrimPrefix(it.ItemID, "ENT01:"))
		}
	}
	slices.Sort(left)
	if want := []string{"F-002", "F-003", "F-128", "F-129", "F-130", "F-131"}; size(rs) != 6 || !slices.Equal(left, want) {
		t.Fatalf("v3: %d изделий %v, ждали %v", size(rs), left, want)
	}
}
