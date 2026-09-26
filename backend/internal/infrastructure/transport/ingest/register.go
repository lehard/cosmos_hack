package ingest

import (
	"context"

	app "ant/internal/application/ingest"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "ingest"

// Register объявляет операции модуля ingest: приём пачек подписанных событий,
// ручной ввод, импорт CSV / Excel, карантин, источники и метрики приёма
// (FR-26…FR-41, FR-123, FR-140, FR-141; AD-7, AD-20, AD-41).
//
// Операции приёма объявлены без метаданных команды человека (Route.NoCommandMeta):
// тело — подписанные факты источника, у каждого свой event_id и source_seq;
// идемпотентность — по source_id + event_id и отпечатку payload (AD-7), а не по
// command_id, и basis_seq у факта нет: поздний факт принимается и даёт реакцию
// AD-5, а не 409. Эмитент самих фактов — модуль-владелец семейства (каталог);
// приём пишет от своего имени только записи ingest.* (карантин, флаги, импорт).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	ingestEmits := []catalog.Type{catalog.IngestMessageQuarantined, catalog.IngestAnomalyFlagged}

	r := httpapi.Post("/ingest/batches", "Принять пачку событий",
		"FR-26…FR-31, AD-46: подписанные конверты DSSE от edge-агента, шлюзов, терминалов. Сырое тело проверяется теми же JSON Schema (AD-20): "+
			"неизвестная версия и нет обязательного поля — карантин с кодом; неизвестное значение перечисления — UNKNOWN с флагом (критичное — карантин); "+
			"повтор — дубль без изменения показателей; другой payload с тем же ключом — конфликт целостности. Ответ — итог по каждому конверту.")
	r.NoCommandMeta = true
	httpapi.Register(api, r,
		platform.Action{ID: "ingest.batch.submit", Class: platform.ClassRecord, Owner: owner, Subject: "quarantine", Emits: ingestEmits},
		func(ctx context.Context, in *struct{ Body app.IngestBatch }) (*httpapi.Out[app.IngestResult], error) {
			v, err := c.SubmitBatch(ctx, in.Body)
			if err != nil {
				return nil, err
			}
			return httpapi.OK(v), nil
		})

	r = httpapi.Post("/ingest/events", "Принять событие ручного ввода",
		"FR-137, FR-140, FR-141: одно событие терминала участка или ручного ввода — такой же источник с проверкой входов, дублей и привязки; пометка «ручной ввод» видна в интерфейсе.")
	r.NoCommandMeta = true
	httpapi.Register(api, r,
		platform.Action{ID: "ingest.event.submit", Class: platform.ClassRecord, Owner: owner, Subject: "quarantine", Emits: ingestEmits},
		func(ctx context.Context, in *struct{ Body app.ManualEvent }) (*httpapi.Out[app.IngestOutcome], error) {
			v, err := c.SubmitEvent(ctx, in.Body)
			if err != nil {
				return nil, err
			}
			return httpapi.OK(v), nil
		})

	r = httpapi.Post("/ingest/imports", "Импорт CSV / Excel",
		"FR-141: файл сохраняется в хранилище материалов по адресу H(байты); строки проходят те же проверки, что события устройств; итог — ingest.import.completed. "+
			"Идемпотентность — по отпечатку файла и строк.")
	r.NoCommandMeta = true
	httpapi.Register(api, r,
		platform.Action{ID: "ingest.import.submit", Class: platform.ClassRecord, Owner: owner, Subject: "quarantine",
			Emits: []catalog.Type{catalog.IngestImportCompleted, catalog.IngestMessageQuarantined, catalog.IngestAnomalyFlagged}},
		func(ctx context.Context, in *struct{ Body app.ImportFile }) (*httpapi.Out[app.ImportResult], error) {
			v, err := c.Import(ctx, in.Body)
			if err != nil {
				return nil, err
			}
			return httpapi.OK(v), nil
		})

	httpapi.Read(api, httpapi.Get("/quarantine", "Карантин сообщений",
		"FR-30: содержимое хранится вне журнала, факт помещения — служебная запись ingest.message.quarantined с кодом причины."),
		platform.Action{ID: "ingest.quarantine.list", Owner: owner, Subject: "quarantine"},
		func(ctx context.Context, in *struct {
			SourceID string `query:"source_id" maxLength:"128" doc:"Источник."`
			Code     string `query:"code" maxLength:"128" doc:"Код причины."`
			State    string `query:"state" enum:"open,accepted,still_invalid,discarded" doc:"Состояние."`
			httpapi.PageQuery
		}, _ platform.Moment) (app.QuarantineList, error) {
			return q.Quarantine(ctx, app.QuarantineFilter{SourceID: in.SourceID, Code: in.Code, State: in.State}, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/quarantine/{quarantine_id}", "Сообщение в карантине", "Исходное содержимое, код причины и история переобработки."),
		platform.Action{ID: "ingest.quarantine.read", Owner: owner, Subject: "quarantine"},
		func(ctx context.Context, in *struct {
			QuarantineID string `path:"quarantine_id" maxLength:"128"`
		}, _ platform.Moment) (app.QuarantineEntry, error) {
			return q.QuarantineEntry(ctx, in.QuarantineID)
		})

	httpapi.Do(api, httpapi.Post("/quarantine/{quarantine_id}/reprocess", "Переобработать из карантина",
		"FR-29, AD-20: после появления повышателя или исправления — принять, оставить или отбросить; итог — ingest.message.reprocessed."),
		platform.Action{ID: "ingest.message.reprocess", Class: platform.ClassRecord, Owner: owner, Subject: "quarantine", Emits: []catalog.Type{catalog.IngestMessageReprocessed}},
		func(ctx context.Context, in *struct {
			QuarantineID string `path:"quarantine_id" maxLength:"128"`
			Body         app.ReprocessMessage
		}) (platform.Receipt, error) {
			return c.Reprocess(ctx, in.QuarantineID, in.Body)
		})

	httpapi.Read(api, httpapi.Get("/sources", "Источники событий",
		"AD-7, AD-9: устройства, шлюзы, терминалы — ключ, состояние, последний source_seq, дыры последовательности, расхождение часов, объём карантина."),
		platform.Action{ID: "ingest.source.list", Owner: owner, Subject: "quarantine"},
		func(ctx context.Context, in *struct{ httpapi.PageQuery }, _ platform.Moment) (app.SourceList, error) {
			return q.Sources(ctx, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/ingest/metrics", "Метрики приёма",
		"FR-41: задержка, дубли, отказы, объём карантина, полнота; «событие → экран» (FR-2). Операционные метрики, не проекции (AD-7)."),
		platform.Action{ID: "ingest.metrics.read", Owner: owner, Subject: "quarantine"},
		func(ctx context.Context, _ *struct{}, _ platform.Moment) (app.IngestMetrics, error) {
			return q.Metrics(ctx)
		})
}
