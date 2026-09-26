package simulation

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	sim "ant/internal/domain/simulation"
)

// Корень репозитория от каталога пакета.
const repo = "../../../../.."

// TestStreams — генератор по определениям scenarios/definitions даёт потоки,
// совпадающие с закоммиченными (детерминизм по seed, AD-4, AD-38), и каждое
// событие по контракту проходит схемы contracts/events (AD-20).
// ANT_UPDATE_STREAMS=1 — перезаписать потоки (make sim-streams).
func TestStreams(t *testing.T) {
	f := NewFiles(filepath.Join(repo, "scenarios"))
	runs, err := f.Runs()
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) == 0 {
		t.Fatal("нет определений прогонов в scenarios/definitions/runs")
	}
	schemas := NewEventSchemas(filepath.Join(repo, "contracts", "events"))
	dir := filepath.Join(repo, "scenarios", "definitions", "streams")
	update := os.Getenv("ANT_UPDATE_STREAMS") == "1"
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
			bad := 0
			for _, e := range p.Emissions {
				err := schemas.Validate(e.Event)
				switch {
				case e.Contract && err != nil:
					bad++
					if bad <= 20 {
						t.Errorf("событие %s (%s, %s): %v", e.Label, e.EventType, e.Scenario, err)
					}
				case !e.Contract && err == nil:
					t.Errorf("событие %s (%s) помечено «не по контракту», а схемы проходит", e.Label, e.Scenario)
				}
			}
			if bad > 20 {
				t.Errorf("ещё %d событий не по контракту", bad-20)
			}
			files, err := Render(p)
			if err != nil {
				t.Fatal(err)
			}
			if update {
				if err := WriteStreams(dir, files); err != nil {
					t.Fatal(err)
				}
				return
			}
			for name, want := range files {
				got, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatalf("%s: %v (make sim-streams)", name, err)
				}
				if !bytes.Equal(got, want) {
					t.Errorf("%s устарел: определения или генератор изменились — make sim-streams", name)
				}
			}
		})
	}
}
