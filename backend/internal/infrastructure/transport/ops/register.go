package ops

import (
	"context"

	app "ant/internal/application/ops"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "ops"

// Register объявляет операции модуля ops: состояние компонентов, остановленные
// изделия и повтор их обработки, отключение и включение источников, настройки
// адаптеров (FR-109, FR-127; AD-25, AD-35, AD-45).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	type sourceCmd struct {
		SourceID string `path:"source_id" maxLength:"128" doc:"Источник событий."`
		Body     app.SwitchSource
	}

	httpapi.Read(api, httpapi.Get("/ops/health", "Состояние системы",
		"FR-127: сервисы и роли, очереди и отставание потребителей, карантин, интеграции, остановленные изделия, последний отчёт верификатора «по данным сервера»."),
		platform.Action{ID: "ops.health.read", Owner: owner, Subject: "integrity"},
		func(ctx context.Context, _ *struct{}, _ platform.Moment) (app.OpsHealth, error) {
			return q.Health(ctx)
		})

	httpapi.Read(api, httpapi.Get("/ops/stopped-items", "Остановленные изделия",
		"AD-45: ошибка свёртки или проекции на записи → ops.processing.failed; изделие «обработка остановлена», партиция продолжает."),
		platform.Action{ID: "ops.stopped_item.list", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct{ httpapi.PageQuery }, _ platform.Moment) (app.StoppedItemList, error) {
			return q.StoppedItems(ctx, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/ops/settings", "Настройки адаптеров",
		"AD-35, AD-36: адаптер каждого ведомого порта по ключу конфигурации, режим fixtures | live по модулям, включённые внешние системы."),
		platform.Action{ID: "ops.setting.list", Owner: owner},
		func(ctx context.Context, _ *struct{}, _ platform.Moment) (app.SettingList, error) {
			return q.Settings(ctx)
		})

	httpapi.Do(api, httpapi.Post("/ops/stopped-items/{item_id}/retry", "Повторить обработку изделия", "AD-45: повтор свёртки изделия (как ant rebuild --item)."),
		platform.Action{ID: "ops.processing.retry", Class: platform.ClassRecord, Owner: owner, Subject: "item", Emits: []catalog.Type{catalog.OpsProcessingRetried}},
		func(ctx context.Context, in *struct {
			ItemID string `path:"item_id" maxLength:"128"`
			Body   app.RetryProcessing
		}) (platform.Receipt, error) {
			return c.RetryProcessing(ctx, in.ItemID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/ops/sources/{source_id}/disable", "Отключить источник",
		"AD-28: отключение источника — критическое действие администратора (группа admin_security); защитное."),
		platform.Action{ID: "ops.source.disable", Class: platform.ClassProtective, Critical: true, CAGroup: "admin_security", Owner: owner, Subject: "quarantine",
			Emits: []catalog.Type{catalog.OpsSourceDisabled}, SignatureLevel: 2},
		func(ctx context.Context, in *sourceCmd) (platform.Receipt, error) {
			return c.DisableSource(ctx, in.SourceID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/ops/sources/{source_id}/enable", "Включить источник",
		"AD-28: включение источника — критическое разрешающее действие администратора."),
		platform.Action{ID: "ops.source.enable", Class: platform.ClassPermissive, Critical: true, CAGroup: "admin_security", Owner: owner, Subject: "quarantine",
			Emits: []catalog.Type{catalog.OpsSourceEnabled}, SignatureLevel: 2},
		func(ctx context.Context, in *sourceCmd) (platform.Receipt, error) {
			return c.EnableSource(ctx, in.SourceID, in.Body)
		})
}
