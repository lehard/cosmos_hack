package simulation

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	sim "ant/internal/domain/simulation"
)

// Service — реализация live ведущих портов модуля simulation (AD-36): пульт
// тестовых сценариев над генератором (domain/simulation), определениями
// scenarios/, обычным приёмом, теми же операциями API и журналом.
//
// Без зависимостей (NewService) — заглушка 501, как в волне 1; с
// зависимостями (NewServiceWith) — прогоны, часы, пауза, скорость, остановка
// на решениях людей, автосверка «ожидалось → получилось» (FR-104…FR-108,
// FR-129; AD-26, AD-37, AD-38).
type Service struct {
	Unimplemented
	d    Deps
	live bool

	mu    sync.Mutex
	plans map[string]*runPlan // run_id → план прогона
	// runMu — один шаг прогона за раз (роль stands — одна копия, AD-6).
	runMu sync.Mutex
}

// Deps — ведомые порты сервиса.
type Deps struct {
	Definitions Definitions
	Gateway     Gateway
	Probe       Probe
	Actor       Actor
	Stands      Stands
	Tamper      Tamperer
	Recorder    Recorder
	Settler     Settler
	Store       RunStore
	Infra       InfraClock
	Domain      DomainNow
	// Profile — профиль развёртывания (prod | demo | fixtures | load): подделка
	// в обход системы — только fixtures и demo (AD-26).
	Profile string
	// Batch — сколько наступивших событий и шагов прогон обрабатывает за один
	// шаг раннера (0 — 5000).
	Batch int
}

// NewService создаёт реализацию live без зависимостей (все операции — 501).
func NewService() *Service { return &Service{} }

// NewServiceWith создаёт реализацию live над портами d.
func NewServiceWith(d Deps) *Service {
	if d.Batch <= 0 {
		d.Batch = 5000
	}
	return &Service{d: d, live: true, plans: map[string]*runPlan{}}
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// Режимы и состояния прогона (views.go).
const (
	ModeInteractive = "interactive"
	ModeAutocheck   = "autocheck"
	ModeLoad        = "load"

	StateRunning  = "running"
	StatePaused   = "paused"
	StateWaiting  = "waiting_for_decision"
	StateComplete = "completed"
	StateStopped  = "stopped"
	StateFailed   = "failed"
)

func (s *Service) now() time.Time {
	if s.d.Infra != nil {
		return s.d.Infra.Now()
	}
	return time.Now()
}

// Scenarios — сценарии пульта из каталога определений (simulation.scenario.list).
func (s *Service) Scenarios(ctx context.Context) (ScenarioList, error) {
	if !s.live {
		return s.Unimplemented.Scenarios(ctx)
	}
	cat, err := s.d.Definitions.Catalog(ctx)
	if err != nil {
		return ScenarioList{}, err
	}
	out := ScenarioList{Items: []Scenario{}}
	bundles := map[string]sim.Bundle{}
	for _, e := range cat.Entries {
		b, ok := bundles[e.Run]
		if !ok {
			if b, err = s.d.Definitions.Bundle(ctx, e.Run); err != nil {
				return ScenarioList{}, fmt.Errorf("сценарий %s: %w", e.ID, err)
			}
			bundles[e.Run] = b
		}
		rows := 0
		for _, c := range cards(e) {
			ex, ok, err := s.d.Definitions.Expected(ctx, c)
			if err != nil {
				return ScenarioList{}, err
			}
			if ok {
				for _, cp := range ex.Checkpoints {
					rows += len(cp.Assertions)
				}
			}
		}
		desc := e.Note
		if desc == "" {
			desc = b.Run.Description
		}
		out.Items = append(out.Items, Scenario{ScenarioID: e.ID, Version: b.Run.Version, Title: e.Title, Description: desc,
			CaseRefs: slices.Clone(e.CaseRefs), DefaultSeed: b.Run.Seed, DefaultItems: len(b.Run.Items), Assertions: rows,
			Decisions: countStops(b, cards(e)), Kind: e.Kind, RunDef: e.Run})
	}
	return out, nil
}

func cards(e sim.CatalogEntry) []string {
	if len(e.Cards) > 0 {
		return e.Cards
	}
	return []string{e.ID}
}

// countStops — сколько раз интерактивный прогон остановится на решении человека.
func countStops(b sim.Bundle, only []string) int {
	n := 0
	for _, sc := range b.Scenarios {
		if !slices.Contains(only, sc.ID) {
			continue
		}
		for _, st := range sc.Steps {
			if st.Decision != "" && st.Refusal == "" && (st.Stop || slices.Contains(b.Run.Stops, st.Label)) {
				n++
			}
		}
	}
	return n
}

// StartRun — новый прогон (simulation.run.start, AD-38): отдельное
// пространство имён run_id, план генератора по seed, служебная запись
// simulation.run.started и первый тик часов сценария.
func (s *Service) StartRun(ctx context.Context, scenarioID string, in StartRun) (StartedRun, error) {
	if !s.live {
		return s.Unimplemented.StartRun(ctx, scenarioID, in)
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if in.CommandID != "" {
		if prev, ok := s.byCommand(ctx, in.CommandID); ok {
			return StartedRun{Receipt: platform.Receipt{CommandID: in.CommandID, Seq: prev.BasisSeq, Replayed: true, RecordedAt: prev.LastTick}, RunID: prev.RunID}, nil
		}
	}
	if active, ok := s.active(ctx); ok {
		e := platform.Fail(errcodes.ApiValidationFailed, "run_id", active.RunID)
		e.Detail = "уже идёт прогон " + active.RunID + " (" + active.State + "): остановите его — доменное время журнала не убывает (AD-37), прогоны идут по очереди (AD-38)"
		return StartedRun{}, e
	}
	cat, err := s.d.Definitions.Catalog(ctx)
	if err != nil {
		return StartedRun{}, err
	}
	i := slices.IndexFunc(cat.Entries, func(e sim.CatalogEntry) bool { return e.ID == scenarioID })
	if i < 0 {
		return StartedRun{}, platform.Fail(errcodes.ApiNotFound, "resource", "сценарий "+scenarioID)
	}
	entry := cat.Entries[i]
	b, err := s.d.Definitions.Bundle(ctx, entry.Run)
	if err != nil {
		return StartedRun{}, err
	}
	seed := b.Run.Seed
	if in.Seed != nil && *in.Seed > 0 {
		seed = *in.Seed
	}
	speed := in.Speed
	if speed <= 0 {
		speed = max(b.Run.Speed, 1)
	}
	mode := in.Mode
	if mode == "" {
		mode = ModeInteractive
	}
	var domainNow time.Time
	if s.d.Domain != nil {
		if t, err := s.d.Domain.Now(ctx); err == nil {
			domainNow = t
		}
	}
	runID, err := s.newRunID(ctx, entry.Run, seed)
	if err != nil {
		return StartedRun{}, err
	}
	st := &RunState{RunID: runID, Entry: entry.ID, RunDef: entry.Run, Version: b.Run.Version, Seed: seed, Mode: mode,
		State: StateRunning, CommandID: in.CommandID, GenNow: domainNow, Items: map[string]string{}, Refs: map[string]string{},
		Steps: map[string]StepResult{}, Rows: map[string]sim.Result{}, Baselines: map[string]any{}, Delivered: map[DeliveryStatus]int{}}
	rp, err := s.build(ctx, st)
	if err != nil {
		return StartedRun{}, err
	}
	real := s.now()
	st.StartedAt = real
	st.Clock = sim.Clock{VirtualAt: rp.plan.Start, RealAt: real, Speed: speed}
	seq, err := s.record(ctx, st, "simulation.run.started", rp.plan.Start, map[string]any{
		"run_id": runID, "scenario_id": entry.ID, "scenario_version": b.Run.Version, "seed": seed, "speed": speed, "mode": mode})
	if err != nil {
		return StartedRun{}, err
	}
	if err := s.tick(ctx, st, rp.plan.Start); err != nil {
		return StartedRun{}, err
	}
	if err := s.d.Store.Save(ctx, st); err != nil {
		return StartedRun{}, err
	}
	s.mu.Lock()
	s.plans[runID] = rp
	s.mu.Unlock()
	return StartedRun{Receipt: platform.Receipt{CommandID: in.CommandID, Seq: seq, RecordedAt: rp.plan.Start}, RunID: runID}, nil
}

// active — прогон, который ещё идёт (идёт, ждёт решения или на паузе).
func (s *Service) active(ctx context.Context) (*RunState, bool) {
	runs, err := s.d.Store.List(ctx)
	if err != nil {
		return nil, false
	}
	for _, r := range runs {
		if !isFinal(r.State) {
			return r, true
		}
	}
	return nil, false
}

func (s *Service) byCommand(ctx context.Context, commandID string) (*RunState, bool) {
	runs, err := s.d.Store.List(ctx)
	if err != nil {
		return nil, false
	}
	for _, r := range runs {
		if r.CommandID == commandID {
			return r, true
		}
	}
	return nil, false
}

// newRunID — ‹прогон›-‹seed›-‹n›: уникален в журнале, шаблон AD-38.
func (s *Service) newRunID(ctx context.Context, run string, seed int64) (string, error) {
	runs, err := s.d.Store.List(ctx)
	if err != nil {
		return "", err
	}
	base := fmt.Sprintf("%s-%d", strings.ToLower(run), seed)
	n := 1
	for _, r := range runs {
		if strings.HasPrefix(r.RunID, base+"-") {
			n++
		}
	}
	return fmt.Sprintf("%s-%d", base, n), nil
}

// build — план прогона и точки табло (генератор детерминирован: после
// перезапуска роли план строится заново из того же входа).
func (s *Service) build(ctx context.Context, st *RunState) (*runPlan, error) {
	cat, err := s.d.Definitions.Catalog(ctx)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(cat.Entries, func(e sim.CatalogEntry) bool { return e.ID == st.Entry })
	if i < 0 {
		return nil, fmt.Errorf("сценарий %s пропал из каталога", st.Entry)
	}
	b, err := s.d.Definitions.Bundle(ctx, st.RunDef)
	if err != nil {
		return nil, err
	}
	plan, err := sim.Generate(b, sim.Params{RunID: st.RunID, Seed: st.Seed, Now: st.GenNow})
	if err != nil {
		return nil, err
	}
	rp := &runPlan{plan: plan, refs: b.Run.Refs, world: b.World}
	for _, c := range cards(cat.Entries[i]) {
		ex, ok, err := s.d.Definitions.Expected(ctx, c)
		if err != nil {
			return nil, err
		}
		if ok {
			rp.add(ex)
		}
	}
	rp.index()
	return rp, nil
}

// plan — план прогона (из кэша или заново).
func (s *Service) plan(ctx context.Context, st *RunState) (*runPlan, error) {
	s.mu.Lock()
	rp, ok := s.plans[st.RunID]
	s.mu.Unlock()
	if ok {
		return rp, nil
	}
	rp, err := s.build(ctx, st)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.plans[st.RunID] = rp
	s.mu.Unlock()
	return rp, nil
}

func (s *Service) load(ctx context.Context, runID string) (*RunState, error) {
	st, ok, err := s.d.Store.Load(ctx, runID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, platform.Fail(errcodes.SimulationRunNotFound, "run_id", runID)
	}
	return st, nil
}

// PauseRun — пауза (simulation.run.pause): поток событий и часы стоят; столы
// показывают состояние на этот момент (FR-129).
func (s *Service) PauseRun(ctx context.Context, runID string, in RunControl) (platform.Receipt, error) {
	if !s.live {
		return s.Unimplemented.PauseRun(ctx, runID, in)
	}
	return s.control(ctx, runID, in.CommandID, func(st *RunState) (string, map[string]any, error) {
		if st.State != StateRunning && st.State != StateWaiting {
			return "", nil, stateConflict(st, "пауза")
		}
		st.Clock = st.Clock.Pause(s.now())
		st.State = StatePaused
		return "simulation.run.paused", map[string]any{"run_id": st.RunID, "reason": "user"}, nil
	})
}

// ResumeRun — продолжение (simulation.run.resume): с того же курсора — без
// потерь и дублей.
func (s *Service) ResumeRun(ctx context.Context, runID string, in RunControl) (platform.Receipt, error) {
	if !s.live {
		return s.Unimplemented.ResumeRun(ctx, runID, in)
	}
	return s.control(ctx, runID, in.CommandID, func(st *RunState) (string, map[string]any, error) {
		if st.State != StatePaused {
			return "", nil, stateConflict(st, "продолжение")
		}
		st.State = StateRunning
		if st.Waiting != nil {
			st.State = StateWaiting
		} else {
			st.Clock = st.Clock.Resume(s.now())
		}
		return "simulation.run.resumed", map[string]any{"run_id": st.RunID}, nil
	})
}

// StopRun — остановка (simulation.run.stop): итог stopped.
func (s *Service) StopRun(ctx context.Context, runID string, in RunControl) (platform.Receipt, error) {
	if !s.live {
		return s.Unimplemented.StopRun(ctx, runID, in)
	}
	return s.control(ctx, runID, in.CommandID, func(st *RunState) (string, map[string]any, error) {
		if isFinal(st.State) {
			return "", nil, stateConflict(st, "остановка")
		}
		st.Clock = st.Clock.Pause(s.now())
		st.State = StateStopped
		t := s.now()
		st.FinishedAt = &t
		st.Waiting = nil
		return "simulation.run.finished", map[string]any{"run_id": st.RunID, "outcome": "stopped"}, nil
	})
}

// SetSpeed — скорость доменных часов ×1…×1000 (simulation.run.set_speed, AD-37).
func (s *Service) SetSpeed(ctx context.Context, runID string, in SetSpeed) (platform.Receipt, error) {
	if !s.live {
		return s.Unimplemented.SetSpeed(ctx, runID, in)
	}
	return s.control(ctx, runID, in.CommandID, func(st *RunState) (string, map[string]any, error) {
		if isFinal(st.State) {
			return "", nil, stateConflict(st, "смена скорости")
		}
		st.Clock = st.Clock.WithSpeed(s.now(), in.Speed)
		return "time.clock.ticked", map[string]any{"now": sim.FormatTime(st.Clock.VirtualAt), "speed": in.Speed}, nil
	})
}

func isFinal(state string) bool {
	return state == StateComplete || state == StateStopped || state == StateFailed
}

func stateConflict(st *RunState, what string) error {
	e := platform.Fail(errcodes.ApiValidationFailed, "run_id", st.RunID)
	e.Detail = fmt.Sprintf("%s невозможна: прогон %s в состоянии %s", what, st.RunID, st.State)
	return e
}

// control — команда пульта над прогоном: изменение состояния и служебная запись.
func (s *Service) control(ctx context.Context, runID, commandID string, fn func(st *RunState) (string, map[string]any, error)) (platform.Receipt, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	st, err := s.load(ctx, runID)
	if err != nil {
		return platform.Receipt{}, err
	}
	typ, data, err := fn(st)
	if err != nil {
		return platform.Receipt{}, err
	}
	at := st.Clock.Now(s.now())
	if at.Before(st.LastTick) {
		at = st.LastTick
	}
	seq, err := s.record(ctx, st, typ, at, data)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := s.d.Store.Save(ctx, st); err != nil {
		return platform.Receipt{}, err
	}
	return platform.Receipt{CommandID: commandID, Seq: seq, RecordedAt: at}, nil
}

// record — служебная запись прогона в журнал (AD-44); без регистратора — только состояние.
func (s *Service) record(ctx context.Context, st *RunState, typ string, at time.Time, data map[string]any) (int64, error) {
	if s.d.Recorder == nil {
		return st.BasisSeq, nil
	}
	seq, err := s.d.Recorder.Record(ctx, Record{Type: typ, RunID: st.RunID, OccurredAt: at, Data: data})
	if err != nil {
		return 0, fmt.Errorf("запись %s прогона %s: %w", typ, st.RunID, err)
	}
	if seq > st.BasisSeq {
		st.BasisSeq = seq
	}
	if typ == "time.clock.ticked" && at.After(st.LastTick) {
		st.LastTick = at
	}
	return seq, nil
}

// tick — доменное «сейчас» сценария: запись time.clock.ticked (AD-37).
func (s *Service) tick(ctx context.Context, st *RunState, at time.Time) error {
	if !at.After(st.LastTick) && !st.LastTick.IsZero() {
		return nil
	}
	_, err := s.record(ctx, st, "time.clock.ticked", at, map[string]any{"now": sim.FormatTime(at), "speed": st.Clock.Speed})
	if err == nil && at.After(st.LastTick) {
		st.LastTick = at
	}
	return err
}

// Runs — прогоны (simulation.run.list): новые первыми.
func (s *Service) Runs(ctx context.Context, p platform.Page) (RunList, error) {
	if !s.live {
		return s.Unimplemented.Runs(ctx, p)
	}
	runs, err := s.d.Store.List(ctx)
	if err != nil {
		return RunList{}, err
	}
	slices.SortFunc(runs, func(a, b *RunState) int { return b.StartedAt.Compare(a.StartedAt) })
	out := RunList{Items: []Run{}}
	for _, st := range runs {
		v, err := s.view(ctx, st)
		if err != nil {
			return RunList{}, err
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}

// Run — состояние прогона (simulation.run.read).
func (s *Service) Run(ctx context.Context, runID string, m platform.Moment) (Run, error) {
	if !s.live {
		return s.Unimplemented.Run(ctx, runID, m)
	}
	st, err := s.load(ctx, runID)
	if err != nil {
		return Run{}, err
	}
	return s.view(ctx, st)
}

func (s *Service) view(ctx context.Context, st *RunState) (Run, error) {
	rp, err := s.plan(ctx, st)
	if err != nil {
		return Run{}, err
	}
	clock := st.Clock.Now(s.now())
	if !st.LastTick.IsZero() && clock.After(st.LastTick) && st.Mode != ModeInteractive {
		clock = st.LastTick
	}
	v := Run{RunID: st.RunID, ScenarioID: st.Entry, ScenarioVersion: st.Version, Seed: st.Seed, Mode: st.Mode, State: st.State,
		Speed: max(st.Clock.Speed, 1), Step: st.Cursor.Actions, Steps: len(rp.plan.Actions), ClockAt: clock, StartedAt: st.StartedAt,
		FinishedAt: st.FinishedAt, Items: len(rp.plan.Truth.Items), BasisSeq: st.BasisSeq}
	if st.Cursor.Actions > 0 && st.Cursor.Actions <= len(rp.plan.Actions) {
		v.StepTitle = actionTitle(rp.plan.Actions[st.Cursor.Actions-1])
	}
	if w := st.Waiting; w != nil {
		v.WaitingFor = &RunWait{Role: w.Role, Action: w.Op, ObjectID: w.Object, Title: w.Title}
		v.StepTitle = w.Title
	}
	for _, r := range rp.rows {
		v.BoardTotal++
		if st.Rows[r.a.ID].Status == sim.StatusPassed {
			v.BoardPassed++
		}
	}
	for _, x := range st.Injections {
		for _, r := range x.Rows {
			v.BoardTotal++
			if st.Rows[r.Assertion.ID].Status == sim.StatusPassed {
				v.BoardPassed++
			}
		}
	}
	return v, nil
}

func actionTitle(a sim.Action) string {
	if a.Note != "" {
		return a.Note
	}
	if a.Label != "" {
		return a.Label
	}
	return a.Operation
}

// errStop — прогон остановился на решении человека.
var errStop = errors.New("ждёт решения человека")
