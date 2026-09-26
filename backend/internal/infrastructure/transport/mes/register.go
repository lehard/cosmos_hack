package mes

import (
	app "ant/internal/application/mes"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля mes.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
