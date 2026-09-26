package analysis

import (
	"context"
	"testing"

	app "ant/internal/application/analysis"
	"ant/internal/application/platform"
	_ "ant/internal/infrastructure/fixtures/world" // встроенный мир заготовок (loader.Builtin)
)

// TestTechnologistOverlay — сужение области риска даёт новую ступень и размер
// инцидента, назначенная мера появляется в списке мер «назначена».
func TestTechnologistOverlay(t *testing.T) {
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "TEC-01", Role: "technologist"})
	a := New()
	rs, err := a.RiskScope(ctx, "RS-01", platform.Moment{})
	if err != nil || len(rs.Items) < 2 {
		t.Fatalf("область RS-01: %d изделий, %v", len(rs.Items), err)
	}
	gone := rs.Items[0].ItemID
	hdr := func(id string) platform.CommandHeader { return platform.CommandHeader{CommandID: id} }
	if _, err := a.NarrowScope(ctx, "RS-01", app.ChangeScope{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000b001"), ItemIDs: []string{gone},
		EvidenceEventIDs: []string{"EV-X"}, Reason: app.Reason{Text: "рентген чистый"}}); err != nil {
		t.Fatal(err)
	}
	after, _ := a.RiskScope(ctx, "RS-01", platform.Moment{})
	last := after.Versions[len(after.Versions)-1]
	if len(after.Versions) != len(rs.Versions)+1 || last.Change != "narrowed" || len(last.ItemsRemoved) != 1 || last.ItemsRemoved[0] != gone || len(after.Items) != len(rs.Items)-1 {
		t.Fatalf("ступень сужения: %+v", last)
	}
	l, err := a.Incidents(ctx, platform.Moment{}, platform.Page{})
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range l.Items {
		if x.IncidentID == "RS-01" && x.Size != len(after.Items) {
			t.Fatalf("размер инцидента %d, в области %d", x.Size, len(after.Items))
		}
	}
	before, _ := a.CorrectiveActions(ctx, platform.Moment{})
	if _, err := a.AssignAction(ctx, "RS-01", app.AssignAction{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000b002"), ActionType: "corrective_action",
		Direction: "prevent_occurrence", OwnerID: "FOR-WC", Title: "Заменить кабель ИС-2", EffectivenessPlan: app.EffectivenessPlanInput{Metric: "отклонения тока", WindowDays: 14}}); err != nil {
		t.Fatal(err)
	}
	ca, _ := a.CorrectiveActions(ctx, platform.Moment{})
	if len(ca.Items) != len(before.Items)+1 || ca.Items[0].Status != "assigned" || ca.Items[0].Plan.WindowDays != 14 || ca.Items[0].IncidentID != "RS-01" {
		t.Fatalf("мера: %+v", ca.Items[0])
	}
}
