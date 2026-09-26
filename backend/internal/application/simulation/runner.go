package simulation

import (
	"context"
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
	for budget := s.d.Batch; budget > 0; budget-- {
		due := sim.NextDue(rp.plan, rp.points, st.Cursor)
		if due.At.After(target) {
			break
		}
		if due.Kind != sim.DuePoint || !rp.points[due.Index].Baseline {
			if err := s.tick(ctx, st, due.At); err != nil {
				return s.fail(ctx, st, err)
			}
		}
		switch due.Kind {
		case sim.DueEmissions:
			if err := s.deliver(ctx, st, rp, due.From, due.To); err != nil {
				return s.fail(ctx, st, err)
			}
			st.Cursor.Emissions = due.To
		case sim.DueAction:
			err := s.action(ctx, st, rp, due.Index)
			if errors.Is(err, errStop) {
				return s.d.Store.Save(ctx, st)
			}
			if err != nil {
				return s.fail(ctx, st, err)
			}
			st.Cursor.Actions = due.Index + 1
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
	if !fast && target.Sub(st.LastTick) >= time.Minute && !target.After(rp.plan.End) {
		if err := s.tick(ctx, st, target.Truncate(time.Second)); err != nil {
			return s.fail(ctx, st, err)
		}
	}
	return s.d.Store.Save(ctx, st)
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
	res, err := s.d.Gateway.Deliver(ctx, st.RunID, rp.plan.Emissions[from:to])
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
		obj := ""
		for _, k := range []string{"nc_id", "item_id", "incident_id", "lot_id"} {
			if v, ok := a.Params[k]; ok {
				obj, _ = s.expand(ctx, st, rp, fmt.Sprint(v))
				break
			}
		}
		st.Waiting = &Waiting{Action: i, Role: a.Role, Op: a.Operation, Object: obj, Title: actionTitle(a), Since: st.BasisSeq}
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
	if s.d.Settler != nil {
		_ = s.d.Settler.Settle(ctx, st.RunID)
	}
}

// waitDone — решение человека принято на столе роли: прогон продолжается.
func (s *Service) waitDone(ctx context.Context, st *RunState) (bool, error) {
	w := st.Waiting
	if s.d.Actor == nil {
		return false, nil
	}
	ok, seq, err := s.d.Actor.Decided(ctx, st.RunID, w.Op, w.Since)
	if err != nil || !ok {
		return false, err
	}
	rp, err := s.plan(ctx, st)
	if err != nil {
		return false, err
	}
	a := rp.plan.Actions[w.Action]
	st.Steps[stepKey(a)] = StepResult{Operation: a.Operation, At: a.At, Status: "done", Seq: seq, Detail: "решение принято на столе роли"}
	st.Cursor.Actions = w.Action + 1
	st.Waiting = nil
	st.State = StateRunning
	st.Clock = st.Clock.Resume(s.now())
	_, err = s.record(ctx, st, "simulation.run.resumed", a.At, map[string]any{"run_id": st.RunID})
	return err == nil, err
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
	defer func() { st.Steps[key] = res }()
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
		res.Status, res.Refusal, res.Detail = "failed", out.Code, "команда отклонена"
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
	m["command_id"] = rp.plan.IDs.CommandID(a.Seq)
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
			var until time.Time
			if a.Stand.Until != "" {
				if t, perr := sim.ParseTime(a.Stand.Until); perr == nil {
					until = t
				}
			}
			err = s.d.Stands.SetFault(ctx, a.Stand.Stand, *a.Stand, until)
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
