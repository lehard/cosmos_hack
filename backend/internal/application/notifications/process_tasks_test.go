package notifications_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	notifapp "ant/internal/application/notifications"
	procapp "ant/internal/application/process"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	notif "ant/internal/domain/notifications"
	dp "ant/internal/domain/process"
)

// Задачи ролей порождает процесс, а не сценарий (показ SHOW-IS2): изделие
// входит в шаг BPMN с действием человека — роль по дорожке шага получает
// задачу с операцией и меткой изделия; действие продвигает токен — задача
// снимается. Полная композиция движка (engine.Fold) на стартовом процессе
// фланца с нормативным слоем notifications.Bundles, как у воркера.

var p0 = time.Date(2026, 9, 21, 5, 0, 0, 0, time.UTC)

type flow struct {
	t       *testing.T
	bundles notifapp.Bundles
	item    string
	hash    string
	in      []kernel.Record
}

func newFlow(t *testing.T) *flow {
	t.Helper()
	xml, err := os.ReadFile("../../../../normative/process/flange-process.bpmn")
	if err != nil {
		t.Fatal(err)
	}
	store := &procapp.MemVersions{}
	seed, err := procapp.EnsureSeed(context.Background(), store, xml, p0.Add(-72*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return &flow{t: t, bundles: notifapp.Bundles{Next: &procapp.Bundles{Store: store, TTL: time.Nanosecond}},
		item: "ENT01:show-is2-20260921-1/I-3CDF7159", hash: seed.Hash}
}

func (f *flow) add(tp catalog.Type, h float64, data map[string]any) {
	seq := int64(len(f.in) + 1)
	info, _ := catalog.Lookup(tp)
	b, err := json.Marshal(data)
	if err != nil {
		f.t.Fatal(err)
	}
	at := p0.Add(time.Duration(h * float64(time.Hour)))
	f.in = append(f.in, kernel.Record{Seq: seq, EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", seq), Type: tp, Kind: info.Kind, Provenance: "personal",
		ItemID: f.item, Stream: "item:" + f.item, OccurredAt: at, ReceivedAt: at, RecordedAt: at, Data: b})
}

// tasks — задачи процесса по итогу свёртки: открытые (task.task.created) и
// снятые (task.task.withdrawn) по task_id.
func (f *flow) tasks() (open map[string]notif.TaskData, closed map[string]bool) {
	f.t.Helper()
	b, _, err := f.bundles.Bundle(context.Background(), f.item, f.in)
	if err != nil {
		f.t.Fatal(err)
	}
	_, rs := engine.Fold(b, f.in)
	open, closed = map[string]notif.TaskData{}, map[string]bool{}
	for _, r := range rs {
		raw, _ := json.Marshal(r.Data)
		switch r.Type {
		case catalog.TaskTaskCreated:
			var d notif.TaskData
			_ = json.Unmarshal(raw, &d)
			if d.Kind == notif.KindProcessStep {
				open[d.TaskID] = d
			}
		case catalog.TaskTaskWithdrawn:
			var d notif.TaskWithdrawnData
			_ = json.Unmarshal(raw, &d)
			closed[d.TaskID] = true
		}
	}
	for id := range closed {
		delete(open, id)
	}
	return open, closed
}

func (f *flow) only(op string) notif.TaskData {
	f.t.Helper()
	open, _ := f.tasks()
	var got []notif.TaskData
	for _, d := range open {
		if d.OperationID == op && d.AssigneeRoleID != "storekeeper" {
			got = append(got, d)
		}
	}
	if len(got) != 1 {
		f.t.Fatalf("задач %s: %d, открыты %+v", op, len(got), open)
	}
	return got[0]
}

func TestProcessStepTasks(t *testing.T) {
	f := newFlow(t)
	f.add(catalog.ItemItemRegistered, 0, map[string]any{"item_id": f.item, "item_type_id": "FL-100.00.000", "item_revision": "Б",
		"process_version_hash": f.hash, "normative_rev": "flange-1", "lot_ids": []string{"LOT-FL-1"}})
	f.add(catalog.ItemCarrierApplied, 0.01, map[string]any{"carrier_type": "tag_qr", "value": "show-is2-20260921/TAG:F-001", "is_temporary": true})
	f.add(catalog.DecisionPresentationResolved, 1, map[string]any{"step_key": "incoming.zt1_lot_acceptance", "closing_point": "ZT-1",
		"resolution": dp.ResolutionAccept, "presentation_no": 1, "method_event_ids": []string{}})
	f.add(catalog.OperationRunStarted, 2, map[string]any{"operation_run_id": "MO-1", "operation_code": "MO", "step_key": "machining.cnc", "operator_id": "O17"})
	f.add(catalog.OperationRunFinished, 3, map[string]any{"operation_run_id": "MO-1", "completion": "completed"})
	f.add(catalog.DecisionPresentationResolved, 4, map[string]any{"step_key": "machining.zt2_acceptance", "closing_point": "ZT-2",
		"resolution": dp.ResolutionAccept, "presentation_no": 1, "method_event_ids": []string{}})

	// Мастер механического цеха отправляет — у мастера сварочного «Принять в цех Ф-001».
	send := f.only(dp.OpMovementSend)
	if send.AssigneeRoleID != "site_foreman" || send.LocationID != "WS-MC" {
		t.Fatalf("отправка: %+v", send)
	}
	f.add(catalog.OperationMovementSent, 4.5, map[string]any{"from_location_id": "WS-MC", "to_location_id": "WS-WC", "sent_by": "FOR-MC",
		"step_key": "machining.send_to_welding"})
	rcv := f.only(dp.OpMovementReceive)
	if rcv.AssigneeRoleID != "site_foreman" || rcv.LocationID != "WS-WC" || rcv.Title != "Принять в цех Ф-001" ||
		rcv.ItemLabel != "Ф-001" || rcv.ItemID != f.item || rcv.StepKey != "welding.receive" || rcv.SubjectRef != "item:"+f.item {
		t.Fatalf("приёмка: %+v", rcv)
	}
	if _, closed := f.tasks(); !closed[send.TaskID] {
		t.Fatal("отправка выполнена — задача не снята")
	}

	// Мастер принял — задача снята, у сварщика — операции сварочного цеха.
	f.add(catalog.OperationMovementReceived, 5, map[string]any{"to_location_id": "WS-WC", "destination_kind": "workshop",
		"inspection_on_receipt": "no_damage", "received_by": "FOR-WC", "step_key": "welding.receive"})
	f.add(catalog.OperationMovementReceived, 5.1, map[string]any{"to_location_id": "WS-WC", "destination_kind": "workshop",
		"inspection_on_receipt": "no_damage", "received_by": "FOR-WC", "step_key": "incoming.issue_pipe"})
	if _, closed := f.tasks(); !closed[rcv.TaskID] {
		t.Fatal("приёмка выполнена — задача не снята")
	}
	// Подготовка кромок — подготовительный шаг сварки (окно «edge_prep<=PT8H»
	// у сварки): сварщику сразу «Начать: Сварка…», кромки засчитываются по её началу.
	weld := f.only(dp.OpOperationStart)
	if weld.AssigneeRoleID != "performer" || weld.LocationID != "WS-WC" || weld.StepKey != "welding.weld" ||
		weld.Title != "Начать: Сварка фланца с патрубком — Ф-001" {
		t.Fatalf("сварка: %+v", weld)
	}
	// Кромки записаны отдельным выполнением — пока оно идёт, задача «Завершить» его;
	// после — снова «Начать: Сварка…» (та же задача).
	f.add(catalog.OperationRunStarted, 6, map[string]any{"operation_run_id": "EP-1", "operation_code": "EP", "step_key": "welding.edge_prep", "operator_id": "W21"})
	if ep := f.only(dp.OpOperationFinish); ep.StepKey != "welding.edge_prep" {
		t.Fatalf("кромки в работе: %+v", ep)
	}
	f.add(catalog.OperationRunFinished, 6.5, map[string]any{"operation_run_id": "EP-1", "completion": "completed"})
	if again := f.only(dp.OpOperationStart); again.TaskID != weld.TaskID {
		t.Fatalf("сварка после кромок: %+v", again)
	}
	f.add(catalog.OperationRunStarted, 7, map[string]any{"operation_run_id": "SV-1", "operation_code": "SV", "step_key": "welding.weld", "operator_id": "W21",
		"equipment_id": "IS-2"})
	fin := f.only(dp.OpOperationFinish)
	if fin.StepKey != "welding.weld" {
		t.Fatalf("выполнено: %+v", fin)
	}
	f.add(catalog.OperationRunFinished, 7.7, map[string]any{"operation_run_id": "SV-1", "completion": "completed"})
	open, closed := f.tasks()
	if !closed[weld.TaskID] || !closed[fin.TaskID] {
		t.Fatalf("сварка выполнена — задачи не сняты: %+v", open)
	}
	// Дальше — машинный контроль КТ-3 и ЗТ-3 в очереди контролёра: задач процесса нет.
	for _, d := range open {
		if d.StepKey != "incoming.issue_assembly_parts" || d.AssigneeRoleID != "storekeeper" {
			t.Errorf("лишняя задача после сварки: %+v", d)
		}
	}
}
