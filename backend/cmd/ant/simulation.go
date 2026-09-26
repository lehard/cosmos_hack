package main

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	crossitemapp "ant/internal/application/crossitem"
	engineapp "ant/internal/application/engine"
	ingestapp "ant/internal/application/ingest"
	appjournal "ant/internal/application/journal"
	simapp "ant/internal/application/simulation"
	sim "ant/internal/domain/simulation"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/clock"
	simstore "ant/internal/infrastructure/storage/simulation"
	"ant/internal/infrastructure/transport/httpapi"
	simhttp "ant/internal/infrastructure/transport/simulation"
)

// Модуль simulation (эпики 32, 16) на ядре процесса: пульт тестовых
// сценариев — прогоны генератора через обычный приём, решения демо-персон
// теми же операциями API, служебные записи прогона и тики часов сценария в
// журнал, автосверка «ожидалось → получилось» после того, как воркер и
// стадия догнали журнал (Settler).
//
// Сервис один на процесс: его операции регистрирует роль api (buildAPI), а
// раннер (Loop) крутит роль stands (одна копия, AD-6). Состояние прогонов —
// в памяти процесса (MemoryRuns), поэтому api и stands — в одном процессе
// (compose: -role=api,worker,crossitem,projector,stands).

// defaultSimTick — период раннера прогонов, если stands.tick не задан.
const defaultSimTick = 500 * time.Millisecond

// simBatch — сколько наступивших событий и шагов прогона раннер проводит за
// один шаг (между шагами — сохранение состояния и команды пульта).
const simBatch = 200

// simHolder — сервис симуляции процесса и сигнал его готовности для роли stands.
type simHolder struct {
	once  sync.Once
	ready chan struct{}
	svc   *simapp.Service
}

func (h *simHolder) init() { h.once.Do(func() { h.ready = make(chan struct{}) }) }

// publish — сервис собран ролью api.
func (e *environment) publishSimulation(svc *simapp.Service) {
	e.sim.init()
	e.sim.svc = svc
	close(e.sim.ready)
}

// awaitSimulation — сервис симуляции, когда его соберёт роль api этого
// процесса; ctx отменён — nil.
func (e *environment) awaitSimulation(ctx context.Context) *simapp.Service {
	e.sim.init()
	select {
	case <-e.sim.ready:
		return e.sim.svc
	case <-ctx.Done():
		return nil
	}
}

// portProxy — Probe и Actor симуляции поверх HTTP API процесса: API
// собирается после сервиса (buildAPI регистрирует его операции), поэтому
// порт подставляется, когда API готов.
type portProxy struct {
	p atomic.Pointer[simhttp.HTTPPort]
}

var (
	_ simapp.Probe = (*portProxy)(nil)
	_ simapp.Actor = (*portProxy)(nil)
)

func (x *portProxy) Read(ctx context.Context, op string, params map[string]string, runID string) (any, error) {
	p := x.p.Load()
	if p == nil {
		return nil, simapp.ErrUnavailable
	}
	return p.Read(ctx, op, params, runID)
}

func (x *portProxy) Act(ctx context.Context, persona, op string, params map[string]string, body map[string]any) (simapp.ActResult, error) {
	p := x.p.Load()
	if p == nil {
		return simapp.ActResult{}, simapp.ErrUnavailable
	}
	return p.Act(ctx, persona, op, params, body)
}

func (x *portProxy) Decided(ctx context.Context, runID, op string, since int64) (bool, int64, error) {
	p := x.p.Load()
	if p == nil {
		return false, 0, nil
	}
	return p.Decided(ctx, runID, op, since)
}

// runGateway — события прогона в обычный приём с прогоном в контексте:
// время приёма — доменные часы этого прогона (AD-37, AD-38).
type runGateway struct{ g simapp.IngestGateway }

func (r runGateway) Deliver(ctx context.Context, runID string, batch []sim.Emission) ([]simapp.Delivered, error) {
	return r.g.Deliver(appjournal.WithRun(ctx, runID), runID, batch)
}

// runStart — доменное «сейчас» на старте прогона (от него сдвигаются даты
// определения, AD-38): не раньше recorded_at головы журнала — иначе первый
// тик нового прогона отвергается как убывание recorded_at (AD-37).
type runStart struct{ c *core }

func (r runStart) Now(ctx context.Context) (time.Time, error) {
	t, err := r.c.domainClock().Now(ctx)
	if err != nil {
		return time.Time{}, err
	}
	if h, ok, err := r.c.journal.RecordedHead(ctx); err == nil && ok && h.After(t) {
		t = h
	}
	return t, nil
}

// simulationLive — live-сервис пульта сценариев (эпик 32) на ядре процесса.
// Без каталога сценариев (stands.scenarios) — nil: операции отвечают 501.
func simulationLive(ctx context.Context, env *environment, ingest *ingestapp.Service) (*simapp.Service, *portProxy, error) {
	dir := env.cfg.Stands.Scenarios
	if dir == "" || ingest == nil {
		env.log.Warn("симуляция выключена: нет каталога сценариев (stands.scenarios) или живого приёма")
		return nil, nil, nil
	}
	c, err := env.readyCore(ctx)
	if err != nil {
		return nil, nil, err
	}
	proxy := &portProxy{}
	settler := &journalstore.Settler{Store: c.journal, Signal: c.listener, Partitions: env.cfg.Engine.Partitions,
		Consumers: func() []string {
			names := []string{crossitemapp.ConsumerStage}
			for _, g := range c.registry.Globals() {
				names = append(names, engineapp.ConsumerName(g.Name))
			}
			return names
		}}
	svc := simapp.NewServiceWith(simapp.Deps{
		Definitions: simstore.NewFiles(dir),
		Gateway:     runGateway{simapp.IngestGateway{Ingest: ingest, SentAt: true}},
		Probe:       proxy,
		Actor:       proxy,
		Stands:      simapp.StandControl{Control: env.standsRegistry()},
		// Подделка в обход системы (S09, F25): демо-инструмент эпика 29 — make
		// tamper; порт пульта к нему — заглушка (Д-60), шаг «пропущен» с пояснением.
		Tamper: simapp.PendingTamperer{},
		Recorder: &simapp.JournalRecorder{Store: c.journal, ScenarioClock: scenarioClock(env.cfg), Now: clock.System{}.Now,
			DomainBuild: c.codec.DomainBuild, Partition: env.cfg.Engine.Partitions},
		Settler: settler,
		Store:   simstore.NewMemoryRuns(),
		Infra:   clock.System{},
		Domain:  runStart{c},
		Profile: env.cfg.Profile,
		Log:     env.moduleLog("simulation"),
		// Шаг раннера — небольшими порциями: состояние прогона (шаг, часы,
		// табло) видно на пульте по ходу, пауза и остановка не ждут конца прогона.
		Batch: simBatch,
	})
	env.log.Info("симуляция: пульт сценариев", "scenarios", dir, "scenario_clock", scenarioClock(env.cfg))
	return svc, proxy, nil
}

// attachSimulation — порт сверки и решений над собранным API; сервис —
// роли stands процесса.
func attachSimulation(env *environment, svc *simapp.Service, proxy *portProxy, api *httpapi.API, h http.Handler) {
	if svc == nil {
		return
	}
	proxy.p.Store(simhttp.NewHTTPPort(api, h))
	env.publishSimulation(svc)
}

// runSimulation — раннер прогонов в роли stands: продвигает активные прогоны
// каждые stands.tick. Сервис собирает роль api этого же процесса.
func runSimulation(ctx context.Context, env *environment) {
	svc := env.awaitSimulation(ctx)
	if svc == nil {
		return
	}
	tick := env.cfg.Stands.Tick
	if tick <= 0 {
		tick = defaultSimTick
	}
	env.log.Info("симуляция: раннер прогонов", "tick", tick)
	_ = svc.Loop(ctx, tick)
}
