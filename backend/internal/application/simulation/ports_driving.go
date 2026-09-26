package simulation

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля simulation (AD-36): пульт тестовых сценариев.
type Queries interface {
	// Scenarios — определения сценариев (simulation.scenario.list).
	Scenarios(ctx context.Context) (ScenarioList, error)
	// Runs — прогоны (simulation.run.list).
	Runs(ctx context.Context, p platform.Page) (RunList, error)
	// Run — состояние прогона: шаг, доменные часы, пауза, скорость (simulation.run.read).
	Run(ctx context.Context, runID string, m platform.Moment) (Run, error)
	// Board — табло «ожидалось → получилось» (simulation.board.read, AD-26).
	Board(ctx context.Context, runID string, m platform.Moment) (Board, error)
	// Injections — кнопки цифрового стенда для прогона (simulation.injection.list, FR-152).
	Injections(ctx context.Context, runID string) (InjectionList, error)
}

// Commands — ведущий порт команд модуля simulation.
type Commands interface {
	// StartRun — новый прогон сценария (simulation.run.start): квитанция и run_id.
	StartRun(ctx context.Context, scenarioID string, in StartRun) (StartedRun, error)
	PauseRun(ctx context.Context, runID string, in RunControl) (platform.Receipt, error)
	ResumeRun(ctx context.Context, runID string, in RunControl) (platform.Receipt, error)
	StopRun(ctx context.Context, runID string, in RunControl) (platform.Receipt, error)
	SetSpeed(ctx context.Context, runID string, in SetSpeed) (platform.Receipt, error)
	ApplyInjection(ctx context.Context, runID string, in ApplyInjection) (platform.Receipt, error)
}

// Unimplemented — заглушка портов simulation: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Scenarios(context.Context) (ScenarioList, error) {
	return ScenarioList{}, ni("simulation.scenario.list")
}
func (Unimplemented) Runs(context.Context, platform.Page) (RunList, error) {
	return RunList{}, ni("simulation.run.list")
}
func (Unimplemented) Run(context.Context, string, platform.Moment) (Run, error) {
	return Run{}, ni("simulation.run.read")
}
func (Unimplemented) Board(context.Context, string, platform.Moment) (Board, error) {
	return Board{}, ni("simulation.board.read")
}
func (Unimplemented) Injections(context.Context, string) (InjectionList, error) {
	return InjectionList{}, ni("simulation.injection.list")
}
func (Unimplemented) StartRun(context.Context, string, StartRun) (StartedRun, error) {
	return StartedRun{}, ni("simulation.run.start")
}
func (Unimplemented) PauseRun(context.Context, string, RunControl) (platform.Receipt, error) {
	return platform.Receipt{}, ni("simulation.run.pause")
}
func (Unimplemented) ResumeRun(context.Context, string, RunControl) (platform.Receipt, error) {
	return platform.Receipt{}, ni("simulation.run.resume")
}
func (Unimplemented) StopRun(context.Context, string, RunControl) (platform.Receipt, error) {
	return platform.Receipt{}, ni("simulation.run.stop")
}
func (Unimplemented) SetSpeed(context.Context, string, SetSpeed) (platform.Receipt, error) {
	return platform.Receipt{}, ni("simulation.run.set_speed")
}
func (Unimplemented) ApplyInjection(context.Context, string, ApplyInjection) (platform.Receipt, error) {
	return platform.Receipt{}, ni("simulation.injection.apply")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
