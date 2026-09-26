package machinelogs

import (
	app "ant/internal/application/machinelogs"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля machinelogs.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
