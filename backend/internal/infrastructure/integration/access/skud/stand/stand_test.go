package stand

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ant/internal/infrastructure/integration/access/skud"
	"ant/internal/infrastructure/integration/ingest/stands"
)

// Эпик 37: stand СКУД по протоколу skud.v1 и адаптер-клиент.
func TestStandAndClient(t *testing.T) {
	st := New(Options{Arrive: true, Zones: []Zone{{ID: "Z-WC", Name: "Сварочный цех"}, {ID: "Z-MC", Name: "Механический цех"}},
		Holders: []Holder{{ID: "W21", Name: "Сварщик", Home: "Z-WC"}, {ID: "INS-01", Name: "Контролёр"}}})
	srv := httptest.NewServer(stands.NewRegistry(st).Handler())
	defer srv.Close()
	c := skud.NewClient(srv.URL+"/stand/skud/api/v1", 0)
	ctx := context.Background()
	if a, err := c.About(ctx); err != nil || a.Contract != skud.Contract || len(a.Zones) != 2 {
		t.Fatalf("about: %+v %v", a, err)
	}
	evs, last, err := c.Events(ctx, 0)
	if err != nil || len(evs) != 1 || last != 1 || !evs[0].Pass.Enter || evs[0].Pass.PersonID != "W21" || evs[0].Pass.ZoneID != "Z-WC" {
		t.Fatalf("смена пришла: %+v %d %v", evs, last, err)
	}
	// Кнопка стенда «сварщик вышел из зоны».
	b, _ := json.Marshal(skud.PassRequest{Holder: "W21", Zone: "Z-WC", Direction: skud.DirOut})
	resp, err := http.Post(srv.URL+"/stand/skud/api/v1/passes", "application/json", bytes.NewReader(b))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("проход: %v %v", resp, err)
	}
	resp.Body.Close()
	// Турникет не выпустит того, кого нет в зоне.
	resp, _ = http.Post(srv.URL+"/stand/skud/api/v1/passes", "application/json", bytes.NewReader(b))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("повторный выход: %d", resp.StatusCode)
	}
	resp.Body.Close()
	evs, last, err = c.Events(ctx, 1)
	if err != nil || len(evs) != 1 || last != 2 || evs[0].Pass.Enter || evs[0].Seq != 2 {
		t.Fatalf("после 1: %+v %d %v", evs, last, err)
	}
	// Страница «глазами СКУД».
	resp, err = http.Get(srv.URL + "/stand/skud/")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("страница: %v %v", resp, err)
	}
	resp.Body.Close()
}
