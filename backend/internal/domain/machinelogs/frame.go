package machinelogs

import (
	"encoding/json"
	"maps"
	"slices"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/item"
	"ant/internal/domain/kernel"
	"ant/internal/domain/process"
	"ant/internal/domain/quality"
	"ant/internal/domain/vision"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "machinelogs"

// State — состояние модуля machinelogs в свёртке одного изделия (AD-5):
// профили выполнения операций изделия по operation_run_id (FR-148). Вход —
// все записи с item_id изделия: выполнения (operation.run.*), привязанные
// стадией события оборудования (equipment.event.bound) и несоответствия
// окна нарушения (decision.nonconformity.registered). Поля экспортируемые и
// сериализуемые в JSON — они входят в хеш состояния (Д-22).
type State struct {
	Runs map[string]Profile `json:"runs,omitempty"`
	// Errors — записи, которые не удалось разобрать (id → причина): свёртка
	// не падает, профиль помечен, вход сохранён (FR-123).
	Errors map[string]string `json:"errors,omitempty"`
}

// Profiles — профили изделия по времени начала (затем по id).
func (s State) Profiles() []Profile {
	out := make([]Profile, 0, len(s.Runs))
	for _, k := range slices.Sorted(maps.Keys(s.Runs)) {
		out = append(out, s.Runs[k])
	}
	slices.SortStableFunc(out, func(a, b Profile) int { return a.StartedAt.Compare(b.StartedAt) })
	return out
}

// Profile — профиль выполнения runID; ok=false — нет такого выполнения у изделия.
func (s State) Profile(runID string) (Profile, bool) {
	p, ok := s.Runs[runID]
	return p, ok
}

// DefaultSpecialSteps — шаги-специальные процессы по умолчанию: признак
// specialProcess="true" в BPMN фланца (normative/process/flange-process.bpmn,
// шаг welding.weld; FR-151). Действует, пока нормативный слой (эпик 17) не
// передаёт признак в Env/StageEnv.
var DefaultSpecialSteps = []string{"welding.weld"}

// Env — закреплённая при запуске изделия часть нормативного слоя, нужная
// machinelogs (AD-17): шаги с признаком «специальный процесс» (FR-151).
// Пустой список — DefaultSpecialSteps.
type Env struct {
	SpecialSteps []string `json:"special_steps,omitempty"`
}

// Special — шаг step — специальный процесс.
func (e Env) Special(step string) bool {
	steps := e.SpecialSteps
	if len(steps) == 0 {
		steps = DefaultSpecialSteps
	}
	return step != "" && slices.Contains(steps, step)
}

// Upstream — состояния модулей раньше machinelogs в композиции на этом шаге
// (только чтение, AD-40): поздний модуль видит вывод раннего, обратно — только
// через функцию-намерение раннего модуля.
type Upstream struct {
	Item    *item.State
	Process *process.State
	Vision  *vision.State
	Quality *quality.State
}

// Reduce применяет запись входа изделия (факт, решение, адресованную запись
// стадии) к состоянию модуля (AD-5). Реакции в свёртку не входят (AD-3).
//
//   - operation.run.started / finished / interval_resolved — выполнение и его
//     интервал (FR-121);
//   - equipment.event.bound — событие оборудования, привязанное стадией к
//     выполнению (AD-29, AD-42): профиль выполнения (FR-148);
//   - decision.nonconformity.registered — несоответствие окна нарушения
//     специального процесса (FR-151).
//
// Ошибка разбора записи не останавливает свёртку: причина — в State.Errors.
func Reduce(s State, r kernel.Record, env Env, up Upstream) State {
	_ = up
	switch {
	case IsRunRecord(r.Type):
		id := RunID(r)
		if id == "" {
			return fail(s, r, "нет operation_run_id")
		}
		p := s.Runs[id]
		run, err := ApplyRun(p.Run, r)
		if err != nil {
			return fail(s, r, err.Error())
		}
		p.Run = run
		p.SpecialProcess = env.Special(run.StepKey)
		return put(s, id, derive(p))
	case r.Type == catalog.EquipmentEventBound:
		b, err := decodeBound(r.Data)
		if err != nil {
			return fail(s, r, err.Error())
		}
		ev, err := b.Subject()
		if err != nil {
			return fail(s, r, err.Error())
		}
		p := s.Runs[b.OperationRunID]
		if p.RunID == "" {
			p.RunID, p.ItemID, p.Origin = b.OperationRunID, r.ItemID, OriginSourceReported
		}
		return put(s, b.OperationRunID, ApplyBound(p, ev, b.Binding))
	case r.Type == catalog.DecisionNonconformityRegistered:
		var d struct {
			NcID           string `json:"nc_id"`
			WindowEventID  string `json:"violation_window_event_id"`
			OperationRunID string `json:"operation_run_id"`
		}
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return fail(s, r, err.Error())
		}
		p, ok := s.Runs[d.OperationRunID]
		if !ok {
			p = Profile{Run: Run{RunID: d.OperationRunID, ItemID: r.ItemID, Origin: OriginSourceReported}}
		}
		return put(s, d.OperationRunID, MarkViolation(p, d.NcID, d.WindowEventID))
	}
	return s
}

// put — новое состояние с профилем id (map копируется: прежнее состояние
// свёртки не меняется, AD-4).
func put(s State, id string, p Profile) State {
	runs := maps.Clone(s.Runs)
	if runs == nil {
		runs = map[string]Profile{}
	}
	runs[id] = p
	s.Runs = runs
	return s
}

func fail(s State, r kernel.Record, why string) State {
	errs := maps.Clone(s.Errors)
	if errs == nil {
		errs = map[string]string{}
	}
	errs[r.EventID] = why
	s.Errors = errs
	return s
}

// React вычисляет реакции модуля по состоянию после записи (AD-3, AD-40).
// У machinelogs нет реакций в потоке изделия: его записи с выводами —
// привязка события к выполнению и окно нарушения — пишет межизделийная
// стадия (Stage), а несоответствие окна — модуль nonconformity через свою
// функцию (FR-151). Поздние модули композиции (nonconformity, analysis)
// читают профили и отклонения из State через Upstream.
func React(s State, env Env, up Upstream) kernel.Output {
	_, _, _ = s, env, up
	return kernel.Output{}
}

// Guard — доменный гард операций модуля machinelogs (AD-39). Команд человека у
// модуля нет (факты оборудования приходят через приём), поэтому отказов нет.
func Guard(s State, env Env, up Upstream, cmd kernel.Command) error {
	_, _, _, _ = s, env, up, cmd
	return nil
}
