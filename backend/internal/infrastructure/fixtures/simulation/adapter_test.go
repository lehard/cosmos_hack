package simulation

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"ant/internal/application/platform"
	app "ant/internal/application/simulation"
	"ant/internal/infrastructure/fixtures/loader"
)

const manifest = `format: 1
id: sc
title: Тест
case: ["§1.5"]
initial_step: 1
enterprise: ENT01
local_ids: ['F-\d{3}', 'NC-\d{2}']
steps:
  - {step: 0, clock: 2026-09-18T05:00:00Z, title: "шаг 0"}
  - {step: 1, clock: 2026-09-18T06:00:00Z, title: "шаг 1", wait: {action: nonconformity.nonconformity.confirm, role: quality_inspector, object: {kind: nonconformity, id: NC-01}, title: "Подтвердить"}}
  - {step: 2, clock: 2026-09-18T07:00:00Z, title: "шаг 2"}
`

func adapter(t *testing.T) (*Adapter, *loader.Runtime) {
	t.Helper()
	fsys := fstest.MapFS{"sc/scenario.yaml": {Data: []byte(manifest)}}
	for i, c := range []string{"05", "06", "07"} {
		fsys["sc/steps/0"+string(rune('0'+i))+".yaml"] = &fstest.MapFile{Data: []byte("format: 1\nstep: " + string(rune('0'+i)) + "\nclock: 2026-09-18T" + c + ":00:00Z\ntitle: шаг\n")}
	}
	lib, err := loader.LoadFS(fsys, ".")
	if err != nil {
		t.Fatal(err)
	}
	rt := loader.New(lib, loader.NewMemoryCursor())
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	return &Adapter{Runtime: rt, Now: func() time.Time { return now }}, rt
}

func TestPult(t *testing.T) {
	a, rt := adapter(t)
	ctx := context.Background()
	sl, err := a.Scenarios(ctx, false)
	if err != nil || len(sl.Items) != 1 || sl.Items[0].Decisions != 1 || sl.Items[0].CaseRefs[0] != "§1.5" {
		t.Fatalf("сценарии: %+v %v", sl, err)
	}
	// Без прогона — сценарий по умолчанию на initial_step, ждёт решения.
	r, err := a.Run(ctx, "sc", platform.Moment{})
	if err != nil || r.Step != 1 || r.State != "waiting_for_decision" || r.WaitingFor == nil || r.WaitingFor.ObjectID != "NC-01" {
		t.Fatalf("прогон по умолчанию: %+v %v", r, err)
	}
	if _, err := a.Run(ctx, "чужой", platform.Moment{}); err == nil {
		t.Fatal("чужой прогон должен быть не найден")
	}
	rc, err := a.StartRun(ctx, "sc", app.StartRun{CommandHeader: platform.CommandHeader{CommandID: "0192e4a0-0000-7000-8000-000000000001"}, Mode: "interactive", Speed: 1000})
	if err != nil || rc.Seq != loader.StepSeq(0) {
		t.Fatalf("старт: %+v %v", rc, err)
	}
	st, _, _ := rt.State(ctx)
	if !strings.HasPrefix(st.RunID, "fx-") || len(st.RunID) != 11 || st.Speed != 1000 {
		t.Fatalf("курсор после старта: %+v", st)
	}
	if _, err := a.PauseRun(ctx, st.RunID, app.RunControl{}); err != nil {
		t.Fatal(err)
	}
	if r, _ := a.Run(ctx, st.RunID, platform.Moment{}); r.State != "paused" || r.StartedAt.IsZero() {
		t.Fatalf("пауза: %+v", r)
	}
	if _, err := a.ResumeRun(ctx, st.RunID, app.RunControl{}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetSpeed(ctx, st.RunID, app.SetSpeed{Speed: 5}); err != nil {
		t.Fatal(err)
	}
	if rl, err := a.Runs(ctx, platform.Page{}); err != nil || len(rl.Items) != 1 || rl.Items[0].Speed != 5 {
		t.Fatalf("прогоны: %+v %v", rl, err)
	}
	if _, err := a.StopRun(ctx, st.RunID, app.RunControl{}); err != nil {
		t.Fatal(err)
	}
	if st2, _, _ := rt.State(ctx); st2.RunID != "" {
		t.Fatalf("после остановки прогон остался: %+v", st2)
	}
}
