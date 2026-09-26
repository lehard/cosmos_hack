// Пакет mltest — сценарии оборудования для тестов и прогонов (FR-149):
// записи журнала, какими их пишут приём (факты edge-агента и терминала) —
// нормальное выполнение и выполнение с двумя отклонениями на станке ЧПУ,
// «сварка вне режима» на сварочном источнике (FR-151). Используют тесты
// application/machinelogs (фейки в памяти) и storage/machinelogs (своя БД).
//
// Слой: application (тестовая опора, без ввода-вывода); в сборку ролей не входит.
// Владелец: эпик 23 (MachineLogs).
package mltest

import (
	"fmt"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
)

// Scenario — записи сценария в порядке поступления и ожидаемое.
type Scenario struct {
	Facts []engineapp.Out
	// WindowItems — изделия окна нарушения (FR-151); WithDefect — из них с
	// найденным дефектом.
	WindowItems []string
	WithDefect  []string
	// Runs — выполнения сценария по изделиям.
	Runs map[string]string
}

type builder struct {
	run string
	sc  Scenario
}

func stamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

func (b *builder) id(name string) string {
	return kernel.UUIDv5(constants.NsAnt, "mltest\x1f"+b.run+"\x1f"+name)
}

func (b *builder) fact(name string, t catalog.Type, stream, item string, at time.Time, data any) {
	id := b.id(name)
	b.sc.Facts = append(b.sc.Facts, engineapp.Out{EventID: id, Type: t, Kind: catalog.KindFact, Stream: stream, ItemID: item,
		RunID: b.run, OccurredAt: at, Correlation: id, Data: data})
}

func (b *builder) runRecords(runID, item, step, eq string, from, to time.Time) {
	b.fact("start-"+runID, catalog.OperationRunStarted, "item:"+item, item, from, map[string]any{
		"operation_run_id": runID, "operation_code": "030", "step_key": step, "equipment_id": eq,
		"operator_id": "W-07", "operation_started_at": stamp(from)})
	b.fact("finish-"+runID, catalog.OperationRunFinished, "item:"+item, item, to, map[string]any{
		"operation_run_id": runID, "completion": "completed", "operation_finished_at": stamp(to)})
	if b.sc.Runs == nil {
		b.sc.Runs = map[string]string{}
	}
	b.sc.Runs[item] = runID
}

func m(v int64, scale int, unit string) map[string]any {
	return map[string]any{"value": v, "scale": scale, "unit": unit}
}

func tol(nom, lo, hi int64, scale int, unit string) map[string]any {
	return map[string]any{"nominal": m(nom, scale, unit), "lower": m(lo, scale, unit), "upper": m(hi, scale, unit)}
}

func (b *builder) equipment(name string, t catalog.Type, eq string, at time.Time, data map[string]any) {
	data["equipment_id"] = eq
	b.fact(name, t, "equipment:"+eq, "", at, data)
}

// WeldingOutOfRegime — «сварка вне режима» (FR-151, PRD §11.12): восемь
// сварок шва W-1 (шаг welding.weld — специальный процесс) на источнике IS-2
// по 10 минут через 15; ток вне уставки 160 ± 10 А с 20-й по 95-ю минуту;
// дефект при контроле найден только у трёх из шести изделий окна.
func WeldingOutOfRegime(t0 time.Time, run string) Scenario {
	b := &builder{run: run}
	at := func(min int) time.Time { return t0.Add(time.Duration(min) * time.Minute) }
	b.equipment("prog", catalog.EquipmentProgramChanged, "IS-2", at(-10), map[string]any{"program_ref": "PS-4", "program_revision": "3", "planned": true})
	b.equipment("st-auto", catalog.EquipmentStateChanged, "IS-2", at(-1), map[string]any{"execution": "idle", "controller_mode": "automatic", "condition": "normal"})
	for i := range 8 {
		item, runID := fmt.Sprintf("ENT01:%sFL-%d", prefix(run), i), fmt.Sprintf("%sRUN-W-%d", prefix(run), i)
		b.runRecords(runID, item, "welding.weld", "IS-2", at(i*15), at(i*15+10))
		if i >= 1 && i <= 6 {
			b.sc.WindowItems = append(b.sc.WindowItems, item)
		}
		if i >= 1 && i <= 3 {
			b.sc.WithDefect = append(b.sc.WithDefect, item)
			b.fact(fmt.Sprintf("defect-%d", i), catalog.QualityDefectIdentified, "item:"+item, item, at(i*15+40), map[string]any{
				"defect_id": fmt.Sprintf("D-%d", i), "zone_id": "W-1", "first_observation_event_id": b.id(fmt.Sprintf("obs-%d", i)),
				"defect_type_code": "burn_through"})
		}
	}
	b.equipment("dev-current", catalog.EquipmentDeviationDetected, "IS-2", at(95), map[string]any{
		"deviation_kind": "out_of_setpoint", "parameter": "current", "value": m(1800, 1, "A"), "setpoint": tol(1600, 1500, 1700, 1, "A"),
		"started_at": stamp(at(20)), "ended_at": stamp(at(95))})
	return b.sc
}

// CNCTwoDeviations — станок ЧПУ CNC-1 (FR-149): нормальное выполнение, затем
// выполнение с двумя отклонениями — ручная коррекция подачи 130 % и
// перегрузка шпинделя; до операции чисто, программа O1001 ред. B,
// инструмент T05 с ресурсом 73/75.
func CNCTwoDeviations(t0 time.Time, run string) Scenario {
	b := &builder{run: run}
	at := func(min int) time.Time { return t0.Add(time.Duration(min) * time.Minute) }
	p := prefix(run)
	b.equipment("prog", catalog.EquipmentProgramChanged, "CNC-1", at(-10), map[string]any{"program_ref": "O1001", "program_revision": "B", "planned": true})
	b.equipment("tool", catalog.EquipmentToolChanged, "CNC-1", at(-9), map[string]any{"tool_id": "T05", "tool_life_used": 72, "tool_life_limit": 75})
	// Нормальное выполнение.
	b.equipment("st-run-1", catalog.EquipmentStateChanged, "CNC-1", at(0), map[string]any{"execution": "running", "controller_mode": "automatic", "condition": "normal"})
	b.equipment("cycle-1", catalog.EquipmentCycleSummarized, "CNC-1", at(9), map[string]any{
		"window_start": stamp(at(0)), "window_end": stamp(at(9)), "cycle_ref": "C-1", "raw_ref": "edge:CNC-1/C-1",
		"parameters": []any{
			map[string]any{"parameter": "spindle_load", "mean": m(64, 0, "%"), "max": m(78, 0, "%"), "min": m(40, 0, "%"), "setpoint": tol(80, 0, 100, 0, "%"), "out_of_setpoint_ms": 0},
			map[string]any{"parameter": "feed_override", "mean": m(100, 0, "%"), "max": m(100, 0, "%"), "min": m(100, 0, "%"), "setpoint": tol(100, 100, 100, 0, "%"), "out_of_setpoint_ms": 0},
		}})
	b.equipment("st-idle-1", catalog.EquipmentStateChanged, "CNC-1", at(10), map[string]any{"execution": "idle", "controller_mode": "automatic", "condition": "normal"})
	b.runRecords(p+"RUN-C-1", "ENT01:"+p+"FL-C-1", "machining.turn", "CNC-1", at(0), at(10))
	// Выполнение с двумя отклонениями.
	b.equipment("tool-2", catalog.EquipmentToolChanged, "CNC-1", at(19), map[string]any{"tool_id": "T05", "tool_life_used": 73, "tool_life_limit": 75})
	b.equipment("st-run-2", catalog.EquipmentStateChanged, "CNC-1", at(20), map[string]any{"execution": "running", "controller_mode": "automatic", "condition": "normal"})
	b.equipment("dev-feed", catalog.EquipmentDeviationDetected, "CNC-1", at(26), map[string]any{
		"deviation_kind": "manual_override", "parameter": "feed_override", "value": m(130, 0, "%"), "setpoint": tol(100, 100, 100, 0, "%"),
		"started_at": stamp(at(23)), "ended_at": stamp(at(26)), "code": "feed"})
	b.equipment("dev-load", catalog.EquipmentDeviationDetected, "CNC-1", at(25), map[string]any{
		"deviation_kind": "overload", "parameter": "spindle_load", "value": m(118, 0, "%"), "setpoint": tol(80, 0, 100, 0, "%"),
		"started_at": stamp(at(24)), "ended_at": stamp(at(25))})
	b.equipment("cycle-2", catalog.EquipmentCycleSummarized, "CNC-1", at(29), map[string]any{
		"window_start": stamp(at(20)), "window_end": stamp(at(29)), "cycle_ref": "C-2", "raw_ref": "edge:CNC-1/C-2",
		"parameters": []any{
			map[string]any{"parameter": "spindle_load", "mean": m(71, 0, "%"), "max": m(118, 0, "%"), "min": m(40, 0, "%"), "setpoint": tol(80, 0, 100, 0, "%"), "out_of_setpoint_ms": 60000},
			map[string]any{"parameter": "feed_override", "mean": m(115, 0, "%"), "max": m(130, 0, "%"), "min": m(100, 0, "%"), "setpoint": tol(100, 100, 100, 0, "%"), "out_of_setpoint_ms": 180000},
		}})
	b.equipment("st-idle-2", catalog.EquipmentStateChanged, "CNC-1", at(30), map[string]any{"execution": "idle", "controller_mode": "automatic", "condition": "normal"})
	b.runRecords(p+"RUN-C-2", "ENT01:"+p+"FL-C-2", "machining.turn", "CNC-1", at(20), at(30))
	return b.sc
}

// prefix — префикс локальных ID прогона сценария (AD-38).
func prefix(run string) string {
	if run == "" {
		return ""
	}
	return run + "-"
}
