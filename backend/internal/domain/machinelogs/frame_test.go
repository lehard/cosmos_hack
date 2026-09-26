package machinelogs_test

import (
	"encoding/json"
	"testing"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
	ml "ant/internal/domain/machinelogs"
)

// cncDeviations — выполнение на станке ЧПУ с двумя отклонениями (FR-149):
// ручная коррекция подачи 130 % и перегрузка шпинделя; до начала — чисто.
func cncDeviations(j *journal) {
	j.add("prog", catalog.EquipmentProgramChanged, "", "", at(-5), map[string]any{"equipment_id": "CNC-1", "program_ref": "O1001", "program_revision": "B", "planned": true})
	j.add("tool", catalog.EquipmentToolChanged, "", "", at(-4), map[string]any{"equipment_id": "CNC-1", "tool_id": "T05", "tool_life_used": 73, "tool_life_limit": 75})
	j.state("st-ready", "CNC-1", "idle", "automatic", "normal", at(-1))
	j.started("RUN-C", "ENT01:FL-C", "machining.turn", "CNC-1", at(0))
	j.state("st-run", "CNC-1", "running", "automatic", "normal", at(0))
	j.deviation("dev-feed", "CNC-1", "manual_override", "feed_override", measure(130, 0, "%"), tol(100, 100, 100, 0, "%"), at(3), at(6))
	j.deviation("dev-load", "CNC-1", "overload", "spindle_load", measure(118, 0, "%"), tol(80, 0, 100, 0, "%"), at(4), at(5))
	j.add("cycle", catalog.EquipmentCycleSummarized, "", "", at(9), map[string]any{
		"equipment_id": "CNC-1", "window_start": stamp(at(0)), "window_end": stamp(at(9)), "cycle_ref": "C-1",
		"parameters": []any{map[string]any{"parameter": "spindle_load", "mean": measure(72, 0, "%"), "max": measure(118, 0, "%"), "min": measure(40, 0, "%"), "setpoint": tol(80, 0, 100, 0, "%"), "out_of_setpoint_ms": 60000}}})
	j.finished("RUN-C", "ENT01:FL-C", at(10))
}

// itemInput — вход свёртки изделия: его записи и адресованные стадией.
func itemInput(j *journal, item string, out []kernel.Addressed) []kernel.Record {
	var in []kernel.Record
	for _, r := range j.recs {
		if r.ItemID == item {
			in = append(in, r)
		}
	}
	for _, a := range out {
		if a.Stream == "item:"+item {
			in = append(in, addressedRecord(j, a))
		}
	}
	return in
}

func fold(in []kernel.Record) ml.State {
	var s ml.State
	for _, r := range in {
		s = ml.Reduce(s, r, ml.Env{}, ml.Upstream{})
	}
	return s
}

// FR-147, FR-148: ручная подача 130 % видна как отклонение в профиле
// выполнения операции; программа, инструмент и ресурс — из обстановки до начала.
func TestRunProfileShowsManualFeed130(t *testing.T) {
	j := &journal{}
	cncDeviations(j)
	_, out := runStage(ml.StagePorts{}, j.recs)
	s := fold(itemInput(j, "ENT01:FL-C", out))
	p, ok := s.Profile("RUN-C")
	if !ok {
		t.Fatal("нет профиля выполнения")
	}
	ov := p.ManualOverrides()
	if len(ov) != 1 || ov[0].Value == nil || ov[0].Value.Value != 130 || ov[0].Value.Unit != "%" || ov[0].Parameter != "feed_override" {
		t.Fatalf("ручная подача 130 %%: %+v", p.Deviations)
	}
	if len(p.Deviations) != 2 {
		t.Fatalf("два отклонения: %+v", p.Deviations)
	}
	if p.ProgramRef != "O1001" || p.ProgramRevision != "B" || p.ToolID != "T05" || *p.ToolLifeUsed != 73 || *p.ToolLifeLimit != 75 {
		t.Fatalf("программа и инструмент: %+v", p)
	}
	if p.ControllerMode != "automatic" || p.EquipmentID != "CNC-1" || p.FinishedAt == nil || p.SpecialProcess {
		t.Fatalf("режим и интервал: %+v", p)
	}
	if len(p.Parameters) != 1 || p.Parameters[0].InRange() == nil || *p.Parameters[0].InRange() {
		t.Fatalf("сводка нагрузки вне уставки: %+v", p.Parameters)
	}
	layers := map[ml.Layer]int{}
	for _, e := range p.Events {
		layers[e.Layer]++
	}
	if layers[ml.LayerWhat] == 0 || layers[ml.LayerWithWhat] == 0 || layers[ml.LayerHow] == 0 || layers[ml.LayerDeviation] != 2 {
		t.Fatalf("четыре слоя: %v", layers)
	}
}

// Нормальное выполнение: профиль без отклонений, сводка в пределах уставки.
func TestRunProfileNormal(t *testing.T) {
	j := &journal{}
	j.started("RUN-N", "ENT01:FL-N", "machining.turn", "CNC-1", at(0))
	j.add("cycle-n", catalog.EquipmentCycleSummarized, "", "", at(9), map[string]any{
		"equipment_id": "CNC-1", "window_start": stamp(at(0)), "window_end": stamp(at(9)),
		"parameters": []any{map[string]any{"parameter": "spindle_load", "max": measure(78, 0, "%"), "min": measure(40, 0, "%"), "setpoint": tol(80, 0, 100, 0, "%"), "out_of_setpoint_ms": 0}}})
	j.finished("RUN-N", "ENT01:FL-N", at(10))
	_, out := runStage(ml.StagePorts{}, j.recs)
	p, _ := fold(itemInput(j, "ENT01:FL-N", out)).Profile("RUN-N")
	if len(p.Deviations) != 0 || len(p.Parameters) != 1 || !*p.Parameters[0].InRange() {
		t.Fatalf("нормальное выполнение: %+v", p)
	}
}

// Несоответствие окна нарушения отмечает профиль (FR-151); свёртка сериализуема.
func TestProfileMarksViolation(t *testing.T) {
	j := &journal{}
	weldingBadDay(j)
	_, out := runStage(ml.StagePorts{Register: fakeRegistrar}, j.recs)
	s := fold(itemInput(j, "ENT01:FL-5", out))
	p, _ := s.Profile("RUN-W-5")
	if !p.SpecialProcess || !p.Violation() || len(p.Nonconformities) != 1 {
		t.Fatalf("профиль сварки в окне: %+v", p)
	}
	if len(p.Deviations) != 1 || p.Deviations[0].Kind != "out_of_setpoint" {
		t.Fatalf("ток вне уставки в профиле: %+v", p.Deviations)
	}
	if _, err := json.Marshal(s); err != nil {
		t.Fatal(err)
	}
	if o := ml.React(s, ml.Env{}, ml.Upstream{}); len(o.Reactions) != 0 {
		t.Fatal("у machinelogs нет реакций в потоке изделия")
	}
}

func TestClassification(t *testing.T) {
	for _, x := range []struct {
		cat  string
		want ml.Class
	}{{"execution", ml.ClassEvent}, {"condition", ml.ClassCondition}, {"alarm", ml.ClassCondition}, {"override", ml.ClassEvent}, {"program", ml.ClassEvent}} {
		if c, ok := ml.CategoryClass(x.cat); !ok || c != x.want {
			t.Fatalf("%s → %s", x.cat, c)
		}
	}
	if _, ok := ml.CategoryClass("vibration_raw"); ok {
		t.Fatal("неизвестная категория пропускается")
	}
	c, l := ml.Classify(ml.Event{Type: catalog.EquipmentCycleSummarized})
	if c != ml.ClassSample || l != ml.LayerHow {
		t.Fatal("сводка цикла — SAMPLE, «как шёл процесс»")
	}
	c, _ = ml.Classify(ml.Event{Type: catalog.EquipmentStateChanged, Condition: "fault"})
	if c != ml.ClassCondition {
		t.Fatal("неисправность — CONDITION")
	}
	// Один тип на смысл от любого источника: пометка источника сохраняется.
	for _, src := range []string{"manual_entry", "machine", "sensor", "external_system"} {
		r := kernel.Record{EventID: "e", Type: catalog.EquipmentStateChanged, SourceKind: src,
			Data: json.RawMessage(`{"equipment_id":"IS-2","execution":"running","controller_mode":"manual","condition":"normal"}`)}
		ev, ok, err := ml.ParseEvent(r)
		if !ok || err != nil || ev.SourceKind != src {
			t.Fatalf("источник %s: %+v %v", src, ev, err)
		}
	}
}

func TestToleranceWithScales(t *testing.T) {
	sp := ml.Tolerance{Lower: &ml.Measure{Value: 150, Scale: 0, Unit: "A"}, Upper: &ml.Measure{Value: 1700, Scale: 1, Unit: "A"}}
	if in, ok := sp.Contains(ml.Measure{Value: 1705, Scale: 1, Unit: "A"}); !ok || in {
		t.Fatal("170,5 А вне 150…170 А")
	}
	if in, ok := sp.Contains(ml.Measure{Value: 16000, Scale: 2, Unit: "A"}); !ok || !in {
		t.Fatal("160,00 А в пределах")
	}
	if _, ok := sp.Contains(ml.Measure{Value: 20, Scale: 0, Unit: "V"}); ok {
		t.Fatal("другая единица — оценка невозможна")
	}
}
