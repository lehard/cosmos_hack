package simulation

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// Stages — этапы маршрута фланца (normative/process/flange-process.bpmn,
// маршрут кейса §1.1: вход → мехобработка → сварка → сборка с установкой
// оборудования → испытание → ОТК → склад). ItemPlan.Until и Skip ссылаются на них.
var Stages = []string{
	"launch", "machining", "remark", "kt2", "cmm", "zt2_presented", "zt2",
	"sent_to_welding", "received_welding", "ring", "edge_prep", "weld", "kt3", "xray", "zt3_presented", "zt3",
	"sent_to_assembly", "received_assembly", "assembly_started", "assembly", "zt4_presented", "zt4",
	"leak_test", "zt5_presented", "zt5", "kt5", "zt6_presented", "zt6", "release",
}

func stageIndex(s string) int { return slices.Index(Stages, s) }

// ordersAndLots — задания 1С и партии: поступление через stand 1С, приёмка,
// входной контроль, ЗТ-1 (Е-01, Е-03…Е-08).
func (g *gen) ordersAndLots() {
	w := g.b.World
	for _, o := range g.b.Run.Orders {
		data := map[string]any{"order_id": "{local:" + o.ID + "}", "external_system": "onec",
			"external_number": o.External, "item_type_id": w.ItemType, "quantity": o.Quantity, "item_revision": w.ItemRevision}
		if o.Due != "" {
			data["due_date"] = o.Due
		}
		g.add(&draft{at: g.t(o.At), source: "onec", typ: "erp.order.received", data: shiftMap(data, g.shift),
			label: "order/" + o.ID, scenario: "route"})
	}
	for _, l := range g.b.Run.Lots {
		at := g.t(l.Arrived)
		ext := l.External
		if ext == "" {
			ext = strings.ToLower(l.ID)
		}
		recv := map[string]any{"lot_id": "{local:" + l.ID + "}", "external_system": "onec", "external_number": ext,
			"supplier_id": l.Supplier, "item_type_id": l.Component, "quantity": l.Quantity}
		if l.Certificate != "" {
			recv["certificate_no"] = l.Certificate
		}
		if l.Heat != "" {
			recv["heat_no"] = l.Heat
		}
		g.add(&draft{at: at, source: "onec", typ: "erp.lot.received", lot: l.ID, data: recv, label: "lot/" + l.ID + "/received", scenario: "route"})
		lotParam := map[string]any{"lot_id": "{local:" + l.ID + "}"}
		g.act(Action{Kind: ActionDecision, At: at.Add(30 * time.Minute), Scenario: "route", Label: "lot/" + l.ID + "/register",
			Operation: "crossitem.lot.register", Role: "storekeeper", Actor: g.ids.Person(w.Route.Storekeeper), Params: lotParam,
			Body: map[string]any{"actual_quantity": l.Quantity, "certificate_present": l.Certificate != "", "packaging_ok": true}})
		acc := at.Add(120 * time.Minute)
		if l.Accepted != "" {
			acc = g.t(l.Accepted)
		}
		g.add(&draft{at: acc.Add(-30 * time.Minute), source: "term-vk", typ: "inspection.result.recorded", lot: l.ID,
			label: "lot/" + l.ID + "/docs", scenario: "route", data: map[string]any{
				"observation_id": "{local:DOC-" + l.ID + "}", "method": "supplier_documents", "phase": "incoming",
				"step_key": "incoming.zt1_lot_acceptance", "lot_id": "{local:" + l.ID + "}", "outcome": "no_defect_indicated",
				"processing_state": "completed", "inspector_id": g.ids.Person(w.Route.QCFinal)}})
		g.add(&draft{at: acc.Add(-20 * time.Minute), source: "cam-kt1", typ: "inspection.result.recorded", lot: l.ID,
			label: "lot/" + l.ID + "/kt1", scenario: "route", data: map[string]any{
				"observation_id": "{local:KT1-" + l.ID + "}", "method": "camera", "phase": "incoming", "inspection_point": "KT-1",
				"step_key": "incoming.kt1_camera", "lot_id": "{local:" + l.ID + "}", "outcome": "no_defect_indicated",
				"processing_state": "completed", "analyzer_confidence_bp": 9400, "observation_quality_bp": 9000,
				"versions": map[string]any{"analyzer_version": "vqc-surface 1.2.0", "contract_version": "1.0", "recipe_ref": "kt1-camera@1"}}})
		g.act(Action{Kind: ActionDecision, At: acc, Scenario: "route", Label: "lot/" + l.ID + "/zt1",
			Operation: "nonconformity.lot.resolve", Role: "quality_inspector", Actor: g.ids.Person(w.Route.QCFinal), Params: lotParam,
			Body: map[string]any{"resolution": "accept", "accepted_quantity": l.Quantity,
				"method_event_ids": []any{"{event:lot/" + l.ID + "/docs}", "{event:lot/" + l.ID + "/kt1}"}}})
		for k, is := range l.Issues {
			g.act(Action{Kind: ActionDecision, At: g.t(is.At), Scenario: "route", Label: fmt.Sprintf("lot/%s/issue-%d", l.ID, k+1),
				Operation: "crossitem.lot.issue", Role: "storekeeper", Actor: g.ids.Person(w.Route.Storekeeper), Params: lotParam,
				Body: map[string]any{"quantity": is.Quantity, "to_location_id": is.To}})
		}
	}
}

// route — маршрут фоновых и сценарных фланцев до этапа Until (FR-104:
// различимые экземпляры, завершённые операции, фон для сравнения).
func (g *gen) route() {
	for i := range g.b.Run.Items {
		p := &g.b.Run.Items[i]
		if _, dup := g.plans[p.ID]; dup {
			g.fail("изделие %s описано дважды", p.ID)
			continue
		}
		g.plans[p.ID] = p
		g.item(p)
	}
}

// num — номер изделия для ID выполнений: F-017 → 017.
func num(item string) string {
	if i := strings.LastIndexByte(item, '-'); i >= 0 {
		return item[i+1:]
	}
	return item
}

func (g *gen) shiftOf(t time.Time) string {
	w := g.b.World
	local := t.Add(-g.shift).In(time.FixedZone("MSK", 3*3600))
	hm := local.Format("15:04")
	if a, ok := w.Shifts["A"]; ok && hm >= a.Starts && hm < a.Ends {
		return "A"
	}
	return "B"
}

func (g *gen) lineOf(equipment string) (string, Line) {
	for _, k := range sortedKeys(g.b.World.Lines) {
		if l := g.b.World.Lines[k]; l.Equipment == equipment {
			return k, l
		}
	}
	return "", Line{}
}

func sortedKeys[V any](m map[string]V) []string { return slices.Sorted(maps.Keys(m)) }

// times — моменты этапов изделия: по нормам маршрута, с явными временами
// плана (ItemPlan.At) и привязкой к сварке.
func (g *gen) times(p *ItemPlan) map[string]time.Time {
	r := g.b.World.Route
	at := map[string]time.Time{}
	min := func(n int) time.Duration { return time.Duration(n) * time.Minute }
	var weldAt, weldEnd time.Time
	if p.Weld != nil {
		weldAt = g.t(p.Weld.At)
		m := p.Weld.Minutes
		if m == 0 {
			m = r.WeldMin
		}
		weldEnd = weldAt.Add(min(m))
	}
	prev := time.Time{}
	for _, s := range Stages {
		var t time.Time
		switch s {
		case "launch":
			t = g.t(p.Launch)
		case "machining":
			if p.Machining != "" {
				t = g.t(p.Machining)
			} else {
				t = prev.Add(30 * time.Minute)
			}
		case "remark":
			t = at["machining"].Add(min(machiningMin(p, r) + 4))
		case "kt2":
			t = prev.Add(4 * time.Minute)
		case "cmm":
			t = prev.Add(min(r.CMMMin))
		case "zt2_presented":
			t = prev.Add(15 * time.Minute)
		case "zt2":
			t = prev.Add(5 * time.Minute)
		case "sent_to_welding":
			if weldAt.IsZero() {
				t = prev.Add(60 * time.Minute)
			} else {
				t = weldAt.Add(-80 * time.Minute)
			}
		case "received_welding":
			t = prev.Add(10 * time.Minute)
		case "ring":
			if weldAt.IsZero() {
				t = prev.Add(10 * time.Minute)
			} else {
				t = weldAt.Add(-50 * time.Minute)
			}
		case "edge_prep":
			if weldAt.IsZero() {
				t = prev.Add(5 * time.Minute)
			} else {
				t = weldAt.Add(-min(r.EdgePrepMin))
			}
		case "weld":
			t = weldAt
		case "kt3":
			t = weldEnd.Add(5 * time.Minute)
		case "xray":
			t = weldEnd.Add(min(r.XRayAfterMin))
		case "zt3_presented":
			t = at["xray"].Add(10 * time.Minute)
		case "zt3":
			t = at["xray"].Add(min(r.ZT3AfterMin))
		case "sent_to_assembly":
			t = prev.Add(30 * time.Minute)
		case "received_assembly":
			t = prev.Add(10 * time.Minute)
		case "assembly_started":
			t = prev.Add(40 * time.Minute)
		case "assembly":
			t = prev.Add(min(r.AssemblyMin))
		case "zt4_presented":
			t = prev.Add(5 * time.Minute)
		case "zt4":
			t = prev.Add(15 * time.Minute)
		case "leak_test":
			t = prev.Add(30 * time.Minute)
		case "zt5_presented":
			t = prev.Add(min(r.LeakTestMin + 5))
		case "zt5":
			t = prev.Add(10 * time.Minute)
		case "kt5":
			t = prev.Add(min(r.FinalAfterMin))
		case "zt6_presented":
			t = prev.Add(10 * time.Minute)
		case "zt6":
			t = prev.Add(35 * time.Minute)
		case "release":
			t = prev.Add(15 * time.Minute)
		}
		if v, ok := p.At[s]; ok {
			t = g.t(v)
		}
		at[s] = t
		prev = t
		if s == p.Until {
			break
		}
	}
	return at
}

// item — события и шаги маршрута одного изделия.
func (g *gen) item(p *ItemPlan) {
	w := g.b.World
	r := w.Route
	until := stageIndex(p.Until)
	if until < 0 {
		g.fail("изделие %s: этап until %q не из списка этапов", p.ID, p.Until)
		return
	}
	for _, s := range p.Skip {
		stage, sub, isSub := strings.Cut(s, ".")
		if stageIndex(stage) < 0 || (isSub && !slices.Contains(AssemblySteps, sub)) {
			g.fail("изделие %s: этап skip %q не из списка этапов", p.ID, s)
		}
	}
	for _, s := range sortedKeys(p.At) {
		if stageIndex(s) < 0 {
			g.fail("изделие %s: этап at %q не из списка этапов", p.ID, s)
		}
	}
	if until >= stageIndex("sent_to_welding") && p.Weld == nil {
		g.fail("изделие %s: маршрут до %s без сварки (weld)", p.ID, p.Until)
		return
	}
	at := g.times(p)
	for i := 1; i <= until; i++ {
		if !at[Stages[i]].After(at[Stages[i-1]]) && !at[Stages[i]].Equal(at[Stages[i-1]]) {
			g.fail("изделие %s: этап %s (%s) раньше этапа %s (%s)", p.ID, Stages[i], FormatTime(at[Stages[i]]), Stages[i-1], FormatTime(at[Stages[i-1]]))
		}
	}
	it := &ItemTruth{ID: p.ID, Order: p.Order, Reference: p.Reference, Stages: map[string]time.Time{}, Until: p.Until}
	g.items[p.ID] = it
	if until >= stageIndex("remark") && !slices.Contains(p.Skip, "remark") {
		it.RemarkedAt = at["remark"]
	}
	if p.At["remark"] != "" {
		it.RemarkedAt = at["remark"]
	}
	n := num(p.ID)
	run := func(code string) string { return "{local:" + code + "-" + n + "-1}" }
	itemParam := map[string]any{"item_id": "{item:" + p.ID + "}"}
	welder, line := "", ""
	var ln Line
	if p.Weld != nil {
		welder = p.Weld.Welder
		line, ln = g.lineOf(p.Weld.Station)
		if line == "" {
			g.fail("изделие %s: сварочный пост %s не привязан к линии", p.ID, p.Weld.Station)
		}
		it.Line = line
	}
	lots := g.orderLots(p.Order)
	blank, ringLot := p.BlankLot, p.RingLot
	if blank == "" {
		blank = lots["blank"]
	}
	if ringLot == "" {
		ringLot = lots["ring"]
	}
	if blank == "" || ringLot == "" {
		g.fail("изделие %s: у задания %s нет партий заготовок и колец (orders[].lots)", p.ID, p.Order)
	}
	decide := func(stage string, t time.Time, op, role, actor string, params, body map[string]any) {
		g.act(Action{Kind: ActionDecision, At: t, Scenario: "route", Label: p.ID + "/" + stage + "/" + op, Operation: op,
			Role: role, Actor: g.ids.Person(actor), Params: params, Body: body, Item: p.ID})
	}
	fact := func(stage string, t time.Time, source, typ string, data map[string]any) *draft {
		return g.add(&draft{at: t, source: source, typ: typ, item: p.ID, data: data, label: p.ID + "/" + stage,
			scenario: "route", background: true})
	}
	inspect := func(stage string, t time.Time, source, method, phase, step, point, opRun string, zones []any, extra map[string]any) {
		d := map[string]any{"observation_id": "{local:" + strings.ToUpper(stage) + "-" + p.ID + "}", "method": method, "phase": phase,
			"step_key": step, "outcome": "no_defect_indicated", "processing_state": "completed"}
		if point != "" {
			d["inspection_point"] = point
		}
		if opRun != "" {
			d["operation_run_id"] = opRun
		}
		if len(zones) > 0 {
			d["zone_ids"] = zones
		}
		for _, k := range sortedKeys(extra) {
			d[k] = extra[k]
		}
		fact(stage, t, source, "inspection.result.recorded", d)
	}
	camera := func(recipe, analyzer string, conf, quality int) map[string]any {
		return map[string]any{"analyzer_confidence_bp": conf, "observation_quality_bp": quality,
			"versions": map[string]any{"analyzer_version": analyzer, "contract_version": "1.0", "recipe_ref": recipe, "item_revision": w.ItemRevision}}
	}
	rnd := NewRand(g.seed, "route/"+p.ID)
	weldZones := []any{}
	for _, z := range []string{"U1", "U2", "U3", "U4", "U5", "U6", "U7", "U8"} {
		weldZones = append(weldZones, g.ids.Zone(z))
	}
	for i := 0; i <= until; i++ {
		s := Stages[i]
		t := at[s]
		it.Stages[s] = t
		if s == "weld" && p.Weld != nil {
			g.recordWeld(p, at)
		}
		if slices.Contains(p.Skip, s) {
			continue
		}
		switch s {
		case "launch":
			g.act(Action{Kind: ActionDecision, At: t, Scenario: "route", Label: p.ID + "/register", Operation: "item.item.register",
				Role: "storekeeper", Actor: g.ids.Person(r.Storekeeper), Binds: p.ID, Item: p.ID,
				Body: map[string]any{"item_type_id": w.ItemType, "item_revision": w.ItemRevision, "order_id": "{local:" + p.Order + "}",
					"lot_ids": []any{"{local:" + blank + "}"}, "entry_step_key": "incoming.issue_blank"}})
			decide(s+"/tag", t.Add(time.Minute), "item.carrier.apply", "storekeeper", r.Storekeeper, itemParam,
				map[string]any{"carrier_type": "tag_qr", "value": "{carrier:TAG:" + p.ID + "}", "is_temporary": true})
			decide(s+"/issue", t.Add(2*time.Minute), "crossitem.lot.issue", "storekeeper", r.Storekeeper,
				map[string]any{"lot_id": "{local:" + blank + "}"},
				map[string]any{"item_ids": []any{"{item:" + p.ID + "}"}, "order_id": "{local:" + p.Order + "}", "quantity": 1, "to_location_id": "WS-MC"})
		case "machining":
			op := r.CNCOperator
			if g.shiftOf(t) == "B" && r.CNCOperatorB != "" {
				op = r.CNCOperatorB
			}
			mm := machiningMin(p, r)
			decide(s, t, "process.operation.start", "performer", op, itemParam, map[string]any{
				"operation_code": "MO", "operation_run_id": run("MO"), "step_key": "machining.cnc", "equipment_id": g.ids.Equipment("CNC-1"),
				"station_id": "ST-CNC", "program_ref": "ЧПУ-ФЛ-100.01-03"})
			end := t.Add(time.Duration(mm) * time.Minute)
			g.add(&draft{at: end, source: "cnc-1", typ: "equipment.cycle.summarized", label: p.ID + "/machining/cycle", scenario: "route", background: true,
				data: map[string]any{"equipment_id": g.ids.Equipment("CNC-1"), "station_id": "ST-CNC", "window_start": FormatTime(t), "window_end": FormatTime(end),
					"cycle_ref": run("MO"), "parameters": []any{map[string]any{"parameter": "spindle_load",
						"mean": measurement(int64(rnd.Between(58, 72)), 0, "%"), "max": measurement(int64(rnd.Between(80, 88)), 0, "%")}}}})
			d := fact(s+"/finished", end, "cnc-1", "operation.run.finished", map[string]any{"operation_run_id": run("MO"), "completion": "completed",
				"operation_started_at": FormatTime(t), "operation_finished_at": FormatTime(end),
				"reported_duration": map[string]any{"value": mm, "unit": "min", "meaning": "active_processing", "origin": "source_reported"}})
			d.carrier = "tag"
		case "remark":
			d := fact(s, t, "marker", "item.carrier.applied", map[string]any{"carrier_type": "dpm_datamatrix", "value": "{carrier:DM:" + p.ID + "}",
				"is_temporary": false, "replaces_value": "{carrier:TAG:" + p.ID + "}", "zone_id": "F-OUTER"})
			d.carrier = "tag"
		case "kt2":
			inspect(s, t, "cam-kt2", "camera", "after_operation", "machining.kt2_camera", "KT-2", run("MO"), []any{g.ids.Zone("EDGE")},
				camera("kt2-oblique@1", "vqc-edge 1.4.0", rnd.Between(9400, 9800), rnd.Between(9000, 9600)))
		case "cmm":
			inspect(s, t, "cmm-1", "cmm", "after_operation", "machining.kt2_cmm", "", run("MO"), nil, map[string]any{
				"inspector_id": g.ids.Person(r.CMMOperator), "measurements": cmmMeasurements(rnd)})
		case "zt2_presented":
			decide(s, t, "item.presentation.record", "site_foreman", r.MasterMC, itemParam,
				map[string]any{"presentation_no": 1, "presented_to": "qc", "step_key": "machining.zt2_acceptance"})
		case "zt2":
			decide(s, t, "nonconformity.presentation.resolve", "quality_inspector", r.QCMachining, itemParam, map[string]any{
				"closing_point": "ZT-2", "step_key": "machining.zt2_acceptance", "presentation_no": 1, "resolution": "accept",
				"method_event_ids": methodEvents(p, "kt2", "cmm")})
		case "sent_to_welding":
			decide(s, t, "process.movement.send", "site_foreman", r.MasterMC, itemParam,
				map[string]any{"from_location_id": "WS-MC", "to_location_id": "WS-WC", "step_key": "machining.send_to_welding"})
		case "received_welding":
			decide(s, t, "process.movement.receive", "site_foreman", r.MasterWC, itemParam, map[string]any{"destination_kind": "workshop",
				"from_location_id": "WS-MC", "to_location_id": "WS-WC", "inspection_on_receipt": "no_damage", "step_key": "welding.receive"})
		case "ring":
			ring := p.Ring
			if ring == "" {
				ring = "R-" + n
			}
			decide(s+"/issue", t, "crossitem.lot.issue", "storekeeper", r.Storekeeper, map[string]any{"lot_id": "{local:" + ringLot + "}"},
				map[string]any{"item_ids": []any{"{item:" + p.ID + "}"}, "quantity": 1, "to_location_id": "WS-WC"})
			decide(s+"/scan", t.Add(3*time.Minute), "item.assembly.record", "performer", welder, itemParam, map[string]any{
				"binding_method": "dpm_datamatrix", "component_lot_id": "{local:" + ringLot + "}", "component_type_id": "FL-100.01.002",
				"position": ring, "quantity": 1})
		case "edge_prep":
			decide(s, t, "access.operator.confirm_step", "performer", welder, map[string]any{"workplace_id": ln.Workplace},
				map[string]any{"step_key": "welding.edge_prep", "item_id": "{item:" + p.ID + "}", "tp_step": "Подготовка кромок под сварку"})
		case "weld":
			g.weldEvents(p, t, ln, run("SV"), itemParam, rnd)
		case "kt3":
			inspect(s, t, "cam-kt3-a", "camera", "after_operation", "welding.kt3_camera", "KT-3", run("SV"), weldZones,
				camera("kt3-weld@1", "vqc-weld 2.3.1", rnd.Between(9300, 9700), rnd.Between(8800, 9500)))
		case "xray":
			inspect(s, t, "xray", "radiography", "after_operation", "welding.kt3_radiography", "", run("SV"), weldZones, map[string]any{
				"inspector_id": g.ids.Person(r.NDT), "conclusion_ref": "РК-" + t.Add(-g.shift).Format("0102") + "-" + n})
		case "zt3_presented":
			decide(s, t, "item.presentation.record", "site_foreman", r.MasterWC, itemParam,
				map[string]any{"presentation_no": 1, "presented_to": "qc", "step_key": "welding.zt3_acceptance"})
		case "zt3":
			decide(s, t, "nonconformity.presentation.resolve", "quality_inspector", r.QCWelding, itemParam, map[string]any{
				"closing_point": "ZT-3", "step_key": "welding.zt3_acceptance", "presentation_no": 1, "resolution": "accept",
				"method_event_ids": methodEvents(p, "kt3", "xray")})
		case "sent_to_assembly":
			decide(s, t, "process.movement.send", "site_foreman", r.MasterWC, itemParam,
				map[string]any{"from_location_id": "WS-WC", "to_location_id": "WS-AC", "step_key": "welding.send_to_assembly"})
		case "received_assembly":
			decide(s, t, "process.movement.receive", "site_foreman", r.MasterAC, itemParam, map[string]any{"destination_kind": "workshop",
				"from_location_id": "WS-WC", "to_location_id": "WS-AC", "inspection_on_receipt": "no_damage", "step_key": "assembly.receive"})
		case "assembly_started":
			for k, kind := range []string{"cover", "seal", "fastener", "valve"} {
				lot := lots[kind]
				qty := 1
				if kind == "fastener" {
					qty = 12
				}
				decide(fmt.Sprintf("%s/issue-%d", s, k), t.Add(time.Duration(k)*time.Minute), "crossitem.lot.issue", "storekeeper", r.Storekeeper,
					map[string]any{"lot_id": "{local:" + lot + "}"},
					map[string]any{"item_ids": []any{"{item:" + p.ID + "}"}, "quantity": qty, "to_location_id": "WS-AC"})
			}
			decide(s, t.Add(10*time.Minute), "process.operation.start", "performer", r.Assembler, itemParam, map[string]any{
				"operation_code": "AS", "operation_run_id": run("AS"), "step_key": "assembly.join_parts", "station_id": "ST-ASM"})
		case "assembly":
			g.assemblyEvents(p, t, run, itemParam, rnd, camera, lots)
		case "zt4_presented":
			decide(s, t, "item.presentation.record", "site_foreman", r.MasterAC, itemParam,
				map[string]any{"presentation_no": 1, "presented_to": "qc", "step_key": "assembly.zt4_acceptance"})
		case "zt4":
			decide(s, t, "nonconformity.presentation.resolve", "quality_inspector", r.QCAssembly, itemParam, map[string]any{
				"closing_point": "ZT-4", "step_key": "assembly.zt4_acceptance", "presentation_no": 1, "resolution": "accept",
				"method_event_ids": methodEvents(p, "kt4d", "torque", "kt4")})
		case "leak_test":
			decide(s+"/start", t, "process.operation.start", "performer", r.Tester, itemParam, map[string]any{
				"operation_code": "LT", "operation_run_id": run("LT"), "step_key": "testing.leak_test", "equipment_id": g.ids.Equipment("LT-1"),
				"station_id": "ST-LEAK"})
			end := t.Add(time.Duration(r.LeakTestMin) * time.Minute)
			inspect("leak", end.Add(-2*time.Minute), "leak-1", "leak_test", "test", "testing.leak_test", "", run("LT"), nil, map[string]any{
				"equipment_id": g.ids.Equipment("LT-1"), "measurements": []any{map[string]any{"characteristic": "Утечка гелия, мбар·л/с",
					"value": measurement(int64(rnd.Between(12, 40)), 10, "mbar.L/s"), "tolerance": map[string]any{"upper": measurement(100, 10, "mbar.L/s")},
					"verdict": "within"}}})
			decide(s+"/finish", end, "process.operation.finish", "performer", r.Tester, map[string]any{"run_id": run("LT")},
				map[string]any{"completion": "completed"})
		case "zt5_presented":
			decide(s, t, "item.presentation.record", "site_foreman", r.MasterAC, itemParam,
				map[string]any{"presentation_no": 1, "presented_to": "qc", "step_key": "testing.zt5_protocol"})
		case "zt5":
			decide(s, t, "nonconformity.presentation.resolve", "quality_inspector", r.QCAssembly, itemParam, map[string]any{
				"closing_point": "ZT-5", "step_key": "testing.zt5_protocol", "presentation_no": 1, "resolution": "accept",
				"method_event_ids": methodEvents(p, "leak")})
		case "kt5":
			inspect(s, t, "cam-kt5", "camera", "final", "final.kt5_camera", "KT-5", "", nil,
				camera("kt5-final@1", "vqc-final 1.1.0", rnd.Between(9300, 9700), rnd.Between(8900, 9400)))
		case "zt6_presented":
			decide(s, t, "item.presentation.record", "site_foreman", r.MasterAC, itemParam,
				map[string]any{"presentation_no": 1, "presented_to": "customer_representative", "step_key": "final.zt6_acceptance"})
		case "zt6":
			decide(s, t, "nonconformity.presentation.resolve", "quality_inspector", r.QCFinal, itemParam, map[string]any{
				"closing_point": "ZT-6", "step_key": "final.zt6_acceptance", "presentation_no": 1, "resolution": "accept",
				"method_event_ids": methodEvents(p, "kt5")})
		case "release":
			decide(s, t, "item.release.record", "storekeeper", r.Storekeeper, itemParam,
				map[string]any{"after_rework": false, "warehouse_id": "WH-FG"})
		}
	}
}

func machiningMin(p *ItemPlan, r Route) int {
	if p.MachiningMin > 0 {
		return p.MachiningMin
	}
	return r.MachiningMin
}

// orderLots — партии компонентов задания.
func (g *gen) orderLots(order string) map[string]string {
	for _, o := range g.b.Run.Orders {
		if o.ID == order {
			if o.Lots == nil {
				return map[string]string{}
			}
			return o.Lots
		}
	}
	g.fail("задание %s не описано в прогоне", order)
	return map[string]string{}
}

// methodEvents — ссылки на результаты методов этапа (основание решения на ЗТ).
func methodEvents(p *ItemPlan, stages ...string) []any {
	out := make([]any, 0, len(stages))
	for _, s := range stages {
		// этап маршрута и шаг карточки, заменивший его, помечены одинаково: ‹изделие›/‹этап›
		out = append(out, "{event:"+p.ID+"/"+s+"}")
	}
	return out
}

// recordWeld — сварка в «истине» (для области риска §2.6 и журнала источника).
func (g *gen) recordWeld(p *ItemPlan, at map[string]time.Time) {
	r := g.b.World.Route
	m := p.Weld.Minutes
	if m == 0 {
		m = r.WeldMin
	}
	start := at["weld"]
	wt := WeldTruth{Item: p.ID, Run: "SV-" + num(p.ID) + "-1", Station: p.Weld.Station, Welder: p.Weld.Welder,
		Shift: g.shiftOf(start), Start: start, End: start.Add(time.Duration(m) * time.Minute)}
	lo, hi := r.CurrentNominal-2, r.CurrentNominal+3
	if len(p.Weld.Current) == 2 {
		lo, hi = p.Weld.Current[0], p.Weld.Current[1]
	}
	wt.CurrentMin, wt.CurrentMax = lo, hi
	if p.Weld.OutFrom != "" {
		wt.OutFrom = g.t(p.Weld.OutFrom)
	}
	wt.ArcFrom, wt.ArcTo = wt.Start, wt.End
	if len(p.Weld.Arc) == 2 {
		wt.ArcFrom, wt.ArcTo = g.t(p.Weld.Arc[0]), g.t(p.Weld.Arc[1])
	}
	g.welds = append(g.welds, wt)
}

// weldEvents — сварка: подтверждение режима, начало, сводки источника по
// минутам (если окно не покрыто потоком журнала), завершение.
func (g *gen) weldEvents(p *ItemPlan, t time.Time, ln Line, runID string, itemParam map[string]any, rnd *Rand) {
	r := g.b.World.Route
	w := g.weldTruth(p.ID)
	g.act(Action{Kind: ActionDecision, At: t.Add(-2 * time.Minute), Scenario: "route", Label: p.ID + "/weld/confirm",
		Operation: "access.operator.confirm_step", Role: "performer", Actor: g.ids.Person(p.Weld.Welder), Item: p.ID,
		Params: map[string]any{"workplace_id": ln.Workplace},
		Body:   map[string]any{"step_key": "welding.weld", "item_id": "{item:" + p.ID + "}", "tp_step": "Режим по карте сверен"}})
	g.act(Action{Kind: ActionDecision, At: t, Scenario: "route", Label: p.ID + "/weld/start", Operation: "process.operation.start",
		Role: "performer", Actor: g.ids.Person(p.Weld.Welder), Params: itemParam, Item: p.ID, Body: map[string]any{
			"operation_code": "SV", "operation_run_id": runID, "step_key": "welding.weld", "equipment_id": g.ids.Equipment(p.Weld.Station),
			"station_id": "ST-WELD", "program_ref": r.Program}})
	g.weldCycles(w, ln.WeldingSource, runID, rnd, true)
	g.act(Action{Kind: ActionDecision, At: w.End, Scenario: "route", Label: p.ID + "/weld/finish", Operation: "process.operation.finish",
		Role: "performer", Actor: g.ids.Person(p.Weld.Welder), Item: p.ID, Params: map[string]any{"run_id": runID},
		Body: map[string]any{"completion": "completed"}})
}

func (g *gen) weldTruth(item string) WeldTruth {
	for i := len(g.welds) - 1; i >= 0; i-- {
		if g.welds[i].Item == item {
			return g.welds[i]
		}
	}
	return WeldTruth{}
}

// weldCycles — сводки параметров сварки по минутам (equipment.cycle.summarized,
// FR-147: не телеметрия, а сводка окна). live=true — только минуты вне окон
// потоков журнала; поток сам даёт свои записи.
func (g *gen) weldCycles(w WeldTruth, source, runID string, rnd *Rand, live bool) []*draft {
	var out []*draft
	for m := w.ArcFrom; m.Before(w.ArcTo); m = m.Add(time.Minute) {
		if live && g.inStream(w.Station, m) {
			continue
		}
		d := g.add(&draft{at: m.Add(time.Minute), source: source, typ: "equipment.cycle.summarized", scenario: "route", background: live,
			data: g.cycleData(w, m, runID, rnd)})
		d.label = fmt.Sprintf("%s/weld/%s", w.Item, m.Add(-g.shift).Format("0102-1504"))
		out = append(out, d)
	}
	return out
}

func (g *gen) cycleData(w WeldTruth, m time.Time, runID string, rnd *Rand) map[string]any {
	r := g.b.World.Route
	lo, hi := w.CurrentMin, w.CurrentMax
	if !w.OutFrom.IsZero() && m.Before(w.OutFrom) {
		lo, hi = r.CurrentNominal-2, r.CurrentNominal+3
	}
	mean := rnd.Between(lo, hi)
	mx := min(hi, mean+rnd.Between(0, 2))
	mn := max(lo, mean-rnd.Between(0, 2))
	param := map[string]any{"parameter": "current", "mean": measurement(int64(mean), 0, "A"), "max": measurement(int64(mx), 0, "A"),
		"min": measurement(int64(mn), 0, "A"), "setpoint": map[string]any{"nominal": measurement(int64(r.CurrentNominal), 0, "A"),
			"lower": measurement(int64(r.CurrentNominal-r.CurrentTol), 0, "A"), "upper": measurement(int64(r.CurrentNominal+r.CurrentTol), 0, "A")}}
	if mx > r.CurrentNominal+r.CurrentTol {
		param["out_of_setpoint_ms"] = 60000
	}
	return map[string]any{"equipment_id": g.ids.Equipment(w.Station), "station_id": "ST-WELD",
		"window_start": FormatTime(m), "window_end": FormatTime(m.Add(time.Minute)), "cycle_ref": runID,
		"parameters": []any{param, map[string]any{"parameter": "voltage", "mean": measurement(int64(rnd.Between(215, 232)), 1, "V")}}}
}

// assemblyEvents — сборка: сканы компонентов, уплотнение, камера зоны, затяжка,
// установка оборудования (клапан КВД, кейс §1.1), камера сборки.
func (g *gen) assemblyEvents(p *ItemPlan, t time.Time, run func(string) string, itemParam map[string]any, rnd *Rand,
	camera func(recipe, analyzer string, conf, quality int) map[string]any, lots map[string]string) {
	r := g.b.World.Route
	n := num(p.ID)
	cover, valve := p.Cover, p.Valve
	if cover == "" {
		cover = "C-" + n
	}
	if valve == "" {
		valve = "V-" + n
	}
	decide := func(label string, at time.Time, op, actor string, params, body map[string]any) {
		g.act(Action{Kind: ActionDecision, At: at, Scenario: "route", Label: p.ID + "/assembly/" + label, Operation: op,
			Role: "performer", Actor: g.ids.Person(actor), Params: params, Body: body, Item: p.ID})
	}
	fact := func(label string, at time.Time, source, typ string, data map[string]any) {
		if slices.Contains(p.Skip, "assembly."+label) {
			return
		}
		g.add(&draft{at: at, source: source, typ: typ, item: p.ID, data: data, label: p.ID + "/" + label, scenario: "route", background: true})
	}
	scan := func(k int, lot, typ, pos string, qty int) {
		decide("scan-"+pos, t.Add(time.Duration(k)*time.Minute), "item.assembly.record", r.Assembler, itemParam, map[string]any{
			"binding_method": "container_cell", "component_lot_id": "{local:" + lot + "}", "component_type_id": typ, "position": pos, "quantity": qty})
	}
	scan(0, lots["cover"], "FL-100.00.003", cover, 1)
	scan(1, lots["seal"], "FL-100.00.004", "S-1", 1)
	scan(2, lots["fastener"], "FL-100.00.005", "J-1", 12)
	scan(3, lots["valve"], "KVD-6", valve, 1)
	decide("seal", t.Add(8*time.Minute), "access.operator.confirm_step", r.Assembler, map[string]any{"workplace_id": "WP-ASM-1"},
		map[string]any{"step_key": "assembly.seal_install", "item_id": "{item:" + p.ID + "}", "tp_step": "Установка уплотнения"})
	fact("kt4d", t.Add(12*time.Minute), "cam-kt4d", "inspection.result.recorded", withCamera(map[string]any{
		"observation_id": "{local:KT4D-" + p.ID + "}", "method": "camera", "phase": "before_zone_closure", "step_key": "assembly.kt4d_zone_camera",
		"inspection_point": "KT-4d", "operation_run_id": run("AS"), "zone_ids": []any{"S-1", "CAV"}, "outcome": "no_defect_indicated",
		"processing_state": "completed"}, camera("kt4d-uv@1", "vqc-uv 1.0.3", rnd.Between(9200, 9700), rnd.Between(8800, 9400))))
	var torques []any
	for b := 1; b <= 12; b++ {
		torques = append(torques, map[string]any{"characteristic": fmt.Sprintf("Момент затяжки болта %d, Н·м", b),
			"value": measurement(int64(rnd.Between(118, 124)), 1, "N.m"), "tolerance": map[string]any{
				"nominal": measurement(120, 1, "N.m"), "lower": measurement(108, 1, "N.m"), "upper": measurement(132, 1, "N.m")}, "verdict": "within"})
	}
	fact("torque", t.Add(30*time.Minute), "tw-1", "inspection.result.recorded", map[string]any{
		"observation_id": "{local:TW-" + p.ID + "}", "method": "torque", "phase": "assembly", "step_key": "assembly.torque",
		"operation_run_id": run("AS"), "zone_ids": []any{"J-1"}, "outcome": "no_defect_indicated", "processing_state": "completed",
		"equipment_id": g.ids.Equipment("TW-1"), "measurements": torques})
	eq := "{local:EQ-" + n + "-1}"
	decide("eq-start", t.Add(32*time.Minute), "process.operation.start", r.Assembler, itemParam, map[string]any{
		"operation_code": "EQ", "operation_run_id": eq, "step_key": "assembly.fasteners_install", "station_id": "ST-ASM"})
	fact("eq-torque", t.Add(35*time.Minute), "tw-1", "inspection.result.recorded", map[string]any{
		"observation_id": "{local:TW-J2-" + p.ID + "}", "method": "torque", "phase": "assembly", "step_key": "assembly.fasteners_install",
		"operation_run_id": eq, "outcome": "no_defect_indicated", "processing_state": "completed", "equipment_id": g.ids.Equipment("TW-1"),
		"measurements": []any{map[string]any{"characteristic": "Момент затяжки J-2 (клапан КВД), Н·м", "value": measurement(int64(rnd.Between(118, 124)), 1, "N.m"),
			"tolerance": map[string]any{"nominal": measurement(120, 1, "N.m"), "lower": measurement(108, 1, "N.m"), "upper": measurement(132, 1, "N.m")}, "verdict": "within"}}})
	decide("eq-finish", t.Add(37*time.Minute), "process.operation.finish", r.Assembler, map[string]any{"run_id": eq},
		map[string]any{"completion": "completed"})
	fact("kt4", t.Add(45*time.Minute), "cam-kt4", "inspection.result.recorded", withCamera(map[string]any{
		"observation_id": "{local:KT4-" + p.ID + "}", "method": "camera", "phase": "assembly", "step_key": "assembly.kt4_camera",
		"inspection_point": "KT-4", "operation_run_id": run("AS"), "zone_ids": []any{"J-1"}, "outcome": "no_defect_indicated",
		"processing_state": "completed"}, camera("kt4-assembly@1", "vqc-asm 3.0.2", rnd.Between(9200, 9700), rnd.Between(8500, 9300))))
	decide("finish", t.Add(55*time.Minute), "process.operation.finish", r.Assembler, map[string]any{"run_id": run("AS")},
		map[string]any{"completion": "completed"})
}

// AssemblySteps — события внутри этапа сборки, которые карточка может взять на
// себя (ItemPlan.Skip: «assembly.kt4»).
var AssemblySteps = []string{"kt4d", "torque", "eq-torque", "kt4"}

func withCamera(d, cam map[string]any) map[string]any {
	for _, k := range sortedKeys(cam) {
		d[k] = cam[k]
	}
	return d
}

// measurement — измерение без float (value × 10^−scale, единица UCUM).
func measurement(value int64, scale int, unit string) map[string]any {
	return map[string]any{"value": value, "scale": scale, "unit": unit}
}

func cmmMeasurements(rnd *Rand) []any {
	chars := []struct {
		name     string
		nom, tol int64
	}{{"Ø посадочный 120H8, мм", 120000, 54}, {"Ø наружный 180h9, мм", 180000, 100}, {"Высота 32±0,1, мм", 32000, 100},
		{"Плоскостность уплотнительной поверхности, мм", 0, 30}, {"Позиция отверстий 12×Ø9, мм", 0, 100}}
	var out []any
	for _, c := range chars {
		v := c.nom + int64(rnd.Between(-int(c.tol)/2, int(c.tol)/2))
		if c.nom == 0 {
			v = int64(rnd.Between(5, int(c.tol)/2))
		}
		tol := map[string]any{"upper": measurement(c.nom+c.tol, 3, "mm")}
		if c.nom != 0 {
			tol = map[string]any{"nominal": measurement(c.nom, 3, "mm"), "lower": measurement(c.nom-c.tol, 3, "mm"), "upper": measurement(c.nom+c.tol, 3, "mm")}
		}
		out = append(out, map[string]any{"characteristic": c.name, "value": measurement(v, 3, "mm"), "tolerance": tol, "verdict": "within"})
	}
	return out
}
