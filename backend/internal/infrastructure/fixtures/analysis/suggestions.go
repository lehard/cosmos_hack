package analysis

import (
	"context"

	app "ant/internal/application/analysis"
	"ant/internal/application/platform"
)

// Предложения, меры и карта дефицита на заготовках (эпик 42, AD-36): ответы
// операций из мира заготовок (scenarios/fixtures); команды двигают сценарий,
// если он ждёт именно этого решения.

var (
	_ app.SuggestionQueries  = (*Adapter)(nil)
	_ app.SuggestionCommands = (*Adapter)(nil)
)

// Suggestions — предложения (analysis.suggestion.list).
func (Adapter) Suggestions(ctx context.Context, m platform.Moment) (app.SuggestionList, error) {
	return respond[app.SuggestionList](ctx, "analysis.suggestion.list", nil, &m)
}

// CorrectiveActions — меры и взгляд руководителя по качеству (analysis.action.list).
func (Adapter) CorrectiveActions(ctx context.Context, m platform.Moment) (app.CorrectiveActionList, error) {
	return respond[app.CorrectiveActionList](ctx, "analysis.action.list", nil, &m)
}

// DataDeficit — карта дефицита данных (analysis.data_deficit.read).
func (Adapter) DataDeficit(ctx context.Context, m platform.Moment) (app.DataDeficitMap, error) {
	return respond[app.DataDeficitMap](ctx, "analysis.data_deficit.read", nil, &m)
}

// GenerateSuggestions — прогон генераторов на заготовках: мир не меняется.
func (Adapter) GenerateSuggestions(ctx context.Context, in app.GenerateSuggestions) (platform.Receipt, error) {
	return decide(ctx, "analysis.suggestion.generate", "suggestion", "all", in.CommandMeta())
}

// ForwardSuggestion — передать предложение (analysis.suggestion.forward).
func (Adapter) ForwardSuggestion(ctx context.Context, id string, in app.ForwardSuggestion) (platform.Receipt, error) {
	return decide(ctx, "analysis.suggestion.forward", "suggestion", id, in.CommandMeta())
}

// ResolveSuggestion — решение по предложению (analysis.suggestion.resolve).
func (Adapter) ResolveSuggestion(ctx context.Context, id string, in app.ResolveSuggestion) (platform.Receipt, error) {
	return decide(ctx, "analysis.suggestion.resolve", "suggestion", id, in.CommandMeta())
}
