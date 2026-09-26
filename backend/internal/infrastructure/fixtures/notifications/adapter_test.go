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

// TestTasksScopedBySession — область сеанса на заготовках (как live,
// app.InScope): начальник сварочного цеха (ent01/b1/wc) не видит задач
// мастера сборочно-испытательного цеха (WS-AC); та же роль с областью ent01 —
// видит.
func TestTasksScopedBySession(t *testing.T) {
	a := New()
	ids := func(scope string) map[string]string {
		ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "HWS-WC", Role: "head_of_workshop",
			Roles: []string{"head_of_workshop", "site_foreman", "staff", "employee"}, Scope: scope})
		l, err := a.Tasks(ctx, app.TaskFilter{}, platform.Moment{}, platform.Page{})
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]string{}
		for _, x := range l.Items {
			loc := ""
			if x.LocationID != nil {
				loc = *x.LocationID
			}
			out[x.TaskID] = loc
		}
		return out
	}
	wc, ent := ids("ent01/b1/wc"), ids("ent01")
	for id, loc := range wc {
		if loc != "" && loc != "WS-WC" {
			t.Errorf("область ent01/b1/wc видит задачу %s на %s", id, loc)
		}
	}
	ac := 0
	for id, loc := range ent {
		if loc == "WS-AC" {
			ac++
			if _, ok := wc[id]; ok {
				t.Errorf("задача WS-AC %s в области сварочного цеха", id)
			}
		}
	}
	if ac == 0 || len(ent) != len(wc)+ac {
		t.Fatalf("область ent01: %d задач (WS-AC — %d), сварочный цех: %d", len(ent), ac, len(wc))
	}
}
