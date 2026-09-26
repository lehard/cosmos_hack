package nonconformity

import (
	"context"

	app "ant/internal/application/nonconformity"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "nonconformity"

// Register объявляет операции модуля nonconformity: очередь «Ждут моего
// решения», карточка несоответствия, разрешения на отклонение и решения
// контролёра и уполномоченных (FR-49…FR-56, FR-144; AD-27, AD-30, AD-39,
// AD-43). Экраны — эпик 11; решения — через подтверждение уровня подписи 2
// (AD-13): заключения ОТК и критические действия.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	emits := func(t ...catalog.Type) []catalog.Type { return t }

	// ── чтение ──
	httpapi.Read(api, httpapi.Get("/decision-queue", "Очередь «Ждут моего решения»",
		"PRD §3a «Контролёр качества»: точки предъявления, сигналы на рассмотрение, изолированные изделия со сроком решения (обратный отсчёт); "+
			"сортировка по риску и сроку."),
		platform.Action{ID: "nonconformity.queue.list", Owner: owner, Subject: "nonconformity"},
		func(ctx context.Context, in *struct {
			Kind string `query:"kind" enum:"presentation,signal,isolated" doc:"Вид строки; пусто — все."`
			Sort string `query:"sort" enum:"risk,deadline" doc:"Сортировка: по риску (по умолчанию) или по сроку."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.DecisionQueue, error) {
			return q.Queue(ctx, app.QueueFilter{Kind: in.Kind, Sort: in.Sort}, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/nonconformities", "Несоответствия", "Список несоответствий с фильтром по изделию и статусу (журнал регистрации несоответствий — проекция)."),
		platform.Action{ID: "nonconformity.nonconformity.list", Owner: owner, Subject: "nonconformity"},
		func(ctx context.Context, in *struct {
			ItemID string `query:"item_id" maxLength:"128" doc:"Изделие."`
			Status string `query:"status" enum:"draft,confirmed,disposition_set,verified,closed" doc:"Статус."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.NCList, error) {
			return q.List(ctx, app.NCFilter{ItemID: in.ItemID, Status: in.Status}, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/nonconformities/{nc_id}", "Карточка несоответствия",
		"FR-51: три зоны — «что произошло» (до операции / операция: станок, инструмент, программа, исполнитель / после), «доказательства», «что решить»; "+
			"блок «почему система это предлагает»; раздельно исходный сигнал, анализ системы (все версии вывода с причиной пересмотра) и решения людей; оси статуса изделия."),
		platform.Action{ID: "nonconformity.card.read", Owner: owner, Subject: "nonconformity"},
		func(ctx context.Context, in *struct {
			NCID string `path:"nc_id" maxLength:"128" doc:"Несоответствие."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.NCCard, error) {
			return q.Card(ctx, in.NCID, m)
		})

	httpapi.Read(api, httpapi.Get("/concessions", "Разрешения на отклонение",
		"FR-54: действующие разрешения на отклонение, применимые к изделию, с лимитом и остатком — для выбора при решении «ремонт» или «как есть»."),
		platform.Action{ID: "nonconformity.concession.list", Owner: owner, Subject: "nonconformity"},
		func(ctx context.Context, in *struct {
			ItemID string `query:"item_id" maxLength:"128" doc:"Изделие; пусто — все действующие."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.ConcessionList, error) {
			return q.Concessions(ctx, in.ItemID, m)
		})

	// ── команды по несоответствию ──
	type ncCmd[B any] struct {
		NCID string `path:"nc_id" maxLength:"128"`
		Body B
	}
	httpapi.Do(api, httpapi.Post("/nonconformities/{nc_id}/confirm", "Подтвердить несоответствие",
		"FR-52: контролёр подтверждает несоответствие по сигналам — защитное действие, критическое (AD-28), подпись уровня 2."),
		platform.Action{ID: "nonconformity.nonconformity.confirm", Class: platform.ClassProtective, Critical: true, CAGroup: "product_decision", Owner: owner,
			Subject: "nonconformity", Emits: emits(catalog.DecisionNonconformityConfirmed), SignatureLevel: 2},
		func(ctx context.Context, in *ncCmd[app.ConfirmNonconformity]) (platform.Receipt, error) {
			return c.Confirm(ctx, in.NCID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/nonconformities/{nc_id}/disposition", "Решение по несоответствию",
		"FR-53: переделка / ремонт / как есть / списать / вернуть поставщику — необратимое решение уполномоченного по маршруту (AD-27, AD-43); "+
			"ремонт и «как есть» — только с действующим разрешением на отклонение (FR-54, nonconformity.concession_required)."),
		platform.Action{ID: "nonconformity.disposition.set", Class: platform.ClassIrreversible, Critical: true, CAGroup: "nc_decision", Owner: owner,
			Subject: "nonconformity", Guards: []string{"concession_required", "approvals_missing", "separation_of_duties"},
			Emits: emits(catalog.DecisionDispositionSet), SignatureLevel: 2},
		func(ctx context.Context, in *ncCmd[app.SetDisposition]) (platform.Receipt, error) {
			return c.SetDisposition(ctx, in.NCID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/nonconformities/{nc_id}/disposition/verify", "Подтвердить выполнение решения",
		"Повторная проверка после переделки или ремонта подтверждает выполнение решения — разрешающее действие."),
		platform.Action{ID: "nonconformity.disposition.verify", Class: platform.ClassPermissive, Critical: true, CAGroup: "product_decision", Owner: owner,
			Subject: "nonconformity", Emits: emits(catalog.DecisionDispositionVerified), SignatureLevel: 2},
		func(ctx context.Context, in *ncCmd[app.VerifyDisposition]) (platform.Receipt, error) {
			return c.VerifyDisposition(ctx, in.NCID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/nonconformities/{nc_id}/close", "Закрыть несоответствие", "Итог несоответствия после выполнения решения."),
		platform.Action{ID: "nonconformity.nonconformity.close", Class: platform.ClassRecord, Owner: owner, Subject: "nonconformity",
			Emits: emits(catalog.DecisionNonconformityClosed)},
		func(ctx context.Context, in *ncCmd[app.CloseNonconformity]) (platform.Receipt, error) {
			return c.Close(ctx, in.NCID, in.Body)
		})

	// ── команды по изделию ──
	type itemCmd[B any] struct {
		ItemID string `path:"item_id" maxLength:"128"`
		Body   B
	}
	httpapi.Do(api, httpapi.Post("/items/{item_id}/signals/reject", "Отклонить сигнал",
		"FR-52: отклонение сигнала с обязательной причиной — разрешающее действие, критическое (AD-27, AD-28); можно пометить для контура адаптации."),
		platform.Action{ID: "nonconformity.signal.reject", Class: platform.ClassPermissive, Critical: true, CAGroup: "product_decision", Owner: owner,
			Subject: "item", Guards: []string{"reject_reason_required"}, Emits: emits(catalog.DecisionSignalRejected), SignatureLevel: 2},
		func(ctx context.Context, in *itemCmd[app.RejectSignal]) (platform.Receipt, error) {
			return c.RejectSignal(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/recheck", "Назначить доп. проверку", "FR-52: дополнительная проверка методом контроля — защитное действие."),
		platform.Action{ID: "nonconformity.recheck.request", Class: platform.ClassProtective, Owner: owner, Subject: "item",
			Emits: emits(catalog.DecisionRecheckRequested)},
		func(ctx context.Context, in *itemCmd[app.RequestRecheck]) (platform.Receipt, error) {
			return c.RequestRecheck(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/isolate", "Изолировать",
		"FR-55: изделие — в изоляцию со сроком решения; «изоляция» — положение, ожидающее решения (AD-30); защитное действие, критическое."),
		platform.Action{ID: "nonconformity.item.isolate", Class: platform.ClassProtective, Critical: true, CAGroup: "product_decision", Owner: owner,
			Subject: "item", Emits: emits(catalog.DecisionItemIsolated), SignatureLevel: 2},
		func(ctx context.Context, in *itemCmd[app.IsolateItem]) (platform.Receipt, error) {
			return c.Isolate(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/presentations/resolve", "Решение на точке предъявления",
		"FR-19, FR-56: «Принять — передать на ‹следующий шаг›», принять по разрешению, вернуть, недостаточно данных; участник изготовления не принимает "+
			"(разделение обязанностей); без результатов обязательных методов — отказ (nonconformity.method_result_missing)."),
		platform.Action{ID: "nonconformity.presentation.resolve", Class: platform.ClassPermissive, Critical: true, CAGroup: "product_decision", Owner: owner,
			Subject: "item", Guards: []string{"separation_of_duties", "method_result_missing", "item_blocked", "intervention_open"},
			Emits: emits(catalog.DecisionPresentationResolved), SignatureLevel: 2},
		func(ctx context.Context, in *itemCmd[app.ResolvePresentation]) (platform.Receipt, error) {
			return c.ResolvePresentation(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/containment", "Установить сдерживание",
		"FR-49: наблюдать / доп. проверка / блок изделия или партии — защитное действие, критическое."),
		platform.Action{ID: "nonconformity.containment.set", Class: platform.ClassProtective, Critical: true, CAGroup: "product_decision", Owner: owner,
			Subject: "item", Emits: emits(catalog.DecisionContainmentSet), SignatureLevel: 2},
		func(ctx context.Context, in *itemCmd[app.SetContainment]) (platform.Receipt, error) {
			return c.SetContainment(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/containment/release", "Снять сдерживание",
		"FR-49, AD-27: снятие блока — разрешающее действие только уполномоченного; снятие блока ≠ годность; при активном инциденте — отказ гарда."),
		platform.Action{ID: "nonconformity.containment.release", Class: platform.ClassPermissive, Critical: true, CAGroup: "product_decision", Owner: owner,
			Subject: "item", Guards: []string{"item_blocked"}, Emits: emits(catalog.DecisionContainmentReleased), SignatureLevel: 2},
		func(ctx context.Context, in *itemCmd[app.ReleaseContainment]) (platform.Receipt, error) {
			return c.ReleaseContainment(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/items/{item_id}/rework-limit/waive", "Разрешить сверх лимита доработок",
		"FR-18: разрешение на доработку сверх лимита по зоне — полномочие «разрешение сверх лимита доработок»."),
		platform.Action{ID: "nonconformity.rework_limit.waive", Class: platform.ClassPermissive, Critical: true, CAGroup: "nc_decision", Owner: owner,
			Subject: "item", Emits: emits(catalog.DecisionReworkLimitWaived), SignatureLevel: 2},
		func(ctx context.Context, in *itemCmd[app.WaiveReworkLimit]) (platform.Receipt, error) {
			return c.WaiveReworkLimit(ctx, in.ItemID, in.Body)
		})

	// ── партия, разрешение на отклонение, точка процесса ──
	httpapi.Do(api, httpapi.Post("/lots/{lot_id}/resolve", "Решение по партии",
		"Входной контроль (ЗТ-1): принять, принять частично, отклонить, недостаточно данных — по результатам методов."),
		platform.Action{ID: "nonconformity.lot.resolve", Class: platform.ClassPermissive, Critical: true, CAGroup: "product_decision", Owner: owner,
			Subject: "lot", Emits: emits(catalog.DecisionLotResolved), SignatureLevel: 2},
		func(ctx context.Context, in *struct {
			LotID string `path:"lot_id" maxLength:"128"`
			Body  app.ResolveLot
		}) (platform.Receipt, error) {
			return c.ResolveLot(ctx, in.LotID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/concessions", "Выдать разрешение на отклонение",
		"FR-54, Д-24: номер, пункт КД/ТУ, область действия, лимит количества, срок; лимит открывается атомарно с записью (AD-39). "+
			"Внешние полномочия (режим 5) — маршрут подписей документа разрешения."),
		platform.Action{ID: "nonconformity.concession.grant", Class: platform.ClassPermissive, Critical: true, CAGroup: "nc_decision", Owner: owner,
			Subject: "nonconformity", Emits: emits(catalog.DecisionConcessionGranted), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.GrantConcession }) (platform.Receipt, error) {
			return c.GrantConcession(ctx, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/concessions/{concession_id}/revoke", "Отозвать разрешение на отклонение",
		"FR-54: отзыв разрешения — защитное действие; расход лимита прекращается."),
		platform.Action{ID: "nonconformity.concession.revoke", Class: platform.ClassProtective, Critical: true, CAGroup: "nc_decision", Owner: owner,
			Subject: "nonconformity", Emits: emits(catalog.DecisionConcessionRevoked), SignatureLevel: 2},
		func(ctx context.Context, in *struct {
			ConcessionID string `path:"concession_id" maxLength:"128"`
			Body         app.RevokeConcession
		}) (platform.Receipt, error) {
			return c.RevokeConcession(ctx, in.ConcessionID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/process-holds", "Остановить точку процесса",
		"FR-49: стоп точки процесса или критическая остановка (сдерживание процесса — отдельный объект от сдерживания изделий)."),
		platform.Action{ID: "nonconformity.process_hold.set", Class: platform.ClassProtective, Critical: true, CAGroup: "product_decision", Owner: owner,
			Subject: "equipment", Emits: emits(catalog.DecisionProcessHoldSet), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.SetProcessHold }) (platform.Receipt, error) {
			return c.SetProcessHold(ctx, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/process-holds/{hold_id}/release", "Снять остановку точки процесса",
		"FR-49: снятие остановки — разрешающее действие уполномоченного, с числом изделий после точки чистоты."),
		platform.Action{ID: "nonconformity.process_hold.release", Class: platform.ClassPermissive, Critical: true, CAGroup: "product_decision", Owner: owner,
			Subject: "equipment", Emits: emits(catalog.DecisionProcessHoldReleased), SignatureLevel: 2},
		func(ctx context.Context, in *struct {
			HoldID string `path:"hold_id" maxLength:"128"`
			Body   app.ReleaseProcessHold
		}) (platform.Receipt, error) {
			return c.ReleaseProcessHold(ctx, in.HoldID, in.Body)
		})
}
