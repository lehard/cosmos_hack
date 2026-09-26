package machinelogs

import (
	"context"
	"time"

	app "ant/internal/application/machinelogs"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
	processfx "ant/internal/infrastructure/fixtures/process"
)

// Adapter — реализация fixtures ведущих портов модуля machinelogs (AD-36):
// оборудование, профиль выполнения, временная линия, нарушения спецпроцесса —
// из мира заготовок. Команд у модуля нет; поверх мира — выполнения операций,
// начатые, приостановленные и завершённые с терминала в этой сессии
// (fixtures/process.OperationRuns): текущее выполнение оборудования поста и
// профиль выполнения меняются до сброса прогона.
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Equipment — оборудование участка (machinelogs.equipment.list).
func (Adapter) Equipment(ctx context.Context, stationID string, m platform.Moment) (app.EquipmentList, error) {
	v, err := respond[app.EquipmentList](ctx, "machinelogs.equipment.list", map[string]string{"station_id": stationID}, &m)
	if err != nil {
		return v, err
	}
	runs := processfx.OperationRuns(ctx, &m)
	out := v.Items[:0]
	for _, x := range v.Items {
		if stationID == "" || x.StationID == stationID {
			out = append(out, withRuns(ctx, x, runs))
		}
	}
	v.Items = out
	return v, nil
}

// EquipmentByID — состояние оборудования (machinelogs.equipment.read).
func (Adapter) EquipmentByID(ctx context.Context, equipmentID string, m platform.Moment) (app.EquipmentState, error) {
	v, err := respond[app.EquipmentState](ctx, "machinelogs.equipment.read", map[string]string{"equipment_id": equipmentID}, &m)
	if err != nil {
		return v, err
	}
	return withRuns(ctx, v, processfx.OperationRuns(ctx, &m)), nil
}

// withRuns — состояние оборудования с выполнениями сессии: завершённое
// выполнение освобождает оборудование, начатое на его посту (или на нём
// самом) становится текущим; пауза — оборудование остановлено.
func withRuns(ctx context.Context, x app.EquipmentState, runs map[string]*processfx.Run) app.EquipmentState {
	if len(runs) == 0 {
		return x
	}
	rt, err := loader.Default()
	if err != nil {
		return x
	}
	if r := runs[rt.Local(ctx, x.CurrentRunID)]; x.CurrentRunID != "" && r != nil {
		switch {
		case r.FinishedAt != nil:
			x.CurrentRunID, x.Execution = "", "idle"
		case r.Paused:
			x.Execution = "stopped"
		default:
			x.Execution = "running"
		}
	}
	var cur *processfx.Run
	for _, r := range runs {
		if !r.Started || r.FinishedAt != nil || !(r.Equipment == x.EquipmentID || (r.Equipment == "" && r.StationID != "" && r.StationID == x.StationID)) {
			continue
		}
		if cur == nil || r.StartedAt.After(cur.StartedAt) || (r.StartedAt.Equal(cur.StartedAt) && r.RunID > cur.RunID) {
			cur = r
		}
	}
	if cur != nil {
		x.CurrentRunID, x.Execution = cur.RunID, "running"
		if cur.Paused {
			x.Execution = "stopped"
		}
		if cur.ProgramRef != "" {
			x.ProgramRef = cur.ProgramRef
		}
		at := cur.StartedAt
		x.UpdatedAt = &at
	}
	return x
}

// RunProfile — профиль выполнения операции (machinelogs.run_profile.read):
// выполнение мира (с завершением сессии поверх) или начатое в этой сессии.
func (Adapter) RunProfile(ctx context.Context, runID string, m platform.Moment) (app.RunProfile, error) {
	rt, err := loader.Default()
	if err != nil {
		return app.RunProfile{}, err
	}
	r := processfx.OperationRuns(ctx, &m)[rt.Local(ctx, runID)]
	if r != nil && r.Started {
		op := r.Operator
		p := app.RunProfile{OperationRunID: runID, ItemID: r.ItemID, StepKey: r.StepKey, EquipmentID: r.Equipment, OperatorID: &op, StartedAt: r.StartedAt,
			FinishedAt: r.FinishedAt, IntervalOrigin: "system_computed", ProgramRef: r.ProgramRef, Parameters: []app.CycleParameter{}, Events: []app.EquipmentEventRow{}}
		if op == "" {
			p.OperatorID = nil
		}
		return p, nil
	}
	v, err := respond[app.RunProfile](ctx, "machinelogs.run_profile.read", map[string]string{"run_id": runID}, &m)
	if err == nil && r != nil && r.FinishedAt != nil && v.FinishedAt == nil {
		v.FinishedAt = r.FinishedAt
	}
	return v, err
}

// Timeline — временная линия оборудования (machinelogs.timeline.read);
// окно from/to передаётся параметрами, ответ без окна подходит как общий.
func (Adapter) Timeline(ctx context.Context, equipmentID string, from, to time.Time, m platform.Moment) (app.EquipmentTimeline, error) {
	return respond[app.EquipmentTimeline](ctx, "machinelogs.timeline.read",
		map[string]string{"equipment_id": equipmentID, "from": ts(from), "to": ts(to)}, &m)
}

// Violations — нарушения специального процесса (machinelogs.violation.list).
func (Adapter) Violations(ctx context.Context, m platform.Moment) (app.ViolationList, error) {
	return respond[app.ViolationList](ctx, "machinelogs.violation.list", nil, &m)
}

func ts(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}
