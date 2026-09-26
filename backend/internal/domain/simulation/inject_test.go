package simulation

import (
	"encoding/json"
	"errors"
	"testing"
)

// Кнопки цифрового стенда (FR-152, эпик 36): заготовленные события сбоев
// приводятся к плану прогона детерминированно (AD-4, AD-38).
func TestPlanInjection(t *testing.T) {
	b := bundle()
	b.Run.Noise = Noise{}
	p, err := Generate(b, Params{RunID: "r-1"})
	if err != nil {
		t.Fatal(err)
	}
	at := p.End
	in := func(k InjectionKind, n int, target string) InjectionInput {
		return InjectionInput{Plan: p, World: b.World, Kind: k, N: n, Target: target, Delivered: len(p.Emissions), At: at,
			StandSeq: map[string]int64{}}
	}
	event := func(e Emission) map[string]any {
		ev, err := decodeEvent(e.Event)
		if err != nil {
			t.Fatal(err)
		}
		return ev
	}

	// Повтор: те же байты, тот же источник и номер (F01).
	dup, err := PlanInjection(in(InjectDuplicate, 1, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(dup.Emissions) != 1 || dup.Target == nil || dup.Emissions[0].EventID != dup.Target.EventID ||
		string(dup.Emissions[0].Event) != string(dup.Target.Event) || dup.Emissions[0].Delivery != DeliveryDuplicate || dup.Item == "" {
		t.Fatalf("повтор: %+v", dup)
	}
	again, _ := PlanInjection(in(InjectDuplicate, 1, ""))
	if again.Emissions[0].EventID != dup.Emissions[0].EventID {
		t.Fatal("тот же вход — другая цель (AD-4)")
	}

	// Опоздавшее: новое событие, случилось за минуту до цели, пришло в At (F03).
	late, err := PlanInjection(in(InjectLate, 2, dup.Target.EventID))
	if err != nil {
		t.Fatal(err)
	}
	le := late.Emissions[0]
	if le.EventID == dup.Target.EventID || !le.OccurredAt.Equal(dup.Target.OccurredAt.Add(-LateBy)) || !le.DeliverAt.Equal(at) ||
		le.SourceID != "r-1/"+StandLate || le.SourceSeq != 1 || le.Delivery != DeliveryLate {
		t.Fatalf("опоздавшее: %+v", le)
	}
	if ev := event(le); ev["run_id"] != "r-1" || ev["event_id"] != le.EventID || ev["item_ref"] == nil {
		t.Fatalf("конверт опоздавшего: %v", ev)
	}

	// Испорченный кадр: кадр камеры с качеством 0,30 (F08).
	fr, err := PlanInjection(in(InjectCorruptFrame, 3, ""))
	if err != nil {
		t.Fatal(err)
	}
	d := event(fr.Emissions[0])["data"].(map[string]any)
	if d["method"] != "camera" || d["observation_quality_bp"] != json.Number("3000") || fr.Emissions[0].SourceID != "r-1/"+StandFrame {
		t.Fatalf("кадр: %v", d)
	}
	if _, err := PlanInjection(in(InjectCorruptFrame, 3, p.Emissions[0].EventID)); !isRefusal(err) {
		t.Fatalf("кадр по событию не камеры — отказ с причиной: %v", err)
	}

	// Ток вне уставки на сварке: 176 А при 160 ± 10 (S05, S07).
	mf, err := PlanInjection(in(InjectMachineFault, 4, ""))
	if err != nil {
		t.Fatal(err)
	}
	md := event(mf.Emissions[0])["data"].(map[string]any)
	w := p.Truth.Welds[0]
	if mf.Emissions[0].EventType != "equipment.deviation.detected" || md["equipment_id"] != "IS-1" ||
		md["value"].(map[string]any)["value"] != json.Number("176") || mf.Item != "F-501" ||
		mf.Emissions[0].OccurredAt.Before(w.ArcFrom) || !mf.Emissions[0].OccurredAt.Before(w.ArcTo) {
		t.Fatalf("ток вне уставки: %v %+v", md, mf.Emissions[0])
	}

	// Потеря данных: запись до и после разрыва, номера между ними потеряны (F07).
	ls, err := PlanInjection(in(InjectDataLoss, 5, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(ls.Emissions) != 2 || ls.Emissions[0].SourceSeq != 1 || ls.Emissions[1].SourceSeq != 2+LostRecords ||
		ls.StandSeq[StandLoss] != 2+LostRecords || ls.Stand == nil || ls.Stand.Fault != "drop" || ls.Source != "r-1/"+StandLoss {
		t.Fatalf("потеря данных: %+v", ls)
	}
	in6 := in(InjectDataLoss, 6, "")
	in6.StandSeq = ls.StandSeq
	ls2, _ := PlanInjection(in6)
	if ls2.Emissions[0].SourceSeq != 3+LostRecords {
		t.Fatalf("номера источника стенда продолжаются: %d", ls2.Emissions[0].SourceSeq)
	}

	// Подделка: правка на месте результата контроля (F25), событий в приём нет.
	tm, err := PlanInjection(in(InjectTamper, 7, ""))
	if err != nil {
		t.Fatal(err)
	}
	if tm.Tamper == nil || tm.Tamper.Kind != "update_in_place" || tm.Target.EventType != "inspection.result.recorded" || len(tm.Emissions) != 0 {
		t.Fatalf("подделка: %+v", tm)
	}

	// Цель ещё не доставлена — отказ, а не выдумка.
	early := in(InjectDuplicate, 8, p.Emissions[len(p.Emissions)-1].EventID)
	early.Delivered = 1
	if _, err := PlanInjection(early); !isRefusal(err) {
		t.Fatalf("недоставленная цель: %v", err)
	}
	none := in(InjectMachineFault, 9, "")
	none.At = p.Start
	if _, err := PlanInjection(none); !isRefusal(err) {
		t.Fatalf("сварки ещё не было: %v", err)
	}
}

func isRefusal(err error) bool {
	var ie *InjectionError
	return errors.As(err, &ie) && ie.Reason != ""
}

// changed — значение изменилось относительно «до».
func TestEvaluateChanged(t *testing.T) {
	before := []any{map[string]any{"size": json.Number("34")}}
	after := []any{map[string]any{"size": json.Number("13")}}
	if r := Evaluate(Check{Op: "changed"}, after, true, before); r.Status != StatusPassed {
		t.Fatalf("изменилось: %+v", r)
	}
	if r := Evaluate(Check{Op: "changed"}, before, true, before); r.Status != StatusFailed {
		t.Fatalf("не изменилось: %+v", r)
	}
	if r := Evaluate(Check{Op: "changed"}, []any{}, true, nil); r.Status != StatusFailed {
		t.Fatalf("без «до»: %+v", r)
	}
}
