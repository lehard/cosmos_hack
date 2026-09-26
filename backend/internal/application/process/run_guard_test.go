package process_test

import (
	"context"
	"testing"

	app "ant/internal/application/process"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
)

func codeOf(err error) errcodes.Code {
	if pe, ok := platform.AsError(err); ok {
		return pe.Code
	}
	return ""
}

// «Начать» с терминала (FR-137): номер выполнения, уже начатый у другого
// изделия, — отказ «выполнение уже начато»; повтор той же команды — не отказ;
// у исполнителя на посту идёт операция — второй «Начать» на этом посту отказ,
// пока первая не завершена (SHOW-IS2: один run_id на Ф-003 и Ф-002).
func TestStartGuardsRunAndPost(t *testing.T) {
	w := newLiveWorld(t)
	f3, f2 := "ENT01:run-1/I-F003", "ENT01:run-1/I-F002"
	w.register(f3, 0)
	w.register(f2, 0.1)
	started := w.fact(f3, catalog.OperationRunStarted, 1, map[string]any{"operation_run_id": "01a0df5c-0000-7000-8000-000000000001",
		"operation_code": "030", "step_key": "welding.weld", "operator_id": "W21", "workplace_id": "WP-WELD-1"})
	w.sync()
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "W21"})
	start := func(item, run, cmd string) error {
		_, err := w.svc.StartOperation(ctx, item, app.StartOperation{CommandHeader: platform.CommandHeader{CommandID: cmd, WorkplaceID: "WP-WELD-1"},
			OperationRunID: run, OperationCode: "030", StepKey: "welding.weld", StationID: "WP-WELD-1"})
		return err
	}
	if c := codeOf(start(f2, "01a0df5c-0000-7000-8000-000000000001", "01a0df5c-0000-7000-8000-00000000000a")); c != errcodes.ProcessRunAlreadyStarted {
		t.Fatalf("тот же run_id у другого изделия: %q", c)
	}
	if c := codeOf(start(f3, "01a0df5c-0000-7000-8000-000000000001", started)); c == errcodes.ProcessRunAlreadyStarted || c == errcodes.ProcessOperationInProgress {
		t.Fatalf("повтор той же команды — не отказ гарда: %q", c)
	}
	if c := codeOf(start(f2, "01a0df5d-0000-7000-8000-000000000002", "01a0df5d-0000-7000-8000-00000000000b")); c != errcodes.ProcessOperationInProgress {
		t.Fatalf("вторая операция на посту: %q", c)
	}
	w.fact(f3, catalog.OperationRunFinished, 2, map[string]any{"operation_run_id": "01a0df5c-0000-7000-8000-000000000001", "completion": "completed"})
	w.sync()
	if c := codeOf(start(f2, "01a0df5d-0000-7000-8000-000000000002", "01a0df5d-0000-7000-8000-00000000000b")); c == errcodes.ProcessOperationInProgress || c == errcodes.ProcessRunAlreadyStarted {
		t.Fatalf("после «Выполнено» пост свободен: %q", c)
	}
}
