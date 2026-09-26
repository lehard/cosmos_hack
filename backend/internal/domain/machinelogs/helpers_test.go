package machinelogs_test

import (
	"encoding/json"
	"fmt"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
	ml "ant/internal/domain/machinelogs"
)

// t0 — начало смены сценария тестов.
var t0 = time.Date(2026, 9, 26, 7, 0, 0, 0, time.UTC)

func at(min int) time.Time { return t0.Add(time.Duration(min) * time.Minute) }

func stamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// journal — последовательность записей с растущим seq (порядок знания).
type journal struct {
	seq  int64
	recs []kernel.Record
}

func (j *journal) add(id string, t catalog.Type, item, run string, occurred time.Time, data any) kernel.Record {
	j.seq++
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	r := kernel.Record{Seq: j.seq, EventID: id, Type: t, ItemID: item, RunID: run, SourceKind: "machine",
		OccurredAt: occurred, ReceivedAt: occurred, RecordedAt: occurred, Data: b}
	if item != "" {
		r.Stream = "item:" + item
	}
	j.recs = append(j.recs, r)
	return r
}

func (j *journal) started(runID, item, step, eq string, from time.Time) kernel.Record {
	return j.add("start-"+runID, catalog.OperationRunStarted, item, "", from, map[string]any{
		"operation_run_id": runID, "operation_code": "030", "step_key": step, "equipment_id": eq,
		"operator_id": "W-07", "operation_started_at": stamp(from)})
}

func (j *journal) finished(runID, item string, to time.Time) kernel.Record {
	return j.add("finish-"+runID, catalog.OperationRunFinished, item, "", to, map[string]any{
		"operation_run_id": runID, "completion": "completed", "operation_finished_at": stamp(to)})
}

func measure(v int64, scale int, unit string) map[string]any {
	return map[string]any{"value": v, "scale": scale, "unit": unit}
}

func tol(nom, lo, hi int64, scale int, unit string) map[string]any {
	return map[string]any{"nominal": measure(nom, scale, unit), "lower": measure(lo, scale, unit), "upper": measure(hi, scale, unit)}
}

func (j *journal) deviation(id, eq, kind, param string, v map[string]any, sp map[string]any, from, to time.Time) kernel.Record {
	d := map[string]any{"equipment_id": eq, "deviation_kind": kind, "parameter": param, "started_at": stamp(from), "ended_at": stamp(to)}
	if v != nil {
		d["value"] = v
	}
	if sp != nil {
		d["setpoint"] = sp
	}
	return j.add(id, catalog.EquipmentDeviationDetected, "", "", to, d)
}

func (j *journal) state(id, eq, exec, mode, cond string, t time.Time) kernel.Record {
	return j.add(id, catalog.EquipmentStateChanged, "", "", t, map[string]any{"equipment_id": eq, "execution": exec, "controller_mode": mode, "condition": cond})
}

// addressedRecord — адресованная запись стадии как вход свёртки изделия.
func addressedRecord(j *journal, a kernel.Addressed) kernel.Record {
	item := ""
	if len(a.Stream) > 5 && a.Stream[:5] == "item:" {
		item = a.Stream[5:]
	}
	return j.add(fmt.Sprintf("stage-%s-%s", a.Type, a.Key), a.Type, item, "", a.OccurredAt, a.Data)
}

// ncModule — модуль-владелец типа несоответствия; переменная, а не константа:
// заглушка порта в тесте строит запись от имени nonconformity (эпик 21).
var (
	ncModule kernel.Module = "nonconformity"
	ncType                 = catalog.DecisionNonconformityRegistered
)

// fakeRegistrar — заготовка функции-намерения nonconformity для тестов (FR-151).
func fakeRegistrar(q ml.NCRequest) (kernel.Addressed, error) {
	return kernel.NewAddressed(ncModule, ncType, "item:"+q.ItemID, q.Key(),
		map[string]any{"nc_id": q.NCID, "violation_window_event_id": q.WindowEventID, "operation_run_id": q.OperationRunID, "step_key": q.StepKey},
		q.CauseRecords()...)
}

// runStage прогоняет записи через стадию и собирает выход.
func runStage(p ml.StagePorts, recs []kernel.Record) (ml.StageState, []kernel.Addressed) {
	var s ml.StageState
	var out []kernel.Addressed
	f := ml.StageWith(p)
	for _, r := range recs {
		var a []kernel.Addressed
		s, a = f(s, r)
		out = append(out, a...)
	}
	return s, out
}
