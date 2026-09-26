package machinelogs

import (
	"context"
	"testing"

	"ant/internal/application/platform"
	processapp "ant/internal/application/process"
	processfx "ant/internal/infrastructure/fixtures/process"
	_ "ant/internal/infrastructure/fixtures/world" // встроенный мир заготовок (loader.Builtin)
)

// TestTerminalRunOverlay — операция, начатая с терминала, становится текущим
// выполнением оборудования поста; пауза останавливает, завершение освобождает.
func TestTerminalRunOverlay(t *testing.T) {
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "W21", Role: "performer"})
	a, p := New(), processfx.New()
	const station = "WP-WELD-2"
	eq, err := a.Equipment(ctx, station, platform.Moment{})
	if err != nil || len(eq.Items) == 0 {
		t.Fatalf("оборудование поста: %+v %v", eq, err)
	}
	hdr := func(id string) platform.CommandHeader {
		return platform.CommandHeader{CommandID: id, WorkplaceID: station}
	}
	if _, err := p.StartOperation(ctx, "ENT01:F-017", processapp.StartOperation{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000e001"),
		OperationRunID: "run-t1", StepKey: "welding.weld", StationID: station}); err != nil {
		t.Fatal(err)
	}
	cur := func() string {
		eq, err := a.Equipment(ctx, station, platform.Moment{})
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range eq.Items {
			if x.CurrentRunID == "run-t1" {
				return x.Execution
			}
		}
		return ""
	}
	if got := cur(); got != "running" {
		t.Fatalf("после начала: %q", got)
	}
	prof, err := a.RunProfile(ctx, "run-t1", platform.Moment{})
	if err != nil || prof.ItemID != "ENT01:F-017" || prof.FinishedAt != nil || prof.OperatorID == nil || *prof.OperatorID != "W21" {
		t.Fatalf("профиль: %+v %v", prof, err)
	}
	if _, err := p.PauseOperation(ctx, "run-t1", processapp.PauseOperation{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000e002"), PauseReason: "waiting"}); err != nil {
		t.Fatal(err)
	}
	if got := cur(); got != "stopped" {
		t.Fatalf("после паузы: %q", got)
	}
	if _, err := p.FinishOperation(ctx, "run-t1", processapp.FinishOperation{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000e003"), Completion: "completed"}); err != nil {
		t.Fatal(err)
	}
	if got := cur(); got != "" {
		t.Fatalf("после завершения выполнение осталось текущим: %q", got)
	}
	if prof, _ := a.RunProfile(ctx, "run-t1", platform.Moment{}); prof.FinishedAt == nil {
		t.Fatal("профиль не завершён")
	}
}
