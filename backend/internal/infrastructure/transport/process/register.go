package process

import (
	app "ant/internal/application/process"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля process.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
