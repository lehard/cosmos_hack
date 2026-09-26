package machinelogs_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
	ml "ant/internal/domain/machinelogs"
)

// weldingBadDay — «сварка вне режима» (FR-151): восемь сварок на ИС-2 по
// 10 минут через 15; ток вне уставки 160 ± 10 А с 20-й по 95-ю минуту;
// дефект найден только у трёх изделий окна. Плюс зачистка на том же
// источнике (не специальный процесс) внутри окна.
func weldingBadDay(j *journal) {
	for i := range 8 {
		run, item := fmt.Sprintf("RUN-W-%d", i), fmt.Sprintf("ENT01:FL-%d", i)
		j.started(run, item, "welding.weld", "IS-2", at(i*15))
		j.finished(run, item, at(i*15+10))
		if i >= 1 && i <= 3 {
			j.add(fmt.Sprintf("defect-%d", i), catalog.QualityDefectIdentified, item, "", at(i*15+30), map[string]any{"zone_id": "W-1"})
		}
	}
	j.started("RUN-CLEAN", "ENT01:FL-9", "welding.cleanup", "IS-2", at(50))
	j.finished("RUN-CLEAN", "ENT01:FL-9", at(52))
	j.deviation("dev-current", "IS-2", "out_of_setpoint", "current", measure(1800, 1, "A"), tol(1600, 1500, 1700, 1, "A"), at(20), at(95))
}

func TestSpecialProcessWindowAllItemsGetNC(t *testing.T) {
	j := &journal{}
	weldingBadDay(j)
	s, out := runStage(ml.StagePorts{Register: fakeRegistrar}, j.recs)

	var window []kernel.Addressed
	ncItems := map[string]bool{}
	for _, a := range out {
		switch a.Type {
		case catalog.EquipmentViolationWindowResolved:
			window = append(window, a)
		case catalog.DecisionNonconformityRegistered:
			ncItems[a.Stream] = true
		}
	}
	if len(window) != 1 || window[0].Stream != "equipment:IS-2" {
		t.Fatalf("окно нарушения: %+v", window)
	}
	b, _ := json.Marshal(window[0].Data)
	var wd struct {
		Runs  []string `json:"affected_operation_run_ids"`
		Items []string `json:"affected_item_ids"`
		Start string   `json:"window_start"`
		Step  string   `json:"step_key"`
	}
	_ = json.Unmarshal(b, &wd)
	wantRuns := []string{"RUN-W-1", "RUN-W-2", "RUN-W-3", "RUN-W-4", "RUN-W-5", "RUN-W-6"}
	if !slices.Equal(wd.Runs, wantRuns) || len(wd.Items) != 6 || wd.Step != "welding.weld" || wd.Start != "2026-09-26T07:20:00.000Z" {
		t.Fatalf("данные окна: %s", b)
	}
	if len(ncItems) != 6 {
		t.Fatalf("несоответствия: %v", ncItems)
	}
	defects := 0
	for i := 1; i <= 6; i++ {
		if !ncItems[fmt.Sprintf("item:ENT01:FL-%d", i)] {
			t.Fatalf("изделию FL-%d окна не зарегистрировано несоответствие", i)
		}
		if i <= 3 {
			defects++
		}
	}
	if 6-defects != 3 {
		t.Fatal("трое изделий окна — без найденного дефекта")
	}
	if ncItems["item:ENT01:FL-9"] || ncItems["item:ENT01:FL-0"] || ncItems["item:ENT01:FL-7"] {
		t.Fatal("несоответствие вне окна или не на специальном процессе")
	}
	if len(s.Pending) != 0 {
		t.Fatalf("с портом запросы не копятся: %+v", s.Pending)
	}
	// id несоответствия ссылается на запись окна.
	for _, a := range out {
		if a.Type == catalog.DecisionNonconformityRegistered {
			d := a.Data.(map[string]any)
			if d["violation_window_event_id"] != ml.WindowEventID("equipment:IS-2", "dev-current") {
				t.Fatalf("ссылка на окно: %v", d)
			}
		}
	}
}

// Без модуля nonconformity (порт пуст) — запросы копятся в состоянии стадии.
func TestSpecialProcessWithoutRegistrarKeepsPending(t *testing.T) {
	j := &journal{}
	weldingBadDay(j)
	s, out := runStage(ml.StagePorts{}, j.recs)
	if len(s.Pending) != 6 {
		t.Fatalf("запросов несоответствия: %d", len(s.Pending))
	}
	for _, a := range out {
		if a.Type == catalog.DecisionNonconformityRegistered {
			t.Fatal("без порта записи nonconformity нет")
		}
	}
}

// Позднее выполнение (записи о сварке пришли после закрытия окна) тоже
// получает несоответствие; порядок поступления окна и выполнений не важен.
func TestLateRunInClosedWindow(t *testing.T) {
	j := &journal{}
	j.deviation("dev-current", "IS-2", "out_of_setpoint", "current", measure(1800, 1, "A"), tol(1600, 1500, 1700, 1, "A"), at(20), at(95))
	j.started("RUN-LATE", "ENT01:FL-L", "welding.weld", "IS-2", at(30))
	j.finished("RUN-LATE", "ENT01:FL-L", at(40))
	_, out := runStage(ml.StagePorts{Register: fakeRegistrar}, j.recs)
	n := 0
	for _, a := range out {
		if a.Type == catalog.DecisionNonconformityRegistered && a.Stream == "item:ENT01:FL-L" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("позднее выполнение в окне: %d несоответствий", n)
	}
}

// Окно без конца отклонения закрывается подтверждением нормы.
func TestOpenWindowClosedByNormalCycle(t *testing.T) {
	j := &journal{}
	j.started("RUN-A", "ENT01:FL-A", "welding.weld", "IS-2", at(0))
	j.add("dev-open", catalog.EquipmentDeviationDetected, "", "", at(2), map[string]any{
		"equipment_id": "IS-2", "deviation_kind": "out_of_setpoint", "parameter": "current", "started_at": stamp(at(2))})
	j.finished("RUN-A", "ENT01:FL-A", at(10))
	s, out := runStage(ml.StagePorts{Register: fakeRegistrar}, j.recs)
	if s.Windows["dev-open"].End != nil {
		t.Fatal("отклонение без конца — окно открыто")
	}
	for _, a := range out {
		if a.Type == catalog.EquipmentViolationWindowResolved {
			t.Fatal("открытое окно не разрешается")
		}
	}
	j.add("cycle-ok", catalog.EquipmentCycleSummarized, "", "", at(30), map[string]any{
		"equipment_id": "IS-2", "window_start": stamp(at(20)), "window_end": stamp(at(30)),
		"parameters": []any{map[string]any{"parameter": "current", "max": measure(1650, 1, "A"), "min": measure(1560, 1, "A"), "setpoint": tol(1600, 1500, 1700, 1, "A"), "out_of_setpoint_ms": 0}}})
	f := ml.StageWith(ml.StagePorts{Register: fakeRegistrar})
	_, out = f(s, j.recs[len(j.recs)-1])
	var nc, win int
	for _, a := range out {
		switch a.Type {
		case catalog.EquipmentViolationWindowResolved:
			win++
		case catalog.DecisionNonconformityRegistered:
			nc++
		}
	}
	if win != 1 || nc != 1 {
		t.Fatalf("окно %d, несоответствий %d", win, nc)
	}
}

// Привязка: события оборудования в интервале выполнения и обстановка до
// начала адресуются в поток изделия; поздние — при поступлении; повторов нет.
func TestBindingToOperationRun(t *testing.T) {
	j := &journal{}
	j.add("prog", catalog.EquipmentProgramChanged, "", "", at(-5), map[string]any{"equipment_id": "CNC-1", "program_ref": "O1001", "program_revision": "B", "planned": true})
	j.state("st-run", "CNC-1", "running", "automatic", "normal", at(0))
	j.started("RUN-C", "ENT01:FL-C", "machining.turn", "CNC-1", at(0))
	j.deviation("dev-feed", "CNC-1", "manual_override", "feed_override", measure(130, 0, "%"), nil, at(3), at(6))
	j.finished("RUN-C", "ENT01:FL-C", at(10))
	j.deviation("dev-late", "CNC-1", "overload", "spindle_load", measure(118, 0, "%"), tol(80, 0, 100, 0, "%"), at(7), at(8))
	j.state("st-after", "CNC-1", "idle", "automatic", "normal", at(20))
	s, out := runStage(ml.StagePorts{}, j.recs)
	got := map[string]string{}
	for _, a := range out {
		if a.Type != catalog.EquipmentEventBound {
			t.Fatalf("лишняя запись %s", a.Type)
		}
		if a.Stream != "item:ENT01:FL-C" {
			t.Fatalf("поток %s", a.Stream)
		}
		d := a.Data
		b, _ := json.Marshal(d)
		var x struct {
			ID      string `json:"subject_event_id"`
			Binding string `json:"binding"`
		}
		_ = json.Unmarshal(b, &x)
		if _, dup := got[x.ID]; dup {
			t.Fatalf("повтор привязки %s", x.ID)
		}
		got[x.ID] = x.Binding
	}
	want := map[string]string{"prog": "context", "st-run": "interval", "dev-feed": "interval", "dev-late": "interval"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("привязка: %v", got)
	}
	if len(s.Runs["RUN-C"].BoundIDs) != 4 {
		t.Fatalf("привязано: %v", s.Runs["RUN-C"].BoundIDs)
	}
}

// Детерминизм и чистота: тот же вход — тот же выход; прежнее состояние не меняется.
func TestStageDeterministic(t *testing.T) {
	j := &journal{}
	weldingBadDay(j)
	s1, o1 := runStage(ml.StagePorts{Register: fakeRegistrar}, j.recs)
	s2, o2 := runStage(ml.StagePorts{Register: fakeRegistrar}, j.recs)
	b1, _ := json.Marshal([]any{s1, o1})
	b2, _ := json.Marshal([]any{s2, o2})
	if string(b1) != string(b2) {
		t.Fatal("стадия недетерминирована")
	}
	before, _ := json.Marshal(s1)
	f := ml.StageWith(ml.StagePorts{Register: fakeRegistrar})
	j.started("RUN-X", "ENT01:FL-X", "welding.weld", "IS-2", at(200))
	_, _ = f(s1, j.recs[len(j.recs)-1])
	after, _ := json.Marshal(s1)
	if string(before) != string(after) {
		t.Fatal("шаг стадии изменил прежнее состояние")
	}
}
