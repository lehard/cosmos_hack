package journal

import (
	"context"

	app "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "journal"

// Register объявляет операции модуля journal: SSE-канал живых обновлений
// (AD-21, FR-2), общий экран «Журнал событий», голова журнала и таймлайн
// живой карты (FR-4). Записей журнал через API не принимает: в журнал пишет
// только journal.Append по командам модулей (AD-44).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_ = c
	httpapi.RegisterStream(api, func(ctx context.Context, after int64, runID string) (func(context.Context) (httpapi.EntityChanged, error), func(), error) {
		sub, err := q.Subscribe(ctx, after, runID)
		if err != nil {
			return nil, nil, err
		}
		next := func(ctx context.Context) (httpapi.EntityChanged, error) {
			ch, err := sub.Next(ctx)
			return httpapi.EntityChanged{Entity: ch.Entity, ID: ch.ID, Seq: ch.Seq, RunID: ch.RunID, Mode: ch.Mode}, err
		}
		return next, sub.Close, nil
	})

	httpapi.Read(api, httpapi.Get("/journal", "Журнал событий",
		"PRD §3a, общий экран: записи журнала по изделию, потоку, типу, виду записи; исходные события, анализ системы, решения людей "+
			"и служебные записи различимы (AD-2, кейс §7.2); пометка источника и статус подписи у каждой записи (FR-140, FR-68)."),
		platform.Action{ID: "journal.entry.list", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct {
			ItemID    string `query:"item_id" maxLength:"128" doc:"Изделие."`
			Stream    string `query:"stream" maxLength:"160" doc:"Поток: item:‹id›, incident:‹id›, global…"`
			EventType string `query:"event_type" maxLength:"128" doc:"Тип записи или префикс семейства (item., quality.)."`
			EntryKind string `query:"entry_kind" enum:"fact,reaction,decision,service" doc:"Вид записи (AD-2)."`
			AfterSeq  int64  `query:"after_seq" minimum:"0" doc:"Записи после seq."`
			EventID   string `query:"event_id" maxLength:"64" doc:"Одна запись по event_id: переход по causation_id, corrects, correlation_id (интерфейс 6)."`
			Order     string `query:"order" enum:"asc,desc" doc:"asc (по умолчанию) — от старых, курсор — seq последней; desc — новые сверху, следующая страница — seq меньше курсора."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.JournalEntryList, error) {
			return q.Entries(ctx, app.EntryFilter{ItemID: in.ItemID, Stream: in.Stream, EventType: in.EventType, AfterSeq: in.AfterSeq, EntryKind: in.EntryKind,
				EventID: in.EventID, Order: in.Order}, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/journal/{seq}", "Запись журнала", "Одна запись по seq: открытые поля (AD-44), подписанты, статус подписи, содержимое."),
		platform.Action{ID: "journal.entry.read", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct {
			Seq int64 `path:"seq" minimum:"1" doc:"Позиция записи в основной цепочке."`
		}, _ platform.Moment) (app.JournalEntryView, error) {
			return q.Entry(ctx, in.Seq)
		})

	httpapi.Read(api, httpapi.Get("/events/{event_id}", "Запись журнала по event_id",
		"Интерфейс 6, Д-70: окно записи ?open=event:‹id› — доказательства ступеней области риска, доводы гипотез, отметки дорожек разбора. "+
			"Запись словами (text, source_label), вид, позиция в журнале (journal_seq), материалы (evidence_refs) и числа режима (reading)."),
		platform.Action{ID: "journal.event.read", Owner: owner, Subject: "item"},
		func(ctx context.Context, in *struct {
			EventID string `path:"event_id" maxLength:"64" doc:"event_id записи."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.JournalEventView, error) {
			return q.Event(ctx, in.EventID, m)
		})

	httpapi.Read(api, httpapi.Get("/journal/head", "Голова журнала", "Последний seq и доменное время записи (AD-37), последний номер CA, режим часов."),
		platform.Action{ID: "journal.head.read", Owner: owner, Subject: "integrity"},
		func(ctx context.Context, _ *struct{ httpapi.MomentQuery }, m platform.Moment) (app.JournalHead, error) {
			return q.Head(ctx, m)
		})

	httpapi.Read(api, httpapi.Get("/timeline", "Таймлайн живой карты",
		"FR-4, FR-155: диапазон доступной истории (или прогона) и метки значимых событий — эскалации, стоп точки процесса, всплески, новые версии процесса; "+
			"ось — из параметра axis."),
		platform.Action{ID: "journal.timeline.read", Owner: owner, Subject: "live_map"},
		func(ctx context.Context, _ *struct{ httpapi.MomentQuery }, m platform.Moment) (app.TimelineData, error) {
			return q.Timeline(ctx, m)
		})
}
