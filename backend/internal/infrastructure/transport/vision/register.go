package vision

import (
	app "ant/internal/application/vision"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля vision.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
