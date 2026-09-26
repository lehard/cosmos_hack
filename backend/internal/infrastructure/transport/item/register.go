package item

import (
	"context"

	app "ant/internal/application/item"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "item"

// Register объявляет операции модуля item: поиск и список изделий, паспорт,
// журнал изменений, генеалогия, регистрация, носители, предъявление,
// вмешательства, идентификация, сборка, выпуск (FR-42…FR-46, FR-130; AD-16,
// AD-41). Экраны — эпики 10, 11, 13.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	type itemIn struct {
		ItemID string `path:"item_id" maxLength:"128" doc:"Внутренний ID изделия (AD-16)."`
		httpapi.MomentQuery
	}

	httpapi.Read(api, httpapi.Get("/items/lookup", "Найти изделие по номеру детали или скану DataMatrix",
		"Разрешение носителя (AD-41) → изделие на момент. Не найдено — 404 api.not_found."),
		platform.Action{ID: "item.item.lookup", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct {
			Q string `query:"q" required:"true" minLength:"1" maxLength:"256" doc:"Номер детали или содержимое DataMatrix (ant:carrier:‹тип›:‹значение›)."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.ItemLookup, error) {
			return q.Lookup(ctx, in.Q, m)
		})

	httpapi.Read(api, httpapi.Get("/items", "Изделия",
		"Проваливание в детали (FR-7): изделия узла карты по step_key, по сводному статусу, партии или заданию 1С; изделия прежних версий — с пометкой версии."),
		platform.Action{ID: "item.item.list", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct {
			StepKey string `query:"step_key" maxLength:"128" doc:"Шаг процесса (AD-17)."`
			Summary string `query:"summary" enum:"in_process,suspect,reinspection_required,hold,pending_decision,nonconforming,cleared,released,in_rework,in_repair,accepted_with_concession,scrapped,returned" doc:"Сводный статус."`
			LotID   string `query:"lot_id" maxLength:"128"`
			OrderID string `query:"order_id" maxLength:"128" doc:"Задание 1С."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.ItemList, error) {
			return q.List(ctx, app.ItemFilter{StepKey: in.StepKey, Summary: in.Summary, LotID: in.LotID, OrderID: in.OrderID}, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/items/{item_id}/passport", "Паспорт изделия",
		"FR-42, кейс «история изделия»: факты, решения и документы с автором, временем, подписью и статусом её проверки; пометка источника факта (FR-140); "+
			"исходный сигнал, анализ системы и решение человека — раздельно; оси статуса §3b; зоны, носители, несоответствия, инциденты."),
		platform.Action{ID: "item.passport.read", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *itemIn, m platform.Moment) (app.ItemPassport, error) {
			return q.Passport(ctx, in.ItemID, m)
		})

	httpapi.Read(api, httpapi.Get("/items/{item_id}/history", "Журнал изменений паспорта",
		"FR-43: проекция — было / стало / кто / причина; исправления — новыми записями (FR-122)."),
		platform.Action{ID: "item.history.list", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct {
			ItemID string `path:"item_id" maxLength:"128"`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.ItemHistory, error) {
			return q.History(ctx, in.ItemID, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/items/{item_id}/genealogy", "Генеалогия изделия",
		"FR-45: из чего собрано и куда вошло, партии и выписки партнёров; владелец генеалогии — межизделийная стадия (AD-42), паспорт её показывает."),
		platform.Action{ID: "item.genealogy.read", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *itemIn, m platform.Moment) (app.ItemGenealogy, error) {
			return q.Genealogy(ctx, in.ItemID, m)
		})

	// ── команды ──
	type itemCmd[B any] struct {
		ItemID string `path:"item_id" maxLength:"128"`
		Body   B
	}
	emits := func(t ...catalog.Type) []catalog.Type { return t }

	httpapi.Do(api, httpapi.Post("/items", "Зарегистрировать изделие и запустить в работу",
		"FR-42, AD-16: ID рождается в системе (код_предприятия:локальный_id) и из метки не выводится; закрепляется версия нормативного слоя (AD-17)."),
		platform.Action{ID: "item.item.register", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.ItemItemRegistered), SignatureLevel: 1},
		func(ctx context.Context, in *struct{ Body app.RegisterItem }) (platform.Receipt, error) {
			return c.Register(ctx, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/carriers", "Нанести носитель", "AD-16: бирка с QR, DPM, тара с ячейкой; перемаркировка — replaces_value."),
		platform.Action{ID: "item.carrier.apply", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.ItemCarrierApplied), SignatureLevel: 1},
		func(ctx context.Context, in *itemCmd[app.ApplyCarrier]) (platform.Receipt, error) {
			return c.ApplyCarrier(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/carriers/remove", "Снять носитель", "AD-16: не снятый временный носитель — задача контроля полноты."),
		platform.Action{ID: "item.carrier.remove", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.ItemCarrierRemoved), SignatureLevel: 1},
		func(ctx context.Context, in *itemCmd[app.RemoveCarrier]) (platform.Receipt, error) {
			return c.RemoveCarrier(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/presentations", "Предъявить ОТК",
		"FR-19: мастер предъявляет изделие на точке предъявления; повторное предъявление — подписант уровнем выше."),
		platform.Action{ID: "item.presentation.record", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.ItemPresentationRecorded), SignatureLevel: 1},
		func(ctx context.Context, in *itemCmd[app.RecordPresentation]) (platform.Receipt, error) {
			return c.RecordPresentation(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/interventions", "Открыть вмешательство",
		"FR-21: разборка собранного изделия — мастер открывает, зоны теряют статус проверенных до повторного контроля."),
		platform.Action{ID: "item.intervention.open", Class: platform.ClassProtective, Owner: owner, Subject: "item", Emits: emits(catalog.ItemInterventionOpened), SignatureLevel: 1},
		func(ctx context.Context, in *itemCmd[app.OpenIntervention]) (platform.Receipt, error) {
			return c.OpenIntervention(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/interventions/{intervention_id}/close", "Закрыть вмешательство",
		"FR-21: контролёр закрывает после повторной проверки зоны — разрешающее действие (AD-27), критическое (AD-28)."),
		platform.Action{ID: "item.intervention.close", Class: platform.ClassPermissive, Critical: true, CAGroup: "product_decision", Owner: owner, Subject: "item",
			Emits: emits(catalog.ItemInterventionClosed), SignatureLevel: 2},
		func(ctx context.Context, in *struct {
			ItemID         string `path:"item_id" maxLength:"128"`
			InterventionID string `path:"intervention_id" maxLength:"128"`
			Body           app.CloseIntervention
		}) (platform.Receipt, error) {
			return c.CloseIntervention(ctx, in.ItemID, in.InterventionID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/identification/confirm", "Подтвердить идентификацию",
		"AD-16: после «идентификация под сомнением» — повторная идентификация человеком с подписью; снимает изоляцию по этой причине (разрешающее, критическое)."),
		platform.Action{ID: "item.identification.confirm", Class: platform.ClassPermissive, Critical: true, CAGroup: "protected_data", Owner: owner, Subject: "item",
			Emits: emits(catalog.ItemIdentificationConfirmed), SignatureLevel: 2},
		func(ctx context.Context, in *itemCmd[app.ConfirmIdentification]) (platform.Receipt, error) {
			return c.ConfirmIdentification(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/assembly", "Установить компонент в сборку",
		"FR-45: факт сборки; связь генеалогии пишет межизделийная стадия (genealogy.link.added обоим изделиям, AD-42)."),
		platform.Action{ID: "item.assembly.record", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.ItemAssemblyRecorded), SignatureLevel: 1},
		func(ctx context.Context, in *itemCmd[app.RecordAssembly]) (platform.Receipt, error) {
			return c.RecordAssembly(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/release", "Принять на склад выпуска",
		"Закрывающая точка выпуска: реакция «выпуск» в 1С (с признаком after_rework для принятого после переделки, AD-18)."),
		platform.Action{ID: "item.release.record", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: emits(catalog.ItemReleaseRecorded), SignatureLevel: 1},
		func(ctx context.Context, in *itemCmd[app.RecordRelease]) (platform.Receipt, error) {
			return c.RecordRelease(ctx, in.ItemID, in.Body)
		})
}
