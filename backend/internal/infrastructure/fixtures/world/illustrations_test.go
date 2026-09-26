package world

import (
	"context"
	"strings"
	"testing"

	materialsapp "ant/internal/application/materials"
	ncapp "ant/internal/application/nonconformity"
	qualityapp "ant/internal/application/quality"
	"ant/internal/infrastructure/fixtures/loader"
)

// TestIllustrations — к наблюдениям КТ-3 по НС-01 (два ракурса) и к блику на
// Ф-025 приложены иллюстрации: карточка и результаты контроля ссылаются на
// них, materials.material.read/content их отдают, адрес = H(байты), пометка
// «ИЛЛЮСТРАЦИЯ» — и в метаданных, и на картинке (FR-102, NFR-UI-4).
func TestIllustrations(t *testing.T) {
	ctx := context.Background()
	lib, err := testLibrary()
	if err != nil {
		t.Fatal(err)
	}
	rt := loader.New(lib, loader.NewMemoryCursor())
	var card ncapp.NCCard
	if err := rt.Respond(ctx, "nonconformity.card.read", map[string]string{"nc_id": "NC-01"}, nil, &card); err != nil {
		t.Fatal(err)
	}
	if len(card.Evidence.Signals) != 1 || len(card.Evidence.Signals[0].EvidenceRefs) != 2 {
		t.Fatalf("НС-01: ожидались два материала у сигнала: %+v", card.Evidence.Signals)
	}
	refs := append([]string{}, card.Evidence.Signals[0].EvidenceRefs...)
	var insp qualityapp.InspectionResultList
	if err := rt.Respond(ctx, "quality.inspection.list", map[string]string{"item_id": "ENT01:F-025"}, nil, &insp); err != nil {
		t.Fatal(err)
	}
	glare := 0
	for _, r := range insp.Items {
		if r.Outcome == "unable_to_assess" && len(r.EvidenceRefs) == 1 {
			glare++
			refs = append(refs, r.EvidenceRefs[0])
		}
	}
	if glare != 1 {
		t.Fatalf("Ф-025: у «оценка невозможна» нет иллюстрации блика")
	}
	for _, a := range refs {
		var info materialsapp.MaterialInfo
		if err := rt.Respond(ctx, "materials.material.read", map[string]string{"address": a}, nil, &info); err != nil {
			t.Fatal(err)
		}
		var b []byte
		if err := rt.Respond(ctx, "materials.material.content", map[string]string{"address": a}, nil, &b); err != nil {
			t.Fatal(err)
		}
		if !info.IsIllustration || info.Kind != "illustration" || !strings.HasPrefix(info.ProvenanceNote, illustrationMark) || info.MediaType != "image/svg+xml" {
			t.Errorf("%s: метаданные %+v", a, info)
		}
		if Digest(b) != a || int64(len(b)) != info.SizeBytes || !strings.Contains(string(b), illustrationMark) {
			t.Errorf("%s: содержимое не совпадает с адресом или без пометки", a)
		}
	}
}
