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
