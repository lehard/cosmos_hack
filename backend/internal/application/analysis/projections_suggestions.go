package analysis

import (
	"encoding/json"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	dom "ant/internal/domain/analysis"
	"ant/internal/domain/kernel"
)

// Проекции предложений и мер (эпик 42, AD-45: писатель — analysis; ведёт роль
// projector, пересобирает `ant rebuild`):
//   - analysis.suggestion — предложение по suggestion_id с историей решений;
//   - analysis.action — корректирующая мера по action_id с планом проверки
//     эффективности и историей оценок (FR-64).
// Ключ dom.ListKey — список id в порядке появления.

// registerSuggestions подключает проекции предложений и мер к реестру движка
// (вызывается из Register).
func registerSuggestions(reg *engineapp.Registry) error {
	if err := reg.AddGlobal(engineapp.GlobalProjection{Name: ProjectionSuggestion, Writer: dom.Module, Keys: dom.SuggestionKeys,
		Step: stepSuggestion, Entity: noEntity}); err != nil {
		return err
	}
	return reg.AddGlobal(engineapp.GlobalProjection{Name: ProjectionAction, Writer: dom.Module, Keys: dom.ActionKeys,
		Step: stepAction, Entity: noEntity})
}

func stepSuggestion(key string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	if key == dom.ListKey {
		return stepOwnList(prev, dom.SuggestionListID(r))
	}
	var v dom.SuggestionRecord
	if err := unmarshal(prev, &v); err != nil {
		return nil, err
	}
	return json.Marshal(dom.StepSuggestion(v, r))
}

func stepAction(key string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	if key == dom.ListKey {
		return stepOwnList(prev, dom.ActionListID(r))
	}
	var v dom.CorrectiveAction
	if err := unmarshal(prev, &v); err != nil {
		return nil, err
	}
	return json.Marshal(dom.StepAction(v, r))
}

func stepOwnList(prev json.RawMessage, id string) (json.RawMessage, error) {
	var l dom.ListRecord
	if err := unmarshal(prev, &l); err != nil {
		return nil, err
	}
	l = dom.StepList(l, id)
	if l.IDs == nil {
		l.IDs = []string{}
	}
	return json.Marshal(l)
}

// noEntity — предложения и меры не сообщаются по SSE отдельной сущностью:
// экран перечитывает их после команды.
func noEntity(string) (platform.EntityKind, string, bool) { return "", "", false }
