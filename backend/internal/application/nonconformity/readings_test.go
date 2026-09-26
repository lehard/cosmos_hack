package nonconformity

import (
	"encoding/json"
	"testing"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Числа режима в карточке: сводка цикла и отклонение — уставка и наблюдённые
// значения числами, масштабы приведены к общему.
func TestReadingOf(t *testing.T) {
	m := func(v int64, scale int) map[string]any {
		return map[string]any{"value": v, "scale": scale, "unit": "A"}
	}
	sum, _ := json.Marshal(map[string]any{"equipment_id": "IS-2", "parameters": []map[string]any{
		{"parameter": "voltage_v", "min": map[string]any{"value": 20, "scale": 0, "unit": "V"}},
		{"parameter": "current_a", "min": m(176, 0), "max": m(1825, 1), "out_of_setpoint_ms": 60000,
			"setpoint": map[string]any{"nominal": m(160, 0), "lower": m(150, 0), "upper": m(170, 0)}},
	}})
	r := readingOf(kernel.Record{Type: catalog.EquipmentCycleSummarized, Data: sum})
	if r == nil || r.Parameter != "current_a" || r.Unit != "A" || r.Scale != 1 || *r.ObservedMin != 1760 || *r.ObservedMax != 1825 ||
		*r.SetpointMin != 1500 || *r.SetpointMax != 1700 || *r.SetpointNominal != 1600 {
		t.Fatalf("сводка цикла: %+v", r)
	}
	dev, _ := json.Marshal(map[string]any{"equipment_id": "IS-2", "parameter": "current_a", "value": m(182, 0),
		"setpoint": map[string]any{"upper": m(170, 0)}})
	r = readingOf(kernel.Record{Type: catalog.EquipmentDeviationDetected, Data: dev})
	if r == nil || *r.ObservedMin != 182 || *r.ObservedMax != 182 || r.SetpointMin != nil || *r.SetpointMax != 170 {
		t.Fatalf("отклонение: %+v", r)
	}
	if readingOf(kernel.Record{Type: catalog.InspectionResultRecorded, Data: dev}) != nil {
		t.Fatal("не запись режима — чисел нет")
	}
}
