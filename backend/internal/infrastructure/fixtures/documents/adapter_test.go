package documents

import (
	"context"
	"strings"
	"testing"

	app "ant/internal/application/documents"
	"ant/internal/application/platform"
	_ "ant/internal/infrastructure/fixtures/world" // встроенный мир заготовок (loader.Builtin)
)

// TestRegistryOnBuiltinWorld — реестр, документ, отрисовка и печатная форма
// на встроенном мире заготовок (шаг курсора по умолчанию — начальный).
func TestRegistryOnBuiltinWorld(t *testing.T) {
	a, ctx := New(), context.Background()
	all, err := a.Registry(ctx, app.DocumentFilter{}, platform.Moment{}, platform.Page{Limit: 500})
	if err != nil || len(all.Items) < 10 {
		t.Fatalf("реестр: %d документов, %v", len(all.Items), err)
	}
	byItem, err := a.Registry(ctx, app.DocumentFilter{ItemID: "ENT01:F-017"}, platform.Moment{}, platform.Page{})
	if err != nil || len(byItem.Items) == 0 || len(byItem.Items) >= len(all.Items) {
		t.Fatalf("отбор по изделию: %d из %d, %v", len(byItem.Items), len(all.Items), err)
	}
	nc, err := a.Documents(ctx, platform.DrillRef{Entity: platform.EntityNonconformity, ID: "NC-01"}, platform.Moment{}, platform.Page{})
	if err != nil || len(nc.Items) == 0 {
		t.Fatalf("документы НС-01: %+v %v", nc, err)
	}
	d, err := a.Document(ctx, "DOC-PVA-FLANGE-1", 0, platform.Moment{})
	if err != nil || d.Status != "route_closed" || len(d.Route) != 3 || d.Route[0].Signatures[0].KeyStorage == "" {
		t.Fatalf("лист утверждения: %+v %v", d, err)
	}
	r, err := a.Render(ctx, "DOC-PVA-FLANGE-1", 0)
	if err != nil || !strings.Contains(r.HTML, "Лист утверждения") {
		t.Fatalf("отрисовка: %v", err)
	}
	pv, err := a.PrintView(ctx, "DOC-PVA-FLANGE-1", 0)
	if err != nil || !strings.Contains(pv.HTML, "<svg") || pv.QR != d.QR {
		t.Fatalf("печатная форма: %v", err)
	}
}

// TestDecisionRequestsForRareSigners — стол «Требуется ваше решение» у редких
// подписантов на встроенном мире: держатель КД, метролог, представитель
// заказчика видят свой запрос (этап, который ждёт именно их); у аудитора
// запросов нет; карточка решения — по тому же документу.
func TestDecisionRequestsForRareSigners(t *testing.T) {
	a := New()
	as := func(person, role string) context.Context {
		return platform.WithPrincipal(context.Background(), platform.Principal{PersonID: person, Role: role})
	}
	for _, c := range []struct{ person, role, doc string }{
		{"DA-81", "design_authority", "DOC-REQ-DA-F-023"},
		{"MET-82", "metrologist", "DOC-REQ-MET-KT3"},
		{"CR-71", "customer_representative", "DOC-REQ-CR-RS-01"},
	} {
		ctx := as(c.person, c.role)
		l, err := a.DecisionRequests(ctx, platform.Moment{})
		if err != nil {
			t.Fatalf("%s: %v", c.person, err)
		}
		var found *app.DecisionRequest
		for i, rq := range l.Items {
			if rq.MyStage == nil || rq.Document.DocumentID == "" {
				t.Fatalf("%s: запрос без этапа: %+v", c.person, rq)
			}
			if rq.Document.DocumentID == c.doc {
				found = &l.Items[i]
			}
		}
		if found == nil {
			t.Fatalf("%s: нет запроса %s среди %d", c.person, c.doc, len(l.Items))
		}
		if found.ExpectedSigner == nil || *found.ExpectedSigner != c.person || found.Proposal.Summary == "" || len(found.Document.Route) == 0 {
			t.Errorf("%s: запрос %+v", c.person, found)
		}
		card, err := a.DecisionCard(ctx, c.doc, platform.Moment{})
		if err != nil || card.Stage.Stage != *found.MyStage || card.Question == "" || card.AfterSignature == "" {
			t.Errorf("%s: карточка %+v %v", c.person, card, err)
		}
	}
	l, err := a.DecisionRequests(as("AUD-01", "security_auditor"), platform.Moment{})
	if err != nil || len(l.Items) != 0 {
		t.Errorf("аудитор: %d запросов, %v", len(l.Items), err)
	}
	if _, err := a.DecisionCard(as("CR-71", "customer_representative"), "DOC-PVA-FLANGE-1", platform.Moment{}); err == nil {
		t.Error("карточка подписанного документа — должна быть «не найдено»")
	}
}
