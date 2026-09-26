package process

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля process (AD-36).
type Queries interface {
	// LiveMap — живая карта на момент (process.live_map.read, FR-1…FR-5, FR-9).
	LiveMap(ctx context.Context, q LiveMapQuery, m platform.Moment) (LiveMap, error)
	// Node — карточка узла (process.node.read, FR-154).
	Node(ctx context.Context, versionID, stepKey string, m platform.Moment) (ProcessNodeCard, error)
	// Processes — процессы для выбора (process.process.list, UI-11).
	Processes(ctx context.Context, m platform.Moment) (ProcessList, error)
	// Versions — версии процесса processID (пусто — основной процесс;
	// process.version.list, FR-22).
	Versions(ctx context.Context, processID string, m platform.Moment) (ProcessVersionList, error)
	// Version — версия в читаемом виде (process.version.read, FR-24).
	Version(ctx context.Context, versionID string, m platform.Moment) (ProcessVersion, error)
	// Diff — разница версий (process.version.diff, FR-24).
	Diff(ctx context.Context, versionID, againstID string, m platform.Moment) (ProcessVersionDiff, error)
	// Bpmn — BPMN XML версии как загружен (process.version.bpmn, AD-17).
	Bpmn(ctx context.Context, versionID string) (ProcessBpmn, error)
}

// Commands — ведущий порт команд модуля process (AD-39).
type Commands interface {
	DraftVersion(ctx context.Context, in DraftVersion) (platform.Receipt, error)
	SubmitVersion(ctx context.Context, versionID string, in SubmitVersion) (platform.Receipt, error)
	ActivateVersion(ctx context.Context, versionID string, in ActivateVersion) (platform.Receipt, error)
	RetireVersion(ctx context.Context, versionID string, in RetireVersion) (platform.Receipt, error)
	StartOperation(ctx context.Context, itemID string, in StartOperation) (platform.Receipt, error)
	PauseOperation(ctx context.Context, runID string, in PauseOperation) (platform.Receipt, error)
	ResumeOperation(ctx context.Context, runID string, in ResumeOperation) (platform.Receipt, error)
	FinishOperation(ctx context.Context, runID string, in FinishOperation) (platform.Receipt, error)
	SendMovement(ctx context.Context, itemID string, in SendMovement) (platform.Receipt, error)
	ReceiveMovement(ctx context.Context, itemID string, in ReceiveMovement) (platform.Receipt, error)
}

// Unimplemented — заглушка портов process: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) LiveMap(context.Context, LiveMapQuery, platform.Moment) (LiveMap, error) {
	return LiveMap{}, ni("process.live_map.read")
}
func (Unimplemented) Node(context.Context, string, string, platform.Moment) (ProcessNodeCard, error) {
	return ProcessNodeCard{}, ni("process.node.read")
}
func (Unimplemented) Processes(context.Context, platform.Moment) (ProcessList, error) {
	return ProcessList{}, ni("process.process.list")
}
func (Unimplemented) Versions(context.Context, string, platform.Moment) (ProcessVersionList, error) {
	return ProcessVersionList{}, ni("process.version.list")
}
func (Unimplemented) Version(context.Context, string, platform.Moment) (ProcessVersion, error) {
	return ProcessVersion{}, ni("process.version.read")
}
func (Unimplemented) Diff(context.Context, string, string, platform.Moment) (ProcessVersionDiff, error) {
	return ProcessVersionDiff{}, ni("process.version.diff")
}
func (Unimplemented) Bpmn(context.Context, string) (ProcessBpmn, error) {
	return ProcessBpmn{}, ni("process.version.bpmn")
}
func (Unimplemented) DraftVersion(context.Context, DraftVersion) (platform.Receipt, error) {
	return platform.Receipt{}, ni("process.version.draft")
}
func (Unimplemented) SubmitVersion(context.Context, string, SubmitVersion) (platform.Receipt, error) {
	return platform.Receipt{}, ni("process.version.submit")
}
func (Unimplemented) ActivateVersion(context.Context, string, ActivateVersion) (platform.Receipt, error) {
	return platform.Receipt{}, ni("process.version.activate")
}
func (Unimplemented) RetireVersion(context.Context, string, RetireVersion) (platform.Receipt, error) {
	return platform.Receipt{}, ni("process.version.retire")
}
func (Unimplemented) StartOperation(context.Context, string, StartOperation) (platform.Receipt, error) {
	return platform.Receipt{}, ni("process.operation.start")
}
func (Unimplemented) PauseOperation(context.Context, string, PauseOperation) (platform.Receipt, error) {
	return platform.Receipt{}, ni("process.operation.pause")
}
func (Unimplemented) ResumeOperation(context.Context, string, ResumeOperation) (platform.Receipt, error) {
	return platform.Receipt{}, ni("process.operation.resume")
}
func (Unimplemented) FinishOperation(context.Context, string, FinishOperation) (platform.Receipt, error) {
	return platform.Receipt{}, ni("process.operation.finish")
}
func (Unimplemented) SendMovement(context.Context, string, SendMovement) (platform.Receipt, error) {
	return platform.Receipt{}, ni("process.movement.send")
}
func (Unimplemented) ReceiveMovement(context.Context, string, ReceiveMovement) (platform.Receipt, error) {
	return platform.Receipt{}, ni("process.movement.receive")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
