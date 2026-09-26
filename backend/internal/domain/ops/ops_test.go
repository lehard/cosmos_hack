package ops

import (
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)

// AD-45: изделие остановлено, пока последний сбой не снят повтором; повтор
// снимает сбой, на который ссылается, и более ранние.
func TestStoppedItems(t *testing.T) {
	fA1 := Failure{EventID: "a1", Seq: 10, ItemID: "ENT01:A", Consumer: "engine.worker", FailedSeq: 9, Error: "паника", At: t0}
	fB := Failure{EventID: "b1", Seq: 12, ItemID: "ENT01:B", Consumer: "projector:x", FailedSeq: 11, At: t0}
	fA2 := Failure{EventID: "a2", Seq: 20, ItemID: "ENT01:A", Consumer: "engine.worker", FailedSeq: 19, At: t0}
	got := StoppedItems([]Failure{fA2, fB, fA1}, nil)
	if len(got) != 2 || got[0].Failure.EventID != "b1" || got[1].Failure.EventID != "a2" {
		t.Fatalf("без повторов: %+v", got)
	}
	// Повтор по первому сбою не снимает второй, более поздний.
	got = StoppedItems([]Failure{fA1, fB, fA2}, []Retry{{EventID: "r1", Seq: 15, ItemID: "ENT01:A", FailureEventID: "a1"}})
	if len(got) != 2 || got[1].Failure.EventID != "a2" || got[1].Retries != 1 {
		t.Fatalf("повтор первого сбоя: %+v", got)
	}
	got = StoppedItems([]Failure{fA1, fB, fA2}, []Retry{
		{EventID: "r1", Seq: 15, ItemID: "ENT01:A", FailureEventID: "a1"},
		{EventID: "r2", Seq: 25, ItemID: "ENT01:A", FailureEventID: "a2"},
	})
	if len(got) != 1 || got[0].Failure.ItemID != "ENT01:B" {
		t.Fatalf("после повтора: %+v", got)
	}
	// Сбой после повтора снова останавливает изделие.
	fA3 := Failure{EventID: "a3", Seq: 30, ItemID: "ENT01:A", At: t0}
	got = StoppedItems([]Failure{fA1, fA2, fA3}, []Retry{{EventID: "r2", Seq: 25, ItemID: "ENT01:A", FailureEventID: "a2"}})
	if len(got) != 1 || got[0].Failure.EventID != "a3" || got[0].Retries != 1 {
		t.Fatalf("новый сбой: %+v", got)
	}
}

func TestFailureIdentity(t *testing.T) {
	a, b := FailureEventID("engine.worker", "ENT01:A", 5), FailureEventID("engine.worker", "ENT01:A", 5)
	if a != b || a == FailureEventID("engine.worker", "ENT01:A", 6) || RetryEventID(a) == a {
		t.Fatal("идентичность записей сбоя и повтора")
	}
	long := strings.Repeat("ж", 3000) // 6000 байт
	m := FailureMessage(long)
	if len(m) > MaxErrorLen || len(m)%2 != 0 {
		t.Fatalf("обрезка по символу: %d", len(m))
	}
}

// AD-18: журнал хранит переходы канала, не каждую сверку.
func TestShouldRecord(t *testing.T) {
	deg := &IntegrationRecord{State: IntegrationDegraded}
	ok := &IntegrationRecord{State: IntegrationOK}
	cases := []struct {
		last  *IntegrationRecord
		state string
		want  bool
	}{
		{nil, IntegrationOK, false}, {nil, IntegrationDegraded, true}, {nil, IntegrationDisabled, false},
		{deg, IntegrationDegraded, false}, {deg, IntegrationOK, true}, {ok, IntegrationDegraded, true}, {ok, IntegrationOK, false},
	}
	for _, c := range cases {
		if got := ShouldRecord(c.last, c.state); got != c.want {
			t.Errorf("%+v → %s: %v", c.last, c.state, got)
		}
	}
	if DegradedEventID("onec", "degraded", 0) == DegradedEventID("onec", "degraded", 7) {
		t.Fatal("id перехода зависит от предыдущей записи")
	}
}

func TestIntegrations(t *testing.T) {
	at := t0.Add(-time.Hour)
	got := Integrations([]string{"onec", "skud"},
		[]Channel{{System: "onec", State: IntegrationDegraded, Detail: "метаданные", CheckedAt: t0}},
		[]IntegrationRecord{{Seq: 3, System: "onec", State: IntegrationDegraded, At: at}, {Seq: 1, System: "mes", State: IntegrationDegraded, At: at}})
	want := map[string]string{"onec": IntegrationDegraded, "galaktika": IntegrationDisabled, "mes": IntegrationDisabled, "skud": IntegrationOK}
	if len(got) != 4 || got[0].System != "onec" || got[3].System != "skud" {
		t.Fatalf("%+v", got)
	}
	for _, g := range got {
		if want[g.System] != g.State {
			t.Errorf("%s: %s", g.System, g.State)
		}
	}
	if got[0].Since == nil || !got[0].Since.Equal(at) {
		t.Fatalf("degraded с момента записи: %+v", got[0].Since)
	}
}

func TestComponents(t *testing.T) {
	live, dead := t0.Add(time.Minute), t0.Add(-time.Minute)
	cs := Components([]Lease{
		{Name: "worker:h:1/worker", ExpiresAt: live}, {Name: "partition:0", ExpiresAt: live}, {Name: "partition:1", ExpiresAt: dead},
		{Name: "crossitem", Holder: "h:1", Epoch: 2, ExpiresAt: live}, {Name: "projector", Holder: "h:2", ExpiresAt: dead},
	}, t0, 2)
	st := map[string]Component{}
	for _, c := range cs {
		st[c.Name] = c
	}
	if st["ant/worker"].State != StateDegraded || st["ant/crossitem"].State != StateOK || st["ant/crossitem"].Leader != "h:1" ||
		st["ant/projector"].State != StateDown || st["ant/outbox"].State != StateUnknown {
		t.Fatalf("%+v", cs)
	}
	if c := Components(nil, t0, 2); c[0].State != StateUnknown {
		t.Fatalf("воркеры не запускались: %+v", c[0])
	}
}

// FR-109: самопроверка находит отсутствие генезиса и неприменённые миграции.
func TestSelfCheckFindings(t *testing.T) {
	if f := GenesisFindings(0, nil, false); f != nil {
		t.Fatalf("пустой журнал без обязательного генезиса: %+v", f)
	}
	if f := GenesisFindings(12, nil, false); len(f) != 1 || f[0].Severity != Warning {
		t.Fatalf("нет генезиса — предупреждение: %+v", f)
	}
	if f := GenesisFindings(0, nil, true); len(f) != 1 || f[0].Severity != Critical || !HasCritical(f) {
		t.Fatalf("нет обязательного генезиса — критично: %+v", f)
	}
	if f := GenesisFindings(10, []int64{1, 9}, false); len(f) != 1 || f[0].Severity != Critical {
		t.Fatalf("два генезиса: %+v", f)
	}
	if f := GenesisFindings(10, []int64{1}, true); f != nil {
		t.Fatalf("генезис есть: %+v", f)
	}
	ms := MigrationFindings([]Migration{
		{Module: "journal", Current: 3, Target: 3},
		{Module: "ingest", Current: 0, Target: 5, Pending: 2},
		{Module: "erp", Err: "нет прав"},
	})
	if len(ms) != 2 || ms[0].Severity != Critical || !strings.Contains(ms[0].Text, "ingest") || ms[1].Severity != Warning {
		t.Fatalf("%+v", ms)
	}
	if s := Summary(ms); !strings.HasPrefix(s, MsgFailed) || !strings.Contains(s, "ingest") {
		t.Fatal(s)
	}
	if s := Summary(ms[1:]); s != MsgOK {
		t.Fatal(s)
	}
	rf := DBRoleFindings([]string{"ant_verifier"}, []Privilege{{Role: "ant_app", Privilege: "UPDATE", Object: "journal.entries", Granted: true}})
	if len(rf) != 2 || !HasCritical(rf) {
		t.Fatalf("%+v", rf)
	}
	pr := RoleFindings([]ProcessRole{{Name: "init", Pending: true}, {Name: "projector", Leader: true}, {Name: "crossitem", Leader: true, Held: true}})
	if len(pr) != 2 || HasCritical(pr) {
		t.Fatalf("%+v", pr)
	}
}
