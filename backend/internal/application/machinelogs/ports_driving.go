package machinelogs

import (
	"context"
	"time"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля machinelogs (AD-36).
type Queries interface {
	// Equipment — оборудование (machinelogs.equipment.list).
	Equipment(ctx context.Context, stationID string, m platform.Moment) (EquipmentList, error)
	// EquipmentByID — карточка оборудования (machinelogs.equipment.read).
	EquipmentByID(ctx context.Context, equipmentID string, m platform.Moment) (EquipmentState, error)
	// RunProfile — профиль выполнения операции (machinelogs.run_profile.read, FR-148).
	RunProfile(ctx context.Context, runID string, m platform.Moment) (RunProfile, error)
	// Timeline — журнал оборудования на окне (machinelogs.timeline.read, AD-29).
	Timeline(ctx context.Context, equipmentID string, from, to time.Time, m platform.Moment) (EquipmentTimeline, error)
	// Violations — окна нарушений специального процесса (machinelogs.violation.list, FR-151).
	Violations(ctx context.Context, m platform.Moment) (ViolationList, error)
}

// Commands — ведущий порт команд модуля machinelogs: факты оборудования
// приходят через приём (ingest) от edge-агента; команд человека нет.
type Commands interface{}

// Unimplemented — заглушка портов machinelogs: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Equipment(context.Context, string, platform.Moment) (EquipmentList, error) {
	return EquipmentList{}, ni("machinelogs.equipment.list")
}
func (Unimplemented) EquipmentByID(context.Context, string, platform.Moment) (EquipmentState, error) {
	return EquipmentState{}, ni("machinelogs.equipment.read")
}
func (Unimplemented) RunProfile(context.Context, string, platform.Moment) (RunProfile, error) {
	return RunProfile{}, ni("machinelogs.run_profile.read")
}
func (Unimplemented) Timeline(context.Context, string, time.Time, time.Time, platform.Moment) (EquipmentTimeline, error) {
	return EquipmentTimeline{}, ni("machinelogs.timeline.read")
}
func (Unimplemented) Violations(context.Context, platform.Moment) (ViolationList, error) {
	return ViolationList{}, ni("machinelogs.violation.list")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
