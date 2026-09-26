package engine_test

import (
	"encoding/json"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/engine"
	"ant/internal/domain/engine/enginetest"
	"ant/internal/domain/kernel"
	"ant/internal/domain/notifications"
	"ant/internal/domain/quality/qualitytest"
)

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

func inspection(seq int64, id string, at time.Duration, outcome string) kernel.Record {
	return kernel.Record{Seq: seq, EventID: id, Type: catalog.InspectionResultRecorded, Kind: catalog.KindFact,
		ItemID: "ENT01:I-1", Stream: "item:ENT01:I-1", OccurredAt: t0.Add(at), RecordedAt: t0.Add(time.Duration(seq) * time.Hour),
		Data: json.RawMessage(`{"outcome":"` + outcome + `"}`)}
}

// recordAs превращает план в записанные версии (как их восстановит воркер из журнала).
func recordAs(t *testing.T, plan []engine.Planned, seq int64) []engine.Recorded {
	t.Helper()
	var out []engine.Recorded
	for _, p := range plan {
		d, err := json.Marshal(p.Reaction.Data)
		if err != nil {
			t.Fatal(err)
		}
		cd, err := engine.Canonical(d)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, engine.Recorded{EventID: p.EventID, Seq: seq, Module: p.Reaction.Module, Type: p.Reaction.Type,
			Slot: p.Reaction.Slot, Version: p.Version, RuleRev: p.Reaction.RuleRev, AutomationMode: p.Reaction.AutomationMode,
			Causes: p.Reaction.Causes, OccurredAt: p.Reaction.OccurredAt, Data: cd})
	}
	return out
}

// Новый слот → версия 1; тот же вывод → ничего; позднее событие → версия 2
// «пересмотрен из-за записи ‹id›» (AD-3, AD-5, FR-32).
func TestDiffNewSameRevised(t *testing.T) {
	in := []kernel.Record{inspection(1, "e1", 2*time.Minute, "defect_indicated")}
	_, rs := enginetest.Fold(engine.Bundle{}, in)
	plan, err := engine.Diff(rs, nil, engine.Trigger{EventID: "e1"})
	if err != nil || len(plan) != 1 || plan[0].Change != engine.ChangeNew || plan[0].Version != 1 || plan[0].Supersedes != "" {
		t.Fatalf("новый слот: %+v %v", plan, err)
	}
	rec := recordAs(t, plan, 2)

	again, err := engine.Diff(rs, rec, engine.Trigger{EventID: "e1"})
	if err != nil || len(again) != 0 {
		t.Fatalf("тот же вывод не должен дописываться: %+v %v", again, err)
	}

	late := inspection(3, "e0", time.Minute, "defect_indicated") // возникло раньше, узнали позже
	_, rs2 := enginetest.Fold(engine.Bundle{}, append(in, late))
	plan2, err := engine.Diff(rs2, rec, engine.Trigger{EventID: "e0", OccurredAt: late.OccurredAt})
	if err != nil || len(plan2) != 1 {
		t.Fatalf("позднее событие: %+v %v", plan2, err)
	}
	p := plan2[0]
	if p.Change != engine.ChangeRevised || p.Version != 2 || p.Supersedes != rec[0].EventID || p.RevisedDueTo != "e0" {
		t.Fatalf("пересмотр: %+v", p)
	}
	if p.EventID != p.Reaction.ID(2) || p.EventID == rec[0].EventID {
		t.Fatalf("reaction_id версии 2: %s", p.EventID)
	}
}

// Исчезнувшая защитная реакция не снимается: задача «основание защиты
// изменилось» уполномоченному; защита вернулась — задача снимается (AD-3).
func TestDiffProtectiveNotWithdrawn(t *testing.T) {
	in := []kernel.Record{inspection(1, "e1", time.Minute, "defect_indicated")}
	_, rs := enginetest.Fold(engine.Bundle{}, in)
	plan, _ := engine.Diff(rs, nil, engine.Trigger{EventID: "e1"})
	rec := recordAs(t, plan, 2)

	fix := inspection(3, "e2", 2*time.Minute, "no_defect_indicated")
	fix.Corrects = "e1"
	in2 := append(in, fix)
	_, rs2 := enginetest.Fold(engine.Bundle{}, in2)
	if len(rs2) != 0 {
		t.Fatalf("после исправления сигнала быть не должно: %+v", rs2)
	}
	plan2, err := engine.Diff(rs2, rec, engine.Trigger{EventID: "e2"})
	if err != nil || len(plan2) != 1 {
		t.Fatalf("ждали одну задачу: %+v %v", plan2, err)
	}
	task := plan2[0]
	if task.Reaction.Type != catalog.TaskTaskCreated || task.Reaction.Module != "notifications" || task.Change != engine.ChangeNew {
		t.Fatalf("задача пересмотра защиты: %+v", task)
	}
	d := task.Reaction.Data.(ev.TaskTaskCreatedV1)
	if d.Kind != ev.TaskTaskCreatedV1KindProtectionBasisChanged || d.AssigneeRoleID != engine.ReviewerRole {
		t.Fatalf("данные задачи: %+v", d)
	}
	rec = append(rec, recordAs(t, plan2, 4)...)

	// Повторная пересвёртка без изменений — ничего нового.
	if again, _ := engine.Diff(rs2, rec, engine.Trigger{EventID: "e2"}); len(again) != 0 {
		t.Fatalf("задача не должна пересматриваться: %+v", again)
	}

	// Защита вернулась — задача снята версией 2 слота с типом отзыва.
	back := inspection(5, "e3", 3*time.Minute, "defect_indicated")
	_, rs3 := enginetest.Fold(engine.Bundle{}, append(in2, back))
	plan3, err := engine.Diff(rs3, rec, engine.Trigger{EventID: "e3"})
	if err != nil || len(plan3) != 2 {
		t.Fatalf("ждали пересмотр сигнала и снятие задачи: %+v %v", plan3, err)
	}
	var withdrawn, revised bool
	for _, p := range plan3 {
		switch {
		case p.Change == engine.ChangeWithdrawn && p.Reaction.Type == catalog.TaskTaskWithdrawn && p.Version == 2:
			withdrawn = p.Reaction.Data.(ev.TaskTaskWithdrawnV1).TaskID == d.TaskID
		case p.Change == engine.ChangeRevised && p.Reaction.Type == catalog.QualitySignalRaised:
			revised = true
		}
	}
	if !withdrawn || !revised {
		t.Fatalf("план: %+v", plan3)
	}
}

// Два типа в одном слоте — ошибка: совпал бы reaction_id (AD-3).
func TestDiffSlotConflict(t *testing.T) {
	slot := kernel.Slot{RuleID: "r", Subject: "item:X", TriggerKey: "k"}
	a, _ := qualitytest.SignalRaised(slot, nil)
	b, _ := notifications.ReviewTask(slot, ev.TaskTaskCreatedV1KindProtectionBasisChanged, "t", "r", "")
	if _, err := engine.Diff([]kernel.Reaction{a, b}, nil, engine.Trigger{}); err == nil {
		t.Fatal("ждали ошибку конфликта слота")
	}
}

// Решение, принятое до поздних данных, не отменяется: задача автору (AD-5, FR-32).
func TestDecisionBeforeNewData(t *testing.T) {
	dec := kernel.Record{Seq: 2, EventID: "d1", Type: catalog.DecisionNonconformityConfirmed, Kind: catalog.KindDecision,
		ItemID: "ENT01:I-1", Stream: "item:ENT01:I-1", OccurredAt: t0.Add(time.Hour), BasisSeq: 1, Actor: "INS-01"}
	in := []kernel.Record{inspection(1, "e1", time.Minute, "no_defect_indicated"), dec}
	if _, rs := engine.Fold(engine.Bundle{}, in); len(rs) != 0 {
		t.Fatalf("без поздних данных задач нет: %+v", rs)
	}
	late := inspection(3, "e0", 30*time.Minute, "no_defect_indicated")
	_, rs := engine.Fold(engine.Bundle{}, append(in, late))
	if len(rs) != 1 || rs[0].Slot.RuleID != engine.RuleDecisionBeforeNewData {
		t.Fatalf("ждали задачу пересмотра решения: %+v", rs)
	}
	d := rs[0].Data.(ev.TaskTaskCreatedV1)
	if d.Kind != ev.TaskTaskCreatedV1KindReviewAfterNewData || d.AssigneePersonID == nil || *d.AssigneePersonID != "INS-01" {
		t.Fatalf("данные: %+v", d)
	}
	if len(rs[0].Causes) != 2 || rs[0].Causes[0] != "d1" || rs[0].Causes[1] != "e0" {
		t.Fatalf("причины: %v", rs[0].Causes)
	}
	// Событие, возникшее после решения, — не повод пересматривать.
	after := inspection(4, "e9", 2*time.Hour, "no_defect_indicated")
	_, rs2 := engine.Fold(engine.Bundle{}, append(in, after))
	if len(rs2) != 0 {
		t.Fatalf("событие после решения: %+v", rs2)
	}
}

// Запрос на момент — свёртка префикса по оси (AD-22).
func TestPrefixAxes(t *testing.T) {
	a := inspection(1, "a", time.Minute, "defect_indicated")    // recorded t0+1h
	b := inspection(2, "b", 30*time.Second, "defect_indicated") // recorded t0+2h, возникло раньше a
	c := inspection(3, "c", 3*time.Hour, "no_defect_indicated") // recorded t0+3h
	in := []kernel.Record{a, b, c}
	got := engine.Prefix(in, engine.Moment{Axis: engine.AxisOccurred, At: t0.Add(time.Minute)})
	if len(got) != 2 {
		t.Fatalf("как было на t0+1м: %d", len(got))
	}
	got = engine.Prefix(in, engine.Moment{Axis: engine.AxisRecorded, At: t0.Add(90 * time.Minute)})
	if len(got) != 1 || got[0].EventID != "a" {
		t.Fatalf("что мы знали на t0+1.5ч: %+v", got)
	}
	_, rs := engine.FoldAt(engine.Bundle{}, in, engine.Moment{Axis: engine.AxisRecorded, At: t0.Add(90 * time.Minute)})
	if len(rs) != 0 {
		t.Fatalf("FoldAt без пустышки: %+v", rs)
	}
	if got := engine.Prefix(in, engine.Moment{MaxSeq: 2}); len(got) != 2 {
		t.Fatalf("префикс по seq: %d", len(got))
	}
}

// Хеш состояния не зависит от порядка подачи и от basis_seq (AD-6 state_hash).
func TestStateHashStable(t *testing.T) {
	in := []kernel.Record{inspection(1, "e1", 2*time.Minute, "defect_indicated"), inspection(2, "e2", time.Minute, "defect_indicated")}
	s1, r1 := enginetest.Fold(engine.Bundle{}, in)
	s2, r2 := enginetest.Fold(engine.Bundle{}, []kernel.Record{in[1], in[0]})
	h1, err1 := engine.StateHash(s1, r1)
	h2, err2 := engine.StateHash(s2, r2)
	if err1 != nil || err2 != nil || h1 != h2 || len(h1) != len(engine.HashPrefix)+64 {
		t.Fatalf("хеши: %s %s %v %v", h1, h2, err1, err2)
	}
	_, r3 := enginetest.Fold(engine.Bundle{}, in[:1])
	if h3, _ := engine.StateHash(s1, r3); h3 == h1 {
		t.Fatal("другой вывод — другой хеш")
	}
}
