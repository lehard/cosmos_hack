package analysis_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/domain/analysis"
	"ant/internal/domain/crossitem"
	"ant/internal/domain/kernel"
)

// Мир главной истории «плохой день сварочного участка» для тестов области
// риска (FR-61, FR-9): сварки ФЛ-100 по мотивам scenarios/fixtures/flange-bad-day
// (world.yaml): 40 фланцев заказа и 4 фоновых, ИС-1 и ИС-2, сварщики W21 и W22,
// одна технологическая программа сварки. Подтверждённо годные (выпущенные) —
// Ф-001 и Ф-006; прожог на Ф-017 (НС-01).

var msk = time.FixedZone("MSK", 3*3600)

// at — момент «ДД ЧЧ:ММ» сентября 2026 (московское время).
func at(day, h, m int) time.Time { return time.Date(2026, 9, day, h, m, 0, 0, msk).UTC() }

const (
	stepWeld = "welding.weld"
	prog     = "WPS-12"
)

// weld — сварка изделия: пост-источник, сварщик, начало и конец.
type weld struct {
	item     string
	src      string
	welder   string
	from, to time.Time
}

// span — интервал «ДД ЧЧ:ММ-ЧЧ:ММ» (через полночь — следующий день).
func span(day, h1, m1, h2, m2 int) (time.Time, time.Time) {
	from, to := at(day, h1, m1), at(day, h2, m2)
	if to.Before(from) {
		to = to.Add(24 * time.Hour)
	}
	return from, to
}

func w(item, src, welder string, day, h1, m1, h2, m2 int) weld {
	f, t := span(day, h1, m1, h2, m2)
	return weld{item: "ENT01:" + item, src: src, welder: welder, from: f, to: t}
}

// storyWelds — сварки главной истории (world.yaml, items).
func storyWelds() []weld {
	return []weld{
		w("F-001", "IS-2", "W21", 21, 9, 0, 9, 45), w("F-002", "IS-1", "W21", 21, 10, 0, 10, 40),
		w("F-003", "IS-1", "W21", 21, 10, 50, 11, 30), w("F-004", "IS-2", "W21", 21, 11, 40, 12, 20),
		w("F-005", "IS-1", "W21", 21, 12, 30, 13, 10), w("F-006", "IS-2", "W21", 21, 14, 10, 14, 55),
		w("F-007", "IS-1", "W21", 21, 15, 0, 15, 35), w("F-008", "IS-2", "W21", 21, 15, 40, 16, 20),
		w("F-009", "IS-1", "W22", 21, 16, 40, 17, 15), w("F-010", "IS-2", "W22", 21, 17, 30, 18, 10),
		w("F-011", "IS-1", "W22", 21, 18, 20, 18, 55), w("F-012", "IS-2", "W22", 21, 22, 10, 22, 50),
		w("F-013", "IS-1", "W22", 21, 19, 0, 19, 40), w("F-014", "IS-2", "W21", 22, 8, 20, 8, 43),
		w("F-015", "IS-2", "W21", 22, 10, 15, 11, 0), w("F-016", "IS-2", "W21", 22, 9, 20, 10, 5),
		w("F-017", "IS-2", "W21", 23, 10, 40, 11, 0), w("F-018", "IS-1", "W21", 22, 11, 40, 12, 15),
		w("F-019", "IS-2", "W22", 22, 16, 35, 17, 20), w("F-020", "IS-1", "W21", 22, 12, 20, 12, 55),
		w("F-021", "IS-2", "W22", 22, 19, 40, 20, 20), w("F-022", "IS-1", "W21", 22, 13, 0, 13, 35),
		w("F-023", "IS-2", "W21", 23, 8, 30, 9, 15), w("F-024", "IS-1", "W21", 22, 13, 40, 14, 15),
		w("F-025", "IS-2", "W21", 23, 9, 35, 10, 20), w("F-026", "IS-1", "W21", 22, 14, 20, 14, 55),
		w("F-027", "IS-1", "W22", 22, 18, 0, 18, 40), w("F-028", "IS-1", "W22", 22, 18, 50, 19, 25),
		w("F-029", "IS-1", "W22", 22, 21, 30, 22, 10), w("F-030", "IS-1", "W22", 21, 23, 0, 23, 35),
		w("F-031", "IS-1", "W22", 22, 20, 35, 21, 10), w("F-032", "IS-1", "W22", 22, 22, 20, 22, 55),
		w("F-033", "IS-1", "W21", 22, 15, 0, 15, 35), w("F-034", "IS-1", "W21", 22, 15, 40, 16, 15),
		w("F-035", "IS-1", "W21", 22, 11, 5, 11, 35), w("F-036", "IS-1", "W22", 22, 23, 50, 0, 25),
		w("F-221", "IS-2", "W22", 21, 19, 50, 20, 30), w("F-222", "IS-2", "W21", 22, 8, 45, 9, 10),
		w("F-223", "IS-1", "W22", 21, 20, 40, 21, 15), w("F-224", "IS-1", "W22", 22, 23, 5, 23, 40),
	}
}

// journal — записи в порядке seq: вход стадии и её адресованный выход.
type journal struct {
	seq     int64
	n       int
	records []kernel.Record
	stage   crossitem.Stage
	out     []kernel.Record
}

func (j *journal) id(kind string) string {
	j.n++
	return kernel.UUIDv5(constants.NsAnt, "test\x1f"+kind+"\x1f"+strconv.Itoa(j.n))
}

// add — запись журнала (факт или решение); стадия сворачивает её сразу —
// как роль crossitem, через точку подключения crossitem.Fold.
func (j *journal) add(t catalog.Type, kind catalog.Kind, stream, item, actor string, occurred time.Time, data any) kernel.Record {
	raw, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	j.seq++
	r := kernel.Record{Seq: j.seq, EventID: j.id(string(t)), Type: t, Kind: kind, Stream: stream, ItemID: item, Actor: actor,
		OccurredAt: occurred, ReceivedAt: occurred, RecordedAt: occurred, Data: raw, SourceKind: "device"}
	j.records = append(j.records, r)
	j.fold(r)
	return r
}

// fold — шаг стадии и запись её выхода следующими seq (AD-42).
func (j *journal) fold(r kernel.Record) {
	var out []kernel.Addressed
	j.stage, out = crossitem.Fold(j.stage, r)
	for _, a := range out {
		raw, err := json.Marshal(a.Data)
		if err != nil {
			panic(err)
		}
		j.seq++
		item := strings.TrimPrefix(a.Stream, "item:")
		if item == a.Stream {
			item = ""
		}
		rec := kernel.Record{Seq: j.seq, EventID: crossitem.AddressedID(a), Type: a.Type, Kind: catalog.KindReaction, Stream: a.Stream,
			ItemID: item, OccurredAt: a.OccurredAt, ReceivedAt: a.OccurredAt, RecordedAt: a.OccurredAt, CausationID: r.EventID, Data: raw}
		j.records = append(j.records, rec)
		j.out = append(j.out, rec)
		j.fold(rec)
	}
}

func (j *journal) fact(t catalog.Type, item string, occurred time.Time, data any) kernel.Record {
	stream := "item:" + item
	if item == "" {
		stream = "global"
	}
	return j.add(t, catalog.KindFact, stream, item, "", occurred, data)
}

func (j *journal) decision(t catalog.Type, stream, item, actor string, occurred time.Time, data any) kernel.Record {
	return j.add(t, catalog.KindDecision, stream, item, actor, occurred, data)
}

// weldRun — выполнение сварки: начало и конец (факты process, publish: stage).
func (j *journal) weldRun(x weld) (start, finish kernel.Record) {
	run := "RUN-" + strings.TrimPrefix(x.item, "ENT01:") + "-" + strconv.FormatInt(x.from.Unix(), 36)
	start = j.fact(catalog.OperationRunStarted, x.item, x.from, map[string]any{
		"operation_run_id": run, "step_key": stepWeld, "operation_code": "030", "equipment_id": x.src,
		"operator_id": x.welder, "program_ref": prog})
	finish = j.fact(catalog.OperationRunFinished, x.item, x.to, map[string]any{"operation_run_id": run, "completion": "completed"})
	return start, finish
}

// of — выход стадии типа t.
func (j *journal) of(t catalog.Type) []kernel.Record {
	var out []kernel.Record
	for _, r := range j.out {
		if r.Type == t {
			out = append(out, r)
		}
	}
	return out
}

// incident — проекция analysis.incident по всем записям журнала (роль projector).
func (j *journal) incident(id string) analysis.IncidentRecord {
	var v analysis.IncidentRecord
	for _, r := range j.records {
		for _, k := range analysis.IncidentKeys(r) {
			if k == id {
				v = analysis.StepIncident(k, v, r)
			}
		}
	}
	return v
}

// storyUntilNC01 — мир до подтверждения НС-01 включительно: сварки, выпуск
// Ф-001 и Ф-006, находка и подтверждение прожога на Ф-017.
func storyUntilNC01() (*journal, kernel.Record) {
	j := &journal{}
	type ev struct {
		at time.Time
		f  func()
	}
	var evs []ev
	for _, x := range storyWelds() {
		evs = append(evs, ev{x.from, func() { j.weldRun(x) }})
	}
	evs = append(evs,
		ev{at(22, 15, 0), func() {
			j.fact(catalog.ItemReleaseRecorded, "ENT01:F-006", at(22, 15, 0), map[string]any{"received_by": "STK-51", "warehouse_id": "WH-FG", "after_rework": false})
		}},
		ev{at(23, 10, 30), func() {
			j.fact(catalog.ItemReleaseRecorded, "ENT01:F-001", at(23, 10, 30), map[string]any{"received_by": "STK-51", "warehouse_id": "WH-FG", "after_rework": false})
		}},
	)
	// Сварки упорядочены по времени начала, как пришли бы с участка.
	for i := 1; i < len(evs); i++ {
		for k := i; k > 0 && evs[k].at.Before(evs[k-1].at); k-- {
			evs[k], evs[k-1] = evs[k-1], evs[k]
		}
	}
	for _, e := range evs {
		e.f()
	}
	j.fact(catalog.InspectionResultRecorded, "ENT01:F-017", at(23, 11, 6), map[string]any{
		"outcome": "defect_indicated", "phase": "after_operation", "method": "camera", "processing_state": "complete",
		"zone_ids": []string{"W-1.U2"}, "defects": []map[string]any{{"zone_id": "W-1.U2", "defect_type_code": "burn_through", "severity": "major"}}})
	nc := j.decision(catalog.DecisionNonconformityConfirmed, "item:ENT01:F-017", "ENT01:F-017", "ins-01@1", at(23, 11, 8), map[string]any{
		"nc_id": "NC-01", "defect_type_code": "burn_through", "severity": "major", "signal_ids": []string{"SIG-01"},
		"reason": map[string]any{"text": "Прожог шва У2"}})
	return j, nc
}

// items — локальные id изделий множества.
func localIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, analysis.LocalID(id))
	}
	return out
}
