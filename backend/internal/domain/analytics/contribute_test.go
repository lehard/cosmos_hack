package analytics_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"reflect"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	domain "ant/internal/domain/analytics"
	"ant/internal/domain/kernel"
)

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

const item = "ENT01:FL-017"

// rec — запись входа изделия: факт источника (machine), решение (manual) или реакция.
func rec(id string, typ catalog.Type, at time.Duration, data string) kernel.Record {
	info, _ := catalog.Lookup(typ)
	r := kernel.Record{EventID: id, Type: typ, Kind: info.Kind, ItemID: item, Stream: "item:" + item,
		OccurredAt: t0.Add(at), ReceivedAt: t0.Add(at), Data: json.RawMessage(data)}
	if info.Kind == catalog.KindFact {
		r.SourceKind = "machine"
	}
	return r
}

// flange — нормальное изготовление: запуск, сварка, контроль, предъявление ЗТ-3, выпуск.
func flange() []kernel.Record {
	return []kernel.Record{
		rec("e01", catalog.ItemItemRegistered, 0, `{"item_id":"ENT01:FL-017","item_type_id":"FL-100","item_revision":"A","process_version_hash":"x","normative_rev":"r1"}`),
		rec("e02", catalog.OperationRunStarted, 10*time.Minute, `{"operation_run_id":"RUN-1","operation_code":"welding","step_key":"welding.weld","station_id":"ST-2","line_id":"LINE-FL-1","equipment_id":"IS-2","operator_id":"W21"}`),
		rec("e03", catalog.OperationRunFinished, 40*time.Minute, `{"operation_run_id":"RUN-1","completion":"completed"}`),
		rec("e04", catalog.InspectionResultRecorded, 45*time.Minute, `{"observation_id":"OBS-1","method":"visual","phase":"after_operation","step_key":"welding.kt3_camera","operation_run_id":"RUN-1","outcome":"no_defect_indicated","processing_state":"completed"}`),
		rec("e05", catalog.ItemPresentationRecorded, 50*time.Minute, `{"step_key":"welding.zt3_acceptance","presentation_no":1,"presented_to":"qc","presented_by":"M01"}`),
		rec("e06", catalog.DecisionPresentationResolved, 70*time.Minute, `{"step_key":"welding.zt3_acceptance","closing_point":"ZT-3","resolution":"accept","presentation_no":1,"method_event_ids":["e04"]}`),
		rec("e07", catalog.ItemReleaseRecorded, 120*time.Minute, `{"warehouse_id":"WH-1","after_rework":false,"received_by":"K01"}`),
	}
}

func count(rows []domain.Row, metric string) (n int64) {
	for _, r := range rows {
		if r.Metric == metric {
			n += r.Value
		}
	}
	return n
}

func find(rows []domain.Row, metric string) []domain.Row {
	var out []domain.Row
	for _, r := range rows {
		if r.Metric == metric {
			out = append(out, r)
		}
	}
	return out
}

func TestNormalFlow(t *testing.T) {
	rows := domain.Contribute(item, flange())
	want := map[string]int64{
		domain.RowInspectedItems: 1, domain.RowFPYTotal: 1, domain.RowFPYPass: 1, domain.RowPassed: 3,
		domain.RowInspections: 1, domain.RowDefectsDetected: 0, domain.RowItemsWithConfirmedNC: 0, domain.RowPresentations: 1,
	}
	for _, metric := range slices.Sorted(maps.Keys(want)) {
		want := want[metric]
		if got := count(rows, metric); got != want {
			t.Errorf("%s = %d, ожидалось %d", metric, got, want)
		}
	}
	lead := find(rows, domain.RowLeadTime)
	if len(lead) != 1 || lead[0].Value != 7200 || lead[0].Dims.DurationOrigin != domain.OriginSystem {
		t.Fatalf("время в системе: %+v", lead)
	}
	d := find(rows, domain.RowOperationDuration)
	if len(d) != 1 || d[0].Value != 1800 || d[0].Dims.Meaning != domain.MeaningStation || d[0].Dims.Performer != "W21" {
		t.Fatalf("длительность: %+v", d)
	}
	q := find(rows, domain.RowQueue)
	if len(q) != 1 || q[0].Dims.Step != "welding.zt3_acceptance" || q[0].Until == nil || q[0].Until.Sub(q[0].At) != 20*time.Minute {
		t.Fatalf("очередь на ЗТ-3: %+v", q)
	}
	for _, r := range rows {
		if len(r.Sources) == 0 {
			t.Fatalf("строка %s без исходных записей — не раскрывается", r.Metric)
		}
	}
	if k := find(rows, domain.RowInspections)[0].Kinds; !reflect.DeepEqual(k, []string{"machine"}) {
		t.Fatalf("вид источника: %v", k)
	}
}

// Повтор записи и повтор факта с новым event_id (тот же operation_run_id,
// observation_id, номер предъявления) не меняют показатели (кейс §5.2).
func TestRepeatDoesNotDoubleCount(t *testing.T) {
	base := domain.Contribute(item, flange())
	in := flange()
	in = append(in, in[1], in[3]) // тот же event_id дважды
	dup := func(r kernel.Record, id string, shift time.Duration) kernel.Record {
		r.EventID = id
		r.OccurredAt = r.OccurredAt.Add(shift)
		r.ReceivedAt = r.ReceivedAt.Add(time.Hour)
		return r
	}
	in = append(in, dup(in[1], "e02-bis", 0), dup(in[2], "e03-bis", time.Minute), dup(in[3], "e04-bis", 0), dup(in[4], "e05-bis", 0), dup(in[5], "e06-bis", 0), dup(in[6], "e07-bis", 0))
	got := domain.Contribute(item, in)
	for _, m := range []string{domain.RowInspectedItems, domain.RowInspections, domain.RowPassed, domain.RowPresentations, domain.RowLeadTime, domain.RowOperationDuration, domain.RowQueue, domain.RowComparableRuns} {
		if count(got, m) != count(base, m) {
			t.Errorf("%s: повтор изменил показатель %d → %d", m, count(base, m), count(got, m))
		}
	}
}

// Позднее событие встаёт на своё occurred_at: результат тот же, что при
// приходе вовремя, — меняется ровно то, что добавило событие.
func TestLateEvent(t *testing.T) {
	onTime := flange()
	defect := rec("e08", catalog.InspectionResultRecorded, 44*time.Minute, `{"observation_id":"OBS-RT","method":"radiography","phase":"after_operation","step_key":"welding.kt3_radiography","operation_run_id":"RUN-1","outcome":"defect_indicated","processing_state":"completed","defects":[{"defect_type_code":"POR","zone_id":"W1","location":"12h","severity":"major"}]}`)
	late := defect
	late.ReceivedAt = t0.Add(5 * time.Hour) // пришло после выпуска
	a := domain.Contribute(item, append(onTime, defect))
	b := domain.Contribute(item, append(flange(), late))
	strip := func(rs []domain.Row) []domain.Row { return rs }
	if !reflect.DeepEqual(strip(a), strip(b)) {
		t.Fatal("позднее событие дало другие строки, чем пришедшее вовремя")
	}
	base := domain.Contribute(item, flange())
	if got := count(b, domain.RowDefectsDetected) - count(base, domain.RowDefectsDetected); got != 1 {
		t.Fatalf("позднее событие добавило %d дефектов, ожидался 1", got)
	}
	if count(b, domain.RowFPYPass) != 0 || count(base, domain.RowFPYPass) != 1 {
		t.Fatal("поздний дефект: «с первого раза» не пересчитано")
	}
	if count(b, domain.RowInspectedItems) != 1 {
		t.Fatal("изделие проверено — одно")
	}
}

// Повторные наблюдения одного дефекта — один дефект; уточнение вида
// обновляет вид (FR-37); подтверждение и отклонение (кейс §5.2).
func TestDefectIdentityAndDecisions(t *testing.T) {
	obs := func(id string, at time.Duration, dtype string) kernel.Record {
		return rec(id, catalog.InspectionResultRecorded, at, fmt.Sprintf(`{"observation_id":"%s","method":"visual","phase":"after_operation","step_key":"welding.kt3_camera","operation_run_id":"RUN-1","outcome":"defect_indicated","processing_state":"completed","defects":[{"defect_type_code":"%s","zone_id":"W1","location":"12h","severity":"major"}]}`, id, dtype))
	}
	in := flange()[:3]
	in = append(in, obs("o1", 41*time.Minute, "UNK"), obs("o2", 42*time.Minute, "POR"), obs("o3", 43*time.Minute, "POR"))
	rows := domain.Contribute(item, in)
	det := find(rows, domain.RowDefectsDetected)
	if len(det) != 1 || det[0].Dims.DefectType != "POR" || len(det[0].Sources) != 3 {
		t.Fatalf("один физический дефект с тремя наблюдениями: %+v", det)
	}
	if count(rows, domain.RowInspections) != 3 || count(rows, domain.RowInspectionsWithDefect) != 3 {
		t.Fatal("наблюдения считаются отдельно от дефектов")
	}
	confirmed := append(append([]kernel.Record{}, in...), rec("d1", catalog.DecisionNonconformityConfirmed, 60*time.Minute, `{"nc_id":"NC-1","signal_ids":["S-1"],"severity":"major","defect_type_code":"POR","reason":{"text":"пора"}}`))
	rows = domain.Contribute(item, confirmed)
	if count(rows, domain.RowConfirmedDefects) != 1 || count(rows, domain.RowItemsWithConfirmedNC) != 1 || count(rows, domain.RowNCConfirmed) != 1 {
		t.Fatalf("подтверждение: %v", rows)
	}
	cd := find(rows, domain.RowConfirmedDefects)[0]
	if cd.Dims.Origin != domain.DefectProduction || cd.Dims.Performer != "W21" || cd.Dims.NC != "NC-1" {
		t.Fatalf("срез подтверждённого дефекта: %+v", cd.Dims)
	}
	if !slicesContains(cd.Kinds, domain.SourceManual) || !slicesContains(cd.Kinds, "machine") {
		t.Fatalf("виды источника: %v", cd.Kinds)
	}
	if dd := find(rows, domain.RowDetectionDelay); len(dd) != 1 || dd[0].Value != 60 {
		t.Fatalf("задержка обнаружения: %+v", dd)
	}
	rejected := append(append([]kernel.Record{}, in...), rec("d2", catalog.DecisionSignalRejected, 60*time.Minute, `{"signal_ids":["S-1"],"reason":{"text":"блик"}}`))
	rows = domain.Contribute(item, rejected)
	if count(rows, domain.RowDefectsDetected) != 0 || count(rows, domain.RowConfirmedDefects) != 0 {
		t.Fatal("отклонённый сигнал — не дефект («под подозрением» ≠ брак)")
	}
}

// Входной брак отличим от производственного (FR-87).
func TestIncomingDefect(t *testing.T) {
	in := []kernel.Record{
		rec("e1", catalog.InspectionResultRecorded, 0, `{"method":"visual","phase":"incoming","step_key":"incoming.kt1","outcome":"defect_indicated","processing_state":"completed","defects":[{"defect_type_code":"CRK","zone_id":"Z","severity":"major"}]}`),
		rec("e2", catalog.DecisionNonconformityConfirmed, time.Minute, `{"nc_id":"NC-9","signal_ids":["S"],"severity":"major","reason":{"text":"трещина заготовки"}}`),
	}
	rows := domain.Contribute(item, in)
	cd := find(rows, domain.RowConfirmedDefects)
	if len(cd) != 1 || cd[0].Dims.Origin != domain.DefectIncoming {
		t.Fatalf("входной брак: %+v", cd)
	}
}

// Происхождение длительности (соглашение «Длительности»): передано
// источником — со смыслом интервала источника; паузы — активная обработка.
func TestDurationOrigin(t *testing.T) {
	in := flange()[:2]
	in = append(in, rec("f", catalog.OperationRunFinished, 40*time.Minute, `{"operation_run_id":"RUN-1","completion":"completed","reported_duration":{"value":25,"unit":"min","meaning":"active_processing","origin":"source_reported"}}`))
	d := find(domain.Contribute(item, in), domain.RowOperationDuration)
	if len(d) != 1 || d[0].Value != 1500 || d[0].Dims.DurationOrigin != domain.OriginSource || d[0].Dims.Meaning != domain.MeaningActive {
		t.Fatalf("длительность источника: %+v", d)
	}
	in = flange()[:2]
	in = append(in,
		rec("p", catalog.OperationRunPaused, 20*time.Minute, `{"operation_run_id":"RUN-1","pause_reason":"setup"}`),
		rec("r", catalog.OperationRunResumed, 25*time.Minute, `{"operation_run_id":"RUN-1"}`),
		flange()[2])
	d = find(domain.Contribute(item, in), domain.RowOperationDuration)
	if len(d) != 1 || d[0].Value != 1500 || d[0].Dims.DurationOrigin != domain.OriginSystem || d[0].Dims.Meaning != domain.MeaningActive {
		t.Fatalf("длительность за вычетом паузы: %+v", d)
	}
}

// Повторное выполнение (FR-47) и исправление записи (FR-122).
func TestReworkAndCorrection(t *testing.T) {
	in := flange()[:3]
	in = append(in, rec("rw", catalog.OperationRunStarted, 50*time.Minute, `{"operation_run_id":"RUN-2","operation_code":"welding","step_key":"welding.weld","operator_id":"W22","rework_of":"RUN-1"}`),
		rec("rwf", catalog.OperationRunFinished, 70*time.Minute, `{"operation_run_id":"RUN-2","completion":"completed"}`))
	rows := domain.Contribute(item, in)
	if count(rows, domain.RowReworkRuns) != 1 || count(rows, domain.RowComparableRuns) != 2 || count(rows, domain.RowReworkTime) != 1200 {
		t.Fatalf("повтор операции: %d %d %d", count(rows, domain.RowReworkRuns), count(rows, domain.RowComparableRuns), count(rows, domain.RowReworkTime))
	}
	fix := rec("fix", catalog.OperationRunStarted, 50*time.Minute, `{"operation_run_id":"RUN-2","operation_code":"welding","step_key":"welding.weld","operator_id":"W21","rework_of":"RUN-1"}`)
	fix.Corrects = "rw"
	rows = domain.Contribute(item, append(in, fix))
	for _, r := range find(rows, domain.RowReworkRuns) {
		if r.Dims.Performer != "W21" {
			t.Fatalf("исправление не применено: %+v", r.Dims)
		}
	}
}

// Детерминизм: перестановка входа не меняет строки (AD-4, N воркеров).
func TestDeterministic(t *testing.T) {
	in := flange()
	rev := make([]kernel.Record, len(in))
	for i := range in {
		rev[len(in)-1-i] = in[i]
	}
	if !reflect.DeepEqual(domain.Contribute(item, in), domain.Contribute(item, rev)) {
		t.Fatal("порядок подачи входа изменил строки")
	}
}

func slicesContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
