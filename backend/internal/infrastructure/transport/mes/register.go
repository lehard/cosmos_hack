package mes

import (
	"context"

	app "ant/internal/application/mes"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля mes: задания и блокировки в MES (FR-92,
// FR-93; AD-18, AD-30). Входящее из MES — через обычный приём, блокировки —
// реакции журнала через outbox; команд у модуля нет.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_ = c
	const owner = "mes"
	httpapi.Read(api, httpapi.Get("/mes/orders", "Задания MES", "FR-92: задания MES (B2MML-JSON, проектное предположение)."),
		platform.Action{ID: "mes.order.list", Owner: owner},
		func(ctx context.Context, in *struct {
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.MesJobList, error) {
			return q.Jobs(ctx, m, in.Page())
		})
	httpapi.Read(api, httpapi.Get("/mes/blocks", "Блокировки в MES", "FR-93, AD-30: блок изделия или партии, отправленный в MES, и квитанция."),
		platform.Action{ID: "mes.block.list", Owner: owner},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }, m platform.Moment) (app.MesBlockList, error) {
			return q.Blocks(ctx, m)
		})
}
