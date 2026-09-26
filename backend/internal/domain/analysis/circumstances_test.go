package analysis_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/analysis"
	"ant/internal/domain/kernel"
)

// item — вход свёртки изделия (AD-5): записи с item_id.
type item struct {
	id  string
	j   journal
	in  []kernel.Record
	eqs []analysis.EquipmentEvent
}

func newItem(id string) *item { return &item{id: "ENT01:" + id} }

func (it *item) rec(t catalog.Type, kind catalog.Kind, at time.Time, data any) kernel.Record {
	raw, _ := json.Marshal(data)
	it.j.seq++
	r := kernel.Record{Seq: it.j.seq, EventID: it.j.id(string(t)), Type: t, Kind: kind, Stream: "item:" + it.id, ItemID: it.id,
		OccurredAt: at, ReceivedAt: at, Data: raw, SourceKind: "device"}
	it.in = append(it.in, r)
	return r
}

// eq — событие оборудования из порта временной линии (эпик 23 — заготовка).
func (it *item) eq(t catalog.Type, at time.Time, data map[string]any) string {
	raw, _ := json.Marshal(data)
	it.j.seq++
	r := kernel.Record{Seq: it.j.seq, EventID: it.j.id(string(t)), Type: t, Kind: catalog.KindFact, Stream: "equipment:IS-2", OccurredAt: at, Data: raw}
	e, ok := analysis.EquipmentEventOf(r)
	if !ok {
		panic("не событие оборудования")
	}
	it.eqs = append(it.eqs, e)
	return r.EventID
}

func (it *item) inspect(at time.Time, outcome, phase string, defects []map[string]any) kernel.Record {
	return it.rec(catalog.InspectionResultRecorded, catalog.KindFact, at, map[string]any{"outcome": outcome, "phase": phase,
		"method": "camera", "processing_state": "complete", "zone_ids": []string{"W-1.U2"}, "defects": defects})
}

func (it *item) nc(at time.Time, id, defect string) {
	it.rec(catalog.DecisionNonconformityConfirmed, catalog.KindDecision, at, map[string]any{"nc_id": id, "defect_type_code": defect,
		"severity": "major", "signal_ids": []string{}, "reason": map[string]any{"text": "подтверждено"}})
}

// state — состояние модуля analysis после свёртки входа изделия (AD-5): вход
// по occurred_at → received_at → event_id, Reduce и React на каждом шаге, на
// слот — последняя реакция (как domain/engine.Fold; его композицию проверяет
// application/analysis).
func (it *item) state() (analysis.State, []kernel.Reaction) {
	in := slices.Clone(it.in)
	slices.SortStableFunc(in, func(x, y kernel.Record) int {
		switch {
		case kernel.Less(x, y):
			return -1
		case kernel.Less(y, x):
			return 1
		}
		return 0
	})
	var s analysis.State
	bySlot := map[string]kernel.Reaction{}
	var keys []string
	for _, r := range in {
		s = analysis.Reduce(s, r, analysis.Env{}, analysis.Upstream{})
		for _, re := range analysis.React(s, analysis.Env{}, analysis.Upstream{}).Reactions {
			if _, ok := bySlot[re.Slot.Key()]; !ok {
				keys = append(keys, re.Slot.Key())
			}
			bySlot[re.Slot.Key()] = re
		}
	}
	slices.Sort(keys)
	var rs []kernel.Reaction
	for _, k := range keys {
		rs = append(rs, bySlot[k])
	}
	return s, rs
}

// weldItem — сварка Ф-017 «сварка вне режима»: чистая КТ-2 до сварки,
// сварка на ИС-2 сварщиком W21, ручная перестановка детали, прожог на КТ-3.
func weldItem(operator any) *item {
	it := newItem("F-017")
	it.inspect(at(23, 8, 50), "no_defect_indicated", "before_operation", nil)
	it.rec(catalog.OperationRunStarted, catalog.KindFact, at(23, 10, 40), map[string]any{"operation_run_id": "RUN-17", "step_key": "welding.weld",
		"operation_code": "030", "equipment_id": "IS-2", "operator_id": operator, "program_ref": "WPS-12"})
	it.rec(catalog.OperatorOverridePerformed, catalog.KindFact, at(23, 10, 50), map[string]any{"bypassed": "other", "operator_id": "W21",
		"operation_run_id": "RUN-17", "reason": map[string]any{"text": "деталь переставлена в приспособлении"}})
	it.rec(catalog.OperationRunFinished, catalog.KindFact, at(23, 11, 0), map[string]any{"operation_run_id": "RUN-17", "completion": "completed"})
	it.inspect(at(23, 11, 6), "defect_indicated", "after_operation", []map[string]any{{"zone_id": "W-1.U2", "defect_type_code": "burn_through", "severity": "major"}})
	it.nc(at(23, 11, 8), "NC-01", "burn_through")
	return it
}

// «Сварка вне режима» (FR-153): окно возможного возникновения и отклонения
// оборудования на одной шкале с результатами контроля; три дорожки.
func TestCircumstancesThreeLanesAndWindow(t *testing.T) {
	it := weldItem("W21")
	it.eq(catalog.EquipmentToolChanged, at(23, 7, 0), map[string]any{"equipment_id": "IS-2", "tool_id": "TORCH-7"})
	dev := it.eq(catalog.EquipmentDeviationDetected, at(23, 10, 45), map[string]any{"equipment_id": "IS-2", "deviation_kind": "out_of_setpoint",
		"started_at": "2026-09-23T07:45:00.000Z", "ended_at": "2026-09-23T07:55:00.000Z", "parameter": "current",
		"value":    map[string]any{"value": 176, "unit": "A", "scale": 0},
		"setpoint": map[string]any{"lower": map[string]any{"value": 150, "unit": "A", "scale": 0}, "upper": map[string]any{"value": 170, "unit": "A", "scale": 0}}})
	s, _ := it.state()
	a, ok := analysis.Analyze(s, it.id, "NC-01", it.eqs)
	if !ok {
		t.Fatal("несоответствие не найдено")
	}
	if a.Window == nil || !a.Window.Start.Equal(at(23, 8, 50)) || !a.Window.End.Equal(at(23, 11, 6)) {
		t.Fatalf("окно: %+v", a.Window)
	}
	lanes := map[string]int{}
	for _, m := range a.Records {
		lanes[m.Lane]++
	}
	if lanes[analysis.LaneItem] != 2 || lanes[analysis.LanePerson] < 2 || lanes[analysis.LaneEquipment] != 1 {
		t.Fatalf("дорожки: %v", lanes)
	}
	var eqH *analysis.Hypothesis
	for i := range a.Hypotheses {
		if a.Hypotheses[i].Category == analysis.CatEquipment {
			eqH = &a.Hypotheses[i]
		}
	}
	if eqH == nil || eqH.ConfidenceBP == nil || *eqH.ConfidenceBP < 8500 || !slices.Contains(eqH.Supporting, dev) {
		t.Fatalf("гипотеза «оборудование»: %+v", eqH)
	}
	if a.Operation == nil || a.Operation.Equipment != "IS-2" || a.Profile.Tool != "TORCH-7" || a.Profile.Performer != "W21" {
		t.Fatalf("операция и профиль: %+v %+v", a.Operation, a.Profile)
	}
	if len(a.Missing) != 0 || !a.Categorical {
		t.Fatalf("сведений хватает — вывод категоричен: %v %v", a.Missing, a.Categorical)
	}
}

// При недостатке сведений система не делает категоричного вывода (FR-58,
// кейс §5.1): исполнитель неизвестен, журнала оборудования за интервал нет.
func TestMissingInformationNotCategorical(t *testing.T) {
	it := weldItem(nil)
	s, _ := it.state()
	a, _ := analysis.Analyze(s, it.id, "NC-01", []analysis.EquipmentEvent{})
	for _, m := range []string{analysis.MissingOperator, analysis.MissingEquipmentLog, analysis.MissingTool} {
		if !slices.Contains(a.Missing, m) {
			t.Fatalf("нет %q в нехватке сведений: %v", m, a.Missing)
		}
	}
	if a.Categorical {
		t.Fatal("категоричный вывод при нехватке сведений")
	}
	for _, h := range a.Hypotheses {
		if h.Category == analysis.CatEquipment && h.ConfidenceBP != nil {
			t.Fatalf("оборудование без журнала оценено: %+v", h)
		}
	}
	// Порт оборудования не подключён — выводов об оборудовании нет, вывод не категоричен.
	b, _ := analysis.Analyze(s, it.id, "NC-01", nil)
	if b.Categorical || slices.Contains(b.Missing, analysis.MissingEquipmentLog) {
		t.Fatalf("без порта: %v %v", b.Categorical, b.Missing)
	}
}

// Входной дефект не связывается с исполнителем операции (FR-58): нет
// гипотезы «исполнитель», нет событий его дорожки, нет факторов операции.
func TestIncomingDefectNotLinkedToPerformer(t *testing.T) {
	it := newItem("F-019")
	it.rec(catalog.OperationRunStarted, catalog.KindFact, at(22, 16, 35), map[string]any{"operation_run_id": "RUN-19", "step_key": "welding.weld",
		"operation_code": "030", "equipment_id": "IS-2", "operator_id": "W22", "program_ref": "WPS-12"})
	it.rec(catalog.OperatorOverridePerformed, catalog.KindFact, at(22, 16, 50), map[string]any{"bypassed": "other", "operator_id": "W22", "operation_run_id": "RUN-19"})
	it.rec(catalog.OperationRunFinished, catalog.KindFact, at(22, 17, 20), map[string]any{"operation_run_id": "RUN-19", "completion": "completed"})
	it.inspect(at(23, 13, 41), "defect_indicated", "after_operation", []map[string]any{{"zone_id": "RING-BODY", "defect_type_code": "base_metal_pore",
		"component_ref": "ENT01:R-101", "severity": "major"}})
	it.nc(at(23, 13, 50), "NC-04", "base_metal_pore")
	s, _ := it.state()
	a, _ := analysis.Analyze(s, it.id, "NC-04", []analysis.EquipmentEvent{})
	if !a.Incoming || a.Operation != nil || a.Profile.Performer != "" || a.Profile.Equipment != "" {
		t.Fatalf("входной дефект связан с операцией: %+v", a)
	}
	for _, h := range a.Hypotheses {
		if h.Category == analysis.CatPerformer || h.Category == analysis.CatEquipment {
			t.Fatalf("гипотеза %s у входного дефекта", h.Category)
		}
	}
	for _, m := range a.Records {
		if m.Lane == analysis.LanePerson {
			t.Fatalf("событие исполнителя у входного дефекта: %+v", m)
		}
	}
	if len(a.Hypotheses) != 1 || a.Hypotheses[0].Category != analysis.CatIncoming {
		t.Fatalf("гипотезы: %+v", a.Hypotheses)
	}
}

// Версия вывода разбора — реакция incident.hypothesis.computed (AD-3): слот
// правило × изделие × несоответствие; «подтверждённой» гипотезы система не
// пишет (FR-59); без порта оборудования вывод не категоричен.
func TestHypothesisComputedReaction(t *testing.T) {
	it := weldItem("W21")
	s, rs := it.state()
	if len(s.Cases) != 1 || len(rs) != 1 {
		t.Fatalf("реакции: %d, несоответствий %d", len(rs), len(s.Cases))
	}
	re := rs[0]
	if re.Type != catalog.IncidentHypothesisComputed || re.Slot.Subject != "item:"+it.id || re.Slot.TriggerKey != "NC-01" || re.Module != analysis.Module {
		t.Fatalf("реакция: %+v", re)
	}
	raw, _ := json.Marshal(re.Data)
	var d struct {
		Hypotheses  []map[string]any `json:"hypotheses"`
		Categorical bool             `json:"conclusion_is_categorical"`
		Start       string           `json:"causal_window_start"`
	}
	_ = json.Unmarshal(raw, &d)
	if d.Categorical || d.Start != "2026-09-23T05:50:00.000Z" || len(d.Hypotheses) == 0 {
		t.Fatalf("data: %s", raw)
	}
	for _, h := range d.Hypotheses {
		if _, has := h["status"]; has {
			t.Fatalf("система поставила статус гипотезе: %v", h)
		}
	}
}

// Общие факторы «сколько из N» (FR-135): общий для всех — первым;
// неизвестное значение совпадением не считается.
func TestCommonFactors(t *testing.T) {
	ps := []analysis.Profile{
		{NCID: "1", Equipment: "IS-2", Performer: "W21", Program: "WPS-12"},
		{NCID: "2", Equipment: "IS-2", Performer: "W22", Program: "WPS-12"},
		{NCID: "3", Equipment: "IS-2", Performer: "W21"},
	}
	rows := analysis.CommonFactors(ps)
	if rows[0].Factor != analysis.FactorMachine || rows[0].Matches != 3 || rows[0].Value != "IS-2" {
		t.Fatalf("первая строка: %+v", rows[0])
	}
	byF := map[string]analysis.FactorRow{}
	for _, r := range rows {
		byF[r.Factor] = r
	}
	if p := byF[analysis.FactorPerformer]; p.Matches != 2 || p.Distinct != 2 || p.Value != "W21" {
		t.Fatalf("исполнитель: %+v", p)
	}
	if p := byF[analysis.FactorProgram]; p.Matches != 2 {
		t.Fatalf("программа (одна неизвестна): %+v", p)
	}
	if tl := byF[analysis.FactorTool]; tl.Matches != 0 || tl.Value != "" {
		t.Fatalf("инструмент неизвестен: %+v", tl)
	}
}

// Похожие случаи по правилам (FR-60): тот же вид дефекта и та же операция
// или то же оборудование, не позже текущего; сначала совпавшие по обоим.
func TestSimilarCases(t *testing.T) {
	target := analysis.Profile{NCID: "NC-03", DefectType: "burn_through", StepKey: "welding.weld", Equipment: "IS-2", At: at(23, 13, 20)}
	others := []analysis.Profile{
		{NCID: "NC-01", DefectType: "burn_through", StepKey: "welding.weld", Equipment: "IS-2", At: at(23, 11, 8)},
		{NCID: "NC-00", DefectType: "burn_through", StepKey: "welding.weld", Equipment: "IS-1", At: at(15, 11, 8)},
		{NCID: "NC-04", DefectType: "base_metal_pore", StepKey: "welding.weld", Equipment: "IS-2", At: at(23, 13, 50)},
		{NCID: "NC-09", DefectType: "burn_through", StepKey: "welding.weld", Equipment: "IS-2", At: at(24, 9, 0)},
		{NCID: "NC-07", DefectType: "burn_through", StepKey: "machining.turn", Equipment: "CNC-1", At: at(20, 9, 0)},
	}
	got := analysis.Similar(target, others)
	var ids []string
	for _, s := range got {
		ids = append(ids, s.NCID)
	}
	if !slices.Equal(ids, []string{"NC-01", "NC-00"}) {
		t.Fatalf("похожие: %v", ids)
	}
}
