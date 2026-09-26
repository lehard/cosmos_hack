package crossitem_test

import (
	"maps"
	"slices"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/crossitem"
	"ant/internal/domain/kernel"
)

// Неподвижная точка стадии (AD-42): стадия, которая на каждую запись изделия
// связывает его с соседом, без правил Settle крутила бы цикл
// «стадия → изделие → стадия». С правилами — повтор подавляется, записи,
// вызванные своими адресованными, выхода не дают.
func TestSettleFixedPoint(t *testing.T) {
	ping := func(s crossitem.Stage, r kernel.Record) (crossitem.Stage, []kernel.Addressed) {
		other := map[string]string{"A": "B", "B": "A"}[r.ItemID]
		a, err := kernel.NewAddressed("crossitem", catalog.GenealogyLinkAdded, "item:"+other, r.ItemID+">"+other, nil, r)
		if err != nil {
			t.Fatal(err)
		}
		return s, []kernel.Addressed{a}
	}
	var s crossitem.Stage
	queue := []kernel.Record{{Seq: 1, EventID: "f1", ItemID: "A", Type: catalog.ItemAssemblyRecorded}}
	written := 0
	for seq := int64(2); len(queue) > 0; seq++ {
		if seq > 100 {
			t.Fatal("стадия не пришла к неподвижной точке")
		}
		r := queue[0]
		queue = queue[1:]
		var out []kernel.Addressed
		s, out = crossitem.Settle(s, r, ping)
		for _, a := range out {
			written++
			id := crossitem.AddressedID(a)
			// Изделие-адресат реагирует записью publish: stage, вызванной адресованной.
			queue = append(queue, kernel.Record{Seq: seq, EventID: "re-" + id, ItemID: a.Stream[len("item:"):], CausationID: id, Type: catalog.ItemAssemblyRecorded})
		}
	}
	if written != 1 {
		t.Fatalf("ждали одну адресованную запись, записано %d", written)
	}
	// Повтор того же факта не даёт повторной записи.
	if _, out := crossitem.Settle(s, kernel.Record{Seq: 99, EventID: "f1", ItemID: "A"}, ping); len(out) != 0 {
		t.Fatalf("повтор: %+v", out)
	}
}

// Рамка без подключённых модулей ничего не выдаёт.
func TestFoldEmptyModules(t *testing.T) {
	s, out := crossitem.Fold(crossitem.Stage{}, kernel.Record{Seq: 1, EventID: "x"})
	if len(out) != 0 || len(s.Own.Emitted) != 0 {
		t.Fatalf("%+v %+v", s, out)
	}
}

// Правила прохода (Д-40): поздний модуль получает адресованные записи ранних
// в том же проходе; модуль не получает своих записей — ни выданных в своём
// шаге, ни тех, эмитентом которых он указан; запись, вызванная адресованной
// записью стадии, выхода не даёт и дальше не доставляется.
func TestPassDelivery(t *testing.T) {
	got := map[kernel.Module][]catalog.Type{}
	step := func(m kernel.Module, emit func(r kernel.Record) []kernel.Addressed) crossitem.ModuleStep {
		return crossitem.ModuleStep{Module: m, Step: func(s crossitem.Stage, r kernel.Record) (crossitem.Stage, []kernel.Addressed) {
			got[m] = append(got[m], r.Type)
			return s, emit(r)
		}}
	}
	addr := func(m kernel.Module, typ catalog.Type, r kernel.Record) []kernel.Addressed {
		a, err := kernel.NewAddressed(m, typ, "item:ENT01:A", string(typ), map[string]any{"k": 1}, r)
		if err != nil {
			t.Fatal(err)
		}
		return []kernel.Addressed{a}
	}
	steps := []crossitem.ModuleStep{
		// machinelogs на факт оборудования выдаёт окно и — функцией-намерением
		// nonconformity (порт Registrar) — несоответствие окна.
		step("machinelogs", func(r kernel.Record) []kernel.Addressed {
			if r.Type != catalog.EquipmentDeviationDetected {
				return nil
			}
			return append(addr("machinelogs", catalog.EquipmentViolationWindowResolved, r), addr("nonconformity", catalog.DecisionNonconformityRegistered, r)...)
		}),
		step("nonconformity", func(kernel.Record) []kernel.Addressed { return nil }),
		step("analysis", func(r kernel.Record) []kernel.Addressed {
			if r.Type != catalog.EquipmentViolationWindowResolved {
				return nil
			}
			return addr("analysis", catalog.IncidentScopeComputed, r)
		}),
		step("notifications", func(kernel.Record) []kernel.Addressed { return nil }),
	}
	fact := kernel.Record{Seq: 7, EventID: "f1", Type: catalog.EquipmentDeviationDetected, OccurredAt: time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)}
	_, out := crossitem.Pass(crossitem.Stage{}, fact, steps)
	want := map[kernel.Module][]catalog.Type{
		"machinelogs":   {catalog.EquipmentDeviationDetected},
		"nonconformity": {catalog.EquipmentDeviationDetected, catalog.EquipmentViolationWindowResolved},
		"analysis":      {catalog.EquipmentDeviationDetected, catalog.EquipmentViolationWindowResolved, catalog.DecisionNonconformityRegistered},
		"notifications": {catalog.EquipmentDeviationDetected, catalog.EquipmentViolationWindowResolved, catalog.DecisionNonconformityRegistered, catalog.IncidentScopeComputed},
	}
	for _, m := range slices.Sorted(maps.Keys(want)) {
		if w := want[m]; !slices.Equal(got[m], w) {
			t.Fatalf("%s получил %v, ожидалось %v", m, got[m], w)
		}
	}
	if len(out) != 3 || out[2].Type != catalog.IncidentScopeComputed {
		t.Fatalf("выход прохода: %+v", out)
	}
	// Запись стадии как вход: тот же id, поток, изделие, причина и basis_seq, что у записи в журнале.
	rec, ok := crossitem.AddressedRecord(out[0], fact)
	if !ok || rec.EventID != crossitem.AddressedID(out[0]) || rec.ItemID != "ENT01:A" || rec.CausationID != "f1" ||
		rec.BasisSeq != 7 || rec.Seq != 7 || string(rec.Data) != `{"k":1}` || !rec.OccurredAt.Equal(fact.OccurredAt) {
		t.Fatalf("запись стадии: %+v", rec)
	}

	// Запись, вызванная адресованной записью стадии: состояние меняется, но
	// выхода нет — и поздние модули её выходов не получают.
	clear(got)
	s := crossitem.Stage{Own: crossitem.Own{Emitted: map[string]bool{"a1": true}}}
	caused := fact
	caused.EventID, caused.CausationID = "f2", "a1"
	if _, out := crossitem.Settle(s, caused, func(s crossitem.Stage, r kernel.Record) (crossitem.Stage, []kernel.Addressed) {
		return crossitem.Pass(s, r, steps)
	}); len(out) != 0 {
		t.Fatalf("выход записи, вызванной своей адресованной: %+v", out)
	}
	if len(got["analysis"]) != 1 {
		t.Fatalf("analysis получил подавленные записи: %v", got["analysis"])
	}
}
