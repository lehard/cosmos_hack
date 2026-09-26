package verify

import (
	"strings"
	"testing"

	dom "ant/internal/domain/security"
)

// AD-28: правка проекции сдерживания — «проекция расходится с журналом:
// изделие X в журнале „заблокировано“ (CA-…), в проекции „разрешено“».
func TestProjectionMismatchMessage(t *testing.T) {
	v := &run{ca: []caRec{
		{seq: 3, r: dom.Record{ObjectRef: "item:F-023", ActionType: "decision.containment.set"}},
		{seq: 4, r: dom.Record{ObjectRef: "item:F-024", ActionType: "decision.containment.set"}},
	}}
	c := &check{name: "projections"}
	v.mismatch(c, "F-023", "nonconformity.item", []byte(`{"containment":"item_hold","item_id":"F-023"}`), []byte(`{"containment":"none","item_id":"F-023"}`))
	if !c.rejected || len(c.findings) != 1 {
		t.Fatal("нарушение не найдено")
	}
	f := c.findings[0]
	want := "проекция расходится с журналом: изделие F-023 в журнале «заблокировано: Блок изделия» (CA-3), в проекции «разрешено»"
	if f.Detail != want || f.CaRef == nil || *f.CaRef != "CA-3" {
		t.Fatalf("находка: %q", f.Detail)
	}
	c2 := &check{name: "projections"}
	v.mismatch(c2, "F-024", "quality.item", []byte(`{"status":"rework"}`), []byte(`{"status":"accepted"}`))
	if !strings.Contains(c2.findings[0].Detail, `поле status — в журнале "rework", в проекции "accepted"`) {
		t.Fatalf("находка: %q", c2.findings[0].Detail)
	}
}
