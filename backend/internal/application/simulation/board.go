package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ant/internal/application/platform"
	sim "ant/internal/domain/simulation"
)

// runPlan — план прогона и его табло: точки проверки по времени и строки
// утверждений (AD-26, FR-108).
type runPlan struct {
	plan *sim.Plan
	refs map[string]sim.Ref
	// world — мир прогона: уставки и линии для кнопок цифрового стенда.
	world sim.World
	// notes — примечания строк карточек по метке (названия событий в плане).
	notes  map[string]string
	rows   []row
	points []sim.Point
	// byPoint — строки точки (номер Point.Index → номера строк).
	byPoint map[int][]int
}

// row — строка табло: утверждение карточки и момент проверки.
type row struct {
	scenario string
	label    string
	mustNot  bool
	// at — момент проверки (со сдвигом прогона); never — момент не наступит в этом прогоне.
	at       time.Time
	never    bool
	baseline time.Time
	a        sim.Assertion
}

func (rp *runPlan) add(ex sim.Expected) {
	for _, cp := range ex.Checkpoints {
		at, never := rp.moment(cp.At)
		for _, a := range cp.Assertions {
			r := row{scenario: ex.Scenario, label: cp.Label, mustNot: cp.MustNot, at: at, never: never, a: a}
			if a.Baseline != "" {
				if t, nv := rp.moment(a.Baseline); !nv {
					r.baseline = t
				}
			}
			rp.rows = append(rp.rows, r)
		}
	}
}

// moment — момент точки: время определения со сдвигом прогона, «end» — конец
// прогона, «standalone» — только в отдельном прогоне (не наступит).
func (rp *runPlan) moment(s string) (time.Time, bool) {
	switch s {
	case "end":
		return rp.plan.End, false
	case "standalone", "":
		return time.Time{}, true
	}
	t, err := sim.ParseTime(s)
	if err != nil {
		return time.Time{}, true
	}
	t = t.Add(rp.plan.Shift)
	if t.After(rp.plan.End) {
		t = rp.plan.End
	}
	return t, false
}

// index — точки: одна на момент проверки и одна на момент «до» строк с delta/unchanged.
func (rp *runPlan) index() {
	rp.byPoint = map[int][]int{}
	type key struct {
		at       time.Time
		baseline bool
	}
	seen := map[key]int{}
	add := func(at time.Time, baseline bool, i int) {
		k := key{at, baseline}
		idx, ok := seen[k]
		if !ok {
			idx = len(rp.points)
			seen[k] = idx
			rp.points = append(rp.points, sim.Point{At: at, Baseline: baseline, Index: idx})
		}
		rp.byPoint[idx] = append(rp.byPoint[idx], i)
	}
	for i, r := range rp.rows {
		if r.never {
			continue
		}
		if !r.baseline.IsZero() {
			add(r.baseline, true, i)
		}
		add(r.at, false, i)
	}
	sim.SortPoints(rp.points)
}

// evaluate — проверка точки: строки табло (или значения «до») теми же операциями API.
func (s *Service) evaluate(ctx context.Context, st *RunState, rp *runPlan, p sim.Point) {
	s.settle(ctx, st)
	for _, i := range rp.byPoint[p.Index] {
		r := rp.rows[i]
		if p.Baseline {
			if v, found, err := s.read(ctx, st, rp, r.a.Check); err == nil && found {
				st.Baselines[r.a.ID] = v
			}
			continue
		}
		res := s.check(ctx, st, rp, r)
		if res.Status == sim.StatusFailed && len(st.Injections) > 0 {
			// FR-152: кнопки стенда меняют ход прогона — ожидания карточки
			// рассчитаны без них; расхождение остаётся расхождением, но с пояснением
			res.Detail = strings.TrimSpace(res.Detail + " (в прогон вносились сбои кнопками стенда: ожидание карточки рассчитано без них)")
		}
		st.Rows[r.a.ID] = res
	}
}

// check — одна строка табло.
func (s *Service) check(ctx context.Context, st *RunState, rp *runPlan, r row) sim.Result {
	if r.a.Mapping == "manual" {
		return sim.Result{Status: sim.StatusPending, Detail: "проверяется глазами на экране: " + r.a.Note}
	}
	v, found, err := s.read(ctx, st, rp, r.a.Check)
	switch {
	case errors.Is(err, ErrUnavailable) && r.a.Check.Operation == "step":
		label, _ := r.a.Check.Params["label"].(string)
		return sim.Result{Status: sim.StatusPending, Detail: "шаг не выполнен: " + st.Steps[label].Detail}
	case errors.Is(err, ErrUnavailable):
		return sim.Result{Status: sim.StatusPending, Detail: "операция " + r.a.Check.Operation + " пока не отвечает (модуль в работе)"}
	case err != nil:
		var un *sim.ErrUnresolved
		if errors.As(err, &un) {
			return sim.Result{Status: sim.StatusPending, Detail: "не найден объект " + un.Placeholder}
		}
		return sim.Result{Status: sim.StatusFailed, Detail: err.Error()}
	}
	ids := s.ids(st, rp)
	want := ids.Translate(sim.ShiftValue(r.a.Check.Value, rp.plan.Shift))
	c := r.a.Check
	c.Value = want
	res := sim.Evaluate(c, v, found, st.Baselines[r.a.ID])
	if res.Status == sim.StatusPassed && !found && r.a.Mapping != "exact" {
		// «значения нет» — зелёное только у точного пути: у пути «по смыслу»
		// отсутствие поля в ответе ничего не доказывает (честное табло)
		return sim.Result{Status: sim.StatusPending, Detail: "сопоставление с ответом не уточнено (draft): путь не найден в ответе"}
	}
	if res.Status == sim.StatusFailed && r.a.Mapping != "exact" {
		// путь сопоставлен по смыслу: расхождение ещё не доказывает ошибку
		// системы — строка ждёт уточнения пути модулем-владельцем
		res.Status = sim.StatusPending
		res.Detail = strings.TrimSpace("сопоставление с ответом не уточнено (draft): " + res.Detail)
	}
	return res
}

// read — значение по пути в ответе операции (или в итоге шага прогона).
func (s *Service) read(ctx context.Context, st *RunState, rp *runPlan, c sim.Check) (any, bool, error) {
	ids := s.ids(st, rp)
	path, err := ids.ExpandString(c.Path, true)
	if err != nil {
		return nil, false, err
	}
	var doc any
	if c.Operation == "step" {
		label, _ := c.Params["label"].(string)
		res, ok := st.Steps[label]
		if label == "" || (ok && (res.Status == "skipped" || res.Status == "waiting")) {
			return nil, false, ErrUnavailable
		}
		if !ok {
			return nil, false, nil
		}
		b, _ := json.Marshal(res)
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.UseNumber()
		if err := dec.Decode(&doc); err != nil {
			return nil, false, err
		}
	} else {
		params, err := s.params(ctx, st, rp, c.Params)
		if err != nil {
			return nil, false, err
		}
		if s.d.Probe == nil {
			return nil, false, ErrUnavailable
		}
		doc, err = s.d.Probe.Read(ctx, c.Operation, params, st.RunID)
		if errors.Is(err, ErrNotFound) {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, err
		}
	}
	vals, err := sim.Extract(doc, path)
	if err != nil {
		return nil, false, err
	}
	return sim.Single(vals), len(vals) > 0, nil
}

// ids — сведение ID прогона с изделиями и объектами, узнанными во время прогона.
func (s *Service) ids(st *RunState, rp *runPlan) *sim.IDMap {
	m := rp.plan.IDs
	for k, v := range st.Items {
		m.Items[k] = v
	}
	for k, v := range st.Refs {
		m.Refs[k] = v
	}
	return m
}

// params — параметры операции со строковыми значениями; объекты системы
// ({ref:…}) находятся по определению прогона теми же операциями.
func (s *Service) params(ctx context.Context, st *RunState, rp *runPlan, in map[string]any) (map[string]string, error) {
	out := map[string]string{}
	for k, v := range in {
		str, err := s.expand(ctx, st, rp, fmt.Sprint(v))
		if err != nil {
			return nil, err
		}
		out[k] = str
	}
	return out, nil
}

// expand — подстановка с поиском объектов системы по требованию.
func (s *Service) expand(ctx context.Context, st *RunState, rp *runPlan, str string) (string, error) {
	ids := s.ids(st, rp)
	for tries := 0; tries < 4; tries++ {
		out, err := ids.ExpandString(str, true)
		var un *sim.ErrUnresolved
		if err == nil || !errors.As(err, &un) {
			return out, err
		}
		name, ok := refName(un.Placeholder)
		if !ok || !s.resolveRef(ctx, st, rp, name) {
			return "", err
		}
		ids = s.ids(st, rp)
	}
	return ids.ExpandString(str, true)
}

func refName(ph string) (string, bool) {
	if len(ph) > 6 && ph[:5] == "{ref:" {
		return ph[5 : len(ph)-1], true
	}
	return "", false
}

// resolveRef — объект, рождённый системой: чтение операцией из RunDef.Refs.
func (s *Service) resolveRef(ctx context.Context, st *RunState, rp *runPlan, name string) bool {
	ref, ok := rp.refs[name]
	if !ok || s.d.Probe == nil {
		return false
	}
	params := map[string]string{}
	for k, v := range ref.Params {
		str, err := s.expand(ctx, st, rp, fmt.Sprint(v))
		if err != nil {
			return false
		}
		params[k] = str
	}
	doc, err := s.d.Probe.Read(ctx, ref.Operation, params, st.RunID)
	if err != nil {
		return false
	}
	vals, err := sim.Extract(doc, ref.Path)
	if err != nil || len(vals) == 0 {
		return false
	}
	var id string
	switch x := sim.Single(vals).(type) {
	case string:
		id = x
	case json.Number:
		id = x.String()
	default:
		return false
	}
	if id == "" {
		return false
	}
	st.Refs[name] = id
	return true
}

// Board — табло «ожидалось → получилось» (simulation.board.read, AD-26).
func (s *Service) Board(ctx context.Context, runID string, m platform.Moment) (Board, error) {
	if !s.live {
		return s.Unimplemented.Board(ctx, runID, m)
	}
	st, err := s.load(ctx, runID)
	if err != nil {
		return Board{}, err
	}
	rp, err := s.plan(ctx, st)
	if err != nil {
		return Board{}, err
	}
	b := Board{RunID: runID, Rows: []BoardRow{}, BasisSeq: st.BasisSeq}
	done := isFinal(st.State)
	for i, r := range rp.rows {
		exp, _ := json.Marshal(r.a.Check.Value)
		br := BoardRow{AssertionID: r.a.ID, Title: r.a.What, OperationID: r.a.Check.Operation, Path: r.a.Check.Path,
			Expected: string(exp), ScenarioID: r.scenario, Checkpoint: r.label, MustNot: r.mustNot, Mapping: r.a.Mapping, Step: s.pointStep(rp, i)}
		if !r.never {
			at := r.at
			br.At = &at
		}
		res, ok := st.Rows[r.a.ID]
		switch {
		case ok:
			br.Status = string(res.Status)
			br.Detail = res.Detail
			if res.Actual != "" {
				a := res.Actual
				br.Actual = &a
			}
		case r.never:
			br.Status = string(sim.StatusNotReached)
			br.Detail = "момент проверки — только в отдельном прогоне"
		case done:
			br.Status = string(sim.StatusNotReached)
		default:
			br.Status = string(sim.StatusPending)
		}
		if br.Detail == "" && r.a.Note != "" {
			br.Detail = r.a.Note
		}
		switch sim.Status(br.Status) {
		case sim.StatusPassed:
			b.Passed++
		case sim.StatusFailed:
			b.Failed++
		case sim.StatusNotReached:
			b.NotReached++
		default:
			b.Pending++
		}
		b.Rows = append(b.Rows, br)
	}
	// строки кнопок цифрового стенда (FR-152) — после строк сценария, по нажатиям
	for _, x := range st.Injections {
		for _, r := range x.Rows {
			br := injectionRow(st, x, r)
			switch sim.Status(br.Status) {
			case sim.StatusPassed:
				b.Passed++
			case sim.StatusFailed:
				b.Failed++
			default:
				b.Pending++
			}
			b.Rows = append(b.Rows, br)
		}
	}
	return b, nil
}

// pointStep — номер точки проверки строки (шаг табло).
func (s *Service) pointStep(rp *runPlan, i int) int {
	for n, p := range rp.points {
		if p.Baseline {
			continue
		}
		for _, j := range rp.byPoint[p.Index] {
			if j == i {
				return n
			}
		}
	}
	return 0
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
