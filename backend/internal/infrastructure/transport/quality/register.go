package quality

import (
	app "ant/internal/application/quality"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля quality.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
