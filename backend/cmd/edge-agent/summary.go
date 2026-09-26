package main

import (
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"ant/internal/contracts/constants"
	"ant/internal/contracts/procs"
	"ant/internal/domain/kernel"
	ml "ant/internal/domain/machinelogs"
)

// Summarizer — выделитель значимых событий оборудования на краю (эпик 23,
// FR-147, AD-29) поверх базового Extractor эпика 06: не логгер, а сводки.
// Сырые отсчёты (SAMPLE) остаются на краю; в ant уходят:
//
//   - equipment.cycle.summarized — сводка параметров на окно цикла (среднее,
//     максимум, минимум, уставка, сколько миллисекунд вне уставки) со ссылкой
//     на сырые данные на краю;
//   - equipment.deviation.detected — отклонения: параметр вне уставки
//     (перегрузка для нагрузок), ручная коррекция режима (подача 130 %),
//     ручной режим управления во время цикла, авария, исчерпан ресурс
//     инструмента;
//   - equipment.program.changed и equipment.tool.changed — программа и её
//     ревизия, инструмент и его ресурс (MTConnect EVENT).
//
// Классификация категорий — domain/machinelogs.CategoryClass (одна на
// систему). operation_run_id агент не подставляет: привязку к выполнению
// делает межизделийная стадия (AD-29, AD-42). event_id сводок и отклонений —
// UUIDv5(устройство, окно, вид): повтор сообщения stand-а не даёт второго события.
type Summarizer struct {
	mu sync.Mutex
	eq map[string]*equipState
}

// NewSummarizer создаёт выделитель сводок.
func NewSummarizer() *Summarizer { return &Summarizer{eq: map[string]*equipState{}} }

// equipState — состояние выделителя по одному оборудованию.
type equipState struct {
	setpoints map[string]procs.StandTelemetryV1SetpointsElem
	cycleOpen bool
	midCycle  bool
	cycleRef  string
	start     time.Time
	params    map[string]*aggregate
	// excursions — открытые выходы за уставку и коррекции режима по ключу.
	excursions map[string]*excursion
	program    string
	tool       string
	toolUsed   int64
	toolLimit  int64
	mode       string
}

// aggregate — сводка отсчётов параметра за окно цикла (целые, без float).
type aggregate struct {
	unit     string
	scale    int
	n, sum   int64
	min, max int64
	last     time.Time
	lastOut  bool
	outMs    int64
}

// excursion — открытое отклонение: вид, параметр, крайнее значение, начало.
type excursion struct {
	kind      string
	parameter string
	code      string
	value     int64
	scale     int
	unit      string
	start     time.Time
	sp        *procs.StandTelemetryV1SetpointsElem
}

func parseAt(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02T15:04:05.000Z", s)
	return t.UTC(), err == nil
}

func stamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// Process — события сводок и отклонений из сообщения stand-а.
func (s *Summarizer) Process(t procs.StandTelemetryV1) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.eq[t.EquipmentID]
	if st == nil {
		st = &equipState{setpoints: map[string]procs.StandTelemetryV1SetpointsElem{}, excursions: map[string]*excursion{}}
		s.eq[t.EquipmentID] = st
	}
	em := &emitter{eq: t.EquipmentID, run: t.RunID}
	// Смена программы внеплановая, только если цикл шёл до этого сообщения.
	st.midCycle = st.cycleOpen
	sent, _ := parseAt(t.SentAt)
	for _, sp := range t.Setpoints {
		st.setpoints[sp.Parameter] = sp
	}
	if t.Cycle != nil && t.Cycle.Phase == procs.StandTelemetryV1CyclePhaseStart {
		st.cycleOpen, st.cycleRef, st.start = true, t.Cycle.CycleRef, sent
		st.params = map[string]*aggregate{}
	}
	for _, e := range t.Events {
		at, ok := parseAt(e.At)
		if !ok {
			continue
		}
		if _, known := ml.CategoryClass(string(e.Category)); !known {
			continue
		}
		s.event(st, em, e, at)
	}
	for _, x := range t.Samples {
		at, ok := parseAt(x.At)
		if !ok {
			continue
		}
		s.sample(st, em, x, at)
	}
	if t.Cycle != nil && t.Cycle.Phase == procs.StandTelemetryV1CyclePhaseEnd && st.cycleOpen {
		s.closeCycle(st, em, sent)
	}
	return em.out
}

func (s *Summarizer) event(st *equipState, em *emitter, e procs.StandTelemetryV1EventsElem, at time.Time) {
	switch string(e.Category) {
	case ml.CategoryProgram:
		prog, rev, _ := strings.Cut(e.Value, "@")
		if e.Value == st.program {
			return
		}
		// Смена программы посреди цикла — внеплановая (FR-147); иначе — плановая.
		planned := !st.midCycle
		st.program = e.Value
		em.emit("equipment.program.changed", 1, at, "program", map[string]any{"program_ref": prog, "program_revision": orDash(rev), "planned": planned})
		if !planned {
			em.emit("equipment.deviation.detected", 1, at, "unplanned_program:"+e.Value, map[string]any{
				"deviation_kind": ml.DeviationProgramChange, "started_at": stamp(at), "ended_at": stamp(at), "code": prog})
		}
	case ml.CategoryTool:
		used, limit := int64(-1), int64(-1)
		if e.ToolLifeUsed != nil {
			used = int64(*e.ToolLifeUsed)
		}
		if e.ToolLifeLimit != nil {
			limit = int64(*e.ToolLifeLimit)
		}
		if e.Value == st.tool && used == st.toolUsed && limit == st.toolLimit {
			return
		}
		st.tool, st.toolUsed, st.toolLimit = e.Value, used, limit
		d := map[string]any{"tool_id": e.Value}
		if used >= 0 {
			d["tool_life_used"] = used
		}
		if limit >= 0 {
			d["tool_life_limit"] = limit
		}
		em.emit("equipment.tool.changed", 1, at, "tool:"+e.Value+":"+strconv.FormatInt(used, 10), d)
		if limit > 0 && used >= limit {
			em.emit("equipment.deviation.detected", 1, at, "tool_life:"+e.Value+":"+strconv.FormatInt(used, 10), map[string]any{
				"deviation_kind": ml.DeviationToolLife, "parameter": "tool_life", "started_at": stamp(at),
				"value": measure(used, 0, "{cycles}"), "setpoint": map[string]any{"upper": measure(limit, 0, "{cycles}")}})
		}
	case ml.CategoryOverride:
		pct, err := strconv.ParseInt(strings.TrimSpace(e.Value), 10, 64)
		if err != nil {
			return
		}
		what := "feed"
		if e.Code != nil && *e.Code != "" {
			what = *e.Code
		}
		key := "override:" + what
		sp := procs.StandTelemetryV1SetpointsElem{Parameter: what + "_override", Nominal: intp(100), Lower: intp(100), Upper: intp(100), Unit: "%"}
		if pct != 100 {
			x := st.excursions[key]
			if x == nil {
				st.excursions[key] = &excursion{kind: ml.DeviationManualOverride, parameter: what + "_override", code: what, value: pct, unit: "%", start: at, sp: &sp}
			} else if abs(pct-100) > abs(x.value-100) {
				x.value = pct
			}
			return
		}
		s.close(st, em, key, at)
	case ml.CategoryControllerMode:
		mode := strings.ToLower(e.Value)
		prev := st.mode
		st.mode = mode
		if mode == "manual" && prev != "manual" && st.cycleOpen {
			st.excursions["mode"] = &excursion{kind: ml.DeviationManualOverride, parameter: "controller_mode", code: "manual", start: at}
		}
		if mode != "manual" {
			s.close(st, em, "mode", at)
		}
	case ml.CategoryAlarm:
		d := map[string]any{"deviation_kind": ml.DeviationAlarm, "started_at": stamp(at), "ended_at": stamp(at), "code": trunc(e.Value, 64)}
		em.emit("equipment.deviation.detected", 1, at, "alarm:"+e.Value+":"+stamp(at), d)
	}
}

// sample — отсчёт: сводка окна цикла и выход за уставку.
func (s *Summarizer) sample(st *equipState, em *emitter, x procs.StandTelemetryV1SamplesElem, at time.Time) {
	if !st.cycleOpen {
		return // вне цикла отсчёты остаются на краю
	}
	a := st.params[x.Parameter]
	v := int64(x.Value)
	if a == nil {
		a = &aggregate{unit: x.Unit, scale: x.Scale, min: v, max: v}
		st.params[x.Parameter] = a
	}
	if x.Scale != a.scale || x.Unit != a.unit {
		return // смена масштаба или единицы в окне — отсчёт не сводится
	}
	a.n++
	a.sum += v
	a.min, a.max = min(a.min, v), max(a.max, v)
	sp, has := st.setpoints[x.Parameter]
	out := false
	if has && sp.Unit == x.Unit {
		out = outOf(v, x.Scale, sp)
	}
	if a.lastOut && !a.last.IsZero() {
		a.outMs += at.Sub(a.last).Milliseconds()
	}
	a.last, a.lastOut = at, out
	if strings.HasSuffix(x.Parameter, "_override") {
		// Коррекция режима выделяется по событию override (ручное изменение
		// режима), второй раз «вне уставки» не считается: один смысл — одно событие.
		return
	}
	key := "setpoint:" + x.Parameter
	if out {
		ex := st.excursions[key]
		if ex == nil {
			kind := ml.DeviationOutOfSetpoint
			if strings.Contains(x.Parameter, "load") {
				kind = ml.DeviationOverload
			}
			spc := sp
			st.excursions[key] = &excursion{kind: kind, parameter: x.Parameter, value: v, scale: x.Scale, unit: x.Unit, start: at, sp: &spc}
		} else if farther(v, ex.value, x.Scale, sp) {
			ex.value = v
		}
		return
	}
	s.close(st, em, key, at)
}

// close — закрыть открытое отклонение key и выдать его событием.
func (s *Summarizer) close(st *equipState, em *emitter, key string, end time.Time) {
	x := st.excursions[key]
	if x == nil {
		return
	}
	delete(st.excursions, key)
	d := map[string]any{"deviation_kind": x.kind, "parameter": x.parameter, "started_at": stamp(x.start), "ended_at": stamp(end)}
	if x.unit != "" {
		d["value"] = measure(x.value, x.scale, x.unit)
	}
	if x.sp != nil {
		d["setpoint"] = tolerance(*x.sp)
	}
	if x.code != "" {
		d["code"] = x.code
	}
	em.emit("equipment.deviation.detected", 1, end, key+":"+stamp(x.start), d)
}

// closeCycle — конец цикла: закрыть отклонения и выдать сводку окна.
func (s *Summarizer) closeCycle(st *equipState, em *emitter, end time.Time) {
	for _, k := range sortedKeys(st.excursions) {
		s.close(st, em, k, end)
	}
	var params []any
	for _, name := range sortedKeys(st.params) {
		a := st.params[name]
		if a.n == 0 {
			continue
		}
		p := map[string]any{"parameter": name, "mean": measure(a.sum/a.n, a.scale, a.unit),
			"max": measure(a.max, a.scale, a.unit), "min": measure(a.min, a.scale, a.unit)}
		if sp, ok := st.setpoints[name]; ok && sp.Unit == a.unit {
			if a.lastOut {
				a.outMs += end.Sub(a.last).Milliseconds()
			}
			p["setpoint"] = tolerance(sp)
			p["out_of_setpoint_ms"] = max(a.outMs, 0)
		}
		params = append(params, p)
	}
	if len(params) > 0 {
		em.emit("equipment.cycle.summarized", 1, end, "cycle:"+st.cycleRef+":"+stamp(st.start), map[string]any{
			"window_start": stamp(st.start), "window_end": stamp(end), "cycle_ref": trunc(st.cycleRef, 128),
			"parameters": params, "raw_ref": trunc("edge:"+em.eq+"/"+st.cycleRef, 256)})
	}
	st.cycleOpen, st.params = false, nil
}

// emitter — события одного сообщения stand-а.
type emitter struct {
	eq  string
	run *string
	out []map[string]any
}

func (em *emitter) emit(eventType string, version int, at time.Time, key string, data map[string]any) {
	data["equipment_id"] = em.eq
	run := ""
	if em.run != nil {
		run = *em.run
	}
	ev := map[string]any{"event_type": eventType, "schema_version": version, "occurred_at": stamp(at),
		"source_kind": "machine", "reliability": "high", "data": data,
		"event_id": kernel.UUIDv5(constants.NsAnt, "edge-agent\x1f"+run+"\x1f"+em.eq+"\x1f"+eventType+"\x1f"+key)}
	if run != "" {
		ev["run_id"] = run
	}
	em.out = append(em.out, ev)
}

func measure(v int64, scale int, unit string) map[string]any {
	return map[string]any{"value": v, "scale": scale, "unit": unit}
}

func tolerance(sp procs.StandTelemetryV1SetpointsElem) map[string]any {
	out := map[string]any{}
	if sp.Nominal != nil {
		out["nominal"] = measure(int64(*sp.Nominal), sp.Scale, sp.Unit)
	}
	if sp.Lower != nil {
		out["lower"] = measure(int64(*sp.Lower), sp.Scale, sp.Unit)
	}
	if sp.Upper != nil {
		out["upper"] = measure(int64(*sp.Upper), sp.Scale, sp.Unit)
	}
	return out
}

// scaled — значение v масштаба from в масштабе to (to ≥ from).
func scaled(v int64, from, to int) int64 {
	for range to - from {
		v *= 10
	}
	return v
}

// outOf — отсчёт вне уставки (границы включены в допуск).
func outOf(v int64, scale int, sp procs.StandTelemetryV1SetpointsElem) bool {
	s := max(scale, sp.Scale)
	x := scaled(v, scale, s)
	if sp.Lower != nil && x < scaled(int64(*sp.Lower), sp.Scale, s) {
		return true
	}
	return sp.Upper != nil && x > scaled(int64(*sp.Upper), sp.Scale, s)
}

// farther — v дальше от допуска, чем прежнее крайнее значение w.
func farther(v, w int64, scale int, sp procs.StandTelemetryV1SetpointsElem) bool {
	s := max(scale, sp.Scale)
	dist := func(x int64) int64 {
		x = scaled(x, scale, s)
		if sp.Upper != nil && x > scaled(int64(*sp.Upper), sp.Scale, s) {
			return x - scaled(int64(*sp.Upper), sp.Scale, s)
		}
		if sp.Lower != nil && x < scaled(int64(*sp.Lower), sp.Scale, s) {
			return scaled(int64(*sp.Lower), sp.Scale, s) - x
		}
		return 0
	}
	return dist(v) > dist(w)
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func intp(v int) *int { return &v }

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
