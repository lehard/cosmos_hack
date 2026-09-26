package journal

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
)

const manifest = `format: 1
id: sc
title: Тест
initial_step: 0
enterprise: ENT01
local_ids: ['F-\d{3}']
steps:
  - {step: 0, clock: 2026-09-18T05:00:00Z, title: "шаг 0"}
  - {step: 1, clock: 2026-09-18T06:00:00Z, title: "шаг 1"}
`

func runtime(t *testing.T) *loader.Runtime {
	t.Helper()
	fsys := fstest.MapFS{
		"sc/scenario.yaml": {Data: []byte(manifest)},
		"sc/steps/00.yaml": {Data: []byte("format: 1\nstep: 0\nclock: 2026-09-18T05:00:00Z\ntitle: шаг 0\n")},
		"sc/steps/01.yaml": {Data: []byte("format: 1\nstep: 1\nclock: 2026-09-18T06:00:00Z\ntitle: шаг 1\nchanges:\n  - {entity: item, id: \"ENT01:F-017\"}\n")},
	}
	lib, err := loader.LoadFS(fsys, ".")
	if err != nil {
		t.Fatal(err)
	}
	return loader.New(lib, loader.NewMemoryCursor())
}

func TestSubscribeEmitsStepChanges(t *testing.T) {
	rt := runtime(t)
	a := &Adapter{Poll: 5 * time.Millisecond, Runtime: rt}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := rt.Start(ctx, "sc", "run-1", "interactive", 0, 1, time.Now()); err != nil {
		t.Fatal(err)
	}
	sub, err := a.Subscribe(ctx, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()
	if _, err := rt.Advance(ctx); err != nil {
		t.Fatal(err)
	}
	c, err := sub.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if c.Entity != platform.EntityItem || c.ID != "ENT01:run-1/F-017" || c.RunID != "run-1" || c.Mode != platform.ModeFixtures || c.Seq != loader.SeqPerStep+loader.SeqChangesAt {
		t.Fatalf("изменение: %+v", c)
	}
	seen := map[platform.EntityKind]bool{}
	for i := 0; i < 2; i++ {
		c, err := sub.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		seen[c.Entity] = true
	}
	if !seen[platform.EntityLiveMap] || !seen[platform.EntityRun] {
		t.Fatalf("нет live_map или run: %v", seen)
	}
	sub.Close()
	if _, err := sub.Next(ctx); err == nil {
		t.Fatal("после Close подписка должна закрыться")
	}
}
