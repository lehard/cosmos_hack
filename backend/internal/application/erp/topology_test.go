package erp

import (
	"os"
	"testing"
)

// Цеха и склады фланца из нормативного слоя (FR-130): дорожки BPMN в порядке
// laneSet, события «в 1С» — на дорожках цехов-получателей.
func TestTopologyFromFlangeBPMN(t *testing.T) {
	b, err := os.ReadFile("../../../../normative/process/flange-process.bpmn")
	if err != nil {
		t.Fatal(err)
	}
	topo, err := TopologyFromBPMN(b)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"WH-SK", "WH-MC", "WH-WC", "WH-AC", "WH-FG"}
	if len(topo.Lanes) != len(want) {
		t.Fatalf("дорожек %d", len(topo.Lanes))
	}
	for i, w := range want {
		if topo.Lanes[i].Warehouse != w {
			t.Fatalf("дорожка %d: склад %s, ожидался %s", i, topo.Lanes[i].Warehouse, w)
		}
	}
	for step, wh := range map[string]string{"incoming.erp_accept_blank": "WH-SK", "welding.erp_transfer_in": "WH-WC",
		"assembly.erp_transfer_in": "WH-AC", "final.erp_release": "WH-FG"} {
		if got := topo.StepWarehouse(step); got != wh {
			t.Errorf("%s: склад %q, ожидался %s", step, got, wh)
		}
	}
	if _, ok := topo.LaneOf("nc.erp_scrap_rework"); ok {
		t.Error("подпроцесс «Брак» вне дорожек цехов")
	}
}
