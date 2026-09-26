package notifications

import (
	"encoding/json"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/analysis"
	"ant/internal/domain/documents"
	"ant/internal/domain/item"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
	"ant/internal/domain/nonconformity"
	"ant/internal/domain/process"
	"ant/internal/domain/quality"
	"ant/internal/domain/vision"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "notifications"

// State — состояние модуля notifications в свёртке одного изделия (AD-5):
// обязательства со сроками и адресные задачи, выведенные из состояния
// модулей-владельцев (AD-4, AD-40: notifications — последний в композиции),
// и где изделие физически. Экспортируемые поля сериализуются в хеш состояния
// (Д-22).
type State struct {
	ItemID string `json:"item_id,omitempty"`
	RunID  string `json:"run_id,omitempty"`
	// At — наибольший occurred_at свёрнутых записей.
	At    time.Time `json:"at"`
	Where Where     `json:"where"`
	// LastInspectionAt — последний результат контроля изделия (доп. проверка выполнена).
	LastInspectionAt time.Time    `json:"last_inspection_at"`
	Obligations      []Obligation `json:"obligations,omitempty"`
	Tasks            []Task       `json:"tasks,omitempty"`
}

// Upstream — состояния модулей раньше notifications в композиции на этом шаге
// (только чтение, AD-40): поздний модуль видит вывод раннего, обратно — только
// через функцию-намерение раннего модуля.
type Upstream struct {
	Item          *item.State
	Process       *process.State
	Vision        *vision.State
	Quality       *quality.State
	Machinelogs   *machinelogs.State
	Documents     *documents.State
	Nonconformity *nonconformity.State
	Analysis      *analysis.State
}

// Reduce применяет запись входа изделия (факт, решение, адресованную запись
// стадии, «наступил срок») к состоянию модуля (AD-5): отмечает, где изделие,
// принимает «наступил срок» и заново выводит обязательства и задачи из
// состояния модулей-владельцев на этом шаге (derive). Реакции в свёртку не
// входят (AD-3).
func Reduce(s State, r kernel.Record, env Env, up Upstream) State {
	s = s.clone()
	env = env.Defaults()
	if s.ItemID == "" && r.ItemID != "" {
		s.ItemID = r.ItemID
	}
	if s.RunID == "" && r.RunID != "" {
		s.RunID = r.RunID
	}
	if r.OccurredAt.After(s.At) {
		s.At = r.OccurredAt
	}
	switch r.Type {
	case catalog.OperationRunStarted:
		var d struct {
			StepKey     string `json:"step_key"`
			WorkplaceID string `json:"workplace_id"`
			StationID   string `json:"station_id"`
			LineID      string `json:"line_id"`
		}
		if decode(r, &d) {
			s.Where = Where{StepKey: d.StepKey, LocationID: first(d.WorkplaceID, d.StationID, d.LineID, s.Where.LocationID), At: r.OccurredAt}
		}
	case catalog.OperationMovementReceived:
		var d struct {
			ToLocationID string `json:"to_location_id"`
			StepKey      string `json:"step_key"`
		}
		if decode(r, &d) {
			s.Where = Where{StepKey: first(d.StepKey, s.Where.StepKey), LocationID: first(d.ToLocationID, s.Where.LocationID), At: r.OccurredAt}
		}
	case catalog.ItemPresentationRecorded:
		var d struct {
			StepKey string `json:"step_key"`
		}
		if decode(r, &d) && d.StepKey != "" {
			s.Where.StepKey, s.Where.At = d.StepKey, r.OccurredAt
		}
	case catalog.InspectionResultRecorded:
		if r.OccurredAt.After(s.LastInspectionAt) {
			s.LastInspectionAt = r.OccurredAt
		}
	case catalog.ObligationDueReached:
		s.reach(r)
	}
	s.derive(r, env, up)
	return s
}

// reach — «наступил срок» (AD-4): уровень лестницы наступил, если запись
// планировщика — про текущий срок действующего обязательства; следующий
// уровень взводится сам (DueAt). Запись про прежний срок (срок успел
// смениться) или снятое обязательство ничего не меняет.
func (s *State) reach(r kernel.Record) {
	var d struct {
		ObligationID string `json:"obligation_id"`
		DueAt        string `json:"due_at"`
	}
	if !decode(r, &d) {
		return
	}
	o := s.obligation(d.ObligationID)
	due, ok := ParseTime(d.DueAt)
	if o == nil || !ok || !o.Open || !o.Armed() || !due.Equal(o.DueAt()) {
		return
	}
	o.Reaches = append(o.Reaches, Reach{Level: o.Level(), DueAt: due, EventID: r.EventID, At: r.OccurredAt})
}

// React вычисляет реакции модуля по состоянию после записи (AD-3, AD-40):
// сроки, эскалации с ценой задержки, тревоги, информацию и задачи — только
// своими типами через kernel.NewReaction. Реакция вычисляется из состояния
// целиком, поэтому в итоге свёртки на слот остаётся последний вывод (срок
// поставлен или снят, задача поставлена или снята).
func React(s State, env Env, up Upstream) kernel.Output {
	_, _ = up, env
	var out kernel.Output
	add := func(r kernel.Reaction, err error) {
		if err != nil {
			// Тип и эмитент постоянны: ошибка — рассинхронизация с каталогом.
			panic(err)
		}
		r.AutomationMode = 1 // только сообщает (FR-50, режим 1)
		out.Reactions = append(out.Reactions, r)
	}
	for _, o := range s.Obligations {
		for _, r := range obligationReactions(s, o) {
			add(r, nil)
		}
	}
	for _, t := range s.Tasks {
		add(taskReaction(t))
	}
	return out
}

// Guard — доменный гард операций модуля notifications (AD-39): состояние
// изделия на basis_seq и команда → nil или *kernel.Refusal. Единственная
// операция модуля — отметка задачи; её гард — GuardAcknowledge над проекцией
// задачи (задача может быть и вне изделия), у изделия ограничений нет.
func Guard(s State, env Env, up Upstream, cmd kernel.Command) error {
	_, _, _, _ = s, env, up, cmd
	return nil
}

func decode(r kernel.Record, v any) bool {
	return len(r.Data) > 0 && json.Unmarshal(r.Data, v) == nil
}

func first(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
