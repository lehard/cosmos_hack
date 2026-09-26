package analysis

import (
	app "ant/internal/application/analysis"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля analysis.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
