package simulation

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	app "ant/internal/application/simulation"
	"ant/internal/application/simulation/simfake"
	sim "ant/internal/domain/simulation"
)

// humanRuns — стол исполнителя выдаёт свой id выполнения (терминал: новый
// UUID на «Начать»): запись «Начать» с этим id видна по seq нажатия.
type humanRuns struct {
	*simfake.Ingest
	mu    sync.Mutex
	runs  map[int64]string // seq нажатия «Начать» → id выполнения стола
	sent  []sim.Emission
	plain string
}

func (h *humanRuns) Read(ctx context.Context, op string, params map[string]string, runID string) (any, error) {
	if op == "journal.entry.read" {
		h.mu.Lock()
		for seq, id := range h.runs {
			if params["seq"] == itoa64(seq) {
				h.mu.Unlock()
				return map[string]any{"seq": seq, "data": map[string]any{"operation_run_id": id}}, nil
			}
		}
		h.mu.Unlock()
	}
	return h.Ingest.Read(ctx, op, params, runID)
}

func (h *humanRuns) Deliver(ctx context.Context, runID string, batch []sim.Emission) ([]app.Delivered, error) {
	h.mu.Lock()
	h.sent = append(h.sent, batch...)
	h.mu.Unlock()
	return h.Ingest.Deliver(ctx, runID, batch)
}

func itoa64(n int64) string {
	s := ""
	if n == 0 {
		return "0"
	}
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

// TestShowHumanRunID — «Начать» с терминала приходит со своим id выполнения
// (не плановым SV-001-1): остановка «Выполнено» ждётся по изделию и
// закрывается нажатием по фактическому выполнению; события прогона после
// «Начать» (сводки тока, КТ-3, рентген) ссылаются на фактическое выполнение.
func TestShowHumanRunID(t *testing.T) {
	ctx := context.Background()
	files := NewFiles(filepath.Join(repo, "scenarios"))
	ing := &humanRuns{Ingest: simfake.NewIngest(), runs: map[int64]string{}}
	clock := &simfake.Clock{T: time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)}
	d := &desk{a: &simfake.Actor{In: ing.Ingest}}
	store := NewMemoryRuns()
	svc := app.NewServiceWith(app.Deps{Definitions: files, Gateway: ing, Probe: ing, Actor: d, Recorder: &simfake.Recorder{},
		Store: store, Infra: clock, Profile: "demo"})
	started, err := svc.StartRun(ctx, "SHOW-IS2", app.StartRun{Mode: app.ModeInteractive})
	if err != nil {
		t.Fatal(err)
	}
	load := func() *app.RunState { st, _, _ := store.Load(ctx, started.RunID); return st }
	step := func(dt time.Duration) *app.RunState {
		clock.Advance(dt)
		if err := svc.Step(ctx, started.RunID); err != nil {
			t.Fatal(err)
		}
		return load()
	}
	const human = "0199aaaa-bbbb-7ccc-8ddd-000000000001"
	var planned, item string
	finished := false
	st := load()
	for i := 0; i < 3000 && !finished; i++ {
		st = step(2 * time.Second)
		if st.State != app.StateWaiting || st.Waiting == nil || d.pressed(st.Waiting.Op, st.Waiting.Object, st.Consumed) {
			continue
		}
		w := *st.Waiting
		switch {
		case w.Op == "process.operation.start" && planned == "":
			// Ф-001 «Начать» с терминала: id выполнения — свой.
			item = w.Object
			d.press(w.Op, w.Object)
			d.mu.Lock()
			seq := d.recs[len(d.recs)-1].seq
			d.mu.Unlock()
			ing.mu.Lock()
			ing.runs[seq] = human
			ing.mu.Unlock()
			p, _ := svc.Plan(ctx, started.RunID, app.PlanQuery{Limit: 500})
			for _, e := range p.Items {
				if e.Operation == "process.operation.finish" && e.ItemID == item {
					planned = "SV-001-1"
					if e.ObjectID != item {
						t.Fatalf("«Выполнено» в плане ждётся не по изделию: %q", e.ObjectID)
					}
					break
				}
			}
		case w.Op == "process.operation.finish" && item != "" && !finished:
			if w.Object != item {
				t.Fatalf("«Выполнено» ждёт %q, а не изделие %q", w.Object, item)
			}
			d.press(w.Op, w.Object) // «Выполнено» по фактическому выполнению — запись по изделию
			finished = true
		default:
			d.press(w.Op, w.Object)
		}
	}
	if !finished || planned == "" {
		t.Fatalf("не дошли до «Выполнено» Ф-001: %s %+v", st.State, st.Waiting)
	}
	// Прогон идёт дальше: КТ-3 Ф-001 — по фактическому выполнению.
	for i := 0; i < 200; i++ {
		st = step(2 * time.Second)
		if st.Waiting != nil && st.Waiting.Op != "process.operation.finish" {
			break
		}
	}
	if len(st.Runs) != 1 {
		t.Fatalf("выполнения стола: %v", st.Runs)
	}
	var kt3 string
	ing.mu.Lock()
	for _, e := range ing.sent {
		if e.Label == "F-001/kt3" {
			kt3 = string(e.Event)
		}
	}
	ing.mu.Unlock()
	if kt3 == "" || !strings.Contains(kt3, human) || strings.Contains(kt3, "/"+planned+"\"") {
		t.Fatalf("КТ-3 Ф-001 не по фактическому выполнению (%v): %s", st.Runs, kt3)
	}
}
