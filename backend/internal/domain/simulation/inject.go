package simulation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
)

// Цифровой стенд (FR-152, AD-26; эпик 36): кнопки-инъекции поверх идущего
// прогона. Каждая кнопка — заранее заготовленное событие сбоя из карточек
// каталога (F01 повтор, F03 позднее событие, F08 плохое наблюдение, S05/S07
// ток вне уставки, F07 разрыв номеров, F25 вмешательство в хранилище),
// приведённое к прогону: изделие, пост и время берутся из плана генератора и
// доменных часов прогона. События уходят в обычный приём (AD-26); подделка —
// только демо-инструментом cmd/tamper. Функция чистая: тот же план, номер
// нажатия, цель и момент дают те же события (AD-4, AD-38).

// InjectionKind — кнопка цифрового стенда (перечисление ApplyInjection.injection).
type InjectionKind string

// Кнопки цифрового стенда.
const (
	// InjectDuplicate — «прислать повтор события» (F01, S06): те же байты от того же источника.
	InjectDuplicate InjectionKind = "duplicate_event"
	// InjectLate — «прислать опоздавшее событие» (F03, S07): случилось раньше цели, пришло сейчас.
	InjectLate InjectionKind = "late_event"
	// InjectCorruptFrame — «испортить кадр» (F08, S04): качество наблюдения 0,3.
	InjectCorruptFrame InjectionKind = "corrupt_frame"
	// InjectMachineFault — «сбой станка / ток вне уставки» (S05, S07): отклонение на сварке.
	InjectMachineFault InjectionKind = "machine_fault"
	// InjectDataLoss — «потерять кусок данных» (F07, S04): разрыв номеров источника.
	InjectDataLoss InjectionKind = "data_loss"
	// InjectTamper — «подделать запись в обход системы» (F25, S09): только cmd/tamper.
	InjectTamper InjectionKind = "tamper_outside"
)

// InjectionKinds — кнопки в порядке пульта.
var InjectionKinds = []InjectionKind{InjectDuplicate, InjectLate, InjectCorruptFrame, InjectMachineFault, InjectDataLoss, InjectTamper}

// Параметры заготовок — из карточек сбоев (scenarios/definitions/scenarios).
const (
	// LateBy — опоздавшее событие случилось за минуту до цели: «возникло
	// раньше, узнали позже» (F03; история перестраивается по времени события).
	LateBy = time.Minute
	// PoorQualityBP, PoorConfidenceBP — испорченный кадр F08: качество 0,30.
	PoorQualityBP    = 3000
	PoorConfidenceBP = 9300
	// PoorLimitation — ограничение наблюдения из F08.
	PoorLimitation = "грязный объектив"
	// LostRecords — сколько номеров теряет источник за одно нажатие (F07: 42–43).
	LostRecords = 2
	// DeviationOverTol — насколько ток выше верхней границы уставки, А
	// (S07, EV-WS2-0412: 176 А при уставке 160 ± 10).
	DeviationOverTol = 6
	// TamperQualityBP — подделанное качество наблюдения (F25: 9900).
	TamperQualityBP = 9900
)

// Источники stand-а цифрового стенда: новые события кнопок идут от своих
// источников прогона ‹run_id›/stand-‹вид›, чтобы не ломать непрерывность
// source_seq настоящих источников (AD-7, AD-9); у каждого — свой счёт номеров.
const (
	StandLate    = "stand-late"
	StandFrame   = "stand-frame"
	StandMachine = "stand-machine"
	StandLoss    = "stand-loss"
)

// InjectionInput — вход кнопки.
type InjectionInput struct {
	Plan  *Plan
	World World
	Kind  InjectionKind
	// N — номер нажатия в прогоне (1, 2, …): event_id событий кнопки.
	N int
	// Target — event_id цели; пусто — последнее подходящее доставленное событие.
	Target string
	// Delivered — сколько событий плана уже доставлено в приём (курсор прогона).
	Delivered int
	// At — доменное «сейчас» прогона (AD-37): время доставки и событий кнопки.
	At time.Time
	// StandSeq — последние source_seq источников стенда в прогоне.
	StandSeq map[string]int64
}

// Injection — что делает кнопка.
type Injection struct {
	Kind InjectionKind
	N    int
	At   time.Time
	// Target — событие прогона, над которым работает кнопка (nil — без цели).
	Target *Emission
	// Item — изделие определения (F-501), к которому относится результат.
	Item string
	// Equipment — оборудование определения (IS-2) для сбоя станка и потери данных.
	Equipment string
	// Source — source_id источника стенда (потеря данных — где искать разрыв).
	Source string
	// Emissions — события в обычный приём сейчас.
	Emissions []Emission
	// StandSeq — новые последние номера источников стенда.
	StandSeq map[string]int64
	// Stand — сбой stand-а через служебный порт (AD-18): потеря данных.
	Stand *StandAction
	// Tamper — подделка демо-инструментом (AD-26, AD-28).
	Tamper *Tamper
	// Detail — что сделано, по-русски (итог шага прогона).
	Detail string
}

// InjectionError — кнопку нельзя применить: нет цели или цель не подходит.
type InjectionError struct{ Reason string }

func (e *InjectionError) Error() string { return e.Reason }

func refuse(format string, a ...any) error { return &InjectionError{Reason: fmt.Sprintf(format, a...)} }

// InjectionEventID — event_id k-го события нажатия n: UUIDv5(NS_ANT, run ‖ seed ‖ n ‖ k).
func InjectionEventID(ids *IDMap, n, k int) string {
	return kernel.UUIDv5(constants.NsAnt, fmt.Sprintf("simulation-injection|%s|%d|%d|%d", ids.RunID, ids.Seed, n, k))
}

// PlanInjection строит нажатие кнопки над планом прогона.
func PlanInjection(in InjectionInput) (*Injection, error) {
	if in.Plan == nil || in.Plan.IDs == nil {
		return nil, refuse("нет плана прогона")
	}
	if in.N <= 0 {
		return nil, refuse("номер нажатия должен быть положительным")
	}
	in.Delivered = min(max(in.Delivered, 0), len(in.Plan.Emissions))
	out := &Injection{Kind: in.Kind, N: in.N, At: in.At, StandSeq: map[string]int64{}}
	var err error
	switch in.Kind {
	case InjectDuplicate:
		err = in.duplicate(out)
	case InjectLate:
		err = in.late(out)
	case InjectCorruptFrame:
		err = in.frame(out)
	case InjectMachineFault:
		err = in.machine(out)
	case InjectDataLoss:
		err = in.loss(out)
	case InjectTamper:
		err = in.tamper(out)
	default:
		return nil, refuse("неизвестная кнопка %q", in.Kind)
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ── цель ──

// candidate — событие плана, годное в цели: по контракту, не в карантин, не конфликт.
func candidate(e Emission) bool {
	return e.Contract && !e.Quarantine && e.Delivery != DeliveryConflict
}

// target — цель кнопки: указанное событие (должно быть уже доставлено) или
// последнее доставленное, подходящее под want; сначала — с изделием.
func (in InjectionInput) target(what string, want func(Emission, map[string]any) bool) (Emission, map[string]any, error) {
	em := in.Plan.Emissions
	if in.Target != "" {
		for i := in.Delivered - 1; i >= 0; i-- {
			e := em[i]
			if e.EventID != in.Target || !candidate(e) {
				continue
			}
			ev, err := decodeEvent(e.Event)
			if err != nil {
				return Emission{}, nil, err
			}
			if !want(e, ev) {
				return Emission{}, nil, refuse("событие %s (%s) не подходит: нужно %s", e.EventID, e.EventType, what)
			}
			return e, ev, nil
		}
		for _, e := range em[in.Delivered:] {
			if e.EventID == in.Target {
				return Emission{}, nil, refuse("событие %s ещё не доставлено в приём — кнопка работает с тем, что уже пришло", in.Target)
			}
		}
		return Emission{}, nil, refuse("события %s нет в прогоне %s", in.Target, in.Plan.RunID)
	}
	for _, withItem := range []bool{true, false} {
		for i := in.Delivered - 1; i >= 0; i-- {
			e := em[i]
			if !candidate(e) || (withItem && e.Item == "") {
				continue
			}
			ev, err := decodeEvent(e.Event)
			if err != nil {
				return Emission{}, nil, err
			}
			if want(e, ev) {
				return e, ev, nil
			}
		}
	}
	return Emission{}, nil, refuse("в прогоне ещё нет доставленного события: нужно %s", what)
}

func anyEvent(Emission, map[string]any) bool { return true }

func cameraResult(e Emission, ev map[string]any) bool {
	d, _ := ev["data"].(map[string]any)
	return e.EventType == "inspection.result.recorded" && d != nil && d["method"] == "camera"
}

func decodeEvent(b []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var ev map[string]any
	if err := dec.Decode(&ev); err != nil {
		return nil, fmt.Errorf("событие прогона не разобрано: %w", err)
	}
	return ev, nil
}

// clone — глубокая копия исходного события.
func clone(ev map[string]any) map[string]any {
	b, _ := json.Marshal(ev)
	c, _ := decodeEvent(b)
	return c
}

// ── события кнопки ──

// emission — событие k нажатия от источника стенда source: конверт v1 как у
// генератора (contracts/events/common/envelope.v1.json), номер — следующий у
// источника стенда.
func (in InjectionInput) emission(out *Injection, k int, source string, ev map[string]any, occurred time.Time, item string) (Emission, error) {
	ids := in.Plan.IDs
	seq := in.nextSeq(out, source)
	id := InjectionEventID(ids, in.N, k)
	ev["event_id"] = id
	ev["correlation_id"] = id
	ev["causation_id"] = nil
	ev["source_id"] = ids.SourceID(source)
	ev["source_seq"] = seq
	ev["occurred_at"] = FormatTime(occurred)
	ev["run_id"] = ids.RunID
	ev["integrity"] = map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []any{"scenario." + source + "@1"}}
	b, err := json.Marshal(ev)
	if err != nil {
		return Emission{}, err
	}
	typ, _ := ev["event_type"].(string)
	delivery := DeliveryNormal
	if in.At.Sub(occurred) > 5*time.Minute {
		delivery = DeliveryLate
	}
	return Emission{DeliverAt: in.At, SourceKey: source, SourceID: ids.SourceID(source), SourceSeq: seq, EventID: id,
		EventType: typ, OccurredAt: occurred, TrueAt: occurred, Item: item, Scenario: "stand",
		Label: fmt.Sprintf("stand#%d/%d", in.N, k), Delivery: delivery, Contract: true, Event: b}, nil
}

// nextSeq — следующий номер источника стенда (с учётом уже выданных в этом нажатии).
func (in InjectionInput) nextSeq(out *Injection, source string) int64 {
	last, ok := out.StandSeq[source]
	if !ok {
		last = in.StandSeq[source]
	}
	last++
	out.StandSeq[source] = last
	return last
}

// newEnvelope — конверт события оборудования без изделия.
func newEnvelope(typ string, ver int, kind, reliability string, data map[string]any) map[string]any {
	return map[string]any{"event_type": typ, "schema_version": ver, "source_kind": kind, "reliability": reliability, "data": data}
}

func suffixID(data map[string]any, key, suffix string) {
	if v, ok := data[key].(string); ok && v != "" {
		data[key] = v + suffix
	}
}

// duplicate — F01: те же байты, тот же источник и номер — приём отвечает «повтор».
func (in InjectionInput) duplicate(out *Injection) error {
	e, _, err := in.target("событие по контракту", anyEvent)
	if err != nil {
		return err
	}
	dup := e
	dup.DeliverAt, dup.Delivery, dup.Scenario, dup.Label = in.At, DeliveryDuplicate, "stand", fmt.Sprintf("stand#%d/0", in.N)
	out.Target, out.Item = &e, e.Item
	out.Emissions = []Emission{dup}
	out.Detail = fmt.Sprintf("повтор события %s (%s) от %s, номер %d", e.EventID, e.EventType, e.SourceID, e.SourceSeq)
	return nil
}

// late — F03: то же содержание, случилось за LateBy до цели, пришло сейчас.
func (in InjectionInput) late(out *Injection) error {
	e, ev, err := in.target("событие по контракту", anyEvent)
	if err != nil {
		return err
	}
	c := clone(ev)
	if d, ok := c["data"].(map[string]any); ok {
		suffixID(d, "observation_id", fmt.Sprintf("-late-%d", in.N))
	}
	occurred := e.OccurredAt.Add(-LateBy)
	em, err := in.emission(out, 0, StandLate, c, occurred, e.Item)
	if err != nil {
		return err
	}
	out.Target, out.Item = &e, e.Item
	out.Emissions = []Emission{em}
	out.Detail = fmt.Sprintf("событие %s случилось %s (за минуту до %s), пришло %s — позже на %s",
		e.EventType, FormatTime(occurred), e.EventID, FormatTime(in.At), in.At.Sub(occurred).Round(time.Second))
	return nil
}

// frame — F08: кадр камеры той же точки контроля с качеством 0,30.
func (in InjectionInput) frame(out *Injection) error {
	e, ev, err := in.target("результат контроля камерой (inspection.result.recorded, method = camera)", cameraResult)
	if err != nil {
		return err
	}
	c := clone(ev)
	d, _ := c["data"].(map[string]any)
	suffixID(d, "observation_id", fmt.Sprintf("-frame-%d", in.N))
	d["observation_quality_bp"] = PoorQualityBP
	d["analyzer_confidence_bp"] = PoorConfidenceBP
	d["limitations"] = []any{PoorLimitation}
	em, err := in.emission(out, 0, StandFrame, c, in.At, e.Item)
	if err != nil {
		return err
	}
	out.Target, out.Item = &e, e.Item
	out.Emissions = []Emission{em}
	out.Detail = fmt.Sprintf("испорченный кадр точки %v изделия %s: качество 0,30 (%s)", d["inspection_point"], e.Item, PoorLimitation)
	return nil
}

// weld — сварка, на которой случится отклонение: изделия цели или идущая
// сейчас, иначе последняя начатая к моменту At.
func (in InjectionInput) weld(item string) (WeldTruth, bool) {
	var best WeldTruth
	found := false
	for _, w := range in.Plan.Truth.Welds {
		if w.Start.After(in.At) || (item != "" && w.Item != item) {
			continue
		}
		if !found || w.Start.After(best.Start) {
			best, found = w, true
		}
	}
	return best, found
}

// machine — S05/S07: «ток вне уставки» на сварке (equipment.deviation.detected,
// как запись журнала источника в потоке S07).
func (in InjectionInput) machine(out *Injection) error {
	item := ""
	if in.Target != "" {
		e, _, err := in.target("событие изделия", func(e Emission, _ map[string]any) bool { return e.Item != "" })
		if err != nil {
			return err
		}
		item = e.Item
		out.Target = &e
	}
	w, ok := in.weld(item)
	if !ok {
		if item != "" {
			return refuse("изделие %s ещё не сваривалось — отклонению не к чему привязаться", item)
		}
		return refuse("в прогоне ещё не было сварки — отклонению не к чему привязаться")
	}
	at := in.At
	if at.Before(w.ArcFrom) || !at.Before(w.ArcTo) {
		at = w.ArcFrom.Add(w.ArcTo.Sub(w.ArcFrom) / 2)
	}
	at = at.Truncate(time.Second)
	r := in.World.Route
	value := r.CurrentNominal + r.CurrentTol + DeviationOverTol
	eq := in.Plan.IDs.Equipment(w.Station)
	data := map[string]any{"equipment_id": eq, "station_id": "ST-WELD", "deviation_kind": "out_of_setpoint",
		"parameter": "current", "value": measurement(int64(value), 0, "A"), "started_at": FormatTime(at),
		"ended_at": FormatTime(w.End),
		"setpoint": map[string]any{"nominal": measurement(int64(r.CurrentNominal), 0, "A"),
			"lower": measurement(int64(r.CurrentNominal-r.CurrentTol), 0, "A"), "upper": measurement(int64(r.CurrentNominal+r.CurrentTol), 0, "A")},
		"code": "I_OUT_OF_SETPOINT"}
	em, err := in.emission(out, 0, StandMachine, newEnvelope("equipment.deviation.detected", 1, "machine", "high", data), at, "")
	if err != nil {
		return err
	}
	out.Item, out.Equipment = w.Item, w.Station
	out.Emissions = []Emission{em}
	out.Detail = fmt.Sprintf("ток %d А вне уставки %d ± %d А на %s (%s, изделие %s) в %s",
		value, r.CurrentNominal, r.CurrentTol, w.Station, w.Run, w.Item, FormatTime(at))
	return nil
}

// lossEquipment — оборудование потери данных: пост последней сварки, иначе
// сварочный пост первой линии мира.
func (in InjectionInput) lossEquipment() string {
	if w, ok := in.weld(""); ok {
		return w.Station
	}
	for _, k := range slices.Sorted(maps.Keys(in.World.Lines)) {
		if eq := in.World.Lines[k].Equipment; eq != "" {
			return eq
		}
	}
	return "IS-1"
}

// loss — F07: источник теряет LostRecords записей — приходят запись до и
// запись после разрыва (сводки простоя, режим не меняют), номера между ними
// у источника есть, в приёме — нет.
func (in InjectionInput) loss(out *Injection) error {
	eqDef := in.lossEquipment()
	eq := in.Plan.IDs.Equipment(eqDef)
	summary := func(at time.Time) map[string]any {
		return newEnvelope("equipment.cycle.summarized", 1, "machine", "high", map[string]any{"equipment_id": eq, "station_id": "ST-WELD",
			"window_start": FormatTime(at.Add(-time.Minute)), "window_end": FormatTime(at), "cycle_ref": "idle",
			"parameters": []any{map[string]any{"parameter": "arc_time", "mean": measurement(0, 0, "s")}}})
	}
	at := in.At.Truncate(time.Second)
	prev := at.Add(-time.Duration(LostRecords+1) * time.Second)
	before, err := in.emission(out, 0, StandLoss, summary(prev), prev, "")
	if err != nil {
		return err
	}
	first := out.StandSeq[StandLoss] + 1
	out.StandSeq[StandLoss] += LostRecords // номера потеряны у источника
	after, err := in.emission(out, 1, StandLoss, summary(at), at, "")
	if err != nil {
		return err
	}
	out.Equipment, out.Source = eqDef, in.Plan.IDs.SourceID(StandLoss)
	out.Emissions = []Emission{before, after}
	out.Stand = &StandAction{Stand: "equipment:" + eqDef, Fault: "drop", Detail: "потеря куска данных (цифровой стенд)"}
	out.Detail = fmt.Sprintf("источник %s (%s) потерял записи %d–%d", out.Source, eqDef, first, first+LostRecords-1)
	return nil
}

// tamper — F25: правка записи журнала на месте демо-инструментом (cmd/tamper).
func (in InjectionInput) tamper(out *Injection) error {
	var (
		e   Emission
		err error
	)
	if in.Target != "" {
		e, _, err = in.target("событие по контракту", anyEvent)
	} else {
		e, _, err = in.target("результат контроля", func(e Emission, _ map[string]any) bool { return e.EventType == "inspection.result.recorded" })
		var ie *InjectionError
		if errors.As(err, &ie) {
			e, _, err = in.target("событие по контракту", anyEvent)
		}
	}
	if err != nil {
		return err
	}
	change := map[string]any{"data/note": "подделано в обход системы"}
	if e.EventType == "inspection.result.recorded" {
		change = map[string]any{"data/observation_quality_bp": TamperQualityBP}
	}
	label := e.Label
	if label == "" {
		label = e.EventID
	}
	out.Target, out.Item = &e, e.Item
	out.Tamper = &Tamper{Kind: "update_in_place", Target: label, Change: change}
	keys := slices.Sorted(maps.Keys(change))
	out.Detail = fmt.Sprintf("правка записи %s (%s) в хранилище в обход API: %s", e.EventID, e.EventType, strings.Join(keys, ", "))
	return nil
}
