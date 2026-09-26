package world

import (
	"testing"
)

// TestPresentationRead — у каждой строки очереди «предъявлено» и «пересмотр»
// есть точка предъявления (nonconformity.presentation.read): поля команды
// решения, результаты методов, у пересмотра Ф-015 — прежнее решение и
// опоздавшая запись ИС-2 с числами режима; source_event_id строки — id этой записи.
func TestPresentationRead(t *testing.T) {
	m := buildMain(t)
	names, err := LoadNames(repoFS())
	if err != nil {
		t.Fatal(err)
	}
	m.names = names
	reviewSeen := false
	for n := range m.Steps {
		c := m.ctx(n)
		for _, r := range c.queue().Items {
			if r.Kind != "presentation" && r.Kind != "review" {
				continue
			}
			v := c.presentationView(c.M.itemByID[r.ItemID[len(enterprise)+1:]], r)
			p := v.Presentation
			if p.StepKey != r.StepKey || p.ClosingPoint == "" || p.StepLabel == nil || len(p.AllowedResolutions) == 0 || len(p.MethodEventIDs) != len(v.MethodResults) {
				t.Errorf("шаг %d %s: %+v", n, r.ItemID, p)
			}
			if r.Kind == "review" {
				reviewSeen = true
				if v.Review == nil || len(v.Review.NewFacts) != 1 || v.Review.NewFacts[0].Reading == nil || r.SourceEventID == nil ||
					*r.SourceEventID != v.Review.NewFacts[0].EventID || v.Review.Decision.Kind != "decision" || len(v.Review.KnownAtDecision) == 0 {
					t.Errorf("шаг %d пересмотр %s: %+v", n, r.ItemID, v.Review)
				}
			}
		}
	}
	if !reviewSeen {
		t.Error("строки пересмотра в сценарии нет")
	}
}
