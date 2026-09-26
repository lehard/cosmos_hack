package process

import (
	"context"
	"time"

	"ant/internal/application/platform"
	app "ant/internal/application/process"
)

// Adapter — реализация fixtures ведущих портов модуля process (AD-36): живая
// карта, карточка узла, версии процесса в читаемом виде, разница и BPMN — из
// мира заготовок; команды двигают сценарий, если он ждёт именно их.
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

func version(id string) map[string]string { return map[string]string{"version_id": id} }

func ts(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// LiveMap — живая карта на момент (process.live_map.read, FR-1…FR-5, FR-9).
func (Adapter) LiveMap(ctx context.Context, q app.LiveMapQuery, m platform.Moment) (app.LiveMap, error) {
	return respond[app.LiveMap](ctx, "process.live_map.read", map[string]string{
		"period": q.Period, "from": ts(q.From), "to": ts(q.To),
		"process_id": q.ProcessID, "process_version_id": q.ProcessVersionID, "incident_id": q.IncidentID,
	}, &m)
}

// Node — карточка узла (process.node.read, FR-154).
func (Adapter) Node(ctx context.Context, versionID, stepKey string, m platform.Moment) (app.ProcessNodeCard, error) {
	return respond[app.ProcessNodeCard](ctx, "process.node.read", map[string]string{"version_id": versionID, "step_key": stepKey}, &m)
}

// Processes — процессы для выбора (process.process.list, UI-11).
func (Adapter) Processes(ctx context.Context, m platform.Moment) (app.ProcessList, error) {
	return respond[app.ProcessList](ctx, "process.process.list", nil, &m)
}

// Versions — версии процесса (process.version.list, FR-22); processID пусто — основной.
func (Adapter) Versions(ctx context.Context, processID string, m platform.Moment) (app.ProcessVersionList, error) {
	return respond[app.ProcessVersionList](ctx, "process.version.list", map[string]string{"process_id": processID}, &m)
}

// Version — версия в читаемом виде (process.version.read, FR-24).
func (Adapter) Version(ctx context.Context, versionID string, m platform.Moment) (app.ProcessVersion, error) {
	return respond[app.ProcessVersion](ctx, "process.version.read", version(versionID), &m)
}

// Diff — разница версий (process.version.diff, FR-24).
func (Adapter) Diff(ctx context.Context, versionID, againstID string, m platform.Moment) (app.ProcessVersionDiff, error) {
	return respond[app.ProcessVersionDiff](ctx, "process.version.diff", map[string]string{"version_id": versionID, "against": againstID}, &m)
}

// Bpmn — BPMN XML версии как загружен (process.version.bpmn, AD-17).
func (Adapter) Bpmn(ctx context.Context, versionID string) (app.ProcessBpmn, error) {
	return respond[app.ProcessBpmn](ctx, "process.version.bpmn", version(versionID), nil)
}

// DraftVersion — черновик версии (process.version.draft).
func (Adapter) DraftVersion(ctx context.Context, in app.DraftVersion) (platform.Receipt, error) {
	return decide(ctx, "process.version.draft", "process_version", "", in.CommandMeta())
}

// SubmitVersion — отправить на кворум (process.version.submit).
func (Adapter) SubmitVersion(ctx context.Context, versionID string, in app.SubmitVersion) (platform.Receipt, error) {
	return decide(ctx, "process.version.submit", "process_version", versionID, in.CommandMeta())
}

// ActivateVersion — ввести версию в действие (process.version.activate).
func (Adapter) ActivateVersion(ctx context.Context, versionID string, in app.ActivateVersion) (platform.Receipt, error) {
	return decide(ctx, "process.version.activate", "process_version", versionID, in.CommandMeta())
}

// RetireVersion — вывести версию (process.version.retire).
func (Adapter) RetireVersion(ctx context.Context, versionID string, in app.RetireVersion) (platform.Receipt, error) {
	return decide(ctx, "process.version.retire", "process_version", versionID, in.CommandMeta())
}

// StartOperation — начать операцию (process.operation.start).
func (Adapter) StartOperation(ctx context.Context, itemID string, in app.StartOperation) (platform.Receipt, error) {
	return decide(ctx, "process.operation.start", "item", itemID, in.CommandMeta())
}

// PauseOperation — пауза операции (process.operation.pause).
func (Adapter) PauseOperation(ctx context.Context, runID string, in app.PauseOperation) (platform.Receipt, error) {
	return decide(ctx, "process.operation.pause", "item", runID, in.CommandMeta())
}

// ResumeOperation — продолжить операцию (process.operation.resume).
func (Adapter) ResumeOperation(ctx context.Context, runID string, in app.ResumeOperation) (platform.Receipt, error) {
	return decide(ctx, "process.operation.resume", "item", runID, in.CommandMeta())
}

// FinishOperation — завершить операцию (process.operation.finish).
func (Adapter) FinishOperation(ctx context.Context, runID string, in app.FinishOperation) (platform.Receipt, error) {
	return decide(ctx, "process.operation.finish", "item", runID, in.CommandMeta())
}

// SendMovement — отправить изделие (process.movement.send).
func (Adapter) SendMovement(ctx context.Context, itemID string, in app.SendMovement) (platform.Receipt, error) {
	return decide(ctx, "process.movement.send", "item", itemID, in.CommandMeta())
}

// ReceiveMovement — принять изделие (process.movement.receive).
func (Adapter) ReceiveMovement(ctx context.Context, itemID string, in app.ReceiveMovement) (platform.Receipt, error) {
	return decide(ctx, "process.movement.receive", "item", itemID, in.CommandMeta())
}
