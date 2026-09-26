package erp

import (
	"context"

	app "ant/internal/application/erp"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "erp"

// Register объявляет операции модуля erp: задания 1С и Галактики, исходящие
// учётные сообщения с квитанциями, каналы обмена, ручная переотправка и
// решение о компенсации (FR-90, FR-91, FR-96; AD-7, AD-18). Сквозной
// сценарий кейса §1.5: задание из 1С → брак → решение → 1С.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	type keyCmd[B any] struct {
		BusinessKey string `path:"business_key" maxLength:"256" doc:"Бизнес-ключ: субъект, учётное действие, закрывающая точка (AD-7)."`
		Body        B
	}

	httpapi.Read(api, httpapi.Get("/erp/orders", "Задания учётных систем", "Кейс §1.5: производственные задания из 1С и Галактики (erp.order.received) и сколько изделий по ним запущено."),
		platform.Action{ID: "erp.order.list", Owner: owner, Subject: "erp_message"},
		func(ctx context.Context, in *struct {
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.ErpOrderList, error) {
			return q.Orders(ctx, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/erp/messages", "Исходящие учётные сообщения",
		"AD-7, AD-18: «принято в работу», «смена склада», «перевод в брак», «возврат поставщику», «выпуск», «результат контроля» — со статусом отправки и квитанцией; "+
			"ось «учёт в 1С» меняется только квитанцией."),
		platform.Action{ID: "erp.message.list", Owner: owner, Subject: "erp_message"},
		func(ctx context.Context, in *struct {
			ItemID string `query:"item_id" maxLength:"128" doc:"Изделие."`
			Status string `query:"status" enum:"queued,sent,acknowledged,rejected,quarantined" doc:"Состояние сообщения."`
			System string `query:"system" enum:"onec,galaktika" doc:"Внешняя система."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.ErpMessageList, error) {
			return q.Messages(ctx, app.MessageFilter{ItemID: in.ItemID, Status: in.Status, System: in.System}, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/erp/messages/{business_key}", "Учётное сообщение", "Версии сообщения с одним бизнес-ключом, попытки, квитанции и ошибки."),
		platform.Action{ID: "erp.message.read", Owner: owner, Subject: "erp_message"},
		func(ctx context.Context, in *struct {
			BusinessKey string `path:"business_key" maxLength:"256"`
			httpapi.MomentQuery
		}, m platform.Moment) (app.ErpMessage, error) {
			return q.Message(ctx, in.BusinessKey, m)
		})

	httpapi.Read(api, httpapi.Get("/erp/channels", "Каналы обмена",
		"AD-18: при старте адаптер сверяет метаданные внешней системы ($metadata 1С) и версию контракта; расхождение — degraded, отправки нет."),
		platform.Action{ID: "erp.channel.list", Owner: owner, Subject: "erp_message"},
		func(ctx context.Context, _ *struct{}, _ platform.Moment) (app.ErpChannelList, error) {
			return q.Channels(ctx)
		})

	httpapi.Do(api, httpapi.Post("/erp/messages/{business_key}/resend", "Переотправить сообщение",
		"AD-18: повтор — только при транспортных ошибках, затем карантин и ручная переотправка администратором."),
		platform.Action{ID: "erp.posting.resend", Class: platform.ClassRecord, Owner: owner, Subject: "erp_message", Emits: []catalog.Type{catalog.ErpPostingResendRequested}},
		func(ctx context.Context, in *keyCmd[app.ResendPosting]) (platform.Receipt, error) {
			return c.ResendPosting(ctx, in.BusinessKey, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/erp/messages/{business_key}/compensation", "Решение о компенсации",
		"AD-7: новая версия реакции с тем же бизнес-ключом и другим содержимым даёт исправление (сторно + новое) только по решению человека."),
		platform.Action{ID: "erp.posting.compensate", Class: platform.ClassRecord, Critical: true, CAGroup: "protected_data", Owner: owner, Subject: "erp_message",
			Emits: []catalog.Type{catalog.ErpPostingCompensationDecided}, SignatureLevel: 2},
		func(ctx context.Context, in *keyCmd[app.CompensatePosting]) (platform.Receipt, error) {
			return c.CompensatePosting(ctx, in.BusinessKey, in.Body)
		})
}
