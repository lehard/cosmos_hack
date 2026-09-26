package documents

import (
	"testing"

	"ant/internal/application/platform"
)

// TestRegistryFilter — состояние реестра и отбор: «на бумаге» раньше «на
// подписи», подписанный с бумагой — «подписан»; отбор по изделию находит
// документы его несоответствий, вид — по id шаблона, поиск — по объекту.
func TestRegistryFilter(t *testing.T) {
	for _, c := range []struct{ status, paper, want string }{
		{"signing", "printed", StatePaper}, {"route_closed", "signed", StateSigned}, {"route_closed", "printed", StateSigned},
		{"live", "", StateDraft}, {"requested", "", StateDraft}, {"returned", "", StateReturned}, {"annulled", "printed", StateAnnulled},
	} {
		if got := RegistryState(c.status, c.paper); got != c.want {
			t.Errorf("RegistryState(%s, %s) = %s, ожидалось %s", c.status, c.paper, got, c.want)
		}
	}
	l := DocumentList{Items: []DocumentSummary{
		{DocumentID: "DOC-NC-01", Template: "nc-statement@1", Title: "Заявление", Status: "route_closed", Subject: platform.DrillRef{Entity: platform.EntityNonconformity, ID: "NC-01"},
			SubjectLabel: "НС-01", ItemIDs: []string{"ENT01:F-017"}, ProcessID: "Process_Flange"},
		{DocumentID: "TRV-ENT01:F-017", Template: "traveler@1", Status: "live", Subject: platform.DrillRef{Entity: platform.EntityItem, ID: "ENT01:F-017"}, ProcessID: "Process_Flange"},
		{DocumentID: "DOC-PVA-BRACKET-1", Template: "process-version-approval@1", Status: "route_closed", ProcessID: "Process_Bracket", ProcessVersionID: "bracket-1",
			Subject: platform.DrillRef{Entity: platform.EntityProcessVersion, ID: "bracket-1"}},
	}}
	count := func(f DocumentFilter) int { return len(FilterDocuments(l, f).Items) }
	if n := count(DocumentFilter{ItemID: "ENT01:F-017"}); n != 2 {
		t.Errorf("по изделию: %d, ожидалось 2", n)
	}
	if n := count(DocumentFilter{ProcessID: "Process_Bracket"}); n != 1 {
		t.Errorf("по процессу: %d", n)
	}
	if n := count(DocumentFilter{Template: "nc-statement"}); n != 1 {
		t.Errorf("по виду: %d", n)
	}
	if n := count(DocumentFilter{State: StateDraft}); n != 1 {
		t.Errorf("по состоянию: %d", n)
	}
	if n := count(DocumentFilter{Q: "нс-01"}); n != 1 {
		t.Errorf("поиск: %d", n)
	}
	if n := count(DocumentFilter{Subject: platform.DrillRef{Entity: platform.EntityItem, ID: "ENT01:F-017"}}); n != 1 {
		t.Errorf("по объекту: %d", n)
	}
	if got := FilterDocuments(l, DocumentFilter{}); *got.CollectedFromHistory != 3 || got.Items[1].State != StateDraft {
		t.Errorf("без отбора: %+v", got)
	}
	if (DocumentFilter{Subject: platform.DrillRef{Entity: platform.EntityItem, ID: "x"}}).Registry() {
		t.Error("только объект — не реестр")
	}
}
