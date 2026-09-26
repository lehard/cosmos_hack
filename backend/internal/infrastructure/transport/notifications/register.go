package notifications

import (
	"context"

	app "ant/internal/application/notifications"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "notifications"

// Register объявляет операции модуля notifications: сводка для шапки, блок
// «требует вашего внимания», лента тревог, задачи (FR-8, FR-55, FR-57; AD-4,
// AD-40: задачи, уведомления и сроки порождает только notifications).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	httpapi.Register(api, httpapi.Get("/notifications/summary", "Сводка уведомлений для шапки", "FR-57: непрочитанные по видам — информация, тревога, задача, запрос решения."),
		platform.Action{ID: "notifications.summary.read", Class: platform.ClassRead, Owner: owner, Subject: "notification"},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }) (*httpapi.Out[app.NotificationSummary], error) {
			m, err := in.Moment()
			if err != nil {
				return nil, err
			}
			v, err := q.Summary(ctx, m)
			return httpapi.OK(v), err
		})

	httpapi.Read(api, httpapi.Get("/attention", "Требует вашего внимания",
		"FR-8: просроченные решения с ценой задержки («просрочено на 37 мин — стоят 18 изделий, 2 операции»), меры без подтверждённой эффективности, "+
			"временные меры без достигнутого условия выхода. Блок поверх живой карты стола руководителя."),
		platform.Action{ID: "notifications.attention.list", Owner: owner, Subject: "notification"},
		func(ctx context.Context, _ *struct{ httpapi.MomentQuery }, m platform.Moment) (app.AttentionList, error) {
			return q.Attention(ctx, m)
		})

	httpapi.Read(api, httpapi.Get("/alerts", "Лента тревог",
		"FR-8: просроченные изоляции, точки предъявления сверх срока, «не перемещено в изолятор», аномалии узлов, эскалации, нарушения целостности."),
		platform.Action{ID: "notifications.alert.list", Owner: owner, Subject: "notification"},
		func(ctx context.Context, in *struct {
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.AlertList, error) {
			return q.Alerts(ctx, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/tasks", "Задачи и уведомления", "FR-57: задачи пользователя и его роли — физические, доп. проверка, эскалации, запросы решения; срок и цена задержки."),
		platform.Action{ID: "notifications.task.list", Owner: owner, Subject: "task"},
		func(ctx context.Context, in *struct {
			State      string `query:"state" enum:"open,done,accepted,declined,withdrawn" doc:"Состояние задачи."`
			LocationID string `query:"location_id" maxLength:"128" doc:"Место: участок или рабочее место."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.TaskList, error) {
			return q.Tasks(ctx, app.TaskFilter{State: in.State, LocationID: in.LocationID}, m, in.Page())
		})

	httpapi.Do(api, httpapi.Post("/tasks/{task_id}/acknowledge", "Отметить задачу", "FR-57: задача выполнена, принята или отклонена с примечанием (task.task.acknowledged)."),
		platform.Action{ID: "notifications.task.acknowledge", Class: platform.ClassRecord, Owner: owner, Subject: "task", Emits: []catalog.Type{catalog.TaskTaskAcknowledged}},
		func(ctx context.Context, in *struct {
			TaskID string `path:"task_id" maxLength:"128"`
			Body   app.AcknowledgeTask
		}) (platform.Receipt, error) {
			return c.AcknowledgeTask(ctx, in.TaskID, in.Body)
		})
}
