package analytics

import (
	app "ant/internal/application/analytics"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля analytics.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
