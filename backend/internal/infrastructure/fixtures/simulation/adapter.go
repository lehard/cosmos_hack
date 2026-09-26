package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"ant/internal/application/platform"
	app "ant/internal/application/simulation"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	"ant/internal/infrastructure/fixtures/loader"
)

// Adapter — реализация fixtures ведущих портов модуля simulation — пульт
// тестовых сценариев на мире заготовок (FR-129, AD-36, AD-38): сценарии —
// сценарии библиотеки заготовок, прогон — курсор (шаг, доменные часы, пауза,
// скорость, ожидание решения); старт даёт новый run_id — ID ответов получают
// префикс прогона. Табло и кнопки стенда — из мира заготовок.
type Adapter struct {
	// Runtime — мир заготовок; nil — loader.Default() (тесты подставляют свой).
	Runtime *loader.Runtime
	// Now — реальные часы (InfraClock); nil — time.Now.
	Now func() time.Time
	seq atomic.Int64
}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

func (a *Adapter) rt() (*loader.Runtime, error) {
	if a.Runtime != nil {
		return a.Runtime, nil
	}
	return loader.Default()
}

func (a *Adapter) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

// Scenarios — сценарии пульта: сценарии библиотеки заготовок (simulation.scenario.list).
func (a *Adapter) Scenarios(ctx context.Context) (app.ScenarioList, error) {
	rt, err := a.rt()
	if err != nil {
		return app.ScenarioList{}, err
	}
	lib := rt.Library()
	out := app.ScenarioList{Items: []app.Scenario{}}
	for _, id := range lib.Scenarios() {
		sc, _ := lib.Scenario(id)
		mf := sc.Manifest
		decisions := 0
		for _, h := range mf.Steps {
			if h.Wait != nil {
				decisions++
			}
		}
		items, rows := a.totals(ctx, rt, sc)
		v := app.Scenario{
			ScenarioID: mf.ID, Version: "fixtures-1", Title: mf.Title, Description: mf.Description,
			CaseRefs: append([]string{}, mf.Case...), Decisions: decisions,
			DefaultSeed: defaultSeed(mf.ID), DefaultItems: items, Assertions: rows,
		}
		if n := mf.StartStep; n > 0 {
			at := mf.Steps[n].Clock
			v.StartStep, v.StartAt, v.StartTitle = &n, &at, mf.Steps[n].Title
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}

// Runs — прогоны: текущий прогон курсора; без прогона — сценарий по
// умолчанию на паузе под своим id (simulation.run.list).
func (a *Adapter) Runs(ctx context.Context, _ platform.Page) (app.RunList, error) {
	rt, err := a.rt()
	if err != nil {
		return app.RunList{}, err
	}
	st, sc, err := rt.State(ctx)
	if err != nil {
		return app.RunList{}, err
	}
	return app.RunList{Items: []app.Run{a.view(ctx, rt, st, sc)}}, nil
}

// Run — состояние прогона (simulation.run.read): шаг, часы, пауза, скорость, ожидание.
func (a *Adapter) Run(ctx context.Context, runID string, _ platform.Moment) (app.Run, error) {
	rt, st, sc, err := a.current(ctx, runID)
	if err != nil {
		return app.Run{}, err
	}
	return a.view(ctx, rt, st, sc), nil
}

// Board — табло «ожидалось → получилось» (simulation.board.read, AD-26).
func (a *Adapter) Board(ctx context.Context, runID string, m platform.Moment) (app.Board, error) {
	v, err := a.respondBoard(ctx, runID, &m)
	if err == nil && v.RunID == "" {
		v.RunID = runID
	}
	return v, err
}

// Injections — кнопки цифрового стенда (simulation.injection.list, FR-152).
func (a *Adapter) Injections(ctx context.Context, runID string) (app.InjectionList, error) {
	rt, err := a.rt()
	if err != nil {
		return app.InjectionList{}, err
	}
	var out app.InjectionList
	err = rt.Respond(ctx, "simulation.injection.list", map[string]string{"run_id": runID}, nil, &out)
	return out, err
}

// Plan — план прогона на заготовках (simulation.run.plan): шаги сценария с
// доменными часами; шаг с ожиданием — остановка до решения человека.
func (a *Adapter) Plan(ctx context.Context, runID string, q app.PlanQuery) (app.RunPlan, error) {
	rt, st, sc, err := a.current(ctx, runID)
	if err != nil {
		return app.RunPlan{}, err
	}
	v := a.view(ctx, rt, st, sc)
	out := app.RunPlan{RunID: v.RunID, State: v.State, ClockAt: v.ClockAt, Speed: v.Speed, WaitingFor: v.WaitingFor, Items: []app.PlanEntry{}}
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	for n := 0; n < sc.Steps() && len(out.Items) < limit; n++ {
		done := n < st.Step || (n == st.Step && sc.Header(n).Wait == nil)
		if done && !q.All {
			continue
		}
		h := sc.Header(n)
		e := app.PlanEntry{At: h.Clock, Kind: "event", Title: h.Title, Label: fmt.Sprintf("step-%02d", n), Done: done}
		if w := h.Wait; w != nil {
			e.Kind, e.Stop, e.Role, e.Operation = "decision", true, w.Role, w.Action
			e.ObjectID, e.Title = sc.PrefixID(w.Object.ID, st.RunID), w.Title
			e.Waiting = n == st.Step && v.State == "waiting_for_decision"
		}
		out.Items = append(out.Items, e)
	}
	return out, nil
}

// StartRun — запуск прогона сценария (simulation.run.start): новый run_id;
// курсор — на шаге старта (startStep).
func (a *Adapter) StartRun(ctx context.Context, scenarioID string, in app.StartRun) (app.StartedRun, error) {
	rt, err := a.rt()
	if err != nil {
		return app.StartedRun{}, err
	}
	var runID string
	if in.CommandID != "" {
		runID = "fx-" + strings.ReplaceAll(kernel.UUIDv5(constants.NsAnt, "fixtures/run/"+in.CommandID), "-", "")[:8]
	} else {
		runID = fmt.Sprintf("fx-%d", a.seq.Add(1))
	}
	var seed int64
	if in.Seed != nil {
		seed = *in.Seed
	}
	if seed == 0 {
		seed = defaultSeed(scenarioID)
	}
	sc, ok := rt.Library().Scenario(scenarioID)
	if !ok {
		return app.StartedRun{}, platform.Fail(errcodes.ApiNotFound, "object", "сценарий", "id", scenarioID)
	}
	st, err := rt.StartFrom(ctx, scenarioID, runID, in.Mode, seed, in.Speed, startStep(sc, in), a.now())
	if err != nil {
		return app.StartedRun{}, err
	}
	return app.StartedRun{Receipt: receipt("simulation.run.start", in.CommandMeta(), st), RunID: runID}, nil
}

// startStep — шаг старта прогона: from_step, иначе start=start_step — точка
// старта сценария (история сразу «у катастрофы»; шаги до него пройдены — мир
// заготовок накопительный), иначе с начала (решение пользователя: основной
// путь показа — с самого начала, точка старта — по запросу).
func startStep(sc *loader.Scenario, in app.StartRun) int {
	switch {
	case in.FromStep != nil:
		return *in.FromStep
	case in.Start == "start_step":
		return sc.Manifest.StartStep
	}
	return 0
}

// PauseRun — пауза прогона (simulation.run.pause): столы показывают шаг курсора.
func (a *Adapter) PauseRun(ctx context.Context, runID string, in app.RunControl) (platform.Receipt, error) {
	rt, _, _, err := a.current(ctx, runID)
	if err != nil {
		return platform.Receipt{}, err
	}
	st, err := rt.Pause(ctx)
	return receipt("simulation.run.pause", in.CommandMeta(), st), err
}

// ResumeRun — продолжение прогона (simulation.run.resume).
func (a *Adapter) ResumeRun(ctx context.Context, runID string, in app.RunControl) (platform.Receipt, error) {
	rt, _, _, err := a.current(ctx, runID)
	if err != nil {
		return platform.Receipt{}, err
	}
	st, err := rt.Resume(ctx)
	return receipt("simulation.run.resume", in.CommandMeta(), st), err
}

// StopRun — остановка прогона (simulation.run.stop): курсор возвращается к
// сценарию по умолчанию без прогона.
func (a *Adapter) StopRun(ctx context.Context, runID string, in app.RunControl) (platform.Receipt, error) {
	rt, st, _, err := a.current(ctx, runID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := rt.Stop(ctx, runID, a.now()); err != nil {
		return platform.Receipt{}, err
	}
	return receipt("simulation.run.stop", in.CommandMeta(), st), nil
}

// SetSpeed — ускорение доменных часов (simulation.run.set_speed).
func (a *Adapter) SetSpeed(ctx context.Context, runID string, in app.SetSpeed) (platform.Receipt, error) {
	rt, _, _, err := a.current(ctx, runID)
	if err != nil {
		return platform.Receipt{}, err
	}
	st, err := rt.SetSpeed(ctx, in.Speed)
	return receipt("simulation.run.set_speed", in.CommandMeta(), st), err
}

// ApplyInjection — кнопка цифрового стенда (simulation.injection.apply): мир
// заготовок не меняется, кроме шага ожидания именно этого действия.
func (a *Adapter) ApplyInjection(ctx context.Context, runID string, in app.ApplyInjection) (platform.Receipt, error) {
	rt, err := a.rt()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Decide(ctx, "simulation.injection.apply", loader.ObjectRef{Kind: "run", ID: runID}, in.CommandMeta())
}

// current — мир и курсор, если runID — текущий прогон (или, без прогона, id
// сценария по умолчанию); иначе simulation.run_not_found.
func (a *Adapter) current(ctx context.Context, runID string) (*loader.Runtime, platform.CursorState, *loader.Scenario, error) {
	rt, err := a.rt()
	if err != nil {
		return nil, platform.CursorState{}, nil, err
	}
	st, sc, err := rt.State(ctx)
	if err != nil {
		return nil, st, nil, err
	}
	if runID == st.RunID || (st.RunID == "" && runID == sc.Manifest.ID) {
		return rt, st, sc, nil
	}
	return nil, st, nil, platform.Fail(errcodes.SimulationRunNotFound, "run_id", runID)
}

func (a *Adapter) respondBoard(ctx context.Context, runID string, m *platform.Moment) (app.Board, error) {
	rt, err := a.rt()
	if err != nil {
		return app.Board{}, err
	}
	var out app.Board
	err = rt.Respond(ctx, "simulation.board.read", map[string]string{"run_id": runID}, m, &out)
	return out, err
}

// view — прогон пульта из положения курсора и сведений о прогоне.
func (a *Adapter) view(ctx context.Context, rt *loader.Runtime, st platform.CursorState, sc *loader.Scenario) app.Run {
	runID := st.RunID
	state := rt.RunState(st, sc)
	if runID == "" {
		runID = sc.Manifest.ID
		if state == "running" {
			state = "paused"
		}
	}
	speed := st.Speed
	if speed < 1 {
		speed = 1
	}
	r := app.Run{
		RunID: runID, ScenarioID: sc.Manifest.ID, ScenarioVersion: "fixtures-1", Mode: "interactive",
		State: state, Speed: speed, Step: st.Step, StepTitle: sc.Header(st.Step).Title, Steps: sc.Steps(), ClockAt: st.ClockAt,
		StartedAt: sc.Header(0).Clock, BasisSeq: loader.StepSeq(st.Step), Seed: defaultSeed(sc.Manifest.ID),
	}
	if r.ClockAt.IsZero() {
		r.ClockAt = sc.Header(st.Step).Clock
	}
	if ri := rt.Run(st.RunID); ri != nil {
		r.Mode, r.Seed, r.StartedAt, r.FinishedAt = ri.Mode, ri.Seed, ri.StartedAt, ri.FinishedAt
	}
	if w := sc.Header(st.Step).Wait; w != nil && state == "waiting_for_decision" {
		r.WaitingFor = &app.RunWait{Role: w.Role, Action: w.Action, ObjectID: sc.PrefixID(w.Object.ID, st.RunID), Title: w.Title}
	}
	if b, err := a.respondBoard(ctx, runID, nil); err == nil {
		r.BoardPassed, r.BoardTotal = b.Passed, len(b.Rows)
	}
	return r
}

// defaultSeed — seed сценария заготовок: мир flange-bad-day — главная история
// MS-1 определений симуляции (scenarios/definitions/runs/MS-1.yaml), тот же seed.
func defaultSeed(scenario string) int64 {
	if scenario == "flange-bad-day" {
		return 20260921
	}
	return 0
}

// totals — изделий и строк табло сценария заготовок: по ответам его последнего шага.
func (a *Adapter) totals(ctx context.Context, rt *loader.Runtime, sc *loader.Scenario) (items, rows int) {
	cur, cs, err := rt.State(ctx)
	if err != nil || cs == nil || cs.Manifest.ID != sc.Manifest.ID || sc.Steps() == 0 {
		return 0, 0
	}
	last := sc.Header(sc.Steps() - 1).Clock
	m := &platform.Moment{Axis: platform.AxisRecorded, AsOf: &last}
	runID := cur.RunID
	if runID == "" {
		runID = sc.Manifest.ID
	}
	var b app.Board
	if err := rt.Respond(ctx, "simulation.board.read", map[string]string{"run_id": runID}, m, &b); err == nil {
		rows = len(b.Rows)
	}
	var list struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := rt.Respond(ctx, "item.item.list", map[string]string{}, m, &list); err == nil {
		items = len(list.Items)
	}
	return items, rows
}

// receipt — квитанция команды пульта (AD-7): seq шага курсора, id — UUIDv5.
func receipt(op string, meta platform.CommandMeta, st platform.CursorState) platform.Receipt {
	id := kernel.UUIDv5(constants.NsAnt, fmt.Sprintf("fixtures/%s/%s/%d/%s", op, st.RunID, st.Step, meta.CommandID))
	return platform.Receipt{CommandID: meta.CommandID, Seq: loader.StepSeq(st.Step), EventIDs: []string{id}, RecordedAt: st.ClockAt}
}
