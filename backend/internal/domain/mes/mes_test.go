package mes_test

import (
	"encoding/json"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
	"ant/internal/domain/mes"
)

func rec(t catalog.Type, id, item string, data any) kernel.Record {
	b, _ := json.Marshal(data)
	return kernel.Record{Type: t, EventID: id, ItemID: item, Data: b, OccurredAt: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)}
}

func reason() ev.Reason { return ev.Reason{Text: "решение контролёра"} }

// Блок изделия уходит в MES один раз на цикл; снятие — когда сняты все
// записи, державшие блок; снятие основания правилом блок не снимает (AD-27).
func TestPlanHold(t *testing.T) {
	item := "ENT01:F-007"
	h := mes.Hold{Subject: item}
	step := func(r kernel.Record) []mes.HoldDraft {
		t.Helper()
		var ds []mes.HoldDraft
		var err error
		h, ds, err = mes.PlanHold(h, r)
		if err != nil {
			t.Fatal(err)
		}
		return ds
	}
	ds := step(rec(catalog.DecisionContainmentApplied, "e1", item, ev.DecisionContainmentAppliedV1{Level: ev.AxisContainmentItemHold, Basis: []ev.UUID{"x"}}))
	if len(ds) != 1 || !ds[0].Data.Hold || ds[0].Key != item+"/hold/1" || string(*ds[0].Data.ItemID) != item {
		t.Fatalf("блок: %+v", ds)
	}
	if ds := step(rec(catalog.DecisionItemIsolated, "e2", item, ev.DecisionItemIsolatedV1{Reason: reason()})); len(ds) != 0 {
		t.Fatalf("повторный блок того же изделия — без нового сообщения: %+v", ds)
	}
	if ds := step(rec(catalog.GenealogyContainmentPropagated, "e3", item, ev.GenealogyContainmentPropagatedV1{Level: ev.AxisContainmentItemHold,
		Source: "lot", SourceEventID: "e0", Basis: []ev.UUID{"e0"}})); len(ds) != 0 {
		t.Fatalf("распространение по генеалогии при блоке: %+v", ds)
	}
	if ds := step(rec(catalog.DecisionContainmentReleased, "e4", item, ev.DecisionContainmentReleasedV1{ReleasedEventIds: []ev.UUID{"e1", "e2"}, Reason: reason()})); len(ds) != 0 {
		t.Fatalf("блок держит ещё e3 — снятия нет: %+v", ds)
	}
	ds = step(rec(catalog.DecisionContainmentReleased, "e5", item, ev.DecisionContainmentReleasedV1{ReleasedEventIds: []ev.UUID{"e3"}, Reason: reason()}))
	if len(ds) != 1 || ds[0].Data.Hold || ds[0].Key != item+"/release/1" {
		t.Fatalf("снятие: %+v", ds)
	}
	ds = step(rec(catalog.DecisionContainmentSet, "e6", item, ev.DecisionContainmentSetV1{Level: ev.AxisContainmentItemHold, Reason: reason()}))
	if len(ds) != 1 || ds[0].Key != item+"/hold/2" {
		t.Fatalf("новый цикл: %+v", ds)
	}
	ds = step(rec(catalog.DecisionContainmentSet, "e7", item, ev.DecisionContainmentSetV1{Level: ev.AxisContainmentObserve, Reason: reason()}))
	if len(ds) != 1 || ds[0].Key != item+"/release/2" {
		t.Fatalf("понижение уровня человеком: %+v", ds)
	}
	rx, err := mes.HoldReaction(ds[0])
	if err != nil || rx.Slot.Subject != "erp_message:"+item+"/release/2" {
		t.Fatalf("реакция: %+v %v", rx, err)
	}
}

// Блок партии: сдерживание lot_hold — блок партии в MES; решение по партии
// (кроме «не годна») — снятие.
func TestPlanLotHold(t *testing.T) {
	lot := ev.ObjectID("LOT-P-2026-0915")
	r := rec(catalog.DecisionContainmentApplied, "l1", "ENT01:F-001", ev.DecisionContainmentAppliedV1{Level: ev.AxisContainmentLotHold, LotID: &lot, Basis: []ev.UUID{"x"}})
	s, isLot, _ := mes.HoldSubject(r)
	if s != string(lot) || !isLot {
		t.Fatalf("субъект: %s %v", s, isLot)
	}
	h, ds, _ := mes.PlanHold(mes.Hold{Subject: s, Lot: true}, r)
	if len(ds) != 1 || ds[0].Data.LotID == nil || ds[0].Data.ItemID != nil {
		t.Fatalf("блок партии: %+v", ds)
	}
	_, ds, _ = mes.PlanHold(h, rec(catalog.DecisionLotResolved, "l2", "", ev.DecisionLotResolvedV1{LotID: lot, Resolution: ev.DecisionLotResolvedV1ResolutionReject}))
	if len(ds) != 0 {
		t.Fatal("партия не годна — блок остаётся")
	}
	_, ds, _ = mes.PlanHold(h, rec(catalog.DecisionLotResolved, "l3", "", ev.DecisionLotResolvedV1{LotID: lot, Resolution: ev.DecisionLotResolvedV1ResolutionAccept}))
	if len(ds) != 1 || ds[0].Data.Hold {
		t.Fatalf("партия годна — снятие: %+v", ds)
	}
}

// Входящие MES: задание → mes.job.received; событие операции → факт
// изделия по соответствиям; без соответствия — отложено с причиной (FR-123).
func TestInbound(t *testing.T) {
	j := mes.Job{MessageID: "mes-1", RequestID: "OR-000812", SegmentID: "SR-1", OperationCode: "030", Station: "ИС-1"}
	f := mes.JobFacts(j)
	d := f[0].Data.(ev.MesJobReceivedV1)
	if d.JobID != "MES-OR-000812-SR-1" || d.OperationCode != "030" || string(*d.StationID) != "IS-1" {
		t.Fatalf("задание: %+v", d)
	}
	if mes.JobFacts(j)[0].EventID != f[0].EventID {
		t.Fatal("повторная доставка — тот же event_id")
	}
	env := mes.Env{Steps: map[string]string{"030": "welding.weld"}, Items: map[string]string{"ФЛ-100.00.000#0007": "ENT01:F-007"},
		Persons: map[string]string{"TAB-1142": "welder-07"}}
	e := mes.OperationEvent{MessageID: "mes-2", EventID: "OE-1", Category: mes.EventStart, RunRef: "SRSP-77", OperationCode: "030",
		SubLot: "ФЛ-100.00.000#0007", Personnel: "TAB-1142", At: time.Date(2026, 9, 25, 7, 0, 0, 0, time.UTC)}
	fs, def := mes.EventFacts(env, e)
	if def != nil || len(fs) != 1 || fs[0].Type != catalog.OperationRunStarted || fs[0].ItemID != "ENT01:F-007" {
		t.Fatalf("начало операции: %+v %+v", fs, def)
	}
	st := fs[0].Data.(ev.OperationRunStartedV1)
	if st.StepKey != "welding.weld" || st.OperationRunID != "MES-SRSP-77" || *st.OperatorID != "welder-07" {
		t.Fatalf("данные начала: %+v", st)
	}
	e.Category, e.Personnel = mes.EventEnd, ""
	fs, _ = mes.EventFacts(env, e)
	if fs[0].Type != catalog.OperationRunFinished || fs[0].Data.(ev.OperationRunFinishedV1).OperationRunID != "MES-SRSP-77" {
		t.Fatalf("конец того же выполнения: %+v", fs)
	}
	e.SubLot = "ФЛ-100.00.000#0099"
	if _, def := mes.EventFacts(env, e); def == nil {
		t.Fatal("экземпляр без соответствия — отложено")
	}
	e.SubLot, e.Category, e.OperationCode = "ENT01:F-008", mes.EventStart, "999"
	if _, def := mes.EventFacts(env, e); def == nil {
		t.Fatal("неизвестный код операции — отложено")
	}
	e.OperationCode = "030"
	if fs, def := mes.EventFacts(env, e); def != nil || fs[0].ItemID != "ENT01:F-008" || fs[0].Data.(ev.OperationRunStartedV1).OperatorID != nil {
		t.Fatalf("наш ID из MES и неизвестный исполнитель — null: %+v %+v", fs, def)
	}
}
