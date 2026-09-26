package ops

import (
	app "ant/internal/application/ops"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля ops.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
