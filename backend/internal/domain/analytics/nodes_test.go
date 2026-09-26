package analytics_test

import (
	"fmt"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	domain "ant/internal/domain/analytics"
	"ant/internal/domain/kernel"
)

// «Специалист заснул» (FR-5): изделия предъявлены на ЗТ-3, решения нет —
// очередь копится; узел точки предъявления — ограничение линии и аномалия.
func TestSpecialistAsleep(t *testing.T) {
	var rows []domain.Row
	for i := range 6 {
		id := fmt.Sprintf("ENT01:FL-%03d", i)
		at := time.Duration(i) * 10 * time.Minute
		in := []kernel.Record{
			rec(fmt.Sprintf("s%d", i), catalog.OperationRunStarted, at, fmt.Sprintf(`{"operation_run_id":"R%d","operation_code":"welding","step_key":"welding.weld","equipment_id":"IS-2","operator_id":"W21"}`, i)),
			rec(fmt.Sprintf("f%d", i), catalog.OperationRunFinished, at+20*time.Minute, fmt.Sprintf(`{"operation_run_id":"R%d","completion":"completed"}`, i)),
			rec(fmt.Sprintf("p%d", i), catalog.ItemPresentationRecorded, at+25*time.Minute, `{"step_key":"welding.zt3_acceptance","presentation_no":1,"presented_to":"qc","presented_by":"M01"}`),
		}
		for j := range in {
			in[j].ItemID = id
		}
		for _, r := range domain.Contribute(id, in) {
			r.Item = id
			rows = append(rows, r)
		}
	}
	// Одно изделие ждёт начала сварки после приёма на участке.
	w := []kernel.Record{rec("m", catalog.OperationMovementReceived, 100*time.Minute, `{"to_location_id":"ST-2","destination_kind":"station","inspection_on_receipt":"no_damage","received_by":"W21","step_key":"welding.weld"}`)}
	rows = append(rows, domain.Contribute("ENT01:FL-100", w)...)

	now := t0.Add(3 * time.Hour)
	nodes := domain.Nodes(rows, t0, now)
	var zt3 domain.NodeCount
	for _, n := range nodes {
		if n.Step == "welding.zt3_acceptance" {
			zt3 = n
		}
	}
	if zt3.Queue != 6 || zt3.InProgress != 0 {
		t.Fatalf("очередь ЗТ-3: %+v", zt3)
	}
	b, ok := domain.Bottleneck(nodes)
	if !ok || b.Step != "welding.zt3_acceptance" {
		t.Fatalf("ограничение линии: %+v %v", b, ok)
	}
	an := domain.Anomalies(nodes, rows, nil, t0, now, time.Hour, func(string) domain.Norm { return domain.DefaultNorm })
	kinds := map[string]bool{}
	for _, a := range an {
		if a.Step == "welding.zt3_acceptance" {
			kinds[a.Kind] = true
		}
	}
	if !kinds[domain.AnomalyQueue] || !kinds[domain.AnomalyWait] {
		t.Fatalf("аномалии ЗТ-3: %+v", an)
	}
	// Специалист проснулся: решения по всем — очереди нет, ограничения нет.
	var woke []domain.Row
	for i := range 6 {
		id := fmt.Sprintf("ENT01:FL-%03d", i)
		at := time.Duration(i) * 10 * time.Minute
		in := []kernel.Record{
			rec(fmt.Sprintf("p%d", i), catalog.ItemPresentationRecorded, at+25*time.Minute, `{"step_key":"welding.zt3_acceptance","presentation_no":1,"presented_to":"qc","presented_by":"M01"}`),
			rec(fmt.Sprintf("r%d", i), catalog.DecisionPresentationResolved, at+30*time.Minute, `{"step_key":"welding.zt3_acceptance","closing_point":"ZT-3","resolution":"accept","presentation_no":1,"method_event_ids":[]}`),
		}
		woke = append(woke, domain.Contribute(id, in)...)
	}
	if _, ok := domain.Bottleneck(domain.Nodes(woke, t0, now)); ok {
		t.Fatal("после решений ограничения быть не должно")
	}
}

// Простой оборудования: переходы вне порядка и повтор дают те же интервалы.
func TestEquipmentDowntime(t *testing.T) {
	st := func(id string, at time.Duration, exec, cond string) kernel.Record {
		r := rec(id, catalog.EquipmentStateChanged, at, fmt.Sprintf(`{"equipment_id":"IS-2","execution":"%s","controller_mode":"automatic","condition":"%s"}`, exec, cond))
		r.ItemID, r.Stream = "", "equipment:IS-2"
		return r
	}
	evs := []kernel.Record{st("a", 0, "running", "normal"), st("b", 10*time.Minute, "stopped", "normal"), st("c", 40*time.Minute, "running", "normal"), st("d", 50*time.Minute, "running", "fault")}
	fold := func(order []int) []domain.Row {
		var s domain.Equipment
		for _, i := range order {
			keys := domain.EquipmentKeys(evs[i])
			s = domain.EquipmentStep(s, keys[0], evs[i])
		}
		return s.Downtime()
	}
	a := fold([]int{0, 1, 2, 3})
	b := fold([]int{3, 1, 0, 2, 1, 3})
	if len(a) != 2 || len(b) != 2 || a[0].Until == nil || a[0].Until.Sub(a[0].At) != 30*time.Minute || a[1].Until != nil {
		t.Fatalf("интервалы простоя: %+v / %+v", a, b)
	}
	if a[0].Overlap(t0, t0.Add(time.Hour)) != 1800 || a[1].Overlap(t0, t0.Add(time.Hour)) != 600 {
		t.Fatal("время простоя в периоде")
	}
}
