package signing

import (
	app "ant/internal/application/signing"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля signing.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_, _, _ = api, q, c
}
