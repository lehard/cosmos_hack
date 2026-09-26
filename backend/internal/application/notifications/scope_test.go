package notifications_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	notifapp "ant/internal/application/notifications"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	notif "ant/internal/domain/notifications"
	dp "ant/internal/domain/process"
)

// Область задач и закрытие задач процесса (решения пользователя, показ):
// список задач сужается по области сеанса — мастер сварочного цеха не видит
// «Отправить …» механического цеха; задача процесса отметкой не закрывается.

// places — справочник мест (normative/reference/flange/locations.yaml, выборка).
type places map[string]string

func (p places) ScopeOf(id string) (string, bool) { s, ok := p[id]; return s, ok }

var flangePlaces = places{"WS-MC": "ent01/b1/mc", "WS-WC": "ent01/b1/wc", "WP-WELD-1": "ent01/b1/wc/weld/wp1"}

// memTasks — проекция задач в памяти.
type memTasks map[string]json.RawMessage

func (m memTasks) Get(_ context.Context, name, key string) (json.RawMessage, bool, error) {
	if name != notifapp.ProjectionTask {
		return nil, false, nil
	}
	v, ok := m[key]
	return v, ok, nil
}

func (m memTasks) All(_ context.Context, name string) (map[string]json.RawMessage, error) {
	if name != notifapp.ProjectionTask {
		return map[string]json.RawMessage{}, nil
	}
	return m, nil
}

func (memTasks) OpenObligations(context.Context) ([]notif.ObligationRecord, error) { return nil, nil }

type noDecisions struct{ called bool }

func (d *noDecisions) Write(context.Context, notifapp.Decision) (platform.Receipt, error) {
	d.called = true
	return platform.Receipt{Seq: 1}, nil
}

func processTasks(t *testing.T) memTasks {
	t.Helper()
	at := time.Now().UTC().Add(-time.Hour)
	m := memTasks{}
	for _, r := range []notif.TaskRecord{
		{TaskID: "T-RCV", Kind: notif.KindProcessStep, Title: "Принять в цех Ф-002", Role: "site_foreman", LocationID: "WS-WC",
			Subject: "item:DM:F-002", ItemID: "DM:F-002", Operation: dp.OpMovementReceive, StepKey: "welding.receive", State: notif.TaskOpen, CreatedAt: at},
		{TaskID: "T-SEND", Kind: notif.KindProcessStep, Title: "Отправить DM:F-003: Отправка в сварочный цех", Role: "site_foreman", LocationID: "WS-MC",
			Subject: "item:DM:F-003", ItemID: "DM:F-003", Operation: dp.OpMovementSend, StepKey: "machining.send_to_welding", State: notif.TaskOpen, CreatedAt: at},
		{TaskID: "T-MOVE", Kind: "isolate_move", Title: "Переместить в изолятор", Role: "site_foreman", LocationID: "WP-WELD-1",
			Subject: "item:DM:F-004", State: notif.TaskOpen, CreatedAt: at},
		{TaskID: "T-ANY", Kind: "other", Title: "Общая задача мастеров", Role: "site_foreman", Subject: "item:DM:F-005", State: notif.TaskOpen, CreatedAt: at},
	} {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		m[r.TaskID] = b
	}
	return m
}

func taskIDs(t *testing.T, s *notifapp.Service, p platform.Principal) map[string]bool {
	t.Helper()
	l, err := s.Tasks(platform.WithPrincipal(context.Background(), p), notifapp.TaskFilter{}, platform.Moment{}, platform.Page{})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, x := range l.Items {
		out[x.TaskID] = true
	}
	return out
}

func TestTasksScopedBySession(t *testing.T) {
	s := notifapp.NewLive(notifapp.Config{Projections: processTasks(t), Places: flangePlaces})
	forWC := platform.Principal{PersonID: "FOR-WC", Role: "site_foreman", Scope: "ent01/b1/wc"}
	got := taskIDs(t, s, forWC)
	if !got["T-RCV"] || got["T-SEND"] || !got["T-MOVE"] || !got["T-ANY"] {
		t.Fatalf("мастер сварочного цеха: %v — нужны «Принять в цех» (WS-WC), пост цеха и общая, без «Отправить» (WS-MC)", got)
	}
	forMC := platform.Principal{PersonID: "FOR-MC", Role: "site_foreman", Scope: "ent01/b1/mc"}
	if got := taskIDs(t, s, forMC); got["T-RCV"] || !got["T-SEND"] || got["T-MOVE"] || !got["T-ANY"] {
		t.Fatalf("мастер механического цеха: %v", got)
	}
	ent := platform.Principal{PersonID: "HWS-01", Role: "head_of_workshop", Roles: []string{"head_of_workshop", "site_foreman"}, Scope: "ent01"}
	if got := taskIDs(t, s, ent); !got["T-RCV"] || !got["T-SEND"] || !got["T-MOVE"] || !got["T-ANY"] {
		t.Fatalf("область ent01 видит всё: %v", got)
	}
	sum, err := s.Summary(platform.WithPrincipal(context.Background(), forWC), platform.Moment{})
	if err != nil || sum.ByKind == nil || sum.ByKind.Task != 3 {
		t.Fatalf("сводка мастера сварочного цеха — три задачи в области: %+v %v", sum.ByKind, err)
	}
}

func TestProcessStepNotAcknowledged(t *testing.T) {
	d := &noDecisions{}
	s := notifapp.NewLive(notifapp.Config{Projections: processTasks(t), Places: flangePlaces, Decisions: d})
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "FOR-WC", Role: "site_foreman", Scope: "ent01/b1/wc"})
	for _, outcome := range []string{"done", "accepted"} {
		_, err := s.AcknowledgeTask(ctx, "T-RCV", notifapp.AcknowledgeTask{Outcome: outcome})
		var r *kernel.Refusal
		if !errors.As(err, &r) || r.Code != errcodes.ProcessTaskClosedByAction || r.Detail != "Задача процесса закрывается действием: «Принять в цех»" {
			t.Fatalf("отметка %s задачи процесса: %v", outcome, err)
		}
	}
	if d.called {
		t.Fatal("отметка задачи процесса записана в журнал")
	}
	if _, err := s.AcknowledgeTask(ctx, "T-MOVE", notifapp.AcknowledgeTask{Outcome: "done"}); err != nil || !d.called {
		t.Fatalf("обычная задача отмечается: %v", err)
	}
}

// TestProcessStepStaysOpen — проекция задач: старая отметка задачи процесса её
// не закрывает; снимает только действие шага (task.task.withdrawn).
func TestProcessStepStaysOpen(t *testing.T) {
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	rec := func(seq int64, tp catalog.Type, data any) kernel.Record {
		b, _ := json.Marshal(data)
		return kernel.Record{Seq: seq, Type: tp, OccurredAt: at.Add(time.Duration(seq) * time.Minute), Data: b}
	}
	v := notif.StepTask(notif.TaskRecord{}, rec(1, catalog.TaskTaskCreated, notif.TaskData{TaskID: "T-RCV", Kind: notif.KindProcessStep,
		AssigneeRoleID: "site_foreman", LocationID: "WS-WC", SubjectRef: "item:DM:F-002", OperationID: dp.OpMovementReceive}))
	v = notif.StepTask(v, rec(2, catalog.TaskTaskAcknowledged, notif.AckData{TaskID: "T-RCV", Outcome: "accepted"}))
	if v.State != notif.TaskOpen {
		t.Fatalf("отметка закрыла задачу процесса: %s", v.State)
	}
	v = notif.StepTask(v, rec(3, catalog.TaskTaskWithdrawn, notif.TaskWithdrawnData{TaskID: "T-RCV"}))
	if v.State != notif.TaskWithdrawn {
		t.Fatalf("действие шага не сняло задачу: %s", v.State)
	}
	// Обычная задача отметкой закрывается, как раньше.
	o := notif.StepTask(notif.TaskRecord{}, rec(4, catalog.TaskTaskCreated, notif.TaskData{TaskID: "T-MOVE", Kind: "isolate_move", SubjectRef: "item:DM:F-004"}))
	if o = notif.StepTask(o, rec(5, catalog.TaskTaskAcknowledged, notif.AckData{TaskID: "T-MOVE", Outcome: "done"})); o.State != "done" {
		t.Fatalf("обычная задача: %s", o.State)
	}
}
