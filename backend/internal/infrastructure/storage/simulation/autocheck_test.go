package simulation

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ant/internal/application/platform"
	app "ant/internal/application/simulation"
	"ant/internal/application/simulation/simfake"
)

// TestAutocheckOnFakes — автосверка каждого сценария пульта на заготовках
// приёма (make sim-check): генератор → обычный приём (в памяти) → решения →
// табло теми же операциями. Строки, которые проверяет приём, должны совпасть;
// остальные ждут движка («операция пока не отвечает»). Не совпавших строк нет.
func TestAutocheckOnFakes(t *testing.T) {
	ctx := context.Background()
	files := NewFiles(filepath.Join(repo, "scenarios"))
	cat, err := files.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var total, passed, pending, failed, notReached int
	var report strings.Builder
	for _, e := range cat.Entries {
		ing := simfake.NewIngest()
		rec := &simfake.Recorder{}
		clock := &simfake.Clock{T: time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)}
		svc := app.NewServiceWith(app.Deps{Definitions: files, Gateway: ing, Probe: ing, Actor: simfake.Actor{In: ing},
			Recorder: rec, Store: NewMemoryRuns(), Infra: clock, Profile: "demo"})
		started, err := svc.StartRun(ctx, e.ID, app.StartRun{Mode: app.ModeAutocheck})
		if err != nil {
			t.Fatalf("%s: запуск: %v", e.ID, err)
		}
		st, err := svc.RunToEnd(ctx, started.RunID, 10000)
		if err != nil {
			t.Fatalf("%s: прогон: %v", e.ID, err)
		}
		if st.State != app.StateComplete {
			t.Fatalf("%s: прогон не закончен: %s %s", e.ID, st.State, st.Error)
		}
		b, err := svc.Board(ctx, started.RunID, platformMoment())
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range b.Rows {
			if r.Status == "failed" {
				actual := "—"
				if r.Actual != nil {
					actual = *r.Actual
				}
				t.Errorf("%s %s: %s — ожидалось %s, получено %s (%s)", e.ID, r.AssertionID, r.Title, r.Expected, actual, r.Detail)
			}
		}
		total += len(b.Rows)
		passed += b.Passed
		pending += b.Pending
		failed += b.Failed
		notReached += b.NotReached
		fmt.Fprintf(&report, "%-5s %-70.70s строк %3d: совпало %3d, ждёт движка %3d, не дошёл %2d, не совпало %d; доставок %v\n",
			e.ID, e.Title, len(b.Rows), b.Passed, b.Pending, b.NotReached, b.Failed, st.Delivered)
	}
	t.Logf("табло на заготовках приёма:\n%s", report.String())
	t.Logf("итого строк %d: совпало %d, ждёт движка %d, не дошёл %d, не совпало %d", total, passed, pending, notReached, failed)
	if passed == 0 {
		t.Error("на заготовках не совпала ни одна строка — сверка не работает")
	}
}

func platformMoment() platform.Moment { return platform.Moment{} }
