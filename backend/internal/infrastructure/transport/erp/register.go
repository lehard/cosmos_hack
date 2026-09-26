package erp

import (
	app "ant/internal/application/erp"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля erp.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
