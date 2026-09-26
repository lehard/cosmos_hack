package analysis

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Корректирующие действия и их эффективность (FR-64, FR-138, эпик 42):
// проекция analysis.action — мера с планом проверки эффективности и историей
// оценок; гарды команд мер. Различаются коррекция, корректирующее и
// предупреждающее действие (ГОСТ Р ИСО 9000); направления — «почему возник»
// (prevent_occurrence) и «почему пропустили» (improve_detection).
// «Исполнено» ≠ «эффективно»: после «внедрено» идёт окно наблюдения, оценка
// «эффективно» — только после окна; не выполнен критерий — мера переоткрыта
// (status reopened) и снова ждёт внедрения; каждое переоткрытие — новый цикл.

// Статусы меры в проекции analysis.action.
const (
	ActionAssigned    = "assigned"
	ActionImplemented = "implemented"
	ActionEffective   = "effective"
	ActionReopened    = "reopened"
)

// EffectivenessPlan — план проверки эффективности (FR-64).
type EffectivenessPlan struct {
	Metric           string `json:"metric"`
	Baseline         string `json:"baseline"`
	WindowDays       int    `json:"window_days"`
	SuccessCriterion string `json:"success_criterion"`
	EnhancedControl  string `json:"enhanced_control,omitempty"`
}

// ActionEvaluation — оценка эффективности (или отметка о внедрении) в истории меры.
type ActionEvaluation struct {
	Type     string    `json:"type"`
	Result   string    `json:"result,omitempty"`
	Text     string    `json:"text,omitempty"`
	Actor    string    `json:"actor,omitempty"`
	At       time.Time `json:"at"`
	EventID  string    `json:"event_id"`
	Cycle    int       `json:"cycle"`
	Evidence string    `json:"evidence,omitempty"`
}

// CorrectiveAction — значение проекции analysis.action (ключ — action_id).
type CorrectiveAction struct {
	ActionID      string             `json:"action_id"`
	IncidentID    string             `json:"incident_id"`
	ActionType    string             `json:"action_type"`
	Direction     string             `json:"direction"`
	Owner         string             `json:"owner"`
	Title         string             `json:"title,omitempty"`
	SuggestionID  string             `json:"suggestion_id,omitempty"`
	DueAt         *time.Time         `json:"due_at,omitempty"`
	Plan          EffectivenessPlan  `json:"plan"`
	Status        string             `json:"status"`
	Cycle         int                `json:"cycle"`
	AssignedAt    time.Time          `json:"assigned_at"`
	AssignedBy    string             `json:"assigned_by,omitempty"`
	ImplementedAt *time.Time         `json:"implemented_at,omitempty"`
	History       []ActionEvaluation `json:"history"`
	EventID       string             `json:"event_id"`
	// BasisSeq — seq последней записи меры.
	BasisSeq int64 `json:"basis_seq"`
}

// EvaluationDueAt — с какого момента можно оценивать эффективность: внедрено + окно.
func (a CorrectiveAction) EvaluationDueAt() *time.Time {
	if a.ImplementedAt == nil || a.Plan.WindowDays <= 0 {
		return nil
	}
	t := a.ImplementedAt.Add(time.Duration(a.Plan.WindowDays) * 24 * time.Hour)
	return &t
}

// Failures — сколько раз мера признана неэффективной.
func (a CorrectiveAction) Failures() int {
	n := 0
	for _, h := range a.History {
		if h.Type == "evaluated" && h.Result == "failed" {
			n++
		}
	}
	return n
}

// ActionKeys — ключи проекции analysis.action, которые меняет запись.
func ActionKeys(r kernel.Record) []string {
	switch r.Type {
	case catalog.IncidentActionAssigned:
		if id := actionOf(r); id != "" {
			return []string{id, ListKey}
		}
	case catalog.IncidentActionImplemented, catalog.IncidentActionEvaluated:
		if id := actionOf(r); id != "" {
			return []string{id}
		}
	}
	return nil
}

// ActionListID — id, который запись добавляет в список мер.
func ActionListID(r kernel.Record) string {
	if r.Type == catalog.IncidentActionAssigned {
		return actionOf(r)
	}
	return ""
}

func actionOf(r kernel.Record) string {
	d, _ := decodeAs[struct {
		ActionID string `json:"action_id"`
	}](r)
	return d.ActionID
}

// StepAction — шаг проекции меры по записи.
func StepAction(a CorrectiveAction, r kernel.Record) CorrectiveAction {
	at := r.OccurredAt
	actor := ActorName(r.Actor)
	switch r.Type {
	case catalog.IncidentActionAssigned:
		d, err := kernel.Decode[ev.IncidentActionAssignedV1](r)
		if err != nil {
			return a
		}
		a = CorrectiveAction{ActionID: string(d.ActionID), IncidentID: string(d.IncidentID), ActionType: string(d.ActionType),
			Direction: string(d.Direction), Owner: string(d.OwnerID), Title: deref(d.Title), Status: ActionAssigned, Cycle: 1,
			AssignedAt: at, AssignedBy: actor, EventID: r.EventID, History: []ActionEvaluation{}}
		if d.SuggestionID != nil {
			a.SuggestionID = string(*d.SuggestionID)
		}
		if d.DueAt != nil {
			t := time.Time(*d.DueAt)
			a.DueAt = &t
		}
		p := d.EffectivenessPlan
		a.Plan = EffectivenessPlan{Metric: p.Metric, Baseline: p.Baseline, WindowDays: p.WindowDays, SuccessCriterion: p.SuccessCriterion,
			EnhancedControl: deref(p.EnhancedControl)}
		a.History = append(a.History, ActionEvaluation{Type: "assigned", Actor: actor, At: at, EventID: r.EventID, Cycle: 1})
	case catalog.IncidentActionImplemented:
		d, err := kernel.Decode[ev.IncidentActionImplementedV1](r)
		if err != nil || a.ActionID == "" {
			return a
		}
		a.Status = ActionImplemented
		t := at
		a.ImplementedAt = &t
		a.History = append(slices.Clone(a.History), ActionEvaluation{Type: "implemented", Text: deref(d.Note), Actor: actor, At: at,
			EventID: r.EventID, Cycle: a.Cycle})
	case catalog.IncidentActionEvaluated:
		d, err := kernel.Decode[ev.IncidentActionEvaluatedV1](r)
		if err != nil || a.ActionID == "" {
			return a
		}
		a.History = append(slices.Clone(a.History), ActionEvaluation{Type: "evaluated", Result: string(d.Result), Evidence: deref(d.Evidence),
			Actor: actor, At: at, EventID: r.EventID, Cycle: a.Cycle})
		if d.Result == ev.IncidentActionEvaluatedV1ResultEffective {
			a.Status = ActionEffective
		} else {
			// FR-64: не выполнен критерий — мера переоткрывается, новый цикл.
			a.Status, a.Cycle, a.ImplementedAt = ActionReopened, a.Cycle+1, nil
		}
	default:
		return a
	}
	a.BasisSeq = r.Seq
	return a
}

// GuardPlan — корректирующее действие не создаётся без плана проверки
// эффективности (FR-64): метрика, базовый уровень, окно наблюдения, критерий успеха.
func GuardPlan(p EffectivenessPlan) error {
	var missing []string
	if strings.TrimSpace(p.Metric) == "" {
		missing = append(missing, "метрика")
	}
	if strings.TrimSpace(p.Baseline) == "" {
		missing = append(missing, "базовый уровень")
	}
	if p.WindowDays < 1 {
		missing = append(missing, "окно наблюдения")
	}
	if strings.TrimSpace(p.SuccessCriterion) == "" {
		missing = append(missing, "критерий успеха")
	}
	if len(missing) > 0 {
		return kernel.Refuse(errcodes.IncidentEffectivenessPlanRequired, "missing", "нет: "+strings.Join(missing, ", "))
	}
	return nil
}

// GuardImplement — отметить внедрение можно у назначенной или переоткрытой меры.
func GuardImplement(a CorrectiveAction) error {
	if a.Status == ActionAssigned || a.Status == ActionReopened {
		return nil
	}
	return actionState(a, "внедрение уже отмечено — дальше оценка эффективности")
}

// GuardEvaluate — оценка эффективности: только после внедрения; «эффективно»
// — только после окна наблюдения («внедрено» ≠ «эффективно»); «не
// эффективно» можно признать и раньше (критерий уже провален).
func GuardEvaluate(a CorrectiveAction, result string, now time.Time) error {
	if a.Status != ActionImplemented {
		return actionState(a, "эффективность оценивают после внедрения")
	}
	if due := a.EvaluationDueAt(); result == "effective" && due != nil && now.Before(*due) {
		return actionState(a, "окно наблюдения ("+strconv.Itoa(a.Plan.WindowDays)+" дн.) истекает "+due.UTC().Format("02.01.2006 15:04")+
			" UTC — до этого «эффективно» не ставится")
	}
	return nil
}

func actionState(a CorrectiveAction, why string) error {
	return kernel.Refuse(errcodes.IncidentActionState, "action_id", a.ActionID, "status", ActionStatusText(a.Status), "why", why)
}

// ActionStatusText — статус меры словами.
func ActionStatusText(s string) string {
	switch s {
	case ActionAssigned:
		return "назначена"
	case ActionImplemented:
		return "внедрена, идёт окно наблюдения"
	case ActionEffective:
		return "эффективна"
	case ActionReopened:
		return "переоткрыта — не помогла"
	}
	return s
}

// ── взгляд руководителя по качеству (FR-138) ──

// Флаги меры для взгляда руководителя по качеству.
const (
	FlagOverdue         = "overdue"
	FlagIneffective     = "ineffective"
	FlagHangingControl  = "hanging_temporary_control"
	FlagEvaluationDue   = "evaluation_due"
	FlagAwaitingWindow  = "awaiting_window"
	outcomeHelped       = "helped"
	outcomeNotHelped    = "not_helped"
	outcomeHelpedLater  = "helped_after_retry"
	outcomeInProgress   = "in_progress"
	temporaryGraceDays  = 0
	recurringThreshold  = 2
	defaultWindowDays   = 30
	maxMemoryEntries    = 50
	maxRecurringEntries = 20
)

// ActionFlags — что требует внимания по мере на момент now: просрочена,
// не помогла, висит временно усиленный контроль, пора оценить, идёт окно.
func ActionFlags(a CorrectiveAction, now time.Time) []string {
	var f []string
	open := a.Status == ActionAssigned || a.Status == ActionReopened
	if open && a.DueAt != nil && now.After(*a.DueAt) {
		f = append(f, FlagOverdue)
	}
	if a.Failures() > 0 && a.Status != ActionEffective {
		f = append(f, FlagIneffective)
	}
	if a.Plan.EnhancedControl != "" && a.Status != ActionEffective {
		days := a.Plan.WindowDays
		if days <= 0 {
			days = defaultWindowDays
		}
		if now.After(a.AssignedAt.Add(time.Duration(days+temporaryGraceDays) * 24 * time.Hour)) {
			f = append(f, FlagHangingControl)
		}
	}
	if a.Status == ActionImplemented {
		if due := a.EvaluationDueAt(); due != nil && !now.Before(*due) {
			f = append(f, FlagEvaluationDue)
		} else {
			f = append(f, FlagAwaitingWindow)
		}
	}
	if f == nil {
		f = []string{}
	}
	return f
}

// MemoryEntry — строка организационной памяти (FR-138): что пробовали и с
// каким результатом («обучение — не помогло, проверка инструмента — частично»).
type MemoryEntry struct {
	ActionID   string `json:"action_id"`
	IncidentID string `json:"incident_id"`
	Title      string `json:"title"`
	ActionType string `json:"action_type"`
	Direction  string `json:"direction"`
	Cause      string `json:"cause_category,omitempty"`
	Factor     string `json:"factor,omitempty"`
	Outcome    string `json:"outcome"`
	Cycles     int    `json:"cycles"`
}

// Memory — организационная память по мерам с итогом: помогло / не помогло /
// помогло не с первого раза / ещё проверяется.
func Memory(actions []CorrectiveAction, incidents map[string]IncidentRecord) []MemoryEntry {
	out := []MemoryEntry{}
	for _, a := range actions {
		if len(a.History) == 0 {
			continue
		}
		e := MemoryEntry{ActionID: a.ActionID, IncidentID: a.IncidentID, Title: firstNonEmpty(a.Title, a.Plan.Metric), ActionType: a.ActionType,
			Direction: a.Direction, Cycles: a.Cycle}
		if v, ok := incidents[a.IncidentID]; ok {
			e.Factor = firstNonEmpty(v.FactorValue, v.Factor)
			if v.Cause != nil {
				e.Cause = v.Cause.Category
			}
		}
		switch {
		case a.Status == ActionEffective && a.Failures() == 0:
			e.Outcome = outcomeHelped
		case a.Status == ActionEffective:
			e.Outcome = outcomeHelpedLater
		case a.Failures() > 0:
			e.Outcome = outcomeNotHelped
		default:
			e.Outcome = outcomeInProgress
		}
		out = append(out, e)
	}
	slices.SortStableFunc(out, func(x, y MemoryEntry) int { return strings.Compare(x.Cause+x.Title, y.Cause+y.Title) })
	if len(out) > maxMemoryEntries {
		out = out[:maxMemoryEntries]
	}
	return out
}

// Recurring — повторяющаяся проблема (FR-138): вид дефекта × узел, где
// несоответствий два и больше; есть ли по ним мера.
type Recurring struct {
	DefectType string   `json:"defect_type"`
	StepKey    string   `json:"step_key"`
	Count      int      `json:"count"`
	NCIDs      []string `json:"nc_ids"`
	WithAction bool     `json:"with_action"`
}

// RecurringProblems — повторяющиеся проблемы по профилям несоответствий;
// ncIncidents — инциденты каждого несоответствия, withAction — инциденты с мерами.
func RecurringProblems(profiles []Profile, ncIncidents map[string][]string, withAction map[string]bool) []Recurring {
	by := map[string]*Recurring{}
	for _, p := range profiles {
		k := p.DefectType + "|" + p.StepKey
		r := by[k]
		if r == nil {
			r = &Recurring{DefectType: firstNonEmpty(p.DefectType, "не указан"), StepKey: firstNonEmpty(p.StepKey, "—")}
			by[k] = r
		}
		r.Count++
		r.NCIDs = appendUnique(r.NCIDs, p.NCID)
		for _, id := range ncIncidents[p.NCID] {
			r.WithAction = r.WithAction || withAction[id]
		}
	}
	out := []Recurring{}
	for _, k := range slices.Sorted(maps.Keys(by)) {
		if by[k].Count >= recurringThreshold {
			out = append(out, *by[k])
		}
	}
	slices.SortStableFunc(out, func(a, b Recurring) int { return b.Count - a.Count })
	if len(out) > maxRecurringEntries {
		out = out[:maxRecurringEntries]
	}
	return out
}
