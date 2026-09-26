package crossitem_test

// Решение Д-40: поздний модуль стадии видит адресованные записи ранних в том
// же проходе. Главная история — сбой сварочного источника: окно нарушения
// machinelogs (equipment.violation.window_resolved) доходит до analysis и
// расширяет или открывает область риска. Модули — настоящие machinelogs и
// analysis, через crossitem.Fold.

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/domain/analysis"
	"ant/internal/domain/crossitem"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
)

var t0 = time.Date(2026, 9, 26, 7, 0, 0, 0, time.UTC)

func at(min int) time.Time { return t0.Add(time.Duration(min) * time.Minute) }

func stamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// feed — вход стадии с растущим seq; каждая запись сразу проходит Fold.
type feed struct {
	t   *testing.T
	seq int64
	st  crossitem.Stage
	out map[string][]kernel.Addressed // имя записи → выход её прохода
}

func newFeed(t *testing.T) *feed { return &feed{t: t, out: map[string][]kernel.Addressed{}} }

func (f *feed) add(name string, typ catalog.Type, item string, occurred time.Time, data any) {
	f.t.Helper()
	f.seq++
	b, err := json.Marshal(data)
	if err != nil {
		f.t.Fatal(err)
	}
	r := kernel.Record{Seq: f.seq, EventID: kernel.UUIDv5(constants.NsAnt, "test\x1f"+name), Type: typ, ItemID: item,
		SourceKind: "machine", OccurredAt: occurred, ReceivedAt: occurred, RecordedAt: occurred, CorrelationID: "corr-" + name, Data: b}
	if item != "" {
		r.Stream = "item:" + item
	}
	var out []kernel.Addressed
	f.st, out = crossitem.Fold(f.st, r)
	f.out[name] = out
}

func (f *feed) weld(i int, eq string) {
	run, item := fmt.Sprintf("RUN-W-%d", i), fmt.Sprintf("ENT01:FL-%d", i)
	from, to := at(i*15), at(i*15+10)
	f.add("start-"+run, catalog.OperationRunStarted, item, from, map[string]any{
		"operation_run_id": run, "operation_code": "030", "step_key": "welding.weld", "equipment_id": eq,
		"operator_id": "W-07", "operation_started_at": stamp(from)})
	f.add("finish-"+run, catalog.OperationRunFinished, item, to, map[string]any{
		"operation_run_id": run, "completion": "completed", "operation_finished_at": stamp(to)})
}

// deviation — ток вне уставки 160 ± 10 А на источнике eq с from по to.
func (f *feed) deviation(name, eq string, from, to time.Time) {
	measure := func(v int64) map[string]any { return map[string]any{"value": v, "scale": 1, "unit": "A"} }
	f.add(name, catalog.EquipmentDeviationDetected, "", to, map[string]any{
		"equipment_id": eq, "deviation_kind": "out_of_setpoint", "parameter": "current",
		"started_at": stamp(from), "ended_at": stamp(to), "value": measure(1800),
		"setpoint": map[string]any{"nominal": measure(1600), "lower": measure(1500), "upper": measure(1700)}})
}

func byType(out []kernel.Addressed, t catalog.Type) []kernel.Addressed {
	var r []kernel.Addressed
	for _, a := range out {
		if a.Type == t {
			r = append(r, a)
		}
	}
	return r
}

type scopeData struct {
	IncidentID  string   `json:"incident_id"`
	Version     int      `json:"scope_version"`
	WindowStart string   `json:"window_start"`
	Added       []string `json:"added_item_ids"`
	Basis       []string `json:"basis"`
	Change      string   `json:"change"`
}

func scopeOf(t *testing.T, a kernel.Addressed) scopeData {
	t.Helper()
	b, err := json.Marshal(a.Data)
	if err != nil {
		t.Fatal(err)
	}
	var d scopeData
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

// Окно нарушения → расширение области: инцидент по ИС-2 открыт
// подтверждённым несоответствием FL-4 с окном от последней годной FL-2;
// журналы источника, пришедшие позже, показывают ток вне уставки с 20-й
// минуты — в том же проходе окно нарушения расширяет область на FL-1 и FL-2.
func TestViolationWindowExpandsScope(t *testing.T) {
	f := newFeed(t)
	for i := range 6 {
		f.weld(i, "IS-2")
	}
	f.add("release-FL-2", catalog.ItemReleaseRecorded, "ENT01:FL-2", at(50), map[string]any{})
	f.add("nc-FL-4", catalog.DecisionNonconformityConfirmed, "ENT01:FL-4", at(90), map[string]any{"nc_id": "NC-0142", "severity": "major"})
	opened := byType(f.out["nc-FL-4"], catalog.IncidentIncidentOpened)
	if len(opened) != 1 {
		t.Fatalf("инцидент не открыт: %+v", f.out["nc-FL-4"])
	}
	v1 := scopeOf(t, byType(f.out["nc-FL-4"], catalog.IncidentScopeComputed)[0])
	if v1.Version != 1 || slices.Contains(v1.Added, "ENT01:FL-1") || slices.Contains(v1.Added, "ENT01:FL-2") {
		t.Fatalf("первая версия области: %+v", v1)
	}

	f.deviation("dev-current", "IS-2", at(20), at(95))
	out := f.out["dev-current"]
	window := byType(out, catalog.EquipmentViolationWindowResolved)
	if len(window) != 1 {
		t.Fatalf("окно нарушения: %+v", out)
	}
	windowID := crossitem.AddressedID(window[0])
	if windowID != machinelogs.WindowEventID(window[0].Stream, window[0].Key) {
		t.Fatal("id окна в проходе расходится с id записи окна")
	}
	scopes := byType(out, catalog.IncidentScopeComputed)
	if len(scopes) != 1 {
		t.Fatalf("окно нарушения не дошло до analysis в том же проходе: %+v", out)
	}
	v2 := scopeOf(t, scopes[0])
	if v2.IncidentID != v1.IncidentID || v2.Version != 2 || !slices.Equal(v2.Added, []string{"ENT01:FL-1", "ENT01:FL-2"}) ||
		v2.WindowStart != "2026-09-26T07:20:00.000Z" || !slices.Contains(v2.Basis, windowID) {
		t.Fatalf("расширение области окном нарушения: %+v (окно %s)", v2, windowID)
	}
	if !slices.Contains(scopes[0].Causes, windowID) {
		t.Fatalf("причина версии — запись окна: %v", scopes[0].Causes)
	}
	if n := len(byType(out, catalog.IncidentMembershipChanged)); n != 2 {
		t.Fatalf("статус изделия в инциденте — двум добавленным, записей %d", n)
	}
	// Порядок выхода — порядок модулей: окно (machinelogs) раньше области (analysis).
	if slices.IndexFunc(out, func(a kernel.Addressed) bool { return a.Type == catalog.EquipmentViolationWindowResolved }) >
		slices.IndexFunc(out, func(a kernel.Addressed) bool { return a.Type == catalog.IncidentScopeComputed }) {
		t.Fatal("порядок выхода прохода")
	}
	inc := f.st.Analysis.Incidents[v1.IncidentID]
	for _, it := range []string{"ENT01:FL-1", "ENT01:FL-2"} {
		if m := inc.Members[it]; m.Status != analysis.StatusSuspect {
			t.Fatalf("%s: %+v", it, m)
		}
	}
	if inc.Members["ENT01:FL-4"].Status != analysis.StatusConfirmed {
		t.Fatal("правило не понижает «подтверждено»")
	}
	if _, in := inc.Members["ENT01:FL-0"]; in {
		t.Fatal("FL-0 вне окна нарушения")
	}
}

// Окно нарушения без открытого инцидента открывает инцидент по оборудованию:
// все изделия сварок в окне — под подозрением (FR-151, FR-61).
func TestViolationWindowOpensIncident(t *testing.T) {
	f := newFeed(t)
	for i := range 8 {
		f.weld(i, "IS-2")
	}
	f.deviation("dev-current", "IS-2", at(20), at(95))
	out := f.out["dev-current"]
	opened := byType(out, catalog.IncidentIncidentOpened)
	scopes := byType(out, catalog.IncidentScopeComputed)
	if len(opened) != 1 || len(scopes) != 1 {
		t.Fatalf("инцидент по окну нарушения: %+v", out)
	}
	v1 := scopeOf(t, scopes[0])
	want := []string{"ENT01:FL-1", "ENT01:FL-2", "ENT01:FL-3", "ENT01:FL-4", "ENT01:FL-5", "ENT01:FL-6"}
	if v1.Version != 1 || !slices.Equal(v1.Added, want) {
		t.Fatalf("область по окну: %+v", v1)
	}
	// Повтор того же факта оборудования ничего не выдаёт (неподвижная точка).
	f.deviation("dev-current", "IS-2", at(20), at(95))
	if len(f.out["dev-current"]) != 0 {
		t.Fatalf("повтор: %+v", f.out["dev-current"])
	}
}

// Эпик 16, стык 21 и 22: несоответствия окна нарушения (функция-намерение
// nonconformity в шаге machinelogs) в том же проходе попадают в инцидент,
// который окно открыло, — вход разбора обстоятельств.
func TestViolationWindowNCsJoinIncident(t *testing.T) {
	f := newFeed(t)
	for i := range 4 {
		f.weld(i, "IS-2")
	}
	f.deviation("dev-current", "IS-2", at(20), at(50))
	out := f.out["dev-current"]
	ncs := byType(out, catalog.DecisionNonconformityRegistered)
	opened := byType(out, catalog.IncidentIncidentOpened)
	if len(ncs) == 0 || len(opened) != 1 {
		t.Fatalf("окно: несоответствия %d, инцидентов %d", len(ncs), len(opened))
	}
	var inc analysis.Incident
	for _, x := range f.st.Analysis.Incidents {
		inc = x
	}
	if len(inc.NCs) != len(ncs) {
		t.Fatalf("несоответствия окна в инциденте: %v, выдано %d", inc.NCs, len(ncs))
	}
}

// Эпик 16, стык 17 и 23: признак «специальный процесс» — из
// operation.run.interval_resolved: шаг, который процесс не считает
// спецпроцессом, несоответствий окна не получает, хотя он в списке по умолчанию.
func TestSpecialProcessFromIntervalResolved(t *testing.T) {
	f := newFeed(t)
	for i := range 4 {
		f.weld(i, "IS-2")
		run := fmt.Sprintf("RUN-W-%d", i)
		f.add("interval-"+run, catalog.OperationRunIntervalResolved, fmt.Sprintf("ENT01:FL-%d", i), at(i*15+10), map[string]any{
			"operation_run_id": run, "equipment_id": "IS-2", "interval_start": stamp(at(i * 15)), "interval_end": stamp(at(i*15 + 10)),
			"interval_origin": "source_reported", "step_key": "welding.weld", "special_process": false})
	}
	f.deviation("dev-current", "IS-2", at(20), at(50))
	if n := len(byType(f.out["dev-current"], catalog.DecisionNonconformityRegistered)); n != 0 {
		t.Fatalf("процесс: не спецпроцесс — несоответствий окна нет, выдано %d", n)
	}
}
