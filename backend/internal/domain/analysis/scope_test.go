package analysis_test

import (
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/analysis"
	"ant/internal/domain/kernel"
)

// openIncident — id единственного открытого инцидента мира.
func openIncident(t *testing.T, j *journal) string {
	t.Helper()
	opened := j.of(catalog.IncidentIncidentOpened)
	if len(opened) != 1 {
		t.Fatalf("ждали один открытый инцидент, есть %d", len(opened))
	}
	return strings.TrimPrefix(opened[0].Stream, "incident:")
}

// narrow — решение технолога «сузить область» с основанием (FR-61).
func narrow(j *journal, inc string, when time.Time, items []string, evidence []string, reason string) kernel.Record {
	return j.decision(catalog.IncidentScopeNarrowed, "incident:"+inc, "", "tec-01@1", when, map[string]any{
		"incident_id": inc, "item_ids": items, "evidence_event_ids": evidence, "reason": map[string]any{"text": reason}})
}

// itemsWhere — изделия текущей области с условием по сварке.
func itemsWhere(v analysis.IncidentRecord, pred func(weld) bool) []string {
	var out []string
	for _, x := range storyWelds() {
		if v.InScope(x.item) && pred(x) {
			out = append(out, x.item)
		}
	}
	slices.Sort(out)
	return out
}

// «Станок сломался» (FR-9, FR-61, кейс §5.1): область риска RS-01 тает
// 34 → 13 → 6, каждое сужение — решение человека с основанием (доказательства,
// автор, время), счётчик сокращения виден; первая версия — правило системы
// с отсчётом от последней подтверждённо годной детали Ф-006.
func TestRiskScopeMelts34to13to6(t *testing.T) {
	j, _ := storyUntilNC01()
	inc := openIncident(t, j)

	v := j.incident(inc)
	if got := v.Size(); got != 34 {
		t.Fatalf("версия 1: размер %d, ждали 34 (%v)", got, localIDs(slices.Sorted(maps.Keys(v.Members))))
	}
	v1 := v.Versions[0]
	if v1.Change != analysis.ChangeComputed || v1.Author != "" || v1.Reason == nil || !strings.Contains(v1.Reason.Text, "F-006") {
		t.Fatalf("версия 1 — правило системы с отсчётом от Ф-006: %+v", v1)
	}
	if v.KnownGoodItem != "ENT01:F-006" || v.Members["ENT01:F-017"].Status != analysis.StatusConfirmed || v.Members["ENT01:F-017"].Action != analysis.ActionBlock {
		t.Fatalf("отсчёт %q, Ф-017 %+v", v.KnownGoodItem, v.Members["ENT01:F-017"])
	}
	if v.Members["ENT01:F-006"].Status != "" || v.Members["ENT01:F-008"].Status != analysis.StatusSuspect {
		t.Fatalf("Ф-006 вне области, Ф-008 под подозрением: %+v %+v", v.Members["ENT01:F-006"], v.Members["ENT01:F-008"])
	}

	// v2: журнал ИС-1 непрерывный и в уставке — ИС-1 исключается (основание — записи журнала ИС-1).
	is1Log := j.fact(catalog.EquipmentCycleSummarized, "", at(23, 11, 40), map[string]any{"equipment_id": "IS-1",
		"window_start": "2026-09-21T12:00:00.000Z", "window_end": "2026-09-23T08:00:00.000Z",
		"parameters": []map[string]any{{"parameter": "current", "out_of_setpoint_ms": 0}}})
	is1 := itemsWhere(v, func(x weld) bool { return x.src == "IS-1" })
	narrow(j, inc, at(23, 11, 45), is1, []string{is1Log.EventID}, "Журнал ИС-1 непрерывный и в уставке; общий фактор — ИС-2")
	v = j.incident(inc)
	if got := v.Size(); got != 13 {
		t.Fatalf("версия 2: размер %d, ждали 13", got)
	}

	// v3: опоздавший журнал ИС-2 — ток впервые вне уставки во Вт 10:20; сварки ИС-2, законченные раньше, — в уставке.
	late := j.fact(catalog.EquipmentDeviationDetected, "", at(22, 10, 20), map[string]any{"equipment_id": "IS-2",
		"deviation_kind": "out_of_setpoint", "started_at": "2026-09-22T07:20:00.000Z", "parameter": "current",
		"value": map[string]any{"value": 176, "unit": "A", "scale": 0}})
	before := itemsWhere(v, func(x weld) bool { return x.src == "IS-2" && x.to.Before(at(22, 10, 20)) })
	narrow(j, inc, at(23, 12, 15), before, []string{late.EventID}, "Опоздавший журнал ИС-2: ток впервые вне уставки во Вт 10:20; сварки до этого — в уставке")
	v = j.incident(inc)
	if got := v.Size(); got != 6 {
		t.Fatalf("версия 3: размер %d, ждали 6", got)
	}
	want := []string{"F-015", "F-017", "F-019", "F-021", "F-023", "F-025"}
	if got := localIDs(itemsWhere(v, func(weld) bool { return true })); !slices.Equal(got, want) {
		t.Fatalf("итоговая область %v, ждали %v", got, want)
	}

	// Каждая версия — с основанием; сужения — человеком, с доказательствами и временем.
	var sizes []int
	for i, x := range v.Versions {
		sizes = append(sizes, x.Size)
		if x.Version != i+1 || x.Reason == nil || x.Reason.Text == "" || len(x.Evidence) == 0 || x.RecordedAt.IsZero() {
			t.Fatalf("версия без основания: %+v", x)
		}
		if x.Change == analysis.ChangeNarrowed && (x.Author != "TEC-01" || x.DecisionID == "" || len(x.Removed) == 0) {
			t.Fatalf("сужение без автора или решения: %+v", x)
		}
	}
	if !slices.Equal(sizes, []int{34, 13, 6}) || v.InitialSize != 34 {
		t.Fatalf("счётчик сокращения: %v (исходно %d)", sizes, v.InitialSize)
	}
	// Исключённые — «проверено и исключено», у каждого — версия сужения.
	for _, id := range append(is1, before...) {
		if m := v.Members[id]; m.Status != analysis.StatusExcluded || m.Version < 2 {
			t.Fatalf("%s: %+v", id, m)
		}
	}
	// Стадия пишет статус изделию адресованной записью (две оси, AD-30).
	var toF017 int
	for _, r := range j.of(catalog.IncidentMembershipChanged) {
		if r.ItemID == "ENT01:F-017" {
			toF017++
		}
	}
	if toF017 != 1 {
		t.Fatalf("Ф-017: адресованных записей статуса %d, ждали 1", toF017)
	}
}

// Сужение без основания стадия не применяет, гард api отклоняет с кодом
// incident.basis_required (FR-61: ни одно изделие не выходит из области без
// записи основания).
func TestNarrowWithoutBasisRefused(t *testing.T) {
	j, _ := storyUntilNC01()
	inc := openIncident(t, j)
	v := j.incident(inc)
	items := []string{"ENT01:F-008"}
	err := analysis.GuardNarrow(v, items, nil, "журнал в уставке")
	if r, ok := err.(*kernel.Refusal); !ok || r.Code != errcodes.IncidentBasisRequired {
		t.Fatalf("гард: %v", err)
	}
	narrow(j, inc, at(23, 11, 50), items, []string{}, "без доказательств")
	if got := j.incident(inc); got.Size() != 34 || len(got.Versions) != 1 {
		t.Fatalf("сужение без основания применено: %d, версий %d", got.Size(), len(got.Versions))
	}
	if err := analysis.GuardNarrow(v, []string{"ENT01:F-001"}, []string{"x"}, "основание"); err == nil {
		t.Fatal("изделие вне области сужено")
	}
}

// Автоматическое правило может расширить область, но не исключить изделие
// (FR-61, FR-144, AD-27): новая сварка на ИС-2, сборка с компонентом из
// области, повторное несоответствие — только новые изделия и «подтверждено»;
// ни одной версии правила с исключёнными, размер версий правила не убывает.
func TestAutoRuleOnlyExpands(t *testing.T) {
	j, _ := storyUntilNC01()
	inc := openIncident(t, j)
	// Человек исключил Ф-014 с основанием.
	logRec := j.fact(catalog.EquipmentCycleSummarized, "", at(23, 11, 30), map[string]any{"equipment_id": "IS-2",
		"window_start": "2026-09-22T05:20:00.000Z", "window_end": "2026-09-22T05:43:00.000Z", "parameters": []map[string]any{}})
	narrow(j, inc, at(23, 11, 35), []string{"ENT01:F-014"}, []string{logRec.EventID}, "Сварка Ф-014 в уставке")
	before := j.incident(inc)

	// Новое выполнение на ИС-2 после обнаружения — защитное расширение.
	j.weldRun(w("F-037", "IS-2", "W22", 23, 11, 20, 11, 50))
	// Сборка: Ф-015 (в области) вошёл в изделие СБ-1 — распространение вверх по дереву сборки.
	j.fact(catalog.ItemAssemblyRecorded, "ENT01:SB-1", at(23, 12, 0), map[string]any{"assembly_item_id": "ENT01:SB-1",
		"component_item_id": "ENT01:F-015", "component_type_id": "FL-100", "binding_method": "qr"})
	// Повторное выполнение исключённого Ф-014 на ИС-2 — снова под подозрением.
	j.weldRun(w("F-014", "IS-2", "W21", 23, 12, 10, 12, 40))
	// НС на Ф-021 (под подозрением) — «подтверждено», без исключений.
	j.decision(catalog.DecisionNonconformityConfirmed, "item:ENT01:F-021", "ENT01:F-021", "ins-01@1", at(23, 13, 5), map[string]any{
		"nc_id": "NC-02", "defect_type_code": "burn_through", "severity": "major", "signal_ids": []string{}, "reason": map[string]any{"text": "РК"}})

	after := j.incident(inc)
	for _, id := range slices.Sorted(maps.Keys(before.Members)) {
		was, now := before.Members[id], after.Members[id]
		if was.Status != analysis.StatusExcluded && now.Status == analysis.StatusExcluded {
			t.Fatalf("правило исключило %s: %+v → %+v", id, was, now)
		}
	}
	for _, id := range []string{"ENT01:F-037", "ENT01:SB-1", "ENT01:F-014"} {
		if !after.InScope(id) {
			t.Fatalf("%s не добавлен правилом: %+v", id, after.Members[id])
		}
	}
	if m := after.Members["ENT01:SB-1"]; m.Via != "ENT01:F-015" {
		t.Fatalf("сборка без компонента-основания: %+v", m)
	}
	if m := after.Members["ENT01:F-021"]; m.Status != analysis.StatusConfirmed || m.Action != analysis.ActionBlock {
		t.Fatalf("Ф-021 после НС: %+v", m)
	}
	prev := 0
	for _, x := range after.Versions {
		if x.Change == analysis.ChangeComputed {
			if len(x.Removed) > 0 || x.Size < prev {
				t.Fatalf("версия правила сузила область: %+v (было %d)", x, prev)
			}
			if x.Author != "" {
				t.Fatalf("версия правила с автором-человеком: %+v", x)
			}
		}
		prev = x.Size
	}
	// «Исключено» изделию пишется только по решению человека (причина — решение).
	decisions := map[string]bool{}
	for _, r := range j.records {
		if r.Kind == catalog.KindDecision {
			decisions[r.EventID] = true
		}
	}
	for _, r := range j.of(catalog.IncidentMembershipChanged) {
		if strings.Contains(string(r.Data), `"excluded"`) && !decisions[r.CausationID] {
			t.Fatalf("исключение без решения человека: %s", r.Data)
		}
	}
}

// Защитная функция правила отказывается исключать (AD-27).
func TestAutoAddRefusesExclusion(t *testing.T) {
	inc := analysis.Incident{Members: map[string]analysis.Member{"A": {Status: analysis.StatusSuspect}, "B": {Status: analysis.StatusConfirmed}}}
	if ch := inc.AutoAdd("A", analysis.StatusExcluded, ""); len(ch) != 0 || inc.Members["A"].Status != analysis.StatusSuspect {
		t.Fatalf("правило исключило: %+v", inc.Members)
	}
	if ch := inc.AutoAdd("B", analysis.StatusSuspect, ""); len(ch) != 0 || inc.Members["B"].Status != analysis.StatusConfirmed {
		t.Fatalf("правило понизило «подтверждено»: %+v", inc.Members)
	}
}

// Входной дефект (пора в теле кольца партии П-117) открывает инцидент по
// партии, а не по сварке: в область — изделия с кольцами той же партии.
func TestIncomingDefectOpensLotIncident(t *testing.T) {
	j := &journal{}
	for _, x := range []weld{w("F-019", "IS-2", "W22", 22, 16, 35, 17, 20), w("F-027", "IS-1", "W22", 22, 18, 0, 18, 40), w("F-030", "IS-1", "W22", 21, 23, 0, 23, 35)} {
		j.weldRun(x)
	}
	for _, p := range [][2]string{{"F-019", "R-101"}, {"F-027", "R-102"}, {"F-030", "R-001"}} {
		lot := "LOT-R-117"
		if p[1] == "R-001" {
			lot = "LOT-R-116"
		}
		j.fact(catalog.ItemAssemblyRecorded, "ENT01:"+p[0], at(22, 16, 0), map[string]any{"assembly_item_id": "ENT01:" + p[0],
			"component_item_id": "ENT01:" + p[1], "component_lot_id": lot, "component_type_id": "RING", "binding_method": "qr"})
	}
	j.fact(catalog.InspectionResultRecorded, "ENT01:F-019", at(23, 13, 41), map[string]any{"outcome": "defect_indicated",
		"phase": "after_operation", "method": "radiography", "processing_state": "complete",
		"defects": []map[string]any{{"zone_id": "RING-BODY", "defect_type_code": "base_metal_pore", "component_ref": "ENT01:R-101", "severity": "major"}}})
	j.decision(catalog.DecisionNonconformityConfirmed, "item:ENT01:F-019", "ENT01:F-019", "ins-01@1", at(23, 13, 50), map[string]any{
		"nc_id": "NC-04", "defect_type_code": "base_metal_pore", "severity": "major", "signal_ids": []string{}, "reason": map[string]any{"text": "РК"}})
	inc := openIncident(t, j)
	v := j.incident(inc)
	if v.Factor != analysis.FactorMaterialBatch || v.FactorValue != "LOT-R-117" {
		t.Fatalf("фактор %s=%s", v.Factor, v.FactorValue)
	}
	if got := localIDs(slices.Sorted(maps.Keys(v.Members))); !slices.Equal(got, []string{"F-019", "F-027"}) {
		t.Fatalf("область партии: %v", got)
	}
}
