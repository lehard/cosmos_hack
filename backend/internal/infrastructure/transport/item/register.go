package item

import (
	"context"

	app "ant/internal/application/item"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля item.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_ = c
	type lookupIn struct {
		Q string `query:"q" required:"true" minLength:"1" maxLength:"256" doc:"Номер детали или содержимое DataMatrix (ant:carrier:‹тип›:‹значение›)."`
		httpapi.MomentQuery
	}
	httpapi.Register(api, httpapi.Get("/items/lookup", "Найти изделие по номеру детали или скану DataMatrix",
		"Разрешение носителя (AD-41) → изделие на момент. Не найдено — 404 api.not_found."),
		platform.Action{ID: "item.item.lookup", Class: platform.ClassRead, Owner: "item", Subject: "item"},
		func(ctx context.Context, in *lookupIn) (*httpapi.Out[app.ItemLookup], error) {
			m, err := in.Moment()
			if err != nil {
				return nil, err
			}
			v, err := q.Lookup(ctx, in.Q, m)
			return httpapi.OK(v), err
		})
}
