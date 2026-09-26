package crossitem

import (
	"context"

	app "ant/internal/application/crossitem"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля crossitem: партии, выдача из партий,
// временные группы, ручная привязка событий (FR-15, FR-34, FR-45; AD-41, AD-42).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	const owner = "crossitem"
	emits := func(t ...catalog.Type) []catalog.Type { return t }

	httpapi.Read(api, httpapi.Get("/lots", "Партии", "FR-15: партии, садки, плавки со статусом, количеством и сдерживанием."),
		platform.Action{ID: "crossitem.lot.list", Owner: owner, Subject: "lot"},
		func(ctx context.Context, in *struct {
			Status string `query:"status" maxLength:"32" doc:"Фильтр по статусу."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.LotList, error) {
			return q.Lots(ctx, in.Status, m, in.Page())
		})
	httpapi.Read(api, httpapi.Get("/lots/{lot_id}", "Карточка партии", "FR-15, FR-45: входной контроль, выдачи, изделия из партии, документы."),
		platform.Action{ID: "crossitem.lot.read", Owner: owner, Subject: "lot"},
		func(ctx context.Context, in *struct {
			LotID string `path:"lot_id" maxLength:"128"`
			httpapi.MomentQuery
		}, m platform.Moment) (app.LotCard, error) {
			return q.Lot(ctx, in.LotID, m)
		})
	httpapi.Read(api, httpapi.Get("/item-groups", "Временные группы изделий", "FR-15: садки, групповые операции, транспорт."),
		platform.Action{ID: "crossitem.group.list", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }, m platform.Moment) (app.ItemGroupList, error) {
			return q.Groups(ctx, m)
		})

	type lotCmd[B any] struct {
		LotID string `path:"lot_id" maxLength:"128"`
		Body  B
	}
	httpapi.Do(api, httpapi.Post("/lots/{lot_id}/registration", "Зарегистрировать партию", "Кладовщик регистрирует поступившую партию: фактическое количество, упаковка, сертификат."),
		platform.Action{ID: "crossitem.lot.register", Class: platform.ClassRecord, Owner: owner, Subject: "lot", Emits: emits(catalog.GenealogyLotRegistered), SignatureLevel: 1},
		func(ctx context.Context, in *lotCmd[app.RegisterLot]) (platform.Receipt, error) {
			return c.RegisterLot(ctx, in.LotID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/lots/{lot_id}/issues", "Выдать из партии", "FR-45: выдача в производство — корень генеалогии изделий."),
		platform.Action{ID: "crossitem.lot.issue", Class: platform.ClassRecord, Owner: owner, Subject: "lot",
			Guards: []string{"lot_accepted"}, Emits: emits(catalog.GenealogyLotIssued), SignatureLevel: 1},
		func(ctx context.Context, in *lotCmd[app.IssueLot]) (platform.Receipt, error) {
			return c.IssueLot(ctx, in.LotID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/bindings", "Привязать событие к изделию",
		"AD-41, FR-34: ручная привязка или перепривязка события без изделия; пересвёртка обоих изделий; критическое действие (AD-28)."),
		platform.Action{ID: "crossitem.binding.assign", Class: platform.ClassRecord, Critical: true, CAGroup: "protected_data", Owner: owner, Subject: "item",
			Emits: emits(catalog.BindingLinkAssigned), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.AssignBinding }) (platform.Receipt, error) {
			return c.AssignBinding(ctx, in.Body)
		})
}
