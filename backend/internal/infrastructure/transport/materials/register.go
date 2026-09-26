package materials

import (
	app "ant/internal/application/materials"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля materials.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
