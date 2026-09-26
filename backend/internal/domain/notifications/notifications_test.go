package notifications_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"maps"
	"slices"

	"ant/internal/domain/kernel"
	"ant/internal/domain/nonconformity"
	notif "ant/internal/domain/notifications"
)

// fold — свёртка входа изделия модулями nonconformity и notifications в
// порядке композиции (как domain/engine.Step: Reduce всех, затем React, затем
// намерения к nonconformity; импорт движка домену notifications запрещён,
// AD-1). На слот — последняя вычисленная реакция (как engine.Fold).
func fold(recs []kernel.Record) (notif.State, []kernel.Reaction) {
	in := slices.Clone(recs)
	slices.SortStableFunc(in, func(a, b kernel.Record) int {
		switch {
		case kernel.Less(a, b):
			return -1
		case kernel.Less(b, a):
			return 1
		}
		return 0
	})
	var ncs nonconformity.State
	var s notif.State
	bySlot := map[string]kernel.Reaction{}
	for _, r := range in {
		ncs = nonconformity.Reduce(ncs, r, nonconformity.Env{}, nonconformity.Upstream{})
		s = notif.Reduce(s, r, notif.Env{}, notif.Upstream{Nonconformity: &ncs})
		out := nonconformity.React(ncs, nonconformity.Env{}, nonconformity.Upstream{})
		for _, re := range notif.React(s, notif.Env{}, notif.Upstream{Nonconformity: &ncs}).Reactions {
			bySlot[re.Slot.Key()] = re
		}
		for _, it := range out.Intents {
			if it.Target == nonconformity.Module {
				ncs = nonconformity.Apply(ncs, it)
			}
		}
	}
	var rs []kernel.Reaction
	for _, k := range slices.Sorted(maps.Keys(bySlot)) {
		rs = append(rs, bySlot[k])
	}
	return s, rs
}

var t0 = time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC) // пятница

const item = "ENT01:FL-012"

type journal struct {
	n    int
	recs []kernel.Record
}

func (j *journal) add(t catalog.Type, at time.Time, data any) kernel.Record {
	j.n++
	b, _ := json.Marshal(data)
	info, _ := catalog.Lookup(t)
	r := kernel.Record{Seq: int64(j.n), EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", j.n), Type: t, Kind: info.Kind,
		ItemID: item, Stream: "item:" + item, OccurredAt: at, Data: b}
	j.recs = append(j.recs, r)
	return r
}

func of(rs []kernel.Reaction, t catalog.Type) []kernel.Reaction {
	var out []kernel.Reaction
	for _, r := range rs {
		if r.Type == t {
			out = append(out, r)
		}
	}
	return out
}

func data[T any](t *testing.T, r kernel.Reaction) T {
	t.Helper()
	b, err := json.Marshal(r.Data)
	if err != nil {
		t.Fatal(err)
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func dueSets(t *testing.T, rs []kernel.Reaction) map[string]notif.DueSetData {
	out := map[string]notif.DueSetData{}
	for _, r := range of(rs, catalog.ObligationDueSet) {
		d := data[notif.DueSetData](t, r)
		out[d.Basis] = d
	}
	return out
}

// isolated — изделие на сварке изолировано со сроком решения (FR-55).
func isolated(j *journal) time.Time {
	j.add(catalog.OperationRunStarted, t0.Add(time.Minute), map[string]any{"operation_run_id": "RUN-1", "operation_code": "welding",
		"step_key": "welding.weld", "operator_id": "W21", "workplace_id": "WP-W1"})
	due := t0.Add(3 * time.Hour)
	j.add(catalog.DecisionItemIsolated, t0.Add(2*time.Minute), map[string]any{"decision_due_at": notif.FormatTime(due), "reason": map[string]any{"text": "прожог"}})
	return due
}

// FR-55, FR-57, AD-4: изоляция ставит сроки решения и перемещения (эмитит
// только notifications по состоянию nonconformity), «наступил срок» даёт
// эскалацию с ценой задержки и тревогу, следующий уровень лестницы взводится
// сам; приёмка в изоляторе снимает срок перемещения и задачу.
func TestIsolationDeadlinesAndEscalation(t *testing.T) {
	var j journal
	due := isolated(&j)
	_, rs := fold(j.recs)
	ds := dueSets(t, rs)
	iso, move := ds[notif.BasisIsolation], ds[notif.BasisIsolationMove]
	if iso.DueAt != notif.FormatTime(due) || iso.Kind != "nc_disposition" || iso.OwnerRoleID != notif.RoleTechnologist || iso.Level != 1 || iso.StepKey != "welding.weld" {
		t.Fatalf("срок решения: %+v", iso)
	}
	if move.DueAt != notif.FormatTime(t0.Add(2*time.Minute+2*time.Hour)) || move.Kind != "isolation_move" || move.OwnerRoleID != notif.RoleForeman {
		t.Fatalf("срок перемещения: %+v", move)
	}
	tasks := of(rs, catalog.TaskTaskCreated)
	if len(tasks) != 2 {
		t.Fatalf("задачи: %+v", tasks)
	}
	for _, r := range tasks {
		d := data[notif.TaskData](t, r)
		if d.Kind == "isolate_move" && (d.AssigneeRoleID != notif.RoleForeman || d.LocationID != "WP-W1") {
			t.Fatalf("задача перемещения — тому, у кого изделие: %+v", d)
		}
	}
	if len(of(rs, catalog.ObligationEscalationRaised)) != 0 || len(of(rs, catalog.TaskNotificationSent)) != 0 {
		t.Fatal("до срока — без эскалации и тревоги")
	}

	// Планировщик: «наступил срок» решения (occurred_at = срок, id = UUIDv5(obligation_id, due_at)).
	j.add(catalog.ObligationDueReached, due, notif.DueReachedData{ObligationID: iso.ObligationID, DueAt: iso.DueAt})
	_, rs = fold(j.recs)
	esc := of(rs, catalog.ObligationEscalationRaised)
	if len(esc) != 1 {
		t.Fatalf("эскалация: %+v", esc)
	}
	e := data[notif.EscalationData](t, esc[0])
	if e.Level != 1 || e.EscalateToRoleID != notif.RoleHeadOfQC || e.BlockedItems == nil || *e.BlockedItems != 1 || e.BlockedOperations == nil || *e.BlockedOperations != 1 {
		t.Fatalf("эскалация с ценой задержки: %+v", e)
	}
	if !esc[0].OccurredAt.Equal(due) {
		t.Fatalf("время эскалации — из записи журнала (AD-4): %s", esc[0].OccurredAt)
	}
	alarms := of(rs, catalog.TaskNotificationSent)
	if len(alarms) != 1 || data[notif.NotificationData](t, alarms[0]).Severity != "alarm" || data[notif.NotificationData](t, alarms[0]).RecipientRoleID != notif.RoleTechnologist {
		t.Fatalf("тревога владельцу: %+v", alarms)
	}
	iso2 := dueSets(t, rs)[notif.BasisIsolation]
	if iso2.Level != 2 || iso2.DueAt != notif.FormatTime(due.Add(8*time.Hour)) || iso2.FirstDueAt != iso.DueAt {
		t.Fatalf("следующий уровень лестницы: %+v", iso2)
	}

	// Повтор «наступил срок» с тем же сроком ничего не меняет.
	j.add(catalog.ObligationDueReached, due, notif.DueReachedData{ObligationID: iso.ObligationID, DueAt: iso.DueAt})
	_, rs2 := fold(j.recs)
	if len(of(rs2, catalog.ObligationEscalationRaised)) != 1 || dueSets(t, rs2)[notif.BasisIsolation].Level != 2 {
		t.Fatal("повтор «наступил срок» дал вторую эскалацию")
	}

	// Приёмка в изоляторе: срок перемещения снят, задача снята, технологу — информация.
	j.add(catalog.OperationMovementReceived, due.Add(time.Minute), map[string]any{"to_location_id": "ISO-1", "destination_kind": "isolator",
		"inspection_on_receipt": "not_inspected", "received_by": "FOR-WC"})
	_, rs = fold(j.recs)
	if _, still := dueSets(t, rs)[notif.BasisIsolationMove]; still {
		t.Fatal("срок перемещения не снят")
	}
	cleared := of(rs, catalog.ObligationDueCleared)
	if len(cleared) != 1 || data[notif.DueClearedData](t, cleared[0]).ObligationID != move.ObligationID {
		t.Fatalf("срок снят: %+v", cleared)
	}
	if w := of(rs, catalog.TaskTaskWithdrawn); len(w) != 1 {
		t.Fatalf("задача перемещения снята: %+v", w)
	}
	info := 0
	for _, r := range of(rs, catalog.TaskNotificationSent) {
		if d := data[notif.NotificationData](t, r); d.Severity == "info" && d.RecipientRoleID == notif.RoleTechnologist {
			info++
		}
	}
	if info != 1 {
		t.Fatal("информация технологу: изделие в изоляторе")
	}
}

// AD-4: решение по несоответствию снимает срок решения и отзывает тревогу.
func TestDecisionClearsDeadlineAndAlarm(t *testing.T) {
	var j journal
	due := isolated(&j)
	_, rs := fold(j.recs)
	iso := dueSets(t, rs)[notif.BasisIsolation]
	j.add(catalog.ObligationDueReached, due, notif.DueReachedData{ObligationID: iso.ObligationID, DueAt: iso.DueAt})
	j.add(catalog.DecisionNonconformityRegistered, due.Add(time.Minute), map[string]any{"nc_id": "NC-1", "violation_window_event_id": "w-1",
		"step_key": "welding.weld", "operation_run_id": "RUN-1"})
	j.add(catalog.DecisionDispositionSet, due.Add(2*time.Minute), map[string]any{"nc_id": "NC-1", "disposition": "scrap",
		"reason": map[string]any{"text": "брак"}})
	_, rs = fold(j.recs)
	if _, still := dueSets(t, rs)[notif.BasisIsolation]; still {
		t.Fatal("срок решения не снят решением")
	}
	if len(of(rs, catalog.TaskNotificationWithdrawn)) != 1 {
		t.Fatal("тревога не отозвана")
	}
	if len(of(rs, catalog.ObligationEscalationRaised)) != 1 {
		t.Fatal("эскалация — история, остаётся")
	}
}

// AD-4: детерминизм — одна и та же свёртка даёт те же реакции, id «наступил
// срок» зависит только от обязательства и срока.
func TestDeterminism(t *testing.T) {
	var j journal
	isolated(&j)
	_, a := fold(j.recs)
	_, b := fold(j.recs)
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Fatal("реакции различаются между свёртками")
	}
	id := notif.ObligationID(item, "isolation/x")
	first, again, later := notif.ReachedID(id, t0), notif.ReachedID(id, t0), notif.ReachedID(id, t0.Add(time.Hour))
	if first != again || first == later {
		t.Fatal("id «наступил срок»")
	}
}

// FR-55: 3 рабочих дня по календарю пн–пт: пятница + 3 = среда.
func TestCalendar(t *testing.T) {
	c := notif.Calendar{}
	if got := c.AddWorkingDays(t0, 3); got.Weekday() != time.Wednesday || got.Hour() != 8 {
		t.Fatalf("пятница + 3 рабочих дня: %s", got)
	}
	c.NonWorkingDates = []string{"2026-09-28"}
	if got := c.AddWorkingDays(t0, 3); got.Weekday() != time.Thursday {
		t.Fatalf("с праздником в понедельник: %s", got)
	}
}

// FR-8: цена задержки — изделия и операции, ждущие того же решения.
func TestCost(t *testing.T) {
	rows := []notif.ObligationRecord{
		{ItemID: "A", StepKey: "welding.weld", WaitsOn: "incident:RS-1", State: notif.ObligationOpen},
		{ItemID: "B", StepKey: "welding.weld", WaitsOn: "incident:RS-1", State: notif.ObligationOpen},
		{ItemID: "B", StepKey: "welding.weld", WaitsOn: "incident:RS-1", State: notif.ObligationOpen},
		{ItemID: "C", StepKey: "machining.turn", WaitsOn: "incident:RS-1", State: notif.ObligationOpen},
		{ItemID: "D", StepKey: "machining.turn", WaitsOn: "incident:RS-1", State: notif.ObligationCleared},
		{ItemID: "E", StepKey: "assembly.fit", WaitsOn: "item:E", State: notif.ObligationOpen},
	}
	if items, ops := notif.Cost(rows, "incident:RS-1"); items != 3 || ops != 2 {
		t.Fatalf("стоят %d изделий, %d операции", items, ops)
	}
}

// FR-62, FR-57: изделие в области риска на блоке — срок решения по области и
// адресная задача тому, у кого изделие физически; все сроки изделия ждут
// решения по инциденту (цена задержки сводится по инциденту).
func TestIncidentScopeAddressedTask(t *testing.T) {
	var j journal
	j.add(catalog.OperationMovementReceived, t0, map[string]any{"to_location_id": "WC-BUF-2", "destination_kind": "station",
		"inspection_on_receipt": "no_damage", "received_by": "FOR-WC", "step_key": "welding.weld"})
	j.add(catalog.IncidentMembershipChanged, t0.Add(time.Minute), map[string]any{"incident_id": "RS-1", "scope_version": 1, "status": "suspect", "action": "block"})
	_, rs := fold(j.recs)
	d := dueSets(t, rs)[notif.BasisIncidentScope]
	if d.WaitsOn != "incident:RS-1" || d.Kind != "containment_review" {
		t.Fatalf("срок по области: %+v", d)
	}
	var task notif.TaskData
	for _, r := range of(rs, catalog.TaskTaskCreated) {
		task = data[notif.TaskData](t, r)
	}
	if task.Kind != "physical_move" || task.LocationID != "WC-BUF-2" || task.AssigneeRoleID != notif.RoleForeman {
		t.Fatalf("адресная задача: %+v", task)
	}
}

// Задача по запросу измерения в потоке инцидента (эпик 22) — один и тот же
// id при повторе.
func TestObjectReact(t *testing.T) {
	b, _ := json.Marshal(map[string]any{"incident_id": "RS-1", "hypothesis_id": "H-1", "what": "Твёрдость шва", "assignee_id": "NDT-61"})
	r := kernel.Record{Seq: 5, EventID: "00000000-0000-7000-8000-000000000005", Type: catalog.IncidentMeasurementRequested,
		Kind: catalog.KindDecision, Stream: "incident:RS-1", OccurredAt: t0, Data: b}
	a, c := notif.ObjectReact(r), notif.ObjectReact(r)
	if len(a) != 1 || a[0].ID(1) != c[0].ID(1) || a[0].Slot.Subject != "incident:RS-1" {
		t.Fatalf("задача измерения: %+v", a)
	}
	d := data[notif.TaskData](t, a[0])
	if d.AssigneePersonID != "NDT-61" || d.SubjectRef != "incident:RS-1" {
		t.Fatalf("адресат: %+v", d)
	}
	r.Stream, r.ItemID = "item:"+item, item
	if len(notif.ObjectReact(r)) != 0 {
		t.Fatal("записи изделия — не для ObjectReact")
	}
}
