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
