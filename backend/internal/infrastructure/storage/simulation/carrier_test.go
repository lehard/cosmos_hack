package simulation

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	sim "ant/internal/domain/simulation"
)

// TestRouteFactsHaveTag — факты маршрута с носителем «бирка» (item_ref
// tag_qr TAG:‹изделие›) идут только после нанесения этой бирки решением
// item.carrier.apply: иначе ядро не привяжет факт к изделию — токен процесса
// не двигается (висит задача «Начать: Сварка…»), метка изделия — I-…, а не
// Ф-…. Правило для всех прогонов, в том числе короткой истории (entry: weld).
func TestRouteFactsHaveTag(t *testing.T) {
	f := NewFiles(filepath.Join(repo, "scenarios"))
	runs, err := f.Runs()
	if err != nil {
		t.Fatal(err)
	}
	for _, run := range runs {
		t.Run(run, func(t *testing.T) {
			b, err := f.Bundle(context.Background(), run)
			if err != nil {
				t.Fatal(err)
			}
			p, err := sim.Generate(b, sim.Params{RunID: GoldenRunID(run, b.Run.Seed)})
			if err != nil {
				t.Fatal(err)
			}
			tagged := map[string]time.Time{} // изделие → когда нанесена бирка
			for _, a := range p.Actions {
				if a.Operation == "item.carrier.apply" && a.Item != "" && a.Body["carrier_type"] == "tag_qr" {
					if at, ok := tagged[a.Item]; !ok || a.At.Before(at) {
						tagged[a.Item] = a.At
					}
				}
			}
			bad := 0
			for _, e := range p.Emissions {
				if e.Scenario != "route" || e.Item == "" {
					continue
				}
				var ev struct {
					ItemRef *struct {
						CarrierType string `json:"carrier_type"`
					} `json:"item_ref"`
				}
				if err := json.Unmarshal(e.Event, &ev); err != nil {
					t.Fatal(err)
				}
				if ev.ItemRef == nil || ev.ItemRef.CarrierType != "tag_qr" {
					continue
				}
				if at, ok := tagged[e.Item]; !ok || at.After(e.OccurredAt) {
					bad++
					if bad <= 5 {
						t.Errorf("%s (%s): бирка изделия %s не нанесена до факта", e.Label, e.EventType, e.Item)
					}
				}
			}
			if bad > 5 {
				t.Errorf("ещё %d фактов без бирки", bad-5)
			}
		})
	}
}
