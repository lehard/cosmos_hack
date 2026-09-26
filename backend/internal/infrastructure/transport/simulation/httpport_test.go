package simulation

import (
	"context"
	"net/http"
	"testing"
	"time"

	app "ant/internal/application/simulation"
	"ant/internal/application/simulation/simfake"
	sim "ant/internal/domain/simulation"
	"ant/internal/infrastructure/transport/httpapi"
)

// defs — определения одного маленького прогона в памяти.
type defs struct{ b sim.Bundle }

func (d defs) Catalog(context.Context) (sim.Catalog, error) {
	return sim.Catalog{Entries: []sim.CatalogEntry{{ID: "R", Title: "Малый прогон", Kind: "failure", Run: "R", Cards: []string{"C"}}}}, nil
}
func (d defs) Bundle(context.Context, string) (sim.Bundle, error) { return d.b, nil }
func (d defs) Expected(context.Context, string) (sim.Expected, bool, error) {
	return sim.Expected{Scenario: "C", Checkpoints: []sim.Checkpoint{{At: "end", Assertions: []sim.Assertion{{ID: "C-01", What: "принято без карантина",
		Check: sim.Check{Operation: "ingest.metrics.read", Path: "/quarantined", Op: "eq", Value: 0}, Mapping: "exact"}}}}}, true, nil
}

func smallBundle() sim.Bundle {
	w := sim.World{ID: "w", Enterprise: "ENT01", ItemType: "FL-100.00.000", ItemRevision: "Б",
		Sources: map[string]sim.Source{"onec": {ID: "onec", Kind: "external_system", Reliability: "high"}, "term-vk": {ID: "t", Kind: "manual_entry", Reliability: "high"},
			"cam-kt1": {ID: "k1", Kind: "camera", Reliability: "medium"}}}
	r := sim.RunDef{ID: "R", Version: "1", Seed: 3, Speed: 1000, Anchor: "2026-09-21T07:00:00+03:00", End: "2026-09-21T10:00:00+03:00",
		Orders: []sim.OrderPlan{{ID: "ORD-1", At: "2026-09-21T07:00:00+03:00", Quantity: 1}},
		Lots:   []sim.LotPlan{{ID: "LOT-B", Component: "FL-100.01.001", Supplier: "S", Quantity: 1, Arrived: "2026-09-21T07:05:00+03:00"}}}
	return sim.Bundle{World: w, Run: r}
}

// TestHTTPPortPult — пульт через HTTP API в процессе: сценарии, запуск с
// run_id в ответе, прогон до конца, табло — операциями с тем же operationId.
func TestHTTPPortPult(t *testing.T) {
	ctx := context.Background()
	mux := http.NewServeMux()
	api := httpapi.New(mux, httpapi.Config{})
	ing := simfake.NewIngest()
	svc := app.NewServiceWith(app.Deps{Definitions: defs{smallBundle()}, Gateway: ing, Probe: ing, Actor: simfake.Actor{In: ing},
		Recorder: &simfake.Recorder{}, Store: memStore{m: map[string]*app.RunState{}}, Infra: &simfake.Clock{T: time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)}})
	Register(api, svc, svc)
	port := NewHTTPPort(api, mux)

	list, err := port.Read(ctx, "simulation.scenario.list", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	items := list.(map[string]any)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["assertions"].(interface{ String() string }).String() != "1" {
		t.Fatalf("сценарии пульта: %v", list)
	}
	res, err := port.Act(ctx, "ADM-01", "simulation.run.start", map[string]string{"scenario_id": "R"},
		map[string]any{"command_id": "0192e4a0-0000-7000-8000-000000000001", "basis_seq": 0, "policy_seq": 0, "mode": "autocheck"})
	if err != nil || res.Code != "" {
		t.Fatalf("запуск: %v %+v", err, res)
	}
	runs, _ := port.Read(ctx, "simulation.run.list", nil, "")
	runID := runs.(map[string]any)["items"].([]any)[0].(map[string]any)["run_id"].(string)
	if _, err := svc.RunToEnd(ctx, runID, 100); err != nil {
		t.Fatal(err)
	}
	board, err := port.Read(ctx, "simulation.board.read", map[string]string{"run_id": runID}, "")
	if err != nil {
		t.Fatal(err)
	}
	if p := board.(map[string]any)["passed"]; p == nil || p.(interface{ String() string }).String() != "1" {
		t.Fatalf("табло: %v", board)
	}
	if _, err := port.Read(ctx, "simulation.injection.list", map[string]string{"run_id": "нет-такого"}, ""); err == nil {
		t.Fatal("неизвестный прогон — ошибка")
	}
}

// memStore — хранилище прогонов теста.
type memStore struct{ m map[string]*app.RunState }

func (s memStore) Save(_ context.Context, r *app.RunState) error {
	s.m[r.RunID] = app.Snapshot(r)
	return nil
}
func (s memStore) Load(_ context.Context, id string) (*app.RunState, bool, error) {
	r, ok := s.m[id]
	if !ok {
		return nil, false, nil
	}
	return app.Snapshot(r), true, nil
}
func (s memStore) List(context.Context) ([]*app.RunState, error) {
	var out []*app.RunState
	for _, r := range s.m {
		out = append(out, app.Snapshot(r))
	}
	return out, nil
}

// TestDecidedObject — решение засчитывается остановке только над её объектом
// (изделие записи, поток или значение в данных), а не любое решение этого типа.
func TestDecidedObject(t *testing.T) {
	e := map[string]any{"seq": 7, "item_id": "ENT01:run-1/F-003", "stream": "item:ENT01:run-1/F-003",
		"data": map[string]any{"nc_id": "NC-9", "equipment_id": "IS-2"}}
	for obj, want := range map[string]bool{"": true, "ENT01:run-1/F-003": true, "NC-9": true, "IS-2": true,
		"ENT01:run-1/F-002": false, "IS-1": false, "NC-1": false} {
		if got := mentions(e, obj); got != want {
			t.Errorf("объект %q: %v, ждали %v", obj, got, want)
		}
	}
	inc := map[string]any{"seq": 8, "stream": "incident:INC-1", "data": map[string]any{}}
	if !mentions(inc, "INC-1") {
		t.Error("поток инцидента")
	}
}
