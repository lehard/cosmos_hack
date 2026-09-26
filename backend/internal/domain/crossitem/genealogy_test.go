package crossitem_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/analysis"
	"ant/internal/domain/crossitem"
	"ant/internal/domain/kernel"
)

// Генеалогия стадии (эпик 18; FR-15, FR-34, FR-45; AD-41, AD-42): записи
// идут через настоящую crossitem.Fold — со всеми подключёнными модулями и
// правилами неподвижной точки.

// Генеалогия стадии отдаётся analysis через его порт (AD-42).
var _ analysis.Genealogy = crossitem.GenealogyView{}

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

// world — стадия и записанные ею адресованные записи.
type world struct {
	t   *testing.T
	s   crossitem.Stage
	seq int64
	out []kernel.Addressed
}

func (w *world) feed(typ catalog.Type, itemID string, at time.Duration, data any, mod ...func(*kernel.Record)) kernel.Record {
	w.t.Helper()
	w.seq++
	raw, err := json.Marshal(data)
	if err != nil {
		w.t.Fatal(err)
	}
	info, _ := catalog.Lookup(typ)
	r := kernel.Record{Seq: w.seq, EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", w.seq), Type: typ, Kind: info.Kind,
		ItemID: itemID, OccurredAt: t0.Add(at), ReceivedAt: t0.Add(at), RecordedAt: t0.Add(at), Data: raw}
	if itemID != "" {
		r.Stream = "item:" + itemID
	}
	for _, m := range mod {
		m(&r)
	}
	var out []kernel.Addressed
	w.s, out = crossitem.Fold(w.s, r)
	w.out = append(w.out, out...)
	return r
}

// to — адресованные записи типа typ в поток изделия id.
func (w *world) to(typ catalog.Type, id string) []map[string]any {
	var res []map[string]any
	for _, a := range w.out {
		if a.Type != typ || a.Stream != "item:"+id {
			continue
		}
		raw, _ := json.Marshal(a.Data)
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		res = append(res, m)
	}
	return res
}

func (w *world) register(id string, extra map[string]any) {
	d := map[string]any{"item_id": id, "item_type_id": "FL-100.00.000", "item_revision": "Б", "process_version_hash": "streebog256:00", "normative_rev": "r1"}
	maps.Copy(d, extra)
	w.feed(catalog.ItemItemRegistered, id, 0, d)
}

func (w *world) assemble(asm, comp string, at time.Duration) {
	w.feed(catalog.ItemAssemblyRecorded, asm, at, map[string]any{"assembly_item_id": asm, "component_item_id": comp,
		"component_type_id": "FL-100.01.000", "binding_method": "dpm_datamatrix"})
}

// Результат образца-свидетеля садки виден в паспортах всех изделий садки (FR-15).
func TestWitnessOfChargeReachesEveryItem(t *testing.T) {
	w := &world{t: t}
	for _, id := range []string{"ENT01:A", "ENT01:B", "ENT01:C", "ENT01:W"} {
		w.register(id, nil)
	}
	w.feed(catalog.GenealogyGroupFormed, "", time.Hour, map[string]any{"group_id": "CH-7", "kind": "charge",
		"item_ids": []string{"ENT01:A", "ENT01:B", "ENT01:C"}, "witness_item_id": "ENT01:W"}, func(r *kernel.Record) { r.Stream = "group:CH-7" })
	insp := w.feed(catalog.InspectionResultRecorded, "ENT01:W", 3*time.Hour, map[string]any{"outcome": "no_defect_indicated",
		"method": "laboratory", "phase": "test", "conclusion_ref": "ПИ-12"})
	for _, id := range []string{"ENT01:A", "ENT01:B", "ENT01:C"} {
		got := w.to(catalog.GenealogyWitnessPropagated, id)
		if len(got) != 1 || got[0]["inspection_event_id"] != insp.EventID || got[0]["witness_item_id"] != "ENT01:W" || got[0]["outcome"] != "no_defect_indicated" {
			t.Fatalf("%s: результат свидетеля %v", id, got)
		}
		if len(w.to(catalog.GenealogyLinkAdded, id)) != 1 {
			t.Fatalf("%s: связь grouped_with", id)
		}
	}
	if len(w.to(catalog.GenealogyWitnessPropagated, "ENT01:W")) != 0 {
		t.Fatal("свидетель сам себе не адресуется")
	}
	// Повтор того же результата не даёт новых записей.
	n := len(w.out)
	w.s, _ = crossitem.Fold(w.s, insp)
	if _, out := crossitem.Fold(w.s, insp); len(out) != 0 || len(w.out) != n {
		t.Fatalf("повтор: %+v", out)
	}
	if got := w.s.Own.Genealogy.GroupItems("CH-7"); len(got) != 4 {
		t.Fatalf("садка → изделия: %+v", got)
	}
}

// Блок партии доходит до собранных из неё изделий — и до тех, что собраны
// после блока (FR-45, FR-49, AD-42).
func TestLotHoldReachesAssembledItems(t *testing.T) {
	w := &world{t: t}
	w.feed(catalog.GenealogyLotRegistered, "", 0, map[string]any{"lot_id": "L-17", "actual_quantity": 20, "certificate_present": true,
		"registered_by": "store-1", "heat_no": "ПЛ-3"}, func(r *kernel.Record) { r.Stream = "lot:L-17" })
	w.register("ENT01:C1", map[string]any{"lot_ids": []string{"L-17"}})
	w.register("ENT01:U1", nil)
	w.register("ENT01:F1", nil)
	w.assemble("ENT01:U1", "ENT01:C1", time.Hour)
	w.assemble("ENT01:F1", "ENT01:U1", 2*time.Hour)

	// Крепёж из партии прямо в сборку — сборка тоже «из партии».
	w.register("ENT01:F2", nil)
	w.feed(catalog.ItemAssemblyRecorded, "ENT01:F2", 2*time.Hour, map[string]any{"assembly_item_id": "ENT01:F2", "component_lot_id": "L-17",
		"component_type_id": "FL-100.00.005", "quantity": 12, "binding_method": "container_cell"})

	rej := w.feed(catalog.DecisionLotResolved, "", 3*time.Hour, map[string]any{"lot_id": "L-17", "resolution": "reject",
		"method_event_ids": []string{}}, func(r *kernel.Record) { r.Stream = "lot:L-17" })
	for _, id := range []string{"ENT01:C1", "ENT01:U1", "ENT01:F1", "ENT01:F2"} {
		got := w.to(catalog.GenealogyContainmentPropagated, id)
		if len(got) != 1 || got[0]["level"] != "lot_hold" || got[0]["source"] != "lot" || got[0]["lot_id"] != "L-17" || got[0]["source_event_id"] != rej.EventID {
			t.Fatalf("%s: блок партии %v", id, got)
		}
	}
	// Изделие из той же партии, собранное после блока, получает блок вместе со сборкой.
	w.register("ENT01:C2", map[string]any{"lot_ids": []string{"L-17"}})
	w.register("ENT01:F3", nil)
	w.assemble("ENT01:F3", "ENT01:C2", 4*time.Hour)
	for _, id := range []string{"ENT01:C2", "ENT01:F3"} {
		if got := w.to(catalog.GenealogyContainmentPropagated, id); len(got) != 1 || got[0]["level"] != "lot_hold" {
			t.Fatalf("%s: позднее изделие партии %v", id, got)
		}
	}
	// Запрос «партия / плавка → все изделия, включая собранные».
	ids := func(xs []crossitem.LotItem) []string {
		var out []string
		for _, x := range xs {
			out = append(out, x.ItemID+"/"+x.Relation)
		}
		slices.Sort(out)
		return out
	}
	want := []string{"ENT01:C1/made_from_lot", "ENT01:C2/made_from_lot", "ENT01:F1/assembled", "ENT01:F2/made_from_lot", "ENT01:F3/assembled", "ENT01:U1/assembled"}
	if got := ids(w.s.Own.Genealogy.LotItems("L-17")); !slices.Equal(got, want) {
		t.Fatalf("партия → изделия: %v", got)
	}
	if got := ids(w.s.Own.Genealogy.HeatItems("ПЛ-3")); !slices.Equal(got, want) {
		t.Fatalf("плавка → изделия: %v", got)
	}
	if got := w.s.Own.Genealogy.Ancestors("ENT01:C1"); !slices.Equal(got, []string{"ENT01:U1", "ENT01:F1"}) {
		t.Fatalf("вверх по дереву: %v", got)
	}
	if got := w.s.Own.Genealogy.Descendants("ENT01:F1"); !slices.Equal(got, []string{"ENT01:U1", "ENT01:C1"}) {
		t.Fatalf("вниз по дереву: %v", got)
	}
	if loc := w.s.Genealogy().Location("ENT01:C1", ""); loc != crossitem.LocAssembled {
		t.Fatalf("местонахождение: %s", loc)
	}
}

// Блок компонента человеком — вверх по дереву сборки; снятие основания —
// та же запись с released (блок снимает человек, AD-27).
func TestComponentHoldPropagatesUp(t *testing.T) {
	w := &world{t: t}
	for _, id := range []string{"ENT01:C1", "ENT01:U1", "ENT01:F1"} {
		w.register(id, nil)
	}
	w.assemble("ENT01:U1", "ENT01:C1", time.Hour)
	w.assemble("ENT01:F1", "ENT01:U1", 2*time.Hour)
	set := w.feed(catalog.DecisionContainmentSet, "ENT01:C1", 3*time.Hour, map[string]any{"level": "item_hold", "reason": map[string]any{"text": "подозрение"}})
	for _, id := range []string{"ENT01:U1", "ENT01:F1"} {
		got := w.to(catalog.GenealogyContainmentPropagated, id)
		if len(got) != 1 || got[0]["source"] != "component" || got[0]["source_item_id"] != "ENT01:C1" || got[0]["level"] != "item_hold" {
			t.Fatalf("%s: %v", id, got)
		}
	}
	if len(w.to(catalog.GenealogyContainmentPropagated, "ENT01:C1")) != 0 {
		t.Fatal("источник сам себе не адресуется")
	}
	w.feed(catalog.DecisionContainmentReleased, "ENT01:C1", 4*time.Hour, map[string]any{"released_event_ids": []string{set.EventID},
		"reason": map[string]any{"text": "проверено"}})
	for _, id := range []string{"ENT01:U1", "ENT01:F1"} {
		got := w.to(catalog.GenealogyContainmentPropagated, id)
		if len(got) != 2 || got[1]["released"] != true {
			t.Fatalf("%s: снятие основания %v", id, got)
		}
	}
}

// Неоднозначное событие получает перечень кандидатов, а не угаданное изделие
// (FR-34, AD-41); однозначное — одно изделие; ручная привязка — человеком.
func TestAmbiguousEventGetsCandidates(t *testing.T) {
	w := &world{t: t}
	for _, id := range []string{"ENT01:X", "ENT01:Y", "ENT01:Z"} {
		w.register(id, nil)
	}
	// Два крепёжных изделия в одной ячейке тары, у третьего — своя бирка.
	for _, id := range []string{"ENT01:X", "ENT01:Y"} {
		w.feed(catalog.ItemCarrierApplied, id, time.Minute, map[string]any{"carrier_type": "container_cell", "value": "BOX-1/3", "is_temporary": true})
	}
	w.feed(catalog.ItemCarrierApplied, "ENT01:Z", time.Minute, map[string]any{"carrier_type": "tag_qr", "value": "Q-9", "is_temporary": true})

	unbound := func(ref, level string, at time.Duration) kernel.Record {
		return w.feed(catalog.InspectionResultRecorded, "", at, map[string]any{"outcome": "defect_indicated", "method": "visual", "phase": "after_operation"},
			func(r *kernel.Record) { r.CarrierRef, r.IdentificationLevel, r.Stream = ref, level, "global" })
	}
	amb := unbound("container_cell:BOX-1/3", "probable", time.Hour)
	for _, id := range []string{"ENT01:X", "ENT01:Y"} {
		got := w.to(catalog.BindingLinkResolved, id)
		if len(got) != 1 || got[0]["item_id"] != nil || got[0]["binding_reliability"] != "ambiguous" || got[0]["subject_event_id"] != amb.EventID {
			t.Fatalf("%s: %v", id, got)
		}
		if c, _ := got[0]["candidates"].([]any); len(c) != 2 {
			t.Fatalf("%s: кандидаты %v", id, got[0]["candidates"])
		}
	}
	one := unbound("ant:carrier:tag_qr:Q-9", "unique", time.Hour)
	if got := w.to(catalog.BindingLinkResolved, "ENT01:Z"); len(got) != 1 || got[0]["item_id"] != "ENT01:Z" || got[0]["subject"] == nil ||
		got[0]["binding_reliability"] != "unique" || got[0]["subject_event_id"] != one.EventID {
		t.Fatalf("однозначная привязка: %v", got)
	}
	// До нанесения носителя — событие ждёт ручной привязки.
	early := unbound("tag_qr:Q-9", "unique", -time.Hour)
	if u := w.s.Own.Genealogy.Unbound[early.EventID]; u.BoundTo != "" || len(u.Candidates) != 0 {
		t.Fatalf("раннее событие: %+v", u)
	}
	// Контролёр выбирает X: X получает событие, Y — что оно не его.
	w.feed(catalog.BindingLinkAssigned, "ENT01:X", 2*time.Hour, map[string]any{"subject_event_id": amb.EventID, "item_id": "ENT01:X",
		"method": "select_expected", "reason": map[string]any{"text": "по маршрутной карте"}})
	if got := w.to(catalog.BindingLinkResolved, "ENT01:X"); len(got) != 2 || got[1]["item_id"] != "ENT01:X" || got[1]["binding_basis"] != "manual" || got[1]["subject"] == nil {
		t.Fatalf("ручная привязка: %v", got)
	}
	if got := w.to(catalog.BindingLinkResolved, "ENT01:Y"); len(got) != 2 || got[1]["item_id"] != "ENT01:X" || got[1]["previous_item_id"] != "ENT01:Y" {
		t.Fatalf("прежний кандидат: %v", got)
	}
	// Снятый носитель после снятия не разрешается.
	w.feed(catalog.ItemCarrierRemoved, "ENT01:Z", 5*time.Hour, map[string]any{"carrier_type": "tag_qr", "value": "Q-9"})
	late := unbound("tag_qr:Q-9", "unique", 6*time.Hour)
	if u := w.s.Own.Genealogy.Unbound[late.EventID]; u.BoundTo != "" {
		t.Fatalf("снятый носитель: %+v", u)
	}
}

// Разделение 1→N переносит происхождение; сборка — связь обоим изделиям.
func TestSplitCarriesOrigin(t *testing.T) {
	w := &world{t: t}
	w.register("ENT01:P", map[string]any{"lot_ids": []string{"L-1"}})
	w.register("ENT01:P-1", map[string]any{"split_from": "ENT01:P"})
	links := w.to(catalog.GenealogyLinkAdded, "ENT01:P-1")
	var rels []string
	for _, l := range links {
		rels = append(rels, fmt.Sprint(l["relation"], "/", l["inherited"]))
	}
	slices.Sort(rels)
	if !slices.Equal(rels, []string{"made_from_lot/true", "split_from/<nil>"}) {
		t.Fatalf("связи части: %v", rels)
	}
	if len(w.to(catalog.GenealogyLinkAdded, "ENT01:P")) != 2 {
		t.Fatal("исходное изделие: связь партии и разделения")
	}
	if got := w.s.Own.Genealogy.LotItems("L-1"); len(got) != 2 {
		t.Fatalf("партия → изделия: %+v", got)
	}
}

// Изменение справочника с действием в прошлом доходит до изделий,
// выполнявших операцию на оборудовании после даты действия (AD-31, AD-42).
func TestRetroactiveReferenceChange(t *testing.T) {
	w := &world{t: t}
	for i, id := range []string{"ENT01:A", "ENT01:B"} {
		w.register(id, nil)
		w.feed(catalog.OperationRunStarted, id, time.Duration(i+1)*time.Hour, map[string]any{"operation_run_id": "run-" + id, "operation_code": "010",
			"step_key": "welding.weld", "equipment_id": "WELD-2", "operator_id": "op-1"})
	}
	// Поверка признана недействительной с 08:30 — задним числом, записано в 12:00.
	w.feed(catalog.ReferenceEquipmentVerified, "", 30*time.Minute, map[string]any{"equipment_id": "WELD-2", "kind": "welding_source", "result": "failed"},
		func(r *kernel.Record) { r.RecordedAt, r.Stream = t0.Add(4*time.Hour), "reference" })
	if len(w.to(catalog.ReferenceChangeAffectsItem, "ENT01:A")) != 1 || len(w.to(catalog.ReferenceChangeAffectsItem, "ENT01:B")) != 1 {
		t.Fatalf("изменение справочника: %+v", w.out)
	}
	// Действует с момента записи — никого не затрагивает.
	n := len(w.out)
	w.feed(catalog.ReferenceEquipmentVerified, "", 5*time.Hour, map[string]any{"equipment_id": "WELD-2", "kind": "welding_source", "result": "passed"},
		func(r *kernel.Record) { r.Stream = "reference" })
	if len(w.out) != n {
		t.Fatalf("изменение без действия в прошлом: %+v", w.out[n:])
	}
}

// Неподвижная точка (AD-42): изделия реагируют на адресованные записи стадии
// записями publish: stage, вызванными ими; стадия на них выхода не даёт.
func TestGenealogyFixedPoint(t *testing.T) {
	w := &world{t: t}
	w.feed(catalog.GenealogyLotRegistered, "", 0, map[string]any{"lot_id": "L-2", "actual_quantity": 5, "certificate_present": true, "registered_by": "s"})
	w.register("ENT01:C", map[string]any{"lot_ids": []string{"L-2"}})
	w.register("ENT01:F", nil)
	w.assemble("ENT01:F", "ENT01:C", time.Hour)
	w.feed(catalog.DecisionLotResolved, "", 2*time.Hour, map[string]any{"lot_id": "L-2", "resolution": "reject", "method_event_ids": []string{}})
	queue := slices.Clone(w.out)
	for i := 0; len(queue) > 0; i++ {
		if i > 100 {
			t.Fatal("нет неподвижной точки")
		}
		a := queue[0]
		queue = queue[1:]
		item := strings.TrimPrefix(a.Stream, "item:")
		// Реакция nonconformity на блок: сдерживание правилом, вызванное адресованной записью.
		r := kernel.Record{Seq: 1000 + int64(i), EventID: fmt.Sprintf("00000000-0000-5000-8000-%012d", i), Type: catalog.DecisionContainmentApplied,
			Kind: catalog.KindReaction, ItemID: item, Stream: a.Stream, CausationID: crossitem.AddressedID(a), OccurredAt: t0.Add(3 * time.Hour),
			Data: json.RawMessage(`{"level":"item_hold","basis":["` + crossitem.AddressedID(a) + `"]}`)}
		var out []kernel.Addressed
		w.s, out = crossitem.Fold(w.s, r)
		if len(out) != 0 {
			t.Fatalf("стадия ответила на своё: %+v", out)
		}
	}
}
