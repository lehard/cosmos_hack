package analysis

import (
	"context"

	app "ant/internal/application/analysis"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

// registerSuggestions объявляет операции эпика 42: предложения (FR-63),
// корректирующие меры и взгляд руководителя по качеству (FR-64, FR-138),
// карту дефицита данных (FR-143). Реализация, не знающая этих портов
// (старый адаптер), отвечает 501.
func registerSuggestions(api *httpapi.API, q app.Queries, c app.Commands) {
	sq, ok := q.(app.SuggestionQueries)
	if !ok {
		sq = app.UnimplementedSuggestions{}
	}
	sc, ok := c.(app.SuggestionCommands)
	if !ok {
		sc = app.UnimplementedSuggestions{}
	}
	type moment struct{ httpapi.MomentQuery }

	httpapi.Read(api, httpapi.Get("/suggestions", "Предложения",
		"FR-63: предложения генераторов (ограничение линии, область риска, кандидаты в правила реакции, адаптация VisionQC, карта дефицита данных) "+
			"с основаниями, ответственным и историей решений; подключённые генераторы. Ничего не применяется автоматически."),
		platform.Action{ID: "analysis.suggestion.list", Owner: owner, Subject: "suggestion"},
		func(ctx context.Context, _ *moment, m platform.Moment) (app.SuggestionList, error) {
			return sq.Suggestions(ctx, m)
		})

	httpapi.Read(api, httpapi.Get("/corrective-actions", "Корректирующие меры и взгляд руководителя по качеству",
		"FR-64, FR-138: меры с планом проверки эффективности, статус («внедрено» ≠ «эффективно»), флаги — просрочена, не помогла, висит временно усиленный контроль, "+
			"пора оценить; повторяющиеся проблемы; организационная память — что пробовали и с каким результатом."),
		platform.Action{ID: "analysis.action.list", Owner: owner, Subject: "incident"},
		func(ctx context.Context, _ *moment, m platform.Moment) (app.CorrectiveActionList, error) {
			return sq.CorrectiveActions(ctx, m)
		})

	httpapi.Read(api, httpapi.Get("/analysis/data-deficit", "Карта дефицита данных",
		"FR-143: по каждому виду недостающих сведений разбора (FR-58) — в скольких расследованиях не хватало, где, какая цифровизация закрыла бы пробел "+
			"и оценка сужения области риска по уже сделанным сужениям с основаниями; нет оценки — так и сказано."),
		platform.Action{ID: "analysis.data_deficit.read", Owner: owner, Subject: "incident"},
		func(ctx context.Context, _ *moment, m platform.Moment) (app.DataDeficitMap, error) {
			return sq.DataDeficit(ctx, m)
		})

	emits := func(t ...catalog.Type) []catalog.Type { return t }

	httpapi.Do(api, httpapi.Post("/suggestions/generate", "Сформировать предложения",
		"FR-63: прогнать подключённые генераторы; новые предложения записываются фактами incident.suggestion.recorded, уже записанные не повторяются. "+
			"Квитанция перечисляет event_id новых предложений."),
		platform.Action{ID: "analysis.suggestion.generate", Class: platform.ClassRecord, Owner: owner, Subject: "suggestion",
			Emits: emits(catalog.IncidentSuggestionRecorded)},
		func(ctx context.Context, in *struct{ Body app.GenerateSuggestions }) (platform.Receipt, error) {
			return sc.GenerateSuggestions(ctx, in.Body)
		})

	type suggestionCmd[B any] struct {
		SuggestionID string `path:"suggestion_id" maxLength:"128"`
		Body         B
	}
	httpapi.Do(api, httpapi.Post("/suggestions/{suggestion_id}/forward", "Передать предложение ответственному",
		"UJ-1: руководитель передаёт предложение мастеру участка (по персоналу) или технологу (если нужна новая версия процесса); задачу ставит notifications."),
		platform.Action{ID: "analysis.suggestion.forward", Class: platform.ClassRecord, Owner: owner, Subject: "suggestion",
			Emits: emits(catalog.IncidentSuggestionForwarded)},
		func(ctx context.Context, in *suggestionCmd[app.ForwardSuggestion]) (platform.Receipt, error) {
			return sc.ForwardSuggestion(ctx, in.SuggestionID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/suggestions/{suggestion_id}/resolve", "Решение по предложению",
		"FR-63: принять в работу или отклонить — с основанием. «Принято» ничего не применяет: норма — новая версия процесса через кворум, мера — назначением меры."),
		platform.Action{ID: "analysis.suggestion.resolve", Class: platform.ClassRecord, Owner: owner, Subject: "suggestion",
			Emits: emits(catalog.IncidentSuggestionResolved)},
		func(ctx context.Context, in *suggestionCmd[app.ResolveSuggestion]) (platform.Receipt, error) {
			return sc.ResolveSuggestion(ctx, in.SuggestionID, in.Body)
		})
}
