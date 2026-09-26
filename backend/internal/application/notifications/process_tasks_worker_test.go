package notifications_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	notif "ant/internal/domain/notifications"
)

// Задачи процесса на живом движке: журнал, воркер (пересвёртка изделия на
// каждую запись и сравнение слотов, AD-3, AD-5) и проекция задач — как на
// стенде. Короткая история: регистрация на кромках, бирка, сварка фактами
// поста, КТ-3; задачи сварщика после неё сняты.
func TestProcessStepTasksWorker(t *testing.T) {
	for _, batch := range []bool{false, true} {
		name := "по записи"
		if batch {
			name = "пачкой"
		}
		t.Run(name, func(t *testing.T) {
			f := newFlow(t)
			ctx := context.Background()
			j := enginemem.New(nil)
			part := engineapp.Partition{Number: 0, Epoch: 1}
			j.SetEpoch(appjournal.PartitionLease(0), 1)
			codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}
			feed := &enginemem.Feed{J: j, Parts: []engineapp.Partition{part}}
			w := engineapp.NewWorker(engineapp.WorkerConfig{Feed: feed, Codec: codec, Bundles: f.bundles, Refresh: 50 * time.Millisecond, Backoff: 10 * time.Millisecond})
			step := func() {
				t.Helper()
				c, cancel := context.WithTimeout(ctx, time.Second)
				defer cancel()
				works, err := feed.Next(c, part)
				if err != nil {
					t.Fatalf("нет работы: %v", err)
				}
				if err := w.Process(ctx, part, works); err != nil {
					t.Fatal(err)
				}
			}
			n := 0
			put := func(tp catalog.Type, h float64, data map[string]any) {
				t.Helper()
				n++
				info, _ := catalog.Lookup(tp)
				p, err := codec.Encode(ctx, engineapp.Out{EventID: fmt.Sprintf("00000000-0000-7000-9000-%012d", n), Type: tp, Kind: info.Kind,
					Stream: "item:" + f.item, ItemID: f.item, OccurredAt: p0.Add(time.Duration(h * float64(time.Hour))), Correlation: "c", Data: data})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
					t.Fatal(err)
				}
				if !batch {
					step()
				}
			}
			put(catalog.ItemItemRegistered, 0, map[string]any{"item_id": f.item, "item_type_id": "FL-100.00.000", "item_revision": "Б",
				"process_version_hash": f.hash, "normative_rev": "flange-1", "lot_ids": []string{"LOT-FL-1"}, "entry_step_key": "welding.edge_prep"})
			put(catalog.ItemCarrierApplied, 0.02, map[string]any{"carrier_type": "tag_qr", "value": "show-is2-20260921/TAG:F-101", "is_temporary": true})
			put(catalog.OperationRunStarted, 0.5, map[string]any{"operation_run_id": "SV-101-1", "operation_code": "SV", "step_key": "welding.weld",
				"operator_id": "W21", "equipment_id": "IS-1", "station_id": "ST-WELD"})
			put(catalog.OperationRunFinished, 0.7, map[string]any{"operation_run_id": "SV-101-1", "completion": "completed"})
			if batch {
				step()
			}
			// Проекция задач по журналу — как у проектора.
			tasks := map[string]notif.TaskRecord{}
			for _, e := range j.Entries() {
				d, err := codec.Decode(ctx, e)
				if err != nil {
					t.Fatal(err)
				}
				if d.Info.Type == catalog.OpsProcessingFailed {
					t.Fatalf("обработка остановлена: %s", d.Record.Data)
				}
				for _, k := range notif.TaskKeys(d.Record) {
					tasks[k] = notif.StepTask(tasks[k], d.Record)
				}
			}
			for _, v := range tasks {
				if v.Kind == notif.KindProcessStep && v.State == notif.TaskOpen && v.Role == "performer" {
					t.Errorf("задача сварщика не снята: %s (%s)", v.Title, v.Operation)
				}
			}
			if len(tasks) == 0 {
				t.Fatal("задач нет вовсе")
			}
		})
	}
}
