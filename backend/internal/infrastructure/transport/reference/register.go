package reference

import (
	app "ant/internal/application/reference"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля reference.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
