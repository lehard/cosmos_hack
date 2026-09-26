package quality

import (
	"context"

	"ant/internal/application/platform"
	app "ant/internal/application/quality"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "quality"

// Register объявляет операции модуля quality: сигналы, результаты контроля,
// полнота, дефекты, пропуски брака, карта реакций (FR-14, FR-35…FR-38, FR-48,
// FR-50). Команд нет: результаты — факты через приём, решения — nonconformity.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_ = c
	type itemIn struct {
		ItemID string `path:"item_id" maxLength:"128" doc:"Изделие."`
		httpapi.MomentQuery
	}

	httpapi.Read(api, httpapi.Get("/signals", "Сигналы о признаке дефекта",
		"Сигналы quality.signal.raised по изделию и состоянию рассмотрения. Сигнал ≠ брак (NFR-UI-4)."),
		platform.Action{ID: "quality.signal.list", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct {
			ItemID string `query:"item_id" maxLength:"128" doc:"Изделие."`
			State  string `query:"state" enum:"open,confirmed,rejected" doc:"Состояние рассмотрения."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.QualitySignalList, error) {
			return q.Signals(ctx, app.SignalFilter{ItemID: in.ItemID, State: in.State}, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/signals/{signal_id}", "Исходный сигнал",
		"FR-38: сигнал VisionQC как цепочка ступеней ансамбля (где дефект / какой тип), уверенность и качество наблюдения в б. п., "+
			"вектор версий наблюдения (AD-29), иллюстрация — адрес материала. Уверенность ≠ вероятность брака."),
		platform.Action{ID: "quality.signal.read", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct {
			SignalID string `path:"signal_id" maxLength:"128" doc:"Сигнал."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.QualitySignal, error) {
			return q.Signal(ctx, in.SignalID, m)
		})

	httpapi.Read(api, httpapi.Get("/items/{item_id}/inspections", "Результаты контроля изделия",
		"FR-36: результаты всех методов одним типом, три исхода — признак дефекта / признака нет / оценка невозможна; пометка источника (FR-140)."),
		platform.Action{ID: "quality.inspection.list", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct {
			ItemID string `path:"item_id" maxLength:"128" doc:"Изделие."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.InspectionResultList, error) {
			return q.Inspections(ctx, in.ItemID, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/items/{item_id}/coverage", "Полнота контроля",
		"FR-35, FR-14: точки контроля плана по изделию — результат получен, ждём или нет (с причиной пропуска)."),
		platform.Action{ID: "quality.coverage.read", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *itemIn, m platform.Moment) (app.InspectionCoverage, error) {
			return q.Coverage(ctx, in.ItemID, m)
		})

	httpapi.Read(api, httpapi.Get("/defects", "Дефекты",
		"FR-37: физические дефекты (ключ — изделие, зона и место, без вида дефекта); дефекты и изделия с дефектами — раздельно."),
		platform.Action{ID: "quality.defect.list", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct {
			ItemID string `query:"item_id" maxLength:"128" doc:"Изделие; пусто — все."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.QualityDefectList, error) {
			return q.Defects(ctx, in.ItemID, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/escapes", "Пропуски брака",
		"Пропущенный брак (quality.escape.recorded) — возврат в контур адаптации анализатора (FR-99)."),
		platform.Action{ID: "quality.escape.list", Owner: owner, Subject: "analyzer_passport"},
		func(ctx context.Context, in *struct {
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.QualityEscapeList, error) {
			return q.Escapes(ctx, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/reaction-map", "Карта реакций",
		"FR-48, FR-50: действующая карта реакций — правила с режимом автоматизации 1–5, классом действия, владельцем, сроком и порогами."),
		platform.Action{ID: "quality.reaction_map.read", Owner: owner, Subject: "process_version"},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }, m platform.Moment) (app.ReactionMap, error) {
			return q.ReactionMap(ctx, m)
		})
}
