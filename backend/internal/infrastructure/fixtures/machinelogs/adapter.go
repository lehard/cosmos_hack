package machinelogs

import (
	"context"
	"time"

	app "ant/internal/application/machinelogs"
	"ant/internal/application/platform"
)

// Adapter — реализация fixtures ведущих портов модуля machinelogs (AD-36):
// оборудование, профиль выполнения, временная линия, нарушения спецпроцесса —
// из мира заготовок. Команд у модуля нет.
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
	if err != nil || stationID == "" {
		return v, err
	}
	out := v.Items[:0]
	for _, x := range v.Items {
		if x.StationID == stationID {
			out = append(out, x)
		}
	}
	v.Items = out
	return v, nil
}

// EquipmentByID — состояние оборудования (machinelogs.equipment.read).
func (Adapter) EquipmentByID(ctx context.Context, equipmentID string, m platform.Moment) (app.EquipmentState, error) {
	return respond[app.EquipmentState](ctx, "machinelogs.equipment.read", map[string]string{"equipment_id": equipmentID}, &m)
}

// RunProfile — профиль выполнения операции (machinelogs.run_profile.read).
func (Adapter) RunProfile(ctx context.Context, runID string, m platform.Moment) (app.RunProfile, error) {
	return respond[app.RunProfile](ctx, "machinelogs.run_profile.read", map[string]string{"run_id": runID}, &m)
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
