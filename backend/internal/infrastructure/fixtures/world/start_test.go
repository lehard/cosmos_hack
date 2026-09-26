package world

import (
	"context"
	"strings"
	"testing"
	"time"

	"ant/internal/application/platform"
	simapp "ant/internal/application/simulation"
	"ant/internal/infrastructure/fixtures/loader"
	fxsim "ant/internal/infrastructure/fixtures/simulation"
)

// TestStartFromStartStep — старт с точки старта (start=start_step) ставит
// историю сразу у катастрофы: шаги до неё пройдены (табло), через
// несколько секунд ×1000 — ожидание контролёра по Ф-017; сессионное
// наложение и живые обновления — как при старте с нуля (FR-129, AD-36, AD-38).
func TestStartFromStartStep(t *testing.T) {
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "INS-01", Role: "quality_inspector"})
	lib, err := testLibrary()
	if err != nil {
		t.Fatal(err)
	}
	rt := loader.New(lib, loader.NewMemoryCursor())
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	a := &fxsim.Adapter{Runtime: rt, Now: func() time.Time { return now }}
	sc, _ := lib.Scenario("flange-bad-day")
	start := sc.Manifest.StartStep
	if start != 7 || sc.Header(start+1).Wait == nil || sc.Header(start+1).Wait.Action != "nonconformity.nonconformity.confirm" {
		t.Fatalf("start_step: %d, следующий шаг ждёт %+v", start, sc.Header(start+1).Wait)
	}
	for n := 0; n < start; n++ {
		if sc.Header(n).Wait != nil {
			t.Fatalf("до точки старта есть ожидание решения: шаг %d", n)
		}
	}
	sl, err := a.Scenarios(ctx, false)
	if err != nil || len(sl.Items) == 0 || sl.Items[0].StartStep == nil || *sl.Items[0].StartStep != start || sl.Items[0].StartTitle == "" {
		t.Fatalf("точка старта в списке сценариев: %+v %v", sl, err)
	}

	// Старт с точки старта сценария.
	rc, err := a.StartRun(ctx, "flange-bad-day", simapp.StartRun{CommandHeader: platform.CommandHeader{CommandID: "0192e4a0-0000-7000-8000-0000000000a1"}, Mode: "interactive", Speed: 1000, Start: "start_step"})
	if err != nil {
		t.Fatal(err)
	}
	runID := rc.RunID
	st, _, _ := rt.State(ctx)
	if st.Step != start || !st.ClockAt.Equal(sc.Header(start).Clock) || rc.Seq != loader.StepSeq(start) {
		t.Fatalf("курсор после старта: %+v, квитанция %+v", st, rc)
	}

	// Табло: строки шагов до точки старта включительно сверены (не «не дошёл»).
	b, err := a.Board(ctx, runID, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	before := 0
	for _, r := range b.Rows {
		switch {
		case r.Step <= start && r.Status != "passed":
			t.Errorf("строка %s шага %d: %s", r.AssertionID, r.Step, r.Status)
		case r.Step <= start:
			before++
		case r.Status != "not_reached":
			t.Errorf("строка %s шага %d после точки старта: %s", r.AssertionID, r.Step, r.Status)
		}
	}
	if before == 0 || b.Passed != before {
		t.Errorf("табло на старте: пройдено %d из %d строк до точки", b.Passed, before)
	}

	// Живые обновления новой подписки: всё, что есть к шагу старта, с префиксом прогона.
	ch := rt.ChangesBetween(sc, -1, st.Step, runID)
	prefixed, early := 0, 0
	for _, c := range ch {
		if c.Seq >= int64(start+1)*loader.SeqPerStep {
			t.Errorf("изменение позже точки старта: %+v", c)
		}
		if strings.Contains(c.ID, runID+"/") {
			prefixed++
		}
		if c.Seq < int64(start)*loader.SeqPerStep {
			early++
		}
	}
	if prefixed == 0 || early == 0 {
		t.Errorf("изменения к точке старта: %d с префиксом, %d с прошлых шагов из %d", prefixed, early, len(ch))
	}

	// ×1000: 35 доменных минут до прожога — ~2 с; прогон встаёт на ожидании контролёра.
	_ = rt.Tick(ctx, now)
	_ = rt.Tick(ctx, now.Add(3*time.Second))
	r, err := a.Run(ctx, runID, platform.Moment{})
	if err != nil || r.Step != start+1 || r.State != "waiting_for_decision" || r.WaitingFor == nil ||
		r.WaitingFor.Role != "quality_inspector" || r.WaitingFor.ObjectID != "ENT01:"+runID+"/F-017" {
		t.Fatalf("через 3 с ×1000: %+v %v", r, err)
	}
	if r.BoardPassed != before {
		t.Errorf("табло на ожидании: %d", r.BoardPassed)
	}

	// Сессионное наложение: чистое на старте, решение контролёра — факт прогона и шаг вперёд.
	if got := rt.Facts(ctx, nil); len(got) != 0 {
		t.Fatalf("факты на старте: %+v", got)
	}
	w := sc.Header(start + 1).Wait
	dec, err := rt.Record(ctx, w.Action, loader.ObjectRef{Kind: w.Object.Kind, ID: "ENT01:" + runID + "/F-017"}, platform.CommandMeta{CommandID: "c-confirm"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if st, _, _ := rt.State(ctx); st.Step != start+2 || dec.Seq <= int64(start+2)*loader.SeqPerStep+loader.SeqSessionAt {
		t.Fatalf("решение контролёра: шаг %d, seq %d", st.Step, dec.Seq)
	}
	fs := rt.FactsOf(ctx, nil, w.Object.Kind, "ENT01:F-017")
	if len(fs) != 1 || fs[0].RunID != runID {
		t.Fatalf("факт решения: %+v", fs)
	}
	sch, _ := rt.SessionChanges(0)
	if len(sch) == 0 || sch[len(sch)-1].ID != "ENT01:"+runID+"/F-017" || sch[len(sch)-1].RunID != runID {
		t.Errorf("живые обновления решения: %+v", sch)
	}

	// По умолчанию — с начала; явный шаг сильнее start.
	if _, err := a.StartRun(ctx, "flange-bad-day", simapp.StartRun{Mode: "interactive"}); err != nil {
		t.Fatal(err)
	}
	if st, _, _ := rt.State(ctx); st.Step != 0 || len(rt.Facts(ctx, nil)) != 0 {
		t.Fatalf("старт с начала: %+v", st)
	}
	twelve := 12
	if _, err := a.StartRun(ctx, "flange-bad-day", simapp.StartRun{Mode: "interactive", Start: "beginning", FromStep: &twelve}); err != nil {
		t.Fatal(err)
	}
	if r, _ := a.Runs(ctx, platform.Page{}); r.Items[0].Step != 12 || r.Items[0].State != "waiting_for_decision" {
		t.Fatalf("старт с шага 12: %+v", r.Items[0])
	}
	bad := sc.Steps()
	if _, err := a.StartRun(ctx, "flange-bad-day", simapp.StartRun{Mode: "interactive", FromStep: &bad}); err == nil || !strings.Contains(err.Error(), "validation") {
		t.Errorf("шаг вне сценария: %v", err)
	}
}
