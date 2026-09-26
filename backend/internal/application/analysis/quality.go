package analysis

import (
	"context"
	"slices"
	"time"

	"ant/internal/application/platform"
	dom "ant/internal/domain/analysis"
)

// Меры и их эффективность, взгляд руководителя по качеству и карта дефицита
// данных (FR-64, FR-138, FR-143; эпик 42): чтение над проекциями
// analysis.action, analysis.incident, analysis.nc и разбором обстоятельств.

// ── формы ответов ──

// Suggestion — предложение генератора (FR-63).
type Suggestion struct {
	SuggestionID    string              `json:"suggestion_id"`
	Generator       string              `json:"generator" doc:"Генератор (порт + адаптер): rules.bottleneck, rules.risk_scope…"`
	Kind            string              `json:"kind" enum:"bottleneck,risk_scope,reaction_rule_candidate,analyzer_adaptation,data_deficit,other"`
	Title           string              `json:"title"`
	Statement       string              `json:"statement"`
	Estimate        *string             `json:"estimate,omitempty" doc:"Оценка эффекта словами с единицами."`
	ResponsibleRole *string             `json:"responsible_role,omitempty" doc:"Роль ответственного."`
	ResponsibleID   *string             `json:"responsible_id,omitempty" doc:"Кому передано."`
	StepKey         *string             `json:"step_key,omitempty" doc:"Узел процесса — ссылка на карту процесса."`
	IncidentID      *string             `json:"incident_id,omitempty"`
	MissingKind     *string             `json:"missing_kind,omitempty" doc:"Вид недостающих сведений (карта дефицита)."`
	Basis           []string            `json:"basis" doc:"Основания — event_id записей журнала."`
	Status          string              `json:"status" enum:"new,forwarded,accepted,rejected" doc:"Новое / передано ответственному / принято в работу / отклонено. «Принято» ничего не применяет само."`
	RecordedAt      time.Time           `json:"recorded_at"`
	EventID         string              `json:"event_id"`
	BasisSeq        int64               `json:"basis_seq" doc:"basis_seq для команд по предложению (AD-39)."`
	History         []SuggestionHistory `json:"history"`
}

// SuggestionHistory — строка истории предложения.
type SuggestionHistory struct {
	Type            string    `json:"type" enum:"recorded,forwarded,accepted,rejected"`
	Actor           *string   `json:"actor,omitempty"`
	At              time.Time `json:"at"`
	ResponsibleID   *string   `json:"responsible_id,omitempty"`
	ResponsibleRole *string   `json:"responsible_role,omitempty"`
	Text            *string   `json:"text,omitempty"`
	EventID         string    `json:"event_id"`
}

// GeneratorInfo — подключённый генератор.
type GeneratorInfo struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Connected bool   `json:"connected" doc:"false — вход генератора не подключён (например, счётчики узлов)."`
}

// SuggestionList — предложения и генераторы.
type SuggestionList struct {
	Items      []Suggestion    `json:"items"`
	Generators []GeneratorInfo `json:"generators"`
	BasisSeq   int64           `json:"basis_seq"`
}

// SuggestionRun — итог прогона генераторов.
type SuggestionRun struct {
	Recorded []string `json:"recorded" doc:"Новые предложения (suggestion_id)."`
	EventIDs []string `json:"event_ids" doc:"event_id записанных фактов."`
	Skipped  int      `json:"skipped" doc:"Уже записанные — не повторяются."`
	Failed   []string `json:"failed" doc:"Генераторы, завершившиеся ошибкой."`
	Seq      int64    `json:"seq"`
}

// EffectivenessPlanView — план проверки эффективности.
type EffectivenessPlanView struct {
	Metric           string  `json:"metric"`
	Baseline         string  `json:"baseline"`
	WindowDays       int     `json:"window_days"`
	SuccessCriterion string  `json:"success_criterion"`
	EnhancedControl  *string `json:"enhanced_control,omitempty"`
}

// ActionHistory — строка истории меры.
type ActionHistory struct {
	Type     string    `json:"type" enum:"assigned,implemented,evaluated"`
	Result   *string   `json:"result,omitempty" enum:"effective,failed"`
	Text     *string   `json:"text,omitempty"`
	Evidence *string   `json:"evidence,omitempty"`
	Actor    *string   `json:"actor,omitempty"`
	At       time.Time `json:"at"`
	Cycle    int       `json:"cycle"`
	EventID  string    `json:"event_id"`
}

// CorrectiveActionView — мера с планом и флагами (FR-64, FR-138).
type CorrectiveActionView struct {
	ActionID        string                `json:"action_id"`
	IncidentID      string                `json:"incident_id"`
	ActionType      string                `json:"action_type" enum:"correction,corrective_action,preventive_action" doc:"Коррекция / корректирующее / предупреждающее действие (ГОСТ Р ИСО 9000)."`
	Direction       string                `json:"direction" enum:"prevent_occurrence,improve_detection" doc:"«Почему возник» / «почему пропустили»."`
	Owner           string                `json:"owner"`
	Title           *string               `json:"title,omitempty"`
	SuggestionID    *string               `json:"suggestion_id,omitempty"`
	DueAt           *time.Time            `json:"due_at,omitempty"`
	Plan            EffectivenessPlanView `json:"plan"`
	Status          string                `json:"status" enum:"assigned,implemented,effective,reopened" doc:"Назначена / внедрена (идёт окно наблюдения) / эффективна / переоткрыта — не помогла. «Внедрено» ≠ «эффективно»."`
	Cycle           int                   `json:"cycle" doc:"Попытка: 1 + сколько раз мера не помогла."`
	AssignedAt      time.Time             `json:"assigned_at"`
	ImplementedAt   *time.Time            `json:"implemented_at,omitempty"`
	EvaluationDueAt *time.Time            `json:"evaluation_due_at,omitempty" doc:"С какого момента можно поставить «эффективно»: внедрено + окно наблюдения."`
	Flags           []string              `json:"flags" doc:"overdue — просрочена; ineffective — не помогла; hanging_temporary_control — висит временно усиленный контроль; evaluation_due — пора оценить; awaiting_window — идёт окно наблюдения."`
	CauseCategory   *string               `json:"cause_category,omitempty"`
	Factor          *string               `json:"factor,omitempty"`
	History         []ActionHistory       `json:"history"`
	BasisSeq        int64                 `json:"basis_seq"`
}

// RecurringProblem — повторяющаяся проблема (FR-138).
type RecurringProblem struct {
	DefectType string   `json:"defect_type"`
	StepKey    string   `json:"step_key"`
	Count      int      `json:"count"`
	NCIDs      []string `json:"nc_ids"`
	WithAction bool     `json:"with_action" doc:"По инцидентам этих несоответствий есть мера."`
}

// MemoryEntry — организационная память: что пробовали и с каким результатом (FR-138).
type MemoryEntry struct {
	ActionID      string  `json:"action_id"`
	IncidentID    string  `json:"incident_id"`
	Title         string  `json:"title"`
	ActionType    string  `json:"action_type"`
	Direction     string  `json:"direction"`
	CauseCategory *string `json:"cause_category,omitempty"`
	Factor        *string `json:"factor,omitempty"`
	Outcome       string  `json:"outcome" enum:"helped,not_helped,helped_after_retry,in_progress" doc:"Помогло / не помогло / помогло не с первого раза / ещё проверяется."`
	Cycles        int     `json:"cycles"`
}

// QualitySummary — взгляд руководителя по качеству (FR-138): счётчики.
type QualitySummary struct {
	Open             int `json:"open"`
	Overdue          int `json:"overdue"`
	Ineffective      int `json:"ineffective"`
	HangingTemporary int `json:"hanging_temporary"`
	EvaluationDue    int `json:"evaluation_due"`
	Recurring        int `json:"recurring"`
}

// CorrectiveActionList — меры, взгляд руководителя по качеству и организационная память.
type CorrectiveActionList struct {
	Items     []CorrectiveActionView `json:"items"`
	Summary   QualitySummary         `json:"summary"`
	Recurring []RecurringProblem     `json:"recurring"`
	Memory    []MemoryEntry          `json:"memory"`
	AsOf      time.Time              `json:"as_of" doc:"Доменное «сейчас», на которое посчитаны флаги."`
	BasisSeq  int64                  `json:"basis_seq"`
}

// DeficitPlace — где не хватало сведений.
type DeficitPlace struct {
	Place string `json:"place"`
	Count int    `json:"count"`
}

// DeficitEstimate — оценка сужения области риска (средние, в десятых долях детали).
type DeficitEstimate struct {
	FromTenths int    `json:"from_tenths"`
	ToTenths   int    `json:"to_tenths"`
	Incidents  int    `json:"incidents" doc:"Сколько инцидентов с сужением по основаниям легло в оценку."`
	Text       string `json:"text" doc:"«в среднем с 13 до 4 деталей»."`
}

// DeficitRow — вид недостающих сведений на карте дефицита (FR-143).
type DeficitRow struct {
	Kind           string           `json:"kind" enum:"tool_unknown,cycle_end_time_unknown,no_observation_after_operation,no_observation_before_operation,operator_unknown,equipment_log_missing,other"`
	Investigations int              `json:"investigations" doc:"В скольких расследованиях не хватало."`
	Places         []DeficitPlace   `json:"places"`
	NCIDs          []string         `json:"nc_ids"`
	IncidentIDs    []string         `json:"incident_ids"`
	Digitization   string           `json:"digitization" doc:"Какая цифровизация закрыла бы пробел."`
	Estimate       *DeficitEstimate `json:"estimate,omitempty" doc:"Нет — оценка невозможна: по этим расследованиям область ещё не сужали по основаниям."`
	SuggestionID   *string          `json:"suggestion_id,omitempty" doc:"Предложение цифровизации по этой строке, если записано."`
}

// DataDeficitMap — карта дефицита данных (FR-143).
type DataDeficitMap struct {
	Investigations int          `json:"investigations" doc:"Всего расследований (разборов несоответствий)."`
	Rows           []DeficitRow `json:"rows"`
	BasisSeq       int64        `json:"basis_seq"`
}

// ── чтение ──

// ProjectionAction — проекция мер analysis.action (ключ — action_id; ListKey — список).
const ProjectionAction = "analysis.action"

// ProjectionSuggestion — проекция предложений analysis.suggestion (ключ — suggestion_id; ListKey — список).
const ProjectionSuggestion = "analysis.suggestion"

// actions — все меры из проекции analysis.action.
func (s *Service) actions(ctx context.Context) ([]dom.CorrectiveAction, error) {
	ids, err := s.list(ctx, ProjectionAction)
	if err != nil {
		return nil, err
	}
	var out []dom.CorrectiveAction
	for _, id := range ids {
		var a dom.CorrectiveAction
		ok, err := s.get(ctx, ProjectionAction, id, &a)
		if err != nil {
			return nil, err
		}
		if ok && a.ActionID != "" {
			out = append(out, a)
		}
	}
	return out, nil
}

// action — мера по id.
func (s *Service) action(ctx context.Context, id string) (dom.CorrectiveAction, error) {
	var a dom.CorrectiveAction
	ok, err := s.get(ctx, ProjectionAction, id, &a)
	if err != nil {
		return a, err
	}
	if !ok || a.ActionID == "" {
		return a, notFound("Мера", id)
	}
	return a, nil
}

// CorrectiveActions — меры, флаги на доменное «сейчас», повторяющиеся
// проблемы и организационная память (FR-64, FR-138).
func (s *Service) CorrectiveActions(ctx context.Context, m platform.Moment) (CorrectiveActionList, error) {
	if !s.live() {
		return UnimplementedSuggestions{}.CorrectiveActions(ctx, m)
	}
	in, err := s.input(ctx, false)
	if err != nil {
		return CorrectiveActionList{}, err
	}
	now := in.Now
	if m.AsOf != nil {
		now = *m.AsOf
	}
	incidents := map[string]dom.IncidentRecord{}
	for _, v := range in.Incidents {
		incidents[v.IncidentID] = v
	}
	out := CorrectiveActionList{Items: []CorrectiveActionView{}, AsOf: now}
	withAction := map[string]bool{}
	for _, a := range in.Actions {
		withAction[a.IncidentID] = true
		v := actionView(a, now)
		if inc, ok := incidents[a.IncidentID]; ok {
			v.Factor = strp(inc.FactorValue)
			if inc.Cause != nil {
				v.CauseCategory = strp(inc.Cause.Category)
			}
		}
		out.Items = append(out.Items, v)
		out.BasisSeq = max(out.BasisSeq, a.BasisSeq)
		if a.Status != dom.ActionEffective {
			out.Summary.Open++
		}
		for _, f := range v.Flags {
			switch f {
			case dom.FlagOverdue:
				out.Summary.Overdue++
			case dom.FlagIneffective:
				out.Summary.Ineffective++
			case dom.FlagHangingControl:
				out.Summary.HangingTemporary++
			case dom.FlagEvaluationDue:
				out.Summary.EvaluationDue++
			}
		}
	}
	slices.SortStableFunc(out.Items, func(a, b CorrectiveActionView) int { return b.AssignedAt.Compare(a.AssignedAt) })
	var profiles []dom.Profile
	ncIncidents := map[string][]string{}
	for _, a := range in.Analyses {
		profiles = append(profiles, a.Profile)
		for _, v := range in.Incidents {
			if item := in.Items[a.NCID]; item != "" {
				if _, ok := v.Members[item]; ok {
					ncIncidents[a.NCID] = append(ncIncidents[a.NCID], v.IncidentID)
				}
			}
		}
	}
	out.Recurring = []RecurringProblem{}
	for _, r := range dom.RecurringProblems(profiles, ncIncidents, withAction) {
		out.Recurring = append(out.Recurring, RecurringProblem(r))
	}
	out.Summary.Recurring = len(out.Recurring)
	out.Memory = []MemoryEntry{}
	for _, e := range dom.Memory(in.Actions, incidents) {
		out.Memory = append(out.Memory, MemoryEntry{ActionID: e.ActionID, IncidentID: e.IncidentID, Title: e.Title, ActionType: e.ActionType,
			Direction: e.Direction, CauseCategory: strp(e.Cause), Factor: strp(e.Factor), Outcome: e.Outcome, Cycles: e.Cycles})
	}
	return out, nil
}

func actionView(a dom.CorrectiveAction, now time.Time) CorrectiveActionView {
	v := CorrectiveActionView{ActionID: a.ActionID, IncidentID: a.IncidentID, ActionType: a.ActionType, Direction: a.Direction, Owner: a.Owner,
		Title: strp(a.Title), SuggestionID: strp(a.SuggestionID), DueAt: a.DueAt, Status: a.Status, Cycle: a.Cycle, AssignedAt: a.AssignedAt,
		ImplementedAt: a.ImplementedAt, EvaluationDueAt: a.EvaluationDueAt(), Flags: dom.ActionFlags(a, now), History: []ActionHistory{},
		BasisSeq: a.BasisSeq,
		Plan: EffectivenessPlanView{Metric: a.Plan.Metric, Baseline: a.Plan.Baseline, WindowDays: a.Plan.WindowDays,
			SuccessCriterion: a.Plan.SuccessCriterion, EnhancedControl: strp(a.Plan.EnhancedControl)}}
	for _, h := range a.History {
		v.History = append(v.History, ActionHistory{Type: h.Type, Result: strp(h.Result), Text: strp(h.Text), Evidence: strp(h.Evidence),
			Actor: strp(h.Actor), At: h.At, Cycle: h.Cycle, EventID: h.EventID})
	}
	return v
}

// DataDeficit — карта дефицита данных (FR-143): по видам сведений — число
// расследований, где не хватало, где именно, какая цифровизация закрыла бы
// пробел и оценка сужения области риска; ссылка на предложение, если оно записано.
func (s *Service) DataDeficit(ctx context.Context, m platform.Moment) (DataDeficitMap, error) {
	if !s.live() {
		return UnimplementedSuggestions{}.DataDeficit(ctx, m)
	}
	in, err := s.input(ctx, false)
	if err != nil {
		return DataDeficitMap{}, err
	}
	dm := dom.BuildDeficitMap(in.Analyses, in.Incidents, in.Items)
	byKind := map[string]string{}
	for _, p := range dom.DeficitProposals(dm) {
		id := dom.SuggestionID(p.Generator, p.DedupKey)
		if _, err := s.suggestion(ctx, id); err == nil {
			byKind[p.MissingKind] = id
		}
	}
	out := DataDeficitMap{Investigations: dm.Investigations, Rows: []DeficitRow{}}
	for _, v := range in.Incidents {
		out.BasisSeq = max(out.BasisSeq, v.BasisSeq)
	}
	for _, r := range dm.Rows {
		row := DeficitRow{Kind: r.Kind, Investigations: r.Investigations, Places: []DeficitPlace{}, NCIDs: r.NCIDs, IncidentIDs: r.IncidentIDs,
			Digitization: r.Digitization, SuggestionID: strp(byKind[r.Kind])}
		if row.NCIDs == nil {
			row.NCIDs = []string{}
		}
		if row.IncidentIDs == nil {
			row.IncidentIDs = []string{}
		}
		for _, p := range r.Places {
			row.Places = append(row.Places, DeficitPlace(p))
		}
		if e := r.Estimate; e != nil {
			row.Estimate = &DeficitEstimate{FromTenths: e.FromTenths, ToTenths: e.ToTenths, Incidents: e.Incidents,
				Text: "в среднем с " + dom.TenthsText(e.FromTenths) + " до " + dom.TenthsText(e.ToTenths) + " деталей"}
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}
