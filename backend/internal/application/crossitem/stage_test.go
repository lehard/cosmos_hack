package crossitem_test

import (
	"context"
	"testing"
	"time"

	"ant/internal/application/crossitem"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	dcross "ant/internal/domain/crossitem"
	"ant/internal/domain/kernel"
)

// Рамка стадии (AD-42): факт без изделия → адресованная запись изделию с
// occurred_at причины, basis_seq и causation; состояние стадии и курсор — в
// одной записи; повтор той же пачки после смены лидера не дублирует выход.
func TestStageRunner(t *testing.T) {
	j := enginemem.New(nil)
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", Partitions: 4}
	bind := func(s dcross.Stage, r kernel.Record) (dcross.Stage, []kernel.Addressed) {
		if r.Type != catalog.EquipmentStateChanged {
			return s, nil
		}
		a, err := kernel.NewAddressed("crossitem", catalog.BindingLinkResolved, "item:ENT01:I-7", r.EventID, map[string]any{"item_id": "ENT01:I-7"}, r)
		if err != nil {
			t.Fatal(err)
		}
		return s, []kernel.Addressed{a}
	}
	stage := &crossitem.StageRunner{Consumer: j, Codec: codec, Store: j,
		Fold: func(s dcross.Stage, r kernel.Record) (dcross.Stage, []kernel.Addressed) {
			return dcross.Settle(s, r, bind)
		}}

	occurred := time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)
	fact, err := codec.Encode(context.Background(), engineapp.Out{EventID: "f1", Type: catalog.EquipmentStateChanged, Kind: catalog.KindFact,
		Stream: "equipment:WLD-1", OccurredAt: occurred, Data: map[string]any{"state": "run"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(context.Background(), appjournal.AppendRequest{Batch: []appjournal.Pending{fact}}); err != nil {
		t.Fatal(err)
	}
	batch := j.Entries()
	if !crossitem.IsStageInput(batch[0]) {
		t.Fatal("факт без изделия — вход стадии")
	}
	_, rq, err := stage.Apply(context.Background(), dcross.Stage{}, batch)
	if err != nil || len(rq.Batch) != 1 {
		t.Fatalf("адресованная запись: %+v %v", rq, err)
	}
	e := rq.Batch[0].Entry
	if e.EventType != string(catalog.BindingLinkResolved) || e.ItemID == nil || *e.ItemID != "ENT01:I-7" ||
		*e.BasisSeq != 1 || *e.CausationID != "f1" || e.OccurredAt != engineapp.FormatTime(occurred) {
		t.Fatalf("запись: %+v", e)
	}
	if e.Partition != kernel.PartitionOf("ENT01:I-7", 4) {
		t.Fatalf("партиция адресата: %d", e.Partition)
	}
	if _, err := j.Append(context.Background(), rq); err != nil {
		t.Fatal(err)
	}
	if crossitem.IsStageInput(j.Entries()[1]) {
		t.Fatal("собственная адресованная запись стадии — не её вход")
	}

	// Лидер сменился: состояние восстановлено из проекции, повтор пачки не
	// даёт повторной записи.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_ = stage.Run(ctx, appjournal.Fence{Lease: "crossitem", Epoch: 1})
	var n int
	for _, x := range j.Entries() {
		if x.EventType == string(catalog.BindingLinkResolved) {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("повтор пачки продублировал адресованную запись: %d", n)
	}
	if c := j.Cursor(crossitem.ConsumerStage, appjournal.GlobalPartition); c != int64(len(j.Entries())) {
		t.Fatalf("курсор стадии: %d", c)
	}
}
