package nonconformity

import (
	app "ant/internal/application/nonconformity"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля nonconformity.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
