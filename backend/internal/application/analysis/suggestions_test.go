package analysis_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	appanalysis "ant/internal/application/analysis"
	"ant/internal/application/analysis/analysistest"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/analysis"
)

// pluginGenerator — генератор-адаптер «со стороны»: подключается
// RegisterGenerator без правки ядра (FR-63, «новый генератор без изменения ядра»).
type pluginGenerator struct{}

func (pluginGenerator) ID() string   { return "plugin.test" }
func (pluginGenerator) Kind() string { return "other" }
func (pluginGenerator) Generate(_ context.Context, in appanalysis.GeneratorInput) ([]dom.Proposal, error) {
	if len(in.Incidents) == 0 {
		return nil, nil
	}
	return []dom.Proposal{{Title: "Проверка подключения генератора", Statement: "Инцидентов: " + in.Incidents[0].IncidentID,
		ResponsibleRole: dom.RoleProductionManager, DedupKey: "plugin|" + in.Incidents[0].IncidentID, Basis: []string{}}}, nil
}

func errCode(err error) errcodes.Code {
	var pe *platform.Error
	if errors.As(err, &pe) {
		return pe.Code
	}
	if pe, ok := platform.AsError(err); ok {
		return pe.Code
	}
	return ""
}

// Предложения (FR-63): встроенные генераторы и подключённый адаптер пишут
// факты с основаниями; повторный прогон ничего не дублирует; руководитель
// передаёт предложение — задача ответственному; решение — с основанием и
// только один раз.
func TestSuggestionsOnFakes(t *testing.T) {
	appanalysis.RegisterGenerator(pluginGenerator{})
	w, j := memWorld(t)
	ctx := context.Background()
	if err := w.Story(ctx); err != nil {
		t.Fatal(err)
	}
	if err := w.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	svc := w.Service(j, analysistest.At(23, 11, 45))
	pm := platform.WithPrincipal(ctx, platform.Principal{PersonID: "PM-01", Role: "production_manager"})
	m := platform.Moment{Axis: platform.AxisOccurred}

	rc, err := svc.GenerateSuggestions(pm, appanalysis.GenerateSuggestions{CommandHeader: platform.CommandHeader{CommandID: w.ID("command")}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rc.EventIDs) < 2 {
		t.Fatalf("новых предложений %d, ждали хотя бы область риска и адаптер", len(rc.EventIDs))
	}
	if err := w.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	list, err := svc.Suggestions(pm, m)
	if err != nil {
		t.Fatal(err)
	}
	var risk, plugin *appanalysis.Suggestion
	for i, s := range list.Items {
		if s.Status != "new" || len(s.History) != 1 {
			t.Fatalf("новое предложение: %+v", s)
		}
		switch s.Generator {
		case "rules.risk_scope":
			risk = &list.Items[i]
		case "plugin.test":
			plugin = &list.Items[i]
		}
	}
	if risk == nil || plugin == nil {
		t.Fatalf("нет предложений области риска или адаптера: %+v", list.Items)
	}
	if len(risk.Basis) == 0 || risk.IncidentID == nil || !strings.Contains(risk.Statement, "34") {
		t.Fatalf("предложение области риска без оснований: %+v", risk)
	}
	var gens []string
	for _, g := range list.Generators {
		gens = append(gens, g.ID)
		if g.ID == "rules.bottleneck" && g.Connected {
			t.Fatal("ограничение линии без порта analytics — не подключено")
		}
	}
	if !slices.Contains(gens, "plugin.test") || !slices.Contains(gens, "rules.data_deficit") {
		t.Fatalf("генераторы: %v", gens)
	}

	// Повторный прогон над тем же состоянием — ничего нового (ключ повторения).
	rc2, err := svc.GenerateSuggestions(pm, appanalysis.GenerateSuggestions{CommandHeader: platform.CommandHeader{CommandID: w.ID("command")}})
	if err != nil || len(rc2.EventIDs) != 0 {
		t.Fatalf("повторный прогон: %+v, %v", rc2, err)
	}

	// Передать ответственному → задача (notifications) и статус «передано».
	if _, err := svc.ForwardSuggestion(pm, risk.SuggestionID, appanalysis.ForwardSuggestion{
		CommandHeader: platform.CommandHeader{CommandID: w.ID("command"), BasisSeq: risk.BasisSeq}, ResponsibleID: "TEC-01", ResponsibleRole: "technologist"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	list, _ = svc.Suggestions(pm, m)
	got := find(list, risk.SuggestionID)
	if got.Status != "forwarded" || got.ResponsibleID == nil || *got.ResponsibleID != "TEC-01" {
		t.Fatalf("после передачи: %+v", got)
	}
	// Решение без основания — отказ; с основанием — принято; второе решение — отказ гарда.
	if _, err := svc.ResolveSuggestion(pm, risk.SuggestionID, appanalysis.ResolveSuggestion{
		CommandHeader: platform.CommandHeader{CommandID: w.ID("command"), BasisSeq: got.BasisSeq}, Resolution: "accepted"}); errCode(err) != errcodes.ApiValidationFailed {
		t.Fatalf("решение без основания: %v", err)
	}
	if _, err := svc.ResolveSuggestion(pm, risk.SuggestionID, appanalysis.ResolveSuggestion{
		CommandHeader: platform.CommandHeader{CommandID: w.ID("command"), BasisSeq: got.BasisSeq}, Resolution: "accepted",
		Reason: appanalysis.Reason{Text: "Сужаем по журналам источников"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	list, _ = svc.Suggestions(pm, m)
	got = find(list, risk.SuggestionID)
	if got.Status != "accepted" || len(got.History) != 3 {
		t.Fatalf("после решения: %+v", got)
	}
	if _, err := svc.ResolveSuggestion(pm, risk.SuggestionID, appanalysis.ResolveSuggestion{
		CommandHeader: platform.CommandHeader{CommandID: w.ID("command"), BasisSeq: got.BasisSeq}, Resolution: "rejected",
		Reason: appanalysis.Reason{Text: "передумали"}}); errCode(err) != errcodes.IncidentSuggestionState {
		t.Fatalf("повторное решение: %v", err)
	}
}

func find(l appanalysis.SuggestionList, id string) appanalysis.Suggestion {
	for _, s := range l.Items {
		if s.SuggestionID == id {
			return s
		}
	}
	return appanalysis.Suggestion{}
}

// Меры (FR-64, FR-138): без плана эффективности мера не создаётся;
// «исполнено» ≠ «эффективно» — «эффективно» только после окна наблюдения;
// не помогла — переоткрыта, новый цикл, в организационной памяти — «не помогло».
func TestCorrectiveActionsOnFakes(t *testing.T) {
	w, j := memWorld(t)
	ctx := context.Background()
	if err := w.Story(ctx); err != nil {
		t.Fatal(err)
	}
	if err := w.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	now := analysistest.At(23, 11, 45)
	svc := w.Service(j, now)
	tec := analysistest.Principal(ctx, "TEC-01")
	m := platform.Moment{Axis: platform.AxisOccurred}
	incs, err := svc.Incidents(tec, m, platform.Page{})
	if err != nil || len(incs.Items) != 1 {
		t.Fatalf("инциденты: %v", err)
	}
	inc := incs.Items[0].IncidentID
	assign := func(plan appanalysis.EffectivenessPlanInput) (platform.Receipt, error) {
		return svc.AssignAction(tec, inc, appanalysis.AssignAction{CommandHeader: platform.CommandHeader{CommandID: w.ID("command")},
			ActionType: "corrective_action", Direction: "prevent_occurrence", OwnerID: "TEC-01", Title: "Проверка источника каждые 75 циклов",
			EffectivenessPlan: plan})
	}
	if _, err := assign(appanalysis.EffectivenessPlanInput{Metric: "доля швов с трещиной"}); errCode(err) != errcodes.IncidentEffectivenessPlanRequired {
		t.Fatalf("мера без плана: %v", err)
	}
	if _, err := assign(appanalysis.EffectivenessPlanInput{Metric: "доля швов с трещиной", Baseline: "6 из 34", WindowDays: 7,
		SuccessCriterion: "0 трещин за 7 дней", EnhancedControl: "100 % визуальный контроль швов ИС-2"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	al, err := svc.CorrectiveActions(tec, m)
	if err != nil || len(al.Items) != 1 {
		t.Fatalf("меры: %+v %v", al, err)
	}
	a := al.Items[0]
	if a.Status != "assigned" || a.Cycle != 1 || a.Plan.WindowDays != 7 || a.Title == nil {
		t.Fatalf("мера: %+v", a)
	}
	if _, err := svc.EvaluateAction(tec, inc, a.ActionID, appanalysis.EvaluateAction{CommandHeader: platform.CommandHeader{CommandID: w.ID("command")},
		Result: "effective"}); errCode(err) != errcodes.IncidentActionState {
		t.Fatalf("оценка до внедрения: %v", err)
	}
	if _, err := svc.ImplementAction(tec, inc, a.ActionID, appanalysis.ImplementAction{CommandHeader: platform.CommandHeader{CommandID: w.ID("command")},
		Note: "Регламент введён"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EvaluateAction(tec, inc, a.ActionID, appanalysis.EvaluateAction{CommandHeader: platform.CommandHeader{CommandID: w.ID("command")},
		Result: "effective"}); errCode(err) != errcodes.IncidentActionState {
		t.Fatalf("«эффективно» до окна наблюдения: %v", err)
	}
	if _, err := svc.EvaluateAction(tec, inc, a.ActionID, appanalysis.EvaluateAction{CommandHeader: platform.CommandHeader{CommandID: w.ID("command")},
		Result: "failed", Evidence: "Трещина на Ф-040"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	al, _ = svc.CorrectiveActions(tec, m)
	a = al.Items[0]
	if a.Status != "reopened" || a.Cycle != 2 || !slices.Contains(a.Flags, dom.FlagIneffective) || al.Summary.Ineffective != 1 {
		t.Fatalf("после провала: %+v, сводка %+v", a, al.Summary)
	}
	if len(al.Memory) != 1 || al.Memory[0].Outcome != "not_helped" {
		t.Fatalf("организационная память: %+v", al.Memory)
	}
	// Переоткрытую меру можно внедрить снова.
	if _, err := svc.ImplementAction(tec, inc, a.ActionID, appanalysis.ImplementAction{CommandHeader: platform.CommandHeader{CommandID: w.ID("command")}}); err != nil {
		t.Fatal(err)
	}

	// Карта дефицита считается по тем же разборам (FR-143).
	dm, err := svc.DataDeficit(tec, m)
	if err != nil || dm.Investigations == 0 {
		t.Fatalf("карта дефицита: %+v %v", dm, err)
	}
	for _, r := range dm.Rows {
		if r.Investigations == 0 || r.Digitization == "" {
			t.Fatalf("строка карты: %+v", r)
		}
	}
	_ = catalog.IncidentActionAssigned
}
