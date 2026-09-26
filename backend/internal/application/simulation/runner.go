package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	appjournal "ant/internal/application/journal"
	sim "ant/internal/domain/simulation"
)

// Раннер прогонов (роль stands, одна копия): ведёт виртуальные часы,
// отправляет события источников в обычный приём, исполняет шаги людей и
// служебные шаги, останавливается на решениях человека, проверяет табло
// (AD-26, AD-37, FR-129).

// Loop — раннер: каждые every продвигает все активные прогоны. Возвращается
// при отмене ctx.
func (s *Service) Loop(ctx context.Context, every time.Duration) error {
	if !s.live {
		return nil
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			runs, err := s.d.Store.List(ctx)
			if err != nil {
				continue
			}
			for _, r := range runs {
				if r.State == StateRunning || r.State == StateWaiting {
					_ = s.Step(ctx, r.RunID)
				}
			}
		}
	}
}

// RunToEnd продвигает прогон, пока он не закончится, не остановится на
// решении человека или не кончится бюджет шагов (автосверка, make sim-check).
func (s *Service) RunToEnd(ctx context.Context, runID string, maxSteps int) (*RunState, error) {
	for i := 0; i < maxSteps; i++ {
		if err := s.Step(ctx, runID); err != nil {
			return nil, err
		}
		st, err := s.load(ctx, runID)
		if err != nil {
			return nil, err
		}
		if st.State != StateRunning {
			return st, nil
		}
	}
	return s.load(ctx, runID)
}

// Step — один шаг прогона: всё, что наступило к виртуальному «сейчас»
// (интерактивно — по часам и скорости; в автосверке — сразу до следующей
// остановки), но не больше Deps.Batch событий и шагов.
func (s *Service) Step(ctx context.Context, runID string) error {
	if !s.live {
		return nil
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()
	st, err := s.load(ctx, runID)
	if err != nil {
		return err
	}
	if st.State != StateRunning && st.State != StateWaiting {
		return nil
	}
	rp, err := s.plan(ctx, st)
	if err != nil {
		return s.fail(ctx, st, err)
	}
	if st.State == StateWaiting {
		done, err := s.waitDone(ctx, st)
		if err != nil || !done {
			return s.d.Store.Save(ctx, st)
		}
	}
	real := s.now()
	target := st.Clock.Now(real)
	fast := st.Mode != ModeInteractive
	if fast {
		target = rp.plan.End
	}
	// Д-85: история до живой части проигрывается сразу, без часов; часы
	// прогона пойдут от начала живой части, когда история кончится.
	catching := !fast && !st.Live && !rp.plan.LiveFrom.IsZero()
	if catching {
		target = rp.plan.LiveFrom.Add(-time.Nanosecond)
	}
	afterAction := true
	for budget := s.d.Batch; budget > 0; budget-- {
		due := sim.NextDue(rp.plan, rp.points, st.Cursor)
		if due.At.After(target) {
			if catching {
				if err := s.goLive(ctx, st, rp); err != nil {
					return s.fail(ctx, st, err)
				}
				return s.d.Store.Save(ctx, st)
			}
			break
		}
		if due.Kind != sim.DuePoint || !rp.points[due.Index].Baseline {
			if err := s.tick(ctx, st, due.At); err != nil {
				return s.fail(ctx, st, err)
			}
		}
		switch due.Kind {
		case sim.DueEmissions:
			// После решения человека (регистрация, носитель) привязка события
			// по носителю (AD-41) должна видеть его результат: сначала движок
			// догоняет журнал (эпик 16). Между доставками — без ожидания:
			// пачки остаются пачками, неоднозначную привязку доделывает стадия.
			if afterAction {
				s.settle(ctx, st)
				afterAction = false
			}
			if err := s.deliver(ctx, st, rp, due.From, due.To); err != nil {
				return s.fail(ctx, st, err)
			}
			st.Cursor.Emissions = due.To
			// Носитель, нанесённый устройством (перемаркировка DM после
			// станка), — такое же основание привязки, как решение человека:
			// следующие события по этому носителю ждут, пока движок его
			// обработает (иначе результат КТ-2 уходит в очередь привязки и
			// ЗТ-2 не видит его — «нет результата контроля»).
			if carrierChanged(rp.plan.Emissions[due.From:due.To]) {
				afterAction = true
			}
		case sim.DueAction:
			err := s.action(ctx, st, rp, due.Index)
			if errors.Is(err, errStop) {
				return s.d.Store.Save(ctx, st)
			}
			if err != nil {
				return s.fail(ctx, st, err)
			}
			st.Cursor.Actions = due.Index + 1
			afterAction = true
		case sim.DuePoint:
			s.evaluate(ctx, st, rp, rp.points[due.Index])
			st.Cursor.Points = due.Index + 1
		case sim.DueEnd:
			t := s.now()
			st.FinishedAt = &t
			st.State = StateComplete
			if _, err := s.record(ctx, st, "simulation.run.finished", rp.plan.End, map[string]any{"run_id": st.RunID, "outcome": "completed"}); err != nil {
				return s.fail(ctx, st, err)
			}
			return s.d.Store.Save(ctx, st)
		}
	}
	// в интерактиве доменное «сейчас» идёт и между событиями: тик раз в
	// виртуальную минуту, чтобы столы видели время сценария (AD-37)
	if !fast && !catching && target.Sub(st.LastTick) >= time.Minute && !target.After(rp.plan.End) {
		if err := s.tick(ctx, st, target.Truncate(time.Second)); err != nil {
			return s.fail(ctx, st, err)
		}
	}
	return s.d.Store.Save(ctx, st)
}

// carrierChanged — в пачке есть нанесение или смена носителя изделия.
func carrierChanged(batch []sim.Emission) bool {
	for _, e := range batch {
		if strings.HasPrefix(e.EventType, "item.carrier.") {
			return true
		}
	}
	return false
}

func (s *Service) fail(ctx context.Context, st *RunState, err error) error {
	st.State = StateFailed
	st.Error = err.Error()
	t := s.now()
	st.FinishedAt = &t
	_, _ = s.record(ctx, st, "simulation.run.finished", st.LastTick, map[string]any{"run_id": st.RunID, "outcome": "failed"})
	if serr := s.d.Store.Save(ctx, st); serr != nil {
		return errors.Join(err, serr)
	}
	return err
}

// deliver — события источников [from, to) в обычный приём (AD-26).
func (s *Service) deliver(ctx context.Context, st *RunState, rp *runPlan, from, to int) error {
	if s.d.Gateway == nil {
		return nil
	}
	res, err := s.d.Gateway.Deliver(ctx, st.RunID, actualRuns(st, rp.plan.Emissions[from:to]))
	if err != nil {
		return fmt.Errorf("доставка событий %d…%d: %w", from, to, err)
	}
	for _, r := range res {
		st.Delivered[r.Status]++
	}
	return nil
}

// action — шаг прогона: решение человека, сбой stand-а, подделка.
func (s *Service) action(ctx context.Context, st *RunState, rp *runPlan, i int) error {
	a := rp.plan.Actions[i]
	key := stepKey(a)
	switch a.Kind {
	case sim.ActionStand:
		return s.stand(ctx, st, a, key)
	case sim.ActionTamper:
		return s.tamper(ctx, st, rp, a, key)
	}
	// Решение человека и поиск объектов системы ({ref:…}) — по состоянию
	// после обработки всего доставленного (эпик 16: иначе команда видит
	// изделие без только что пришедших событий).
	s.settle(ctx, st)
	if st.Mode == ModeInteractive && a.Stop && st.Waiting == nil {
		// FR-129: сценарий ждёт решения на столе роли; часы стоят.
		obj := s.waitObject(ctx, st, rp, a)
		since := st.BasisSeq
		if st.Live {
			// живая часть (Д-85): человек мог нажать раньше, чем прогон дошёл до шага
			since = st.LiveSeq
		}
		st.Waiting = &Waiting{Action: i, Role: roleOf(rp, a.Role), Op: a.Operation, Object: obj, Title: actionTitle(a), Since: since}
		st.State = StateWaiting
		st.Clock = st.Clock.Pause(s.now())
		st.Steps[key] = StepResult{Operation: a.Operation, At: a.At, Status: "waiting"}
		if _, err := s.record(ctx, st, "simulation.run.paused", a.At, map[string]any{"run_id": st.RunID, "reason": "waiting_for_decision"}); err != nil {
			return err
		}
		return errStop
	}
	s.decide(ctx, st, rp, a, key)
	return nil
}

// settle — дождаться, пока воркер и стадия обработают доставленное прогоном
// (порт Settler); без порта или по сроку — как есть.
func (s *Service) settle(ctx context.Context, st *RunState) {
	if s.d.Settler == nil {
		return
	}
	if err := s.d.Settler.Settle(ctx, st.RunID); err != nil && s.d.Log != nil {
		s.d.Log.Warn("прогон: движок не догнал журнал — продолжаю", "run_id", st.RunID, "err", err)
	}
}

// waitDone — решение человека принято на столе роли: прогон продолжается.
func (s *Service) waitDone(ctx context.Context, st *RunState) (bool, error) {
	w := st.Waiting
	if s.d.Actor == nil {
		return false, nil
	}
	seqs, err := s.d.Actor.Decided(ctx, st.RunID, w.Op, w.Object, w.Since)
	if err != nil {
		return false, err
	}
	// одна запись закрывает одну остановку (Consumed)
	seq := int64(-1)
	for _, x := range seqs {
		if !slices.Contains(st.Consumed, x) {
			seq = x
			break
		}
	}
	if seq < 0 {
		return false, nil
	}
	rp, err := s.plan(ctx, st)
	if err != nil {
		return false, err
	}
	a := rp.plan.Actions[w.Action]
	if a.Operation == opStart {
		// Фактический id выполнения не прочитан — остановка не снимается:
		// иначе ток, КТ-3 и рентген уйдут на плановый id, ЗТ-3 не построится
		// и следующие остановки не снимутся никогда. Повтор — на следующем тике.
		if err := s.aliasRun(ctx, st, rp, a, seq); err != nil {
			if s.d.Log != nil {
				s.d.Log.Warn("прогон: «Начать» принято, фактический id выполнения не прочитан — жду", "run_id", st.RunID,
					"step", stepKey(a), "seq", seq, "err", err)
			}
			return false, nil
		}
	}
	st.Consumed = append(st.Consumed, seq)
	st.Steps[stepKey(a)] = StepResult{Operation: a.Operation, At: a.At, Status: "done", Seq: seq, Detail: "решение принято на столе роли"}
	st.Cursor.Actions = w.Action + 1
	st.Waiting = nil
	st.State = StateRunning
	st.Clock = st.Clock.Resume(s.now())
	_, err = s.record(ctx, st, "simulation.run.resumed", a.At, map[string]any{"run_id": st.RunID})
	return err == nil, err
}

// Операции выполнения, у которых id выполнения выдаёт стол исполнителя.
const (
	opStart  = "process.operation.start"
	opFinish = "process.operation.finish"
)

// aliasRun — «Начать» нажал человек: фактический operation_run_id записи
// (терминал выдаёт свой) вместо планового для всех следующих событий и
// решений прогона по этому выполнению. Ошибка — у шага есть плановый id, а
// фактический не прочитан (запись ещё не видна, чтение не удалось, нет data):
// остановку снимать нельзя. Без порта чтения подменять нечем — не ошибка.
func (s *Service) aliasRun(ctx context.Context, st *RunState, rp *runPlan, a sim.Action, seq int64) error {
	v, ok := a.Body["operation_run_id"]
	if !ok || s.d.Probe == nil {
		return nil
	}
	planned, err := s.expand(ctx, st, rp, fmt.Sprint(v))
	if err != nil {
		return fmt.Errorf("плановый id выполнения: %w", err)
	}
	if planned == "" {
		return errors.New("плановый id выполнения пуст")
	}
	doc, err := s.d.Probe.Read(ctx, "journal.entry.read", map[string]string{"seq": itoa(seq)}, st.RunID)
	if err != nil {
		return fmt.Errorf("journal.entry.read seq %d: %w", seq, err)
	}
	vals, err := sim.Extract(doc, "/data/operation_run_id")
	if err != nil || len(vals) != 1 {
		return fmt.Errorf("в записи seq %d нет data.operation_run_id", seq)
	}
	actual, _ := vals[0].(string)
	if actual == "" {
		return fmt.Errorf("в записи seq %d пустой data.operation_run_id", seq)
	}
	if actual == planned {
		return nil
	}
	if st.Runs == nil {
		st.Runs = map[string]string{}
	}
	st.Runs[planned] = actual
	return nil
}

// actualRuns — события прогона с фактическими id выполнений вместо плановых.
func actualRuns(st *RunState, batch []sim.Emission) []sim.Emission {
	if len(st.Runs) == 0 {
		return batch
	}
	out := make([]sim.Emission, len(batch))
	for i, e := range batch {
		for planned, actual := range st.Runs {
			p, _ := json.Marshal(planned)
			if bytes.Contains(e.Event, p) {
				q, _ := json.Marshal(actual)
				e.Event = bytes.ReplaceAll(e.Event, p, q)
			}
		}
		out[i] = e
	}
	return out
}

// waitObject — объект ожидания: изделие, несоответствие, инцидент, партия,
// пост или оборудование шага (из параметров, иначе из тела команды) —
// решение человека засчитывается только над ним (не любое решение этого типа).
func (s *Service) waitObject(ctx context.Context, st *RunState, rp *runPlan, a sim.Action) string {
	if a.Operation == opFinish && a.Item != "" {
		// «Выполнено» — по изделию, как «Начать»: id выполнения выдаёт стол
		// исполнителя, плановый id прогона в журнале может не встретиться.
		obj, _ := s.expand(ctx, st, rp, "{item:"+a.Item+"}")
		return obj
	}
	for _, m := range []map[string]any{a.Params, a.Body} {
		for _, k := range waitKeys {
			if v, ok := m[k]; ok {
				obj, _ := s.expand(ctx, st, rp, fmt.Sprint(v))
				return obj
			}
		}
	}
	return ""
}

// goLive — история проиграна (Д-85): часы прогона встают на начало живой
// части и идут от «сейчас»; решения людей ищутся после этой точки журнала.
func (s *Service) goLive(ctx context.Context, st *RunState, rp *runPlan) error {
	s.settle(ctx, st)
	if err := s.tick(ctx, st, rp.plan.LiveFrom); err != nil {
		return err
	}
	st.Live = true
	st.LiveSeq = st.BasisSeq
	st.Clock = st.Clock.Rebase(s.now(), rp.plan.LiveFrom)
	_, err := s.record(ctx, st, "simulation.run.resumed", rp.plan.LiveFrom, map[string]any{"run_id": st.RunID})
	return err
}

func stepKey(a sim.Action) string {
	if a.Label != "" {
		return a.Label
	}
	return fmt.Sprintf("%s#%d", a.Operation, a.Seq)
}

// decide — решение от имени демо-персоны теми же операциями API (demo-signer,
// AD-26): подписывается только шаг из определения — с его параметрами и телом.
func (s *Service) decide(ctx context.Context, st *RunState, rp *runPlan, a sim.Action, key string) {
	res := StepResult{Operation: a.Operation, At: a.At}
	defer func() {
		st.Steps[key] = res
		if s.d.Log != nil && (res.Status == "failed" || res.Status == "skipped") {
			s.d.Log.Warn("прогон: шаг не выполнен", "run_id", st.RunID, "step", key, "operation", a.Operation,
				"status", res.Status, "refusal", res.Refusal, "detail", res.Detail)
		}
	}()
	if s.d.Actor == nil {
		res.Status, res.Detail = "skipped", "нет порта решений (demo-signer)"
		return
	}
	params, err := s.params(ctx, st, rp, a.Params)
	if err != nil {
		res.Status, res.Detail = "skipped", err.Error()
		return
	}
	body, err := s.body(ctx, st, rp, a)
	if err != nil {
		res.Status, res.Detail = "skipped", err.Error()
		return
	}
	// Прогон в контексте команды (AD-38): факты и решения демо-подписанта
	// принадлежат прогону, доменное «сейчас» — часы прогона (AD-37).
	out, err := s.d.Actor.Act(appjournal.WithRun(ctx, st.RunID), a.Actor, a.Operation, params, body)
	switch {
	case errors.Is(err, ErrUnavailable):
		res.Status, res.Detail = "skipped", "операция "+a.Operation+" пока не отвечает (модуль в работе)"
		return
	case err != nil:
		res.Status, res.Detail = "failed", err.Error()
		return
	}
	res.Seq = out.Seq
	switch {
	case out.Code != "" && a.Refusal != "":
		res.Status, res.Refusal = "refused", out.Code
		if out.Code != a.Refusal {
			res.Detail = "ожидался отказ " + a.Refusal
		}
	case out.Code != "":
		res.Status, res.Refusal, res.Detail = "failed", out.Code, strings.TrimSpace("команда отклонена: "+out.Detail)
	case a.Refusal != "":
		res.Status, res.Detail = "failed", "ожидался отказ "+a.Refusal+", команда принята"
	default:
		res.Status = "done"
	}
	if a.Binds != "" && out.Seq > 0 && out.Code == "" {
		s.bind(ctx, st, a.Binds, out.Seq)
	}
}

// bind — ID изделия, рождённого решением (регистрация): из записи журнала.
func (s *Service) bind(ctx context.Context, st *RunState, item string, seq int64) {
	if s.d.Probe == nil {
		return
	}
	doc, err := s.d.Probe.Read(ctx, "journal.entry.read", map[string]string{"seq": itoa(seq)}, st.RunID)
	if err != nil {
		return
	}
	if v, err := sim.Extract(doc, "/item_id"); err == nil && len(v) == 1 {
		if id, ok := v[0].(string); ok && id != "" {
			st.Items[item] = id
		}
	}
}

// body — тело команды: заголовок команды (AD-7, AD-39) и тело шага с подстановками.
func (s *Service) body(ctx context.Context, st *RunState, rp *runPlan, a sim.Action) (map[string]any, error) {
	b, err := s.expandAny(ctx, st, rp, a.Body)
	if err != nil {
		return nil, err
	}
	m, _ := b.(map[string]any)
	if m == nil {
		m = map[string]any{}
	}
	m["command_id"] = rp.plan.IDs.ActionCommandID(a.Label, a.Seq)
	if _, ok := m["basis_seq"]; !ok {
		m["basis_seq"] = 0
	}
	if _, ok := m["policy_seq"]; !ok {
		m["policy_seq"] = 0
	}
	return m, nil
}

func (s *Service) expandAny(ctx context.Context, st *RunState, rp *runPlan, v any) (any, error) {
	switch x := v.(type) {
	case string:
		return s.expand(ctx, st, rp, x)
	case []any:
		out := make([]any, len(x))
		for i := range x {
			e, err := s.expandAny(ctx, st, rp, x[i])
			if err != nil {
				return nil, err
			}
			out[i] = e
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(x))
		for _, k := range slices.Sorted(maps.Keys(x)) {
			e, err := s.expandAny(ctx, st, rp, x[k])
			if err != nil {
				return nil, err
			}
			out[k] = e
		}
		return out, nil
	}
	return v, nil
}

// stand — сбой stand-а через служебный порт (AD-18).
func (s *Service) stand(ctx context.Context, st *RunState, a sim.Action, key string) error {
	res := StepResult{Operation: "stand:" + a.Stand.Stand, At: a.At, Status: "done", Detail: a.Stand.Fault}
	if s.d.Stands == nil {
		res.Status, res.Detail = "skipped", "нет служебного порта stand-ов"
	} else {
		var err error
		if a.Stand.Clear {
			err = s.d.Stands.ClearFaults(ctx, a.Stand.Stand)
		} else {
			// until определения — время прогона, а stand живёт по InfraClock:
			// сбой снимает шаг «снять сбой» генератора в момент until
			err = s.d.Stands.SetFault(ctx, a.Stand.Stand, *a.Stand, time.Time{})
		}
		if err != nil {
			res.Status, res.Detail = "skipped", err.Error()
		}
	}
	st.Steps[key] = res
	return nil
}

// tamper — подделка в обход системы (S09, F25): только демо-инструмент в
// профилях fixtures и demo (AD-26).
func (s *Service) tamper(ctx context.Context, st *RunState, rp *runPlan, a sim.Action, key string) error {
	res := StepResult{Operation: "tamper:" + a.Tamper.Kind, At: a.At, Status: "done"}
	switch {
	case s.d.Profile == "prod":
		res.Status, res.Detail = "skipped", "в профиле prod демо-инструмента нет (AD-26)"
	case s.d.Tamper == nil:
		res.Status, res.Detail = "skipped", "демо-инструмент cmd/tamper не подключён"
	default:
		target := rp.plan.IDs.Labels[a.Tamper.Target]
		if target == "" && !strings.Contains(a.Tamper.Target, ":") {
			res.Status, res.Detail = "skipped", "нет события с меткой "+a.Tamper.Target
			break
		}
		if err := s.d.Tamper.Apply(ctx, st.RunID, *a.Tamper, target); err != nil {
			res.Status, res.Detail = "skipped", err.Error()
		}
	}
	st.Steps[key] = res
	return nil
}
