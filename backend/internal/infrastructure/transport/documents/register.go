package documents

import (
	app "ant/internal/application/documents"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля documents.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
