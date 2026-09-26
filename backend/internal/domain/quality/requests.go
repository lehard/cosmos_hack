package quality

import (
	"slices"
	"strings"

	"ant/internal/contracts/statuses"
)

// Виды запросов quality к поздним модулям композиции.
const (
	// RequestDraftNC — черновик карточки несоответствия (nonconformity, эпик 21).
	RequestDraftNC = "draft_nc"
	// RequestContain — уровень сдерживания изделия (nonconformity: ось «сдерживание»).
	RequestContain = "contain"
	// RequestTask — задача человеку (notifications: task.task.created).
	RequestTask = "task"
)

// Request — запрос модуля quality к позднему модулю композиции (AD-40, AD-30).
//
// Порядок композиции item → process → vision → quality → … → nonconformity →
// notifications не даёт quality вызывать функции-намерения nonconformity и
// notifications (ранний модуль не импортирует поздний, AD-1). Поэтому quality
// выражает намерение данными своего состояния: поздний модуль читает
// Upstream.Quality.Requests и сам строит свои записи — nonconformity
// (decision.nonconformity.drafted, сдерживание) и notifications
// (task.task.created, obligation.*). Это порт quality для эпиков 21 и 18/…;
// пока их правил нет, запросы видны в проекции quality.item и в тестах —
// заготовкой потребителя (requests_test.go).
type Request struct {
	// Key — стабильный ключ запроса (для слота реакции потребителя).
	Key  string `json:"key"`
	Kind string `json:"kind"`
	// SignalID, DefectID, StepKey — предмет запроса.
	SignalID string `json:"signal_id,omitempty"`
	DefectID string `json:"defect_id,omitempty"`
	StepKey  string `json:"step_key,omitempty"`
	// Containment — уровень сдерживания (для contain).
	Containment statuses.Containment `json:"containment,omitempty"`
	// TaskKind — вид задачи task.task.created: recheck | isolate_move |
	// decision_required | inspection_missing; RoleID — роль исполнителя.
	TaskKind string `json:"task_kind,omitempty"`
	RoleID   string `json:"role_id,omitempty"`
	Title    string `json:"title,omitempty"`
	// RuleRef — правило карты реакций, AutomationMode — режим (FR-50).
	RuleRef        string `json:"rule_ref,omitempty"`
	AutomationMode int    `json:"automation_mode,omitempty"`
	// Causes — записи-основания (отсортированы).
	Causes []string `json:"causes"`
}

// Роли исполнителей задач (normative/policy).
const (
	roleInspector   = "quality_inspector"
	roleTechnologist = "technologist"
)

// requests — запросы по открытым сигналам, точкам без данных и рекомендациям
// анализатора уровня доверия 1 (AD-29).
func requests(s State) []Request {
	out := []Request{}
	for _, sg := range s.Signals {
		if sg.State != SignalOpen {
			continue
		}
		a := sg.Assessment
		if a.Recommend && !sg.Raised {
			out = append(out, Request{Key: "recommend/" + sg.SignalID, Kind: RequestTask, SignalID: sg.SignalID, DefectID: sg.DefectID,
				StepKey: sg.StepKey, TaskKind: "decision_required", RoleID: roleInspector, RuleRef: a.MapRef, AutomationMode: 1,
				Title: "Рекомендация анализатора: посмотрите " + what(sg), Causes: sg.Causes})
			continue
		}
		if !sg.Raised {
			continue
		}
		if a.DraftNC && sg.Basis != BasisSkipped && !sg.Unable {
			out = append(out, Request{Key: "draft/" + sg.SignalID, Kind: RequestDraftNC, SignalID: sg.SignalID, DefectID: sg.DefectID,
				StepKey: sg.StepKey, RuleRef: a.MapRef, AutomationMode: a.Mode, Title: "Черновик несоответствия: " + what(sg), Causes: sg.Causes})
		}
		if a.Containment != "" && a.Containment != statuses.ContainmentNone {
			out = append(out, Request{Key: "contain/" + sg.SignalID, Kind: RequestContain, SignalID: sg.SignalID, DefectID: sg.DefectID,
				StepKey: sg.StepKey, Containment: a.Containment, RuleRef: a.MapRef, AutomationMode: a.Mode, Causes: sg.Causes})
		}
		if a.Task != "" && sg.Basis != BasisSkipped {
			// Пропуск проверки: задачу «нет данных» ставит точка полноты ниже.
			role := roleInspector
			if a.Outcome == ReactQuestion {
				role = roleTechnologist
			}
			out = append(out, Request{Key: "task/" + sg.SignalID, Kind: RequestTask, SignalID: sg.SignalID, DefectID: sg.DefectID,
				StepKey: sg.StepKey, TaskKind: a.Task, RoleID: role, RuleRef: a.MapRef, AutomationMode: 1, Title: taskTitle(sg), Causes: sg.Causes})
		}
	}
	for _, p := range s.Points {
		if p.Status != "missing" || !p.Required {
			continue
		}
		out = append(out, Request{Key: "missing/" + p.StepKey + "/" + p.Method + "@" + p.Window, Kind: RequestTask, StepKey: p.StepKey,
			TaskKind: "inspection_missing", RoleID: roleInspector, AutomationMode: 1,
			Title: "Нет данных контроля " + firstNonEmpty(p.InspectionPoint, p.ClosingPoint, p.StepKey) + " (" + p.Method + ")", Causes: p.Causes})
	}
	for i := range out {
		c := slices.Clone(out[i].Causes)
		slices.Sort(c)
		out[i].Causes = slices.Compact(c)
	}
	return out
}

func what(sg Signal) string {
	parts := []string{}
	if sg.TypeCode != "" {
		parts = append(parts, sg.TypeCode)
	}
	if sg.Zone != "" {
		parts = append(parts, "зона "+sg.Zone)
	}
	if len(parts) == 0 {
		return "результат контроля"
	}
	return strings.Join(parts, ", ")
}

func taskTitle(sg Signal) string {
	switch {
	case sg.Unable:
		return "Повторный контроль: оценка невозможна (" + firstNonEmpty(sg.UnableReason, "причина не указана") + ")"
	case sg.Basis == BasisSkipped:
		return "Пропущена обязательная проверка " + sg.StepKey
	case sg.Assessment.Outcome == ReactQuestion:
		return "Вопрос технологу: нет требования КД — " + what(sg)
	case sg.Assessment.Outcome == ReactIsolate:
		return "Переместить в изолятор: " + what(sg)
	}
	return "Требуется решение контролёра: " + what(sg)
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// Blocker — почему решение «годно» на точке предъявления пока преждевременно.
type Blocker struct {
	// Code — open_signal | inspection_missing | inspection_pending | unable_to_assess.
	Code     string `json:"code"`
	StepKey  string `json:"step_key,omitempty"`
	SignalID string `json:"signal_id,omitempty"`
}

// PresentationBlockers — что мешает принять изделие на закрывающей точке
// stepKey (FR-35, FR-44, FR-48): открытые сигналы участка, точки контроля
// участка до неё без результата, с «нет данных» или «оценка невозможна».
// Чистая функция для гарда операции решения на точке предъявления
// (nonconformity, AD-39): модуль-владелец операции вызывает её над
// Upstream.Quality на basis_seq.
func PresentationBlockers(s State, env Env, stepKey string) []Blocker {
	cs, ok := env.step(stepKey)
	if !ok {
		return nil
	}
	out := []Blocker{}
	for _, p := range s.Points {
		if p.Stage != cs.Stage || p.Order >= cs.Order || !p.Required {
			continue
		}
		switch p.Status {
		case "missing":
			out = append(out, Blocker{Code: "inspection_missing", StepKey: p.StepKey})
		case "pending":
			out = append(out, Blocker{Code: "inspection_pending", StepKey: p.StepKey})
		case "unable":
			out = append(out, Blocker{Code: "unable_to_assess", StepKey: p.StepKey})
		}
	}
	for _, sg := range s.Signals {
		if sg.Raised && sg.State == SignalOpen && env.stageOf(sg.StepKey) == cs.Stage && !sg.Unable && sg.Basis != BasisSkipped {
			out = append(out, Blocker{Code: "open_signal", StepKey: sg.StepKey, SignalID: sg.SignalID})
		}
	}
	return out
}
