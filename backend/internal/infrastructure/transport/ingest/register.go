package ingest

import (
	app "ant/internal/application/ingest"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля ingest.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
