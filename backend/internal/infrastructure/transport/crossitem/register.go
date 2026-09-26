package crossitem

import (
	app "ant/internal/application/crossitem"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля crossitem.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
