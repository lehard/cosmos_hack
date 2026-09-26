package notifications

import (
	"context"
	"testing"

	app "ant/internal/application/notifications"
	"ant/internal/application/platform"
	_ "ant/internal/infrastructure/fixtures/world" // встроенный мир заготовок (loader.Builtin)
)

func as(person, role string, roles ...string) context.Context {
	return platform.WithPrincipal(context.Background(), platform.Principal{PersonID: person, Role: role, Roles: append([]string{role}, roles...)})
}

// TestTasksAddressedAndAcknowledged — адресность задач на сервере и отметка
// задачи в сессии: исполнитель не видит задач контролёра, отмеченная задача
// уходит из открытых, сводка шапки пересчитывается.
func TestTasksAddressedAndAcknowledged(t *testing.T) {
	a := New()
	w21 := as("W21", "performer", "staff", "employee")
	l, err := a.Tasks(w21, app.TaskFilter{}, platform.Moment{}, platform.Page{})
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range l.Items {
		if x.AssigneeID != nil && *x.AssigneeID != "W21" {
			t.Fatalf("исполнитель видит чужую задачу: %+v", x)
		}
	}
	ins := as("INS-01", "quality_inspector", "staff", "employee")
	open, err := a.Tasks(ins, app.TaskFilter{State: "open"}, platform.Moment{}, platform.Page{})
	if err != nil || len(open.Items) == 0 {
		t.Fatalf("открытые задачи контролёра: %+v %v", open, err)
	}
	for _, x := range open.Items {
		if x.AssigneeID != nil && *x.AssigneeID != "INS-01" {
			t.Fatalf("контролёр видит персональную задачу коллеги: %+v", x)
		}
	}
	before, err := a.Summary(ins, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	task := open.Items[0]
	rc, err := a.AcknowledgeTask(ins, task.TaskID, app.AcknowledgeTask{CommandHeader: platform.CommandHeader{CommandID: "0192e4a0-0000-7000-8000-00000000a001"}, Outcome: "done"})
	if err != nil {
		t.Fatal(err)
	}
	rc2, err := a.AcknowledgeTask(ins, open.Items[len(open.Items)-1].TaskID, app.AcknowledgeTask{CommandHeader: platform.CommandHeader{CommandID: "0192e4a0-0000-7000-8000-00000000a002"}, Outcome: "declined", Note: "не моё"})
	if err != nil || rc2.Seq == rc.Seq {
		t.Fatalf("вторая отметка: %+v %v (номер %d)", rc2, err, rc.Seq)
	}
	after, err := a.Tasks(ins, app.TaskFilter{State: "open"}, platform.Moment{}, platform.Page{})
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range after.Items {
		if x.TaskID == task.TaskID {
			t.Fatalf("отмеченная задача осталась открытой: %+v", x)
		}
	}
	done, _ := a.Tasks(ins, app.TaskFilter{State: "done"}, platform.Moment{}, platform.Page{})
	found := false
	for _, x := range done.Items {
		found = found || x.TaskID == task.TaskID
	}
	if !found {
		t.Fatal("отмеченной задачи нет среди выполненных")
	}
	s, err := a.Summary(ins, platform.Moment{})
	if err != nil || s.Unread >= before.Unread {
		t.Fatalf("сводка не пересчитана: было %+v, стало %+v", before, s)
	}
}
