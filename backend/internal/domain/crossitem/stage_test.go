package crossitem_test

import (
	"testing"

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
