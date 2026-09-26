package nonconformity

import (
	"context"
	"testing"

	app "ant/internal/application/nonconformity"
	"ant/internal/application/platform"
	_ "ant/internal/infrastructure/fixtures/world" // встроенный мир заготовок (loader.Builtin)
)

// TestDecisionsLeaveQueue — решение контролёра убирает строку из очереди,
// точка предъявления больше не предлагает решений, карточка НС показывает
// решение сессии и новый статус.
func TestDecisionsLeaveQueue(t *testing.T) {
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "INS-01", Role: "quality_inspector"})
	a := New()
	q, err := a.Queue(ctx, app.QueueFilter{}, platform.Moment{}, platform.Page{})
	if err != nil || len(q.Items) == 0 {
		t.Fatalf("очередь: %+v %v", q, err)
	}
	hdr := func(id string) platform.CommandHeader { return platform.CommandHeader{CommandID: id} }
	var decided, ncID string
	for _, x := range q.Items {
		switch {
		case x.Kind == "presentation" && decided == "":
			decided = x.ItemID
			if _, err := a.ResolvePresentation(ctx, x.ItemID, app.ResolvePresentation{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000f001"), StepKey: x.StepKey, Resolution: "accept", PresentationNo: 1, MethodEventIDs: []string{"x"}}); err != nil {
				t.Fatal(err)
			}
		case x.NCID != nil && ncID == "" && x.Kind != "presentation":
			ncID = *x.NCID
		}
	}
	if decided == "" {
		t.Skip("в очереди на шаге по умолчанию нет точки предъявления")
	}
	q2, _ := a.Queue(ctx, app.QueueFilter{}, platform.Moment{}, platform.Page{})
	for _, x := range q2.Items {
		if x.Kind == "presentation" && x.ItemID == decided {
			t.Fatalf("решённая точка осталась в очереди: %+v", x)
		}
	}
	if pv, err := a.Presentation(ctx, decided, platform.Moment{}); err == nil && len(pv.Presentation.AllowedResolutions) != 0 {
		t.Errorf("точка предлагает решения после решения: %+v", pv.Presentation.AllowedResolutions)
	}
	if ncID == "" {
		return
	}
	before, err := a.Card(ctx, ncID, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetDisposition(ctx, ncID, app.SetDisposition{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000f002"), Disposition: "scrap", Reason: app.NCReason{Text: "трещина"}}); err != nil {
		t.Fatal(err)
	}
	c, err := a.Card(ctx, ncID, platform.Moment{})
	if err != nil || c.Status != "disposition_set" || c.Axes.Disposition != "scrap" || len(c.HumanDecisions) != len(before.HumanDecisions)+1 {
		t.Fatalf("карточка после решения: %s %s %d решений, %v", c.Status, c.Axes.Disposition, len(c.HumanDecisions), err)
	}
}
