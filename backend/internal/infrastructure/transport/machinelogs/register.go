package machinelogs

import (
	"context"
	"time"

	app "ant/internal/application/machinelogs"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "machinelogs"

// Register объявляет операции модуля machinelogs: оборудование, профиль
// выполнения операции, журнал оборудования, окна нарушений специального
// процесса (FR-121, FR-147…FR-149, FR-151; AD-29). Факты оборудования приходят
// через приём (ingest); команд человека у модуля нет.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_ = c
	httpapi.Read(api, httpapi.Get("/equipment", "Оборудование",
		"Стол мастера «Люди и оборудование»: режим, исправность, программа, инструмент и его ресурс, поверка на дату, предупреждения."),
		platform.Action{ID: "machinelogs.equipment.list", Owner: owner, Subject: "equipment"},
		func(ctx context.Context, in *struct {
			StationID string `query:"station_id" maxLength:"128" doc:"Участок или пост."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.EquipmentList, error) {
			return q.Equipment(ctx, in.StationID, m)
		})

	httpapi.Read(api, httpapi.Get("/equipment/{equipment_id}", "Карточка оборудования", "Состояние по классификации MTConnect (AD-29), поверка, предупреждения."),
		platform.Action{ID: "machinelogs.equipment.read", Owner: owner, Subject: "equipment"},
		func(ctx context.Context, in *struct {
			EquipmentID string `path:"equipment_id" maxLength:"128"`
			httpapi.MomentQuery
		}, m platform.Moment) (app.EquipmentState, error) {
			return q.EquipmentByID(ctx, in.EquipmentID, m)
		})

	httpapi.Read(api, httpapi.Get("/operation-runs/{run_id}/profile", "Профиль выполнения операции",
		"FR-148: оборудование, программа, инструмент, сводки циклов против уставки и отклонения на окне выполнения; привязка — межизделийная стадия (AD-29)."),
		platform.Action{ID: "machinelogs.run_profile.read", Owner: owner, Subject: "equipment"},
		func(ctx context.Context, in *struct {
			RunID string `path:"run_id" maxLength:"128" doc:"operation_run_id."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.RunProfile, error) {
			return q.RunProfile(ctx, in.RunID, m)
		})

	httpapi.Read(api, httpapi.Get("/equipment/{equipment_id}/timeline", "Журнал оборудования на окне",
		"FR-121, AD-29: четыре слоя — что делал станок, чем, как шёл процесс (сводки на окно цикла), отклонения; сырые данные остаются на краю (FR-147)."),
		platform.Action{ID: "machinelogs.timeline.read", Owner: owner, Subject: "equipment"},
		func(ctx context.Context, in *struct {
			EquipmentID string `path:"equipment_id" maxLength:"128"`
			From        string `query:"from" required:"true" format:"date-time" doc:"Начало окна."`
			To          string `query:"to" required:"true" format:"date-time" doc:"Конец окна."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.EquipmentTimeline, error) {
			from, err := time.Parse(time.RFC3339Nano, in.From)
			if err != nil {
				return app.EquipmentTimeline{}, platform.Fail("api.validation_failed", "field", "from", "reason", err.Error())
			}
			to, err := time.Parse(time.RFC3339Nano, in.To)
			if err != nil {
				return app.EquipmentTimeline{}, platform.Fail("api.validation_failed", "field", "to", "reason", err.Error())
			}
			return q.Timeline(ctx, in.EquipmentID, from, to, m)
		})

	httpapi.Read(api, httpapi.Get("/equipment/violations", "Окна нарушений специального процесса",
		"FR-151: нарушение режима на операции со специальным процессом — несоответствие для всех изделий окна, даже без найденного дефекта; решение — комиссией."),
		platform.Action{ID: "machinelogs.violation.list", Owner: owner, Subject: "equipment"},
		func(ctx context.Context, _ *struct{ httpapi.MomentQuery }, m platform.Moment) (app.ViolationList, error) {
			return q.Violations(ctx, m)
		})
}
