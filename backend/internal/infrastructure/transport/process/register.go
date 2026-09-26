package process

import (
	"context"
	"time"

	"ant/internal/application/platform"
	app "ant/internal/application/process"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "process"

// Register объявляет операции модуля process: живая карта, карточка узла,
// версии процесса в читаемом виде и их разница, BPMN версии, жизненный цикл
// версии (кворум), операции и перемещения с терминала исполнителя и мастера
// (FR-1…FR-5, FR-10…FR-25, FR-44, FR-47, FR-137, FR-154; AD-17, AD-21, AD-22).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	httpapi.Read(api, httpapi.Get("/live-map", "Живая карта процесса",
		"FR-1…FR-5, FR-9: схема действующей (или выбранной) версии, изделия-точки по step_key, счётчики узлов за период (считает analytics), "+
			"ограничение линии и аномалии, узлы «оценка невозможна», режим инцидента. Запрос на момент — та же свёртка без записи (AD-22)."),
		platform.Action{ID: "process.live_map.read", Owner: owner, Subject: "live_map"},
		func(ctx context.Context, in *struct {
			Period           string `query:"period" enum:"shift,day,week,month,custom" doc:"Период счётчиков (FR-3); по умолчанию shift."`
			From             string `query:"from" format:"date-time" doc:"Начало произвольного периода."`
			To               string `query:"to" format:"date-time" doc:"Конец произвольного периода."`
			ProcessVersionID string `query:"process_version_id" maxLength:"128" doc:"Версия процесса; не задана — действующая."`
			IncidentID       string `query:"incident_id" maxLength:"128" doc:"Режим инцидента (FR-9)."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.LiveMap, error) {
			lq := app.LiveMapQuery{ProcessVersionID: in.ProcessVersionID, Period: in.Period, IncidentID: in.IncidentID}
			if in.From != "" {
				t, err := time.Parse(time.RFC3339Nano, in.From)
				if err != nil {
					return app.LiveMap{}, platform.Fail("api.validation_failed", "field", "from", "reason", err.Error())
				}
				lq.From = &t
			}
			if in.To != "" {
				t, err := time.Parse(time.RFC3339Nano, in.To)
				if err != nil {
					return app.LiveMap{}, platform.Fail("api.validation_failed", "field", "to", "reason", err.Error())
				}
				lq.To = &t
			}
			return q.LiveMap(ctx, lq, m)
		})

	httpapi.Read(api, httpapi.Get("/process/versions/{version_id}/nodes/{step_key}", "Карточка узла",
		"FR-154: описание шага из documentation версии изделия, наши свойства, нормативные опоры (FR-156) и счётчики с переходом к изделиям и несоответствиям."),
		platform.Action{ID: "process.node.read", Owner: owner, Subject: "process_version"},
		func(ctx context.Context, in *struct {
			VersionID string `path:"version_id" maxLength:"128"`
			StepKey   string `path:"step_key" maxLength:"128"`
			httpapi.MomentQuery
		}, m platform.Moment) (app.ProcessNodeCard, error) {
			return q.Node(ctx, in.VersionID, in.StepKey, m)
		})

	httpapi.Read(api, httpapi.Get("/process/versions", "Версии процесса", "FR-22: черновик → на утверждении → действующая → выведена; кворум, изделия в работе по версии."),
		platform.Action{ID: "process.version.list", Owner: owner, Subject: "process_version"},
		func(ctx context.Context, _ *struct{ httpapi.MomentQuery }, m platform.Moment) (app.ProcessVersionList, error) {
			return q.Versions(ctx, m)
		})

	type versionIn struct {
		VersionID string `path:"version_id" maxLength:"128"`
		httpapi.MomentQuery
	}
	httpapi.Read(api, httpapi.Get("/process/versions/{version_id}", "Версия процесса в читаемом виде",
		"FR-24: элементы в порядке маршрута с нашими свойствами и порогами карты реакций."),
		platform.Action{ID: "process.version.read", Owner: owner, Subject: "process_version"},
		func(ctx context.Context, in *versionIn, m platform.Moment) (app.ProcessVersion, error) {
			return q.Version(ctx, in.VersionID, m)
		})

	httpapi.Read(api, httpapi.Get("/process/versions/{version_id}/diff", "Разница версий",
		"FR-24: читаемая разница с действующей (или указанной) версией — входит в лист утверждения кворума."),
		platform.Action{ID: "process.version.diff", Owner: owner, Subject: "process_version"},
		func(ctx context.Context, in *struct {
			VersionID string `path:"version_id" maxLength:"128"`
			Against   string `query:"against" maxLength:"128" doc:"С чем сравнить; пусто — действующая версия."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.ProcessVersionDiff, error) {
			return q.Diff(ctx, in.VersionID, in.Against, m)
		})

	httpapi.Read(api, httpapi.Get("/process/versions/{version_id}/bpmn", "BPMN версии",
		"AD-17: подписываемая версия — XML целиком (схема, свойства, BPMNDI, documentation); хеш — H(байты как загружены)."),
		platform.Action{ID: "process.version.bpmn", Owner: owner, Subject: "process_version"},
		func(ctx context.Context, in *struct {
			VersionID string `path:"version_id" maxLength:"128"`
		}, _ platform.Moment) (app.ProcessBpmn, error) {
			return q.Bpmn(ctx, in.VersionID)
		})

	// ── команды ──
	emits := func(t ...catalog.Type) []catalog.Type { return t }
	type versionCmd[B any] struct {
		VersionID string `path:"version_id" maxLength:"128"`
		Body      B
	}

	httpapi.Do(api, httpapi.Post("/process/versions", "Создать черновик версии",
		"FR-22, FR-25: черновик из редактора; проверка описания при загрузке (FR-13) — отказ с кодом и id элемента. В журнал черновик не пишется."),
		platform.Action{ID: "process.version.draft", Class: platform.ClassRecord, Owner: owner, Subject: "process_version"},
		func(ctx context.Context, in *struct{ Body app.DraftVersion }) (platform.Receipt, error) {
			return c.DraftVersion(ctx, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/process/versions/{version_id}/submit", "Отправить на утверждение",
		"FR-23: лист утверждения версии (документ с маршрутом кворума: технолог-автор → контролёр с полномочием «кворум», руководитель производства)."),
		platform.Action{ID: "process.version.submit", Class: platform.ClassRecord, Owner: owner, Subject: "process_version", Emits: emits(catalog.NormativeVersionSubmitted), SignatureLevel: 2},
		func(ctx context.Context, in *versionCmd[app.SubmitVersion]) (platform.Receipt, error) {
			return c.SubmitVersion(ctx, in.VersionID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/process/versions/{version_id}/activate", "Ввести версию в действие",
		"FR-23, AD-17: только после закрытия маршрута кворума; движок не исполняет версию без полного набора действительных подписей. Изделия в работе остаются на своей версии."),
		platform.Action{ID: "process.version.activate", Class: platform.ClassIrreversible, Critical: true, CAGroup: "control_change", Owner: owner, Subject: "process_version",
			Emits: emits(catalog.NormativeVersionActivated), Guards: []string{"quorum_complete"}, SignatureLevel: 2},
		func(ctx context.Context, in *versionCmd[app.ActivateVersion]) (platform.Receipt, error) {
			return c.ActivateVersion(ctx, in.VersionID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/process/versions/{version_id}/retire", "Вывести версию", "FR-22: изделия прежних версий показываются на карте с пометкой версии."),
		platform.Action{ID: "process.version.retire", Class: platform.ClassRecord, Critical: true, CAGroup: "control_change", Owner: owner, Subject: "process_version",
			Emits: emits(catalog.NormativeVersionRetired), SignatureLevel: 2},
		func(ctx context.Context, in *versionCmd[app.RetireVersion]) (platform.Receipt, error) {
			return c.RetireVersion(ctx, in.VersionID, in.Body)
		})

	type itemCmd[B any] struct {
		ItemID string `path:"item_id" maxLength:"128"`
		Body   B
	}
	type runCmd[B any] struct {
		RunID string `path:"run_id" maxLength:"128" doc:"operation_run_id."`
		Body  B
	}

	httpapi.Do(api, httpapi.Post("/items/{item_id}/operations", "Начать операцию",
		"FR-137, FR-44: исполнитель на своём рабочем месте; предусловия (FR-17) и лимит доработок (FR-18) — гард; повтор — новый operation_run_id + rework_of (FR-47)."),
		platform.Action{ID: "process.operation.start", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.OperationRunStarted),
			Guards: []string{"preconditions", "rework_limit", "item_blocked"}, SignatureLevel: 1},
		func(ctx context.Context, in *itemCmd[app.StartOperation]) (platform.Receipt, error) {
			return c.StartOperation(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/operation-runs/{run_id}/pause", "Приостановить операцию", "FR-137: причина паузы."),
		platform.Action{ID: "process.operation.pause", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.OperationRunPaused), SignatureLevel: 1},
		func(ctx context.Context, in *runCmd[app.PauseOperation]) (platform.Receipt, error) {
			return c.PauseOperation(ctx, in.RunID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/operation-runs/{run_id}/resume", "Продолжить операцию", "FR-137."),
		platform.Action{ID: "process.operation.resume", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.OperationRunResumed), SignatureLevel: 1},
		func(ctx context.Context, in *runCmd[app.ResumeOperation]) (platform.Receipt, error) {
			return c.ResumeOperation(ctx, in.RunID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/operation-runs/{run_id}/finish", "Завершить операцию", "FR-137: завершена или прервана."),
		platform.Action{ID: "process.operation.finish", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.OperationRunFinished), SignatureLevel: 1},
		func(ctx context.Context, in *runCmd[app.FinishOperation]) (platform.Receipt, error) {
			return c.FinishOperation(ctx, in.RunID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/movements", "Отправить изделие", "FR-16: перемещение между участками и цехами; «в перемещении» до приёма."),
		platform.Action{ID: "process.movement.send", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.OperationMovementSent),
			Guards: []string{"item_blocked"}, SignatureLevel: 1},
		func(ctx context.Context, in *itemCmd[app.SendMovement]) (platform.Receipt, error) {
			return c.SendMovement(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/movements/receive", "Принять изделие",
		"FR-16, FR-137: подтверждение приёма, в том числе перемещения в изолятор — снимает расхождение «изолировано в системе, физически нет»."),
		platform.Action{ID: "process.movement.receive", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.OperationMovementReceived), SignatureLevel: 1},
		func(ctx context.Context, in *itemCmd[app.ReceiveMovement]) (platform.Receipt, error) {
			return c.ReceiveMovement(ctx, in.ItemID, in.Body)
		})
}
