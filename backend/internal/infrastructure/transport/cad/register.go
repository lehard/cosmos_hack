package cad

import (
	app "ant/internal/application/cad"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля cad.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
