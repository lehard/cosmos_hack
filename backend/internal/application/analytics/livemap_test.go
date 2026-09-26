package analytics_test

import (
	"reflect"
	"testing"

	analytics "ant/internal/application/analytics"
	processapp "ant/internal/application/process"
)

// Счётчики узлов analytics и живая карта process (process.live_map.read)
// — одна форма по step_key (FR-2, FR-154, AD-21): поля, их JSON-имена и
// перечни видов аномалий совпадают, так что process копирует ответ
// analytics.node_counters.read без пересчёта.
func TestNodeCountersMatchLiveMap(t *testing.T) {
	pairs := []struct{ a, p any }{
		{analytics.NodeCounters{}, processapp.MapNodeCounters{}},
		{analytics.NodeAnomaly{}, processapp.MapNodeAnomaly{}},
		{analytics.Bottleneck{}, processapp.MapBottleneck{}},
	}
	for _, pr := range pairs {
		at, pt := reflect.TypeOf(pr.a), reflect.TypeOf(pr.p)
		if at.NumField() != pt.NumField() {
			t.Fatalf("%s и %s: разное число полей", at.Name(), pt.Name())
		}
		for i := range at.NumField() {
			fa, fp := at.Field(i), pt.Field(i)
			if fa.Name != fp.Name || jsonName(fa) != jsonName(fp) || fa.Tag.Get("enum") != fp.Tag.Get("enum") {
				t.Errorf("%s.%s (%s) ≠ %s.%s (%s)", at.Name(), fa.Name, fa.Tag, pt.Name(), fp.Name, fp.Tag)
			}
		}
	}
}

func jsonName(f reflect.StructField) string {
	n := f.Tag.Get("json")
	for i := range len(n) {
		if n[i] == ',' {
			return n[:i]
		}
	}
	return n
}

// Имена узлов рядом с step_key (UI-21): счётчики, ограничение, аномалии —
// step_name; data_gaps (строки) — словарь step_names; нет имени — кода хватит.
func TestNodeCounterSetNameSteps(t *testing.T) {
	set := analytics.NodeCounterSet{Counters: []analytics.NodeCounters{{StepKey: "welding.weld"}, {StepKey: "x.unknown"}},
		Bottleneck: &analytics.Bottleneck{StepKey: "welding.weld"}, Anomalies: []analytics.NodeAnomaly{{StepKey: "welding.weld"}},
		DataGaps: []string{"welding.kt3_camera"}}
	set.NameSteps(map[string]string{"welding.weld": "Сварка", "welding.kt3_camera": "КТ-3 камера", "other": "Лишний"})
	if set.Counters[0].StepName == nil || *set.Counters[0].StepName != "Сварка" || set.Counters[1].StepName != nil ||
		set.Bottleneck.StepName == nil || set.Anomalies[0].StepName == nil {
		t.Fatalf("step_name: %+v", set)
	}
	if len(set.StepNames) != 2 || set.StepNames["welding.kt3_camera"] != "КТ-3 камера" || set.StepNames["other"] != "" {
		t.Fatalf("step_names: %v", set.StepNames)
	}
}
