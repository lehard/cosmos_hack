package notifications

import (
	"context"

	app "ant/internal/application/notifications"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля notifications.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_ = c
	httpapi.Register(api, httpapi.Get("/notifications/summary", "Сводка уведомлений для шапки", "FR-57: непрочитанные по видам — информация, тревога, задача, запрос решения."),
		platform.Action{ID: "notifications.summary.read", Class: platform.ClassRead, Owner: "notifications", Subject: "notification"},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }) (*httpapi.Out[app.NotificationSummary], error) {
			m, err := in.Moment()
			if err != nil {
				return nil, err
			}
			v, err := q.Summary(ctx, m)
			return httpapi.OK(v), err
		})
}
