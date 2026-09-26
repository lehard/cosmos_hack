package analysis

import (
	"context"

	dom "ant/internal/domain/analysis"
)

// Гарды команд корректирующих мер (эпик 42, FR-64) над проекцией
// analysis.action: план проверки эффективности обязателен; «внедрено» —
// у назначенной или переоткрытой меры; «эффективно» — только после окна
// наблюдения.

// planOf — план проверки эффективности из тела команды: проверка полноты
// (GuardPlan) и нормализованный вид для записи.
func planOf(in EffectivenessPlanInput) (map[string]any, error) {
	p := dom.EffectivenessPlan{Metric: in.Metric, Baseline: in.Baseline, WindowDays: in.WindowDays, SuccessCriterion: in.SuccessCriterion, EnhancedControl: in.EnhancedControl}
	if err := dom.GuardPlan(p); err != nil {
		return nil, err
	}
	out := map[string]any{"metric": p.Metric, "baseline": p.Baseline, "window_days": p.WindowDays, "success_criterion": p.SuccessCriterion}
	if p.EnhancedControl != "" {
		out["enhanced_control"] = p.EnhancedControl
	}
	return out, nil
}

// addActionExtras — необязательные поля меры (заголовок, предложение).
func addActionExtras(data map[string]any, in AssignAction) {
	if in.Title != "" {
		data["title"] = trim(in.Title, 256)
	}
	if in.SuggestionID != "" {
		data["suggestion_id"] = in.SuggestionID
	}
}

// guardAction — гард внедрения и оценки над мерой из проекции analysis.action.
// Меры нет в проекции (проекция отстаёт) — решает гард инцидента, как раньше.
func (s *Service) guardAction(ctx context.Context, actionID, op, result string) error {
	a, err := s.action(ctx, actionID)
	if err != nil {
		return nil
	}
	switch op {
	case "implement":
		return refusal(dom.GuardImplement(a))
	case "evaluate":
		now, err := s.now(ctx)
		if err != nil {
			return err
		}
		return refusal(dom.GuardEvaluate(a, result, now))
	}
	return nil
}
