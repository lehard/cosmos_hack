package simulation

import (
	"context"

	"ant/internal/application/platform"
	app "ant/internal/application/simulation"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "simulation"

// Register объявляет операции модуля simulation — пульт тестовых сценариев:
// запуск, скорость, пауза, продолжение, остановка, табло «ожидалось →
// получилось», кнопки цифрового стенда (FR-104…FR-108, FR-129, FR-152; AD-26,
// AD-37, AD-38). Экран — эпик 14; кнопки стенда — эпик 36.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	type runIn struct {
		RunID string `path:"run_id" maxLength:"128" doc:"Прогон сценария."`
		httpapi.MomentQuery
	}
	type runCmd[B any] struct {
		RunID string `path:"run_id" maxLength:"128"`
		Body  B
	}
	emits := func(t ...catalog.Type) []catalog.Type { return t }

	httpapi.Read(api, httpapi.Get("/scenarios", "Сценарии",
		"AD-26: определения scenarios/definitions — 8 ситуаций §4.2, 9 проверок §5.1, демо-сценарии, сбои; число утверждений табло и остановок на решениях."),
		platform.Action{ID: "simulation.scenario.list", Owner: owner, Subject: "run"},
		func(ctx context.Context, _ *struct{}, _ platform.Moment) (app.ScenarioList, error) {
			return q.Scenarios(ctx)
		})

	httpapi.Read(api, httpapi.Get("/runs", "Прогоны сценариев", "AD-38: прогоны — отдельные пространства имён run_id."),
		platform.Action{ID: "simulation.run.list", Owner: owner, Subject: "run"},
		func(ctx context.Context, in *struct{ httpapi.PageQuery }, _ platform.Moment) (app.RunList, error) {
			return q.Runs(ctx, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/runs/{run_id}", "Состояние прогона",
		"Шаг сценария, доменные часы (AD-37), пауза, скорость, решение человека, которого ждёт сценарий, счёт табло."),
		platform.Action{ID: "simulation.run.read", Owner: owner, Subject: "run"},
		func(ctx context.Context, in *runIn, m platform.Moment) (app.Run, error) {
			return q.Run(ctx, in.RunID, m)
		})

	httpapi.Read(api, httpapi.Get("/runs/{run_id}/board", "Табло «ожидалось → получилось»",
		"AD-26, кейс §5.1: утверждения scenarios/expected над теми же operationId проверяются теми же Queries; ожидаемое хранится отдельно от входных событий."),
		platform.Action{ID: "simulation.board.read", Owner: owner, Subject: "run"},
		func(ctx context.Context, in *runIn, m platform.Moment) (app.Board, error) {
			return q.Board(ctx, in.RunID, m)
		})

	httpapi.Read(api, httpapi.Get("/runs/{run_id}/plan", "План прогона",
		"Д-85: чего ждёт прогон сейчас (роль, действие) и что будет дальше — решения людей с остановками, события машин и внешних систем, запланированные сбои — с доменным временем; часы прогона и скорость."),
		platform.Action{ID: "simulation.run.plan", Owner: owner, Subject: "run"},
		func(ctx context.Context, in *struct {
			RunID string `path:"run_id" maxLength:"128" doc:"Прогон сценария."`
			All   bool   `query:"all" doc:"Весь план (прошедшее — done); по умолчанию — от текущего места."`
			Limit int    `query:"limit" minimum:"0" maximum:"1000" doc:"Сколько строк (по умолчанию 50)."`
		}, _ platform.Moment) (app.RunPlan, error) {
			return q.Plan(ctx, in.RunID, app.PlanQuery{All: in.All, Limit: in.Limit})
		})

	httpapi.Read(api, httpapi.Get("/runs/{run_id}/injections", "Кнопки цифрового стенда",
		"FR-152: повтор события, опоздавшее событие, испорченный кадр, сбой станка, потеря куска данных, подделка в обход системы (только профили fixtures и demo)."),
		platform.Action{ID: "simulation.injection.list", Owner: owner, Subject: "run"},
		func(ctx context.Context, in *struct {
			RunID string `path:"run_id" maxLength:"128"`
		}, _ platform.Moment) (app.InjectionList, error) {
			return q.Injections(ctx, in.RunID)
		})

	type startedRun struct {
		httpapi.Receipt
		RunID string `json:"run_id" doc:"Новый прогон — отдельное пространство имён (AD-38)."`
	}
	httpapi.Register(api, httpapi.Post("/scenarios/{scenario_id}/runs", "Запустить сценарий",
		"AD-38: новый прогон с run_id, seed, скоростью и режимом; события идут через stand-ы → edge-агент → обычный приём (AD-26)."),
		platform.Action{ID: "simulation.run.start", Class: platform.ClassRecord, Owner: owner, Subject: "run", Emits: emits(catalog.SimulationRunStarted)},
		func(ctx context.Context, in *struct {
			ScenarioID string `path:"scenario_id" maxLength:"128"`
			Body       app.StartRun
		}) (*httpapi.Out[startedRun], error) {
			r, err := c.StartRun(ctx, in.ScenarioID, in.Body)
			if err != nil {
				return nil, err
			}
			return httpapi.OK(startedRun{Receipt: httpapi.ReceiptOf(r.Receipt).Body, RunID: r.RunID}), nil
		})

	httpapi.Do(api, httpapi.Post("/runs/{run_id}/pause", "Пауза", "AD-26: пауза останавливает поток событий и доменные часы; столы показывают состояние на этот момент."),
		platform.Action{ID: "simulation.run.pause", Class: platform.ClassRecord, Owner: owner, Subject: "run", Emits: emits(catalog.SimulationRunPaused)},
		func(ctx context.Context, in *runCmd[app.RunControl]) (platform.Receipt, error) {
			return c.PauseRun(ctx, in.RunID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/runs/{run_id}/resume", "Продолжить", "Продолжение без потерь и дублей."),
		platform.Action{ID: "simulation.run.resume", Class: platform.ClassRecord, Owner: owner, Subject: "run", Emits: emits(catalog.SimulationRunResumed)},
		func(ctx context.Context, in *runCmd[app.RunControl]) (platform.Receipt, error) {
			return c.ResumeRun(ctx, in.RunID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/runs/{run_id}/stop", "Остановить прогон", "Прогон завершается с итогом stopped."),
		platform.Action{ID: "simulation.run.stop", Class: platform.ClassRecord, Owner: owner, Subject: "run", Emits: emits(catalog.SimulationRunFinished)},
		func(ctx context.Context, in *runCmd[app.RunControl]) (platform.Receipt, error) {
			return c.StopRun(ctx, in.RunID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/runs/{run_id}/speed", "Изменить скорость", "Ускорение доменных часов ×1…×1000 (AD-37); тики — time.clock.ticked."),
		platform.Action{ID: "simulation.run.set_speed", Class: platform.ClassRecord, Owner: owner, Subject: "run", Emits: emits(catalog.TimeClockTicked)},
		func(ctx context.Context, in *runCmd[app.SetSpeed]) (platform.Receipt, error) {
			return c.SetSpeed(ctx, in.RunID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/runs/{run_id}/injections", "Нажать кнопку цифрового стенда",
		"FR-152, AD-26: инъекция через служебный порт stand-ов и обычный приём; «подделка в обход системы» — только демо-инструментом cmd/tamper. Результат виден на столах и табло."),
		platform.Action{ID: "simulation.injection.apply", Class: platform.ClassRecord, Owner: owner, Subject: "run", Emits: emits(catalog.SimulationInjectionApplied)},
		func(ctx context.Context, in *runCmd[app.ApplyInjection]) (platform.Receipt, error) {
			return c.ApplyInjection(ctx, in.RunID, in.Body)
		})
}
