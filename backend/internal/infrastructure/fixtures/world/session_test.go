package world

import (
	"context"
	"testing"
	"time"

	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
)

// TestSessionOverlay — сессионное наложение мира заготовок: уникальные
// номера квитанций, повтор command_id, факты текущего прогона, as_of,
// живые обновления и сброс при старте и остановке прогона.
func TestSessionOverlay(t *testing.T) {
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "INS-01", Role: "quality_inspector"})
	lib, err := testLibrary()
	if err != nil {
		t.Fatal(err)
	}
	rt := loader.New(lib, loader.NewMemoryCursor())
	ver := rt.SessionVersion()
	a, err := rt.Record(ctx, "notifications.task.acknowledge", loader.ObjectRef{Kind: "task", ID: "TASK-001"}, platform.CommandMeta{CommandID: "c1"}, "done")
	if err != nil {
		t.Fatal(err)
	}
	b, err := rt.Record(ctx, "notifications.task.acknowledge", loader.ObjectRef{Kind: "task", ID: "TASK-002"}, platform.CommandMeta{CommandID: "c2"}, "done")
	if err != nil {
		t.Fatal(err)
	}
	if a.Seq == b.Seq || b.Seq <= a.Seq {
		t.Fatalf("номера квитанций не уникальны: %d, %d", a.Seq, b.Seq)
	}
	again, err := rt.Record(ctx, "notifications.task.acknowledge", loader.ObjectRef{Kind: "task", ID: "TASK-001"}, platform.CommandMeta{CommandID: "c1"}, "done")
	if err != nil || !again.Replayed || again.Seq != a.Seq {
		t.Fatalf("повтор command_id: %+v %v", again, err)
	}
	fs := rt.FactsOf(ctx, nil, "task", "TASK-001")
	if len(fs) != 1 || fs[0].Actor != "INS-01" || fs[0].Body != "done" {
		t.Fatalf("факты задачи: %+v", fs)
	}
	past := fs[0].At.Add(-time.Minute)
	if got := rt.Facts(ctx, &platform.Moment{AsOf: &past}, "task"); len(got) != 0 {
		t.Errorf("факт виден в прошлом: %+v", got)
	}
	ch, v2 := rt.SessionChanges(ver)
	if len(ch) < 2 || v2 <= ver || ch[len(ch)-1].Entity != "task" {
		t.Errorf("живые обновления: %+v", ch)
	}
	// Старт прогона — чистый мир; факты прогона видны с префиксом и без.
	if _, err := rt.Start(ctx, lib.Default, "run-1", "interactive", 1, 1, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := rt.Facts(ctx, nil); len(got) != 0 {
		t.Fatalf("старт прогона не сбросил факты: %+v", got)
	}
	if _, err := rt.Record(ctx, "notifications.task.acknowledge", loader.ObjectRef{Kind: "task", ID: "run-1/TASK-003"}, platform.CommandMeta{CommandID: "c3"}, "done"); err != nil {
		t.Fatal(err)
	}
	if got := rt.FactsOf(ctx, nil, "task", "TASK-003"); len(got) != 1 {
		t.Fatalf("факт прогона: %+v", got)
	}
	if got := rt.Facts(ctx, &platform.Moment{RunID: "run-2"}); len(got) != 0 {
		t.Errorf("факт виден чужому прогону: %+v", got)
	}
	if err := rt.Stop(ctx, "run-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := rt.Facts(ctx, nil); len(got) != 0 {
		t.Fatalf("остановка прогона не сбросила факты: %+v", got)
	}
}
