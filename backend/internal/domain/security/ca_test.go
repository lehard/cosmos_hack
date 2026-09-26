package security

import (
	"encoding/json"
	"strings"
	"testing"

	"ant/internal/contracts/catalog"
)

const mainID = "0190a8c4-1111-7000-8000-000000000001"

// AD-28: не меньше девяти действий, все — критические типы каталога, у
// каждого есть группа CA.
func TestActionsMVP(t *testing.T) {
	if len(Actions) < 9 {
		t.Fatalf("действий MVP %d < 9", len(Actions))
	}
	operator := 0
	for _, a := range Actions {
		info, ok := catalog.Lookup(a.Type)
		if !ok || !info.Critical || info.CAGroup == "" {
			t.Errorf("%s: не критический тип каталога", a.Type)
		}
		if strings.HasPrefix(string(a.Type), "operator.") {
			operator++
		}
	}
	if operator < 3 {
		t.Errorf("действий исполнителя %d < 3", operator)
	}
}

func TestBuildCA(t *testing.T) {
	m := Main{EventID: mainID, EventType: catalog.DecisionContainmentReleased, Stream: "item:F-023",
		Commit: "streebog256:" + strings.Repeat("ab", 32), Signers: []string{"KO-01@1"},
		Data: json.RawMessage(`{"released_event_ids":[],"reason":{"text":"x"}}`)}
	r, err := BuildCA(m, 7, Command{Basis: []string{"0190a8c4-1111-7000-8000-000000000002"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.CANo != 7 || r.MainCommit != m.Commit || r.ActorID != "KO-01" || r.Before != "заблокировано" || r.After != "разрешено" ||
		r.CAGroup != "product_decision" || len(r.BasisEventIDs) != 1 || r.ObjectRef != "item:F-023" {
		t.Fatalf("запись CA: %+v", r)
	}
	set := m
	set.EventType = catalog.DecisionContainmentSet
	set.Data = json.RawMessage(`{"level":"item_hold","reason":{"text":"x"}}`)
	if r, _ := BuildCA(set, 8, Command{}); r.After != "заблокировано: Блок изделия" {
		t.Fatalf("уровень блока: %q", r.After)
	}
	if _, err := BuildCA(Main{EventID: mainID, EventType: catalog.InspectionResultRecorded}, 1, Command{}); err == nil {
		t.Fatal("некритический тип получил CA")
	}
	// Отмена — только новой записью с причиной.
	if _, err := BuildCA(m, 9, Command{Cancels: "CA-7"}); err == nil {
		t.Fatal("отмена без причины")
	}
	c, err := BuildCA(m, 9, Command{Cancels: "CA-7", CancelReason: "ошибочный выпуск"})
	if err != nil || c.Cancels != "CA-7" || c.CancelReason.Text != "ошибочный выпуск" {
		t.Fatalf("отмена: %+v %v", c, err)
	}
	fix := m
	fix.Corrects = "0190a8c4-1111-7000-8000-000000000003"
	if r, _ := BuildCA(fix, 10, Command{}); !strings.HasPrefix(r.After, CorrectionName) || len(r.BasisEventIDs) != 1 {
		t.Fatalf("исправление: %+v", r)
	}
	if EventID(mainID) != EventID(mainID) || EventID(mainID) == EventID(fix.Corrects) {
		t.Fatal("event_id записи CA не детерминирован")
	}
}
