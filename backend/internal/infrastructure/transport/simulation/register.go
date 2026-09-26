package simulation

import (
	app "ant/internal/application/simulation"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля simulation.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
