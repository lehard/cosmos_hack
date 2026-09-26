package security

import (
	"context"

	"ant/internal/application/platform"
	app "ant/internal/application/security"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля security.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_ = c
	httpapi.Register(api, httpapi.Get("/integrity", "Состояние целостности журнала «по данным сервера»",
		"AD-46: ant забирает последний подписанный отчёт верификатора у хранителя и журналирует security.integrity.checked. "+
			"Индикатор на столах помечен «по данным сервера» и желтеет сам, если свежего отчёта нет дольше двух интервалов."),
		platform.Action{ID: "security.integrity.read", Class: platform.ClassRead, Owner: "security", Subject: "integrity"},
		func(ctx context.Context, _ *struct{}) (*httpapi.Out[app.IntegrityStatus], error) {
			v, err := q.Integrity(ctx)
			return httpapi.OK(v), err
		})
}
