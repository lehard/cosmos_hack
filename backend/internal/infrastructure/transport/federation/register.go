package federation

import (
	app "ant/internal/application/federation"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля federation.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
