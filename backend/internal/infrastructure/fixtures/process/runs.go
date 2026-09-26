package process

import (
	"context"
	"time"

	"ant/internal/application/platform"
	app "ant/internal/application/process"
	"ant/internal/infrastructure/fixtures/loader"
)

// Выполнения операций сессии (FR-137, терминал исполнителя): начало, пауза,
// продолжение и завершение операции мир заготовок не меняют, но терминал и
// оборудование поста показывают их до сброса прогона (loader.Runtime.Record).

// KindOperationRun — вид объекта фактов выполнения операции.
const KindOperationRun = "operation_run"

// Run — выполнение операции по фактам сессии.
type Run struct {
	RunID      string
	ItemID     string
	StepKey    string
	StationID  string
	Equipment  string
	ProgramRef string
	Operator   string
	StartedAt  time.Time
	// Started — выполнение начато в этой сессии (иначе — выполнение мира,
	// над которым сессия только приостановила или завершила).
	Started    bool
	Paused     bool
	FinishedAt *time.Time
	Completion string
}

// OperationRuns — выполнения операций по фактам сессии на момент m: id → выполнение.
func OperationRuns(ctx context.Context, m *platform.Moment) map[string]*Run {
	rt, err := loader.Default()
	if err != nil {
		return nil
	}
	out := map[string]*Run{}
	for _, f := range rt.Facts(ctx, m, KindOperationRun) {
		r := out[f.ID]
		if r == nil {
			r = &Run{RunID: f.ID}
			out[f.ID] = r
		}
		switch b := f.Body.(type) {
		case startedRun:
			r.ItemID, r.StepKey, r.StationID, r.Equipment, r.ProgramRef = b.ItemID, b.In.StepKey, b.In.StationID, b.In.EquipmentID, b.In.ProgramRef
			if r.StationID == "" {
				r.StationID = b.In.WorkplaceID
			}
			r.Operator, r.StartedAt, r.Started, r.Paused, r.FinishedAt = f.Actor, f.At, true, false, nil
		case app.PauseOperation:
			r.Paused = true
		case app.ResumeOperation:
			r.Paused = false
		case app.FinishOperation:
			at := f.At
			r.FinishedAt, r.Completion, r.Paused = &at, b.Completion, false
		}
	}
	return out
}

// startedRun — тело факта начала операции: изделие и команда.
type startedRun struct {
	ItemID string
	In     app.StartOperation
}

// record — команда над выполнением операции: квитанция и факт сессии.
func record(ctx context.Context, op, runID string, meta platform.CommandMeta, body any) (platform.Receipt, error) {
	rt, err := loader.Default()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Record(ctx, op, loader.ObjectRef{Kind: KindOperationRun, ID: runID}, meta, body,
		loader.Change{Entity: string(platform.EntityEquipment), ID: "global"}, loader.Change{Entity: string(platform.EntityItem), ID: "global"},
		loader.Change{Entity: string(platform.EntityLiveMap), ID: "global"})
}
