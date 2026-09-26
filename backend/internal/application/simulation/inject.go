package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	sim "ant/internal/domain/simulation"
)

// Цифровой стенд (FR-152, AD-26; эпик 36): кнопки-инъекции страницы тестовых
// сценариев поверх идущего прогона. Кнопка строит заготовленное событие сбоя
// (domain/simulation.PlanInjection) и отправляет его тем же путём, что и
// события прогона: stand → обычный приём (порт Gateway); потеря данных ещё и
// включает сбой stand-а оборудования через служебный порт (Stands, AD-18);
// подделка — только демо-инструментом cmd/tamper (порт Tamperer, профили
// fixtures и demo). Нажатие — служебная запись simulation.injection.applied
// в журнале прогона; ожидаемый видимый результат — строки табло из
// scenarios/expected/stand/‹кнопка›.yaml (хранятся отдельно от входа, кейс
// §5.1), проверяемые теми же операциями API, что показывают столы.

// InjectionState — нажатие кнопки в прогоне: цель, события, строки табло.
type InjectionState struct {
	N         int       `json:"n"`
	Injection string    `json:"injection"`
	CommandID string    `json:"command_id,omitempty"`
	At        time.Time `json:"at"`
	Target    string    `json:"target_event_id,omitempty"`
	// EventIDs — новые записи журнала, внесённые кнопкой (повтор — ни одной).
	EventIDs []string `json:"event_ids,omitempty"`
	// Seq — запись simulation.injection.applied.
	Seq int64 `json:"seq"`
	// Step — шаг сценария, на котором нажата кнопка.
	Step int            `json:"step"`
	Rows []InjectionRow `json:"rows,omitempty"`
}

// InjectionRow — строка табло кнопки: утверждение с подставленными целью,
// изделием и источником нажатия.
type InjectionRow struct {
	Scenario  string        `json:"scenario"`
	Label     string        `json:"label"`
	MustNot   bool          `json:"must_not,omitempty"`
	Assertion sim.Assertion `json:"assertion"`
}

// injectionTexts — кнопки пульта: название и что будет видно.
var injectionTexts = map[sim.InjectionKind][2]string{
	sim.InjectDuplicate: {"Прислать повтор события",
		"Тот же источник и номер ещё раз (по умолчанию — последнее событие изделия): приём отвечает «повтор», ничего не задвоилось (F01, S06)."},
	sim.InjectLate: {"Прислать опоздавшее событие",
		"Событие случилось раньше цели, пришло сейчас: встаёт на своё время, история изделия перестроена, видно «поздно на …» (F03, S07)."},
	sim.InjectCorruptFrame: {"Испортить кадр",
		"Кадр камеры с качеством 0,3 (по умолчанию — последний кадр): «оценка невозможна», изделие не стало «годным» (F08, S04)."},
	sim.InjectMachineFault: {"Сбой станка / ток вне уставки",
		"Ток 176 А при уставке 160 ± 10 на последней сварке (или сварке изделия цели): окно нарушения режима, область риска изменилась (S05, S07)."},
	sim.InjectDataLoss: {"Потерять кусок данных",
		"Источник сварочного поста теряет записи: разрыв номеров учтён, полнота приёма ниже 100 % (F07, S04)."},
	sim.InjectTamper: {"Подделать запись в обход системы",
		"Правка записи журнала в хранилище демо-инструментом cmd/tamper (только fixtures и demo): проверка целостности находит подмену (F25, S09)."},
	sim.InjectLightChange: {"Сменить свет на камере",
		"Три кадра подряд с качеством 0,55, анализатор по-прежнему отвечает «признаков нет»: дрейф → паспорт анализатора приостановлен, контроль ручной, начальнику ОТК — задача (FR-101)."},
}

// StandFaultFor — сколько держится сбой stand-а оборудования при потере данных (по часам stand-а).
const StandFaultFor = time.Minute

func (s *Service) demoProfile() bool { return s.d.Profile == "demo" || s.d.Profile == "fixtures" }

// Injections — кнопки цифрового стенда для прогона (simulation.injection.list):
// доступны, пока прогон не закончен; подделка — только в профилях fixtures и
// demo. Цель у всех необязательна: пусто — последнее подходящее событие.
func (s *Service) Injections(ctx context.Context, runID string) (InjectionList, error) {
	if !s.live {
		return s.Unimplemented.Injections(ctx, runID)
	}
	st, err := s.load(ctx, runID)
	if err != nil {
		return InjectionList{}, err
	}
	final := isFinal(st.State)
	out := InjectionList{Items: []Injection{}}
	for _, k := range sim.InjectionKinds {
		t := injectionTexts[k]
		out.Items = append(out.Items, Injection{Injection: string(k), Title: t[0], Description: t[1],
			Available: !final && (k != sim.InjectTamper || s.demoProfile())})
	}
	return out, nil
}

// ApplyInjection — нажать кнопку цифрового стенда (simulation.injection.apply,
// FR-152): событие сбоя в обычный приём, запись simulation.injection.applied,
// строки табло «ожидалось → получилось». Повтор с тем же command_id — прежний
// ответ (AD-7).
func (s *Service) ApplyInjection(ctx context.Context, runID string, in ApplyInjection) (platform.Receipt, error) {
	if !s.live {
		return s.Unimplemented.ApplyInjection(ctx, runID, in)
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()
	st, err := s.load(ctx, runID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if in.CommandID != "" {
		for _, x := range st.Injections {
			if x.CommandID == in.CommandID {
				return platform.Receipt{CommandID: in.CommandID, Seq: x.Seq, EventIDs: slices.Clone(x.EventIDs), RecordedAt: x.At, Replayed: true}, nil
			}
		}
	}
	if isFinal(st.State) {
		return platform.Receipt{}, stateConflict(st, "инъекция")
	}
	kind := sim.InjectionKind(in.Injection)
	if !slices.Contains(sim.InjectionKinds, kind) {
		return platform.Receipt{}, refusal(st, "injection", "неизвестная кнопка цифрового стенда "+in.Injection)
	}
	if kind == sim.InjectTamper && !s.demoProfile() {
		return platform.Receipt{}, refusal(st, "injection", "в профиле "+s.d.Profile+" демо-инструмента cmd/tamper нет: подделка — только в fixtures и demo (AD-26)")
	}
	rp, err := s.plan(ctx, st)
	if err != nil {
		return platform.Receipt{}, err
	}
	at := s.injectAt(st, rp)
	n := len(st.Injections) + 1
	inj, err := sim.PlanInjection(sim.InjectionInput{Plan: rp.plan, World: rp.world, Kind: kind, N: n, Target: in.TargetEventID,
		Delivered: st.Cursor.Emissions, At: at, StandSeq: st.StandSeq})
	var ie *sim.InjectionError
	if errors.As(err, &ie) {
		return platform.Receipt{}, refusal(st, "target_event_id", ie.Reason)
	}
	if err != nil {
		return platform.Receipt{}, err
	}
	x := InjectionState{N: n, Injection: string(kind), CommandID: in.CommandID, At: at, Step: st.Cursor.Actions}
	if inj.Target != nil {
		x.Target = inj.Target.EventID
	}
	x.Rows, err = s.injectionRows(ctx, st, rp, inj, x)
	if err != nil {
		return platform.Receipt{}, err
	}

	// Момент «до»: всё доставленное обработано, значения — до нажатия.
	s.settleRun(ctx, st)
	for _, r := range x.Rows {
		if r.Assertion.Baseline == "" {
			continue
		}
		if v, found, err := s.read(ctx, st, rp, r.Assertion.Check); err == nil && found {
			st.Baselines[r.Assertion.ID] = v
		}
	}

	step := StepResult{Operation: "inject:" + string(kind), At: at, Status: "done", Detail: inj.Detail}
	var notes []string
	if len(inj.Emissions) > 0 {
		if s.d.Gateway == nil {
			step.Status, step.Detail = "skipped", "нет порта приёма"
		} else {
			res, err := s.d.Gateway.Deliver(ctx, st.RunID, inj.Emissions)
			if err != nil {
				return platform.Receipt{}, fmt.Errorf("кнопка %s: доставка в приём: %w", kind, err)
			}
			for _, r := range res {
				st.Delivered[r.Status]++
				step.Delivered = append(step.Delivered, string(r.Status))
				if r.Status == DeliveryAccepted {
					x.EventIDs = append(x.EventIDs, r.EventID)
				}
				if step.Seq == 0 && r.Seq > 0 {
					step.Seq = r.Seq
				}
			}
		}
	}
	if a := inj.Stand; a != nil && s.d.Stands != nil {
		// Служебный порт stand-ов (AD-18): stand оборудования, если он есть в
		// роли stands, тоже теряет сообщения; поток прогона теряет их выше.
		if err := s.d.Stands.SetFault(ctx, a.Stand, *a, s.now().Add(StandFaultFor)); err != nil {
			notes = append(notes, "stand "+a.Stand+": "+err.Error()+" — потеря только в потоке прогона")
		} else {
			notes = append(notes, "служебный порт stand-ов: "+a.Stand+" — сбой "+a.Fault+" на "+StandFaultFor.String())
		}
	}
	if t := inj.Tamper; t != nil {
		tp := s.d.Tamper
		if tp == nil {
			tp = PendingTamperer{}
		}
		if err := tp.Apply(ctx, st.RunID, *t, inj.Target.EventID); err != nil {
			step.Status = "skipped"
			notes = append(notes, err.Error())
		}
	}
	if len(notes) > 0 {
		step.Detail = strings.Join(append([]string{step.Detail}, notes...), "; ")
	}
	for k, v := range inj.StandSeq {
		if st.StandSeq == nil {
			st.StandSeq = map[string]int64{}
		}
		st.StandSeq[k] = v
	}
	// press — номер нажатия: два одинаковых нажатия на одном тике — две
	// записи, а не повтор (event_id служебной записи — от её содержимого).
	data := map[string]any{"run_id": st.RunID, "injection": string(kind), "press": n}
	if x.Target != "" {
		data["target_event_id"] = x.Target
	}
	if len(x.EventIDs) > 0 {
		data["event_ids"] = slices.Clone(x.EventIDs)
	}
	if x.Seq, err = s.record(ctx, st, "simulation.injection.applied", at, data); err != nil {
		return platform.Receipt{}, err
	}
	st.Steps[injectionStep(n)] = step

	// Результат: воркер и стадия догнали журнал — строки табло теми же операциями.
	s.settleRun(ctx, st)
	for _, r := range x.Rows {
		if step.Status == "skipped" {
			// кнопка не сработала (нет приёма, нет демо-инструмента): проверять
			// нечего — строка ждёт, а не «не совпало»
			st.Rows[r.Assertion.ID] = sim.Result{Status: sim.StatusPending, Detail: "кнопка не сработала: " + step.Detail}
			continue
		}
		st.Rows[r.Assertion.ID] = s.check(ctx, st, rp, row{scenario: r.Scenario, label: r.Label, mustNot: r.MustNot, at: at, a: r.Assertion})
	}
	st.Injections = append(st.Injections, x)
	if err := s.d.Store.Save(ctx, st); err != nil {
		return platform.Receipt{}, err
	}
	return platform.Receipt{CommandID: in.CommandID, Seq: x.Seq, EventIDs: slices.Clone(x.EventIDs), RecordedAt: at}, nil
}

// settleRun — дождаться, пока воркер и стадия обработают доставленное прогоном.
func (s *Service) settleRun(ctx context.Context, st *RunState) {
	if s.d.Settler != nil {
		_ = s.d.Settler.Settle(ctx, st.RunID)
	}
}

func injectionStep(n int) string { return "inject:" + strconv.Itoa(n) }

// refusal — кнопку нельзя применить: 400 с понятным пояснением.
func refusal(st *RunState, field, detail string) error {
	e := platform.Fail(errcodes.ApiValidationFailed, "field", field, "run_id", st.RunID)
	e.Detail = detail
	return e
}

// injectAt — доменное «сейчас» прогона для кнопки (AD-37): последний тик
// журнала прогона. Раньше него события прогона уже доставлены, позже — ещё
// нет: кнопка не двигает часы сама, иначе служебные записи прогона, которые
// ещё не наступили, пришлось бы писать задним числом (recorded_at не убывает).
func (s *Service) injectAt(st *RunState, rp *runPlan) time.Time {
	if st.LastTick.IsZero() {
		return rp.plan.Start
	}
	return st.LastTick
}

// injectionRows — строки табло кнопки из scenarios/expected/stand/‹кнопка›.yaml
// с подставленными целью, изделием и источником нажатия. Строка, которой
// нечего проверять (у цели нет изделия), не ставится.
func (s *Service) injectionRows(ctx context.Context, st *RunState, rp *runPlan, inj *sim.Injection, x InjectionState) ([]InjectionRow, error) {
	ex, ok, err := s.d.Definitions.Expected(ctx, "stand/"+string(inj.Kind))
	if err != nil || !ok {
		return nil, err
	}
	vals := map[string]string{"step": injectionStep(inj.N), "target": x.Target, "source": inj.Source}
	if inj.Item != "" {
		vals["item"] = "{item:" + inj.Item + "}"
	}
	if inj.Equipment != "" {
		vals["equipment"] = rp.plan.IDs.Equipment(inj.Equipment)
	}
	if len(inj.Emissions) > 0 {
		e := inj.Emissions[0]
		vals["event"] = e.EventID
		// ожидания пишутся во времени определения: табло сдвигает их на прогон (AD-38)
		vals["occurred"] = sim.FormatTime(e.OccurredAt.Add(-rp.plan.Shift))
	}
	if inj.Kind == sim.InjectDataLoss {
		gaps := 1
		for _, p := range st.Injections {
			if p.Injection == string(sim.InjectDataLoss) {
				gaps++
			}
		}
		vals["gaps"] = strconv.Itoa(gaps)
	}
	title := injectionTexts[inj.Kind][0]
	label := fmt.Sprintf("кнопка стенда №%d «%s»", inj.N, title)
	var out []InjectionRow
	for _, cp := range ex.Checkpoints {
		for _, a := range cp.Assertions {
			c, ok := substInject(a.Check, vals)
			if !ok {
				continue
			}
			a.Check = c
			a.ID = fmt.Sprintf("%s#%d", a.ID, inj.N)
			if a.Baseline != "" {
				a.Baseline = "apply"
			}
			out = append(out, InjectionRow{Scenario: ex.Scenario, Label: label, MustNot: cp.MustNot, Assertion: a})
		}
	}
	return out, nil
}

// substInject подставляет {inj:‹имя›} в параметры, путь и ожидаемое значение;
// ok = false — у нажатия нет нужного значения.
func substInject(c sim.Check, vals map[string]string) (sim.Check, bool) {
	ok := true
	var walk func(v any) any
	walk = func(v any) any {
		switch x := v.(type) {
		case string:
			out, good := substString(x, vals)
			ok = ok && good
			return out
		case []any:
			r := make([]any, len(x))
			for i := range x {
				r[i] = walk(x[i])
			}
			return r
		case map[string]any:
			r := make(map[string]any, len(x))
			for k, e := range x {
				r[k] = walk(e)
			}
			return r
		}
		return v
	}
	out := c
	out.Path = walk(c.Path).(string)
	if c.Params != nil {
		out.Params = walk(c.Params).(map[string]any)
	}
	out.Value = walk(c.Value)
	return out, ok
}

func substString(s string, vals map[string]string) (string, bool) {
	ok := true
	for strings.Contains(s, "{inj:") {
		i := strings.Index(s, "{inj:")
		j := strings.Index(s[i:], "}")
		if j < 0 {
			break
		}
		name := s[i+5 : i+j]
		v, has := vals[name]
		if !has || v == "" {
			ok = false
		}
		s = s[:i] + v + s[i+j+1:]
	}
	return s, ok
}

// injectionRow — строка табло кнопки.
func injectionRow(st *RunState, x InjectionState, r InjectionRow) BoardRow {
	a := r.Assertion
	exp, _ := json.Marshal(a.Check.Value)
	at := x.At
	br := BoardRow{AssertionID: a.ID, Title: a.What, OperationID: a.Check.Operation, Path: a.Check.Path, Expected: string(exp),
		ScenarioID: r.Scenario, Checkpoint: r.Label, MustNot: r.MustNot, Mapping: a.Mapping, Step: x.Step, At: &at,
		Status: string(sim.StatusPending)}
	if res, ok := st.Rows[a.ID]; ok {
		br.Status, br.Detail = string(res.Status), res.Detail
		if res.Actual != "" {
			v := res.Actual
			br.Actual = &v
		}
	}
	if br.Detail == "" && a.Note != "" {
		br.Detail = a.Note
	}
	return br
}
