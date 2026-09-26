package simulation

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	app "ant/internal/application/simulation"
	"ant/internal/application/simulation/simfake"
)

// desk — столы ролей на фейках: решения людей — записи (операция, объект)
// с номерами журнала; demo-signer (Act) — через simfake.Actor, с учётом вызовов.
type desk struct {
	a    *simfake.Actor
	mu   sync.Mutex
	recs []deskRec
	acts []string // операции, которые подписал demo-signer
	live bool     // прогон в живой части: вызовы Act считаются отдельно
	auto []string // операции demo-signer в живой части
}

type deskRec struct {
	op, obj string
	seq     int64
}

func (d *desk) Act(ctx context.Context, persona, op string, params map[string]string, body map[string]any) (app.ActResult, error) {
	d.mu.Lock()
	d.acts = append(d.acts, op)
	if d.live {
		d.auto = append(d.auto, op)
	}
	d.mu.Unlock()
	return d.a.Act(ctx, persona, op, params, body)
}

func (d *desk) Decided(_ context.Context, _, op, object string, since int64) ([]int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []int64
	for _, r := range d.recs {
		if r.op == op && (object == "" || r.obj == object) && r.seq > since {
			out = append(out, r.seq)
		}
	}
	return out, nil
}

// pressed — человек уже нажал эту кнопку над этим объектом, и нажатие ещё
// не закрыло другую остановку.
func (d *desk) pressed(op, obj string, consumed []int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.ContainsFunc(d.recs, func(r deskRec) bool {
		return r.op == op && r.obj == obj && !slices.Contains(consumed, r.seq)
	})
}

// press — человек нажал кнопку на своём столе.
func (d *desk) press(op, obj string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.recs = append(d.recs, deskRec{op: op, obj: obj, seq: 1_000_000_000 + int64(len(d.recs))})
}

// TestShowRunManual — сценарий показа «Партия фланцев: сбой ИС-2» (Д-85):
// история проигрывается сразу, часы встают на Пн 08:00; дальше каждое
// решение людей живой партии — остановка до нажатия (demo-signer делает
// только вспомогательные шаги live.auto); решение, принятое раньше, чем
// прогон дошёл до шага, засчитывается; одна запись закрывает одну остановку.
func TestShowRunManual(t *testing.T) {
	ctx := context.Background()
	files := NewFiles(filepath.Join(repo, "scenarios"))
	ing := simfake.NewIngest()
	clock := &simfake.Clock{T: time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)}
	d := &desk{a: &simfake.Actor{In: ing}}
	store := NewMemoryRuns()
	svc := app.NewServiceWith(app.Deps{Definitions: files, Gateway: ing, Probe: ing, Actor: d, Recorder: &simfake.Recorder{},
		Store: store, Infra: clock, Profile: "demo"})
	started, err := svc.StartRun(ctx, "SHOW-IS2", app.StartRun{Mode: app.ModeInteractive})
	if err != nil {
		t.Fatal(err)
	}
	load := func() *app.RunState {
		st, _, _ := store.Load(ctx, started.RunID)
		return st
	}
	step := func(dt time.Duration) *app.RunState {
		clock.Advance(dt)
		if err := svc.Step(ctx, started.RunID); err != nil {
			t.Fatal(err)
		}
		return load()
	}
	// История — за несколько шагов раннера, без часов (реальное время почти не идёт);
	// пока она идёт, часы пульта — последний тик истории, а не ×скорость.
	st := load()
	for i := 0; i < 50 && !st.Live; i++ {
		st = step(10 * time.Second)
		if v, _ := svc.Run(ctx, started.RunID, platformMoment()); !st.Live && !v.ClockAt.Equal(st.LastTick) {
			t.Fatalf("часы во время истории: %s, последний тик %s", v.ClockAt, st.LastTick)
		}
	}
	if !st.Live || st.Clock.Speed != 60 {
		t.Fatalf("история не проиграна сразу: live=%v, скорость %d, курсор %+v", st.Live, st.Clock.Speed, st.Cursor)
	}
	p, err := svc.Plan(ctx, started.RunID, app.PlanQuery{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	if p.ClockAt.Hour() != 5 || p.ClockAt.Minute() != 0 { // Пн 08:00 +03:00
		t.Fatalf("часы в начале живой части: %s", p.ClockAt)
	}
	history := len(d.acts)
	if history == 0 {
		t.Fatal("историю подписывает demo-signer")
	}
	d.mu.Lock()
	d.live = true
	d.mu.Unlock()

	// Живая часть: ждём остановку — нажимаем — прогон идёт дальше.
	var stops []string
	early := false
	for i := 0; i < 3000 && st.State != app.StateComplete; i++ {
		st = step(2 * time.Second)
		if st.State != app.StateWaiting || st.Waiting == nil {
			continue
		}
		w := *st.Waiting
		if n := len(stops); n > 0 && stops[n-1] == w.Op+" "+w.Object {
			continue
		}
		stops = append(stops, w.Op+" "+w.Object)
		// пока человек не нажал — прогон стоит, часы стоят
		if pressed := d.pressed(w.Op, w.Object, st.Consumed); pressed {
			// нажато раньше, чем прогон дошёл до шага: засчитывается без ожидания
			continue
		}
		before := st.Cursor
		st = step(time.Minute)
		if st.State != app.StateWaiting || st.Cursor != before {
			t.Fatalf("остановка %s %q: прогон ушёл без решения человека: %s %+v → %+v, ждёт %+v", w.Op, w.Object, st.State, before, st.Cursor, st.Waiting)
		}
		if w.Op == "process.movement.receive" && len(stops) == 1 {
			// мастер принимает второй фланец раньше, чем прогон до него дошёл
			d.press(w.Op, w.Object)
			pv, _ := svc.Plan(ctx, started.RunID, app.PlanQuery{Limit: 200})
			for _, e := range pv.Items {
				if e.Stop && e.Operation == "process.movement.receive" && e.ObjectID != w.Object {
					d.press(e.Operation, e.ObjectID)
					early = true
					break
				}
			}
			continue
		}
		d.press(w.Op, w.Object)
	}
	if st.State != app.StateComplete {
		t.Fatalf("показ не дошёл до конца: %s, остановок %d, ждёт %+v", st.State, len(stops), st.Waiting)
	}
	if !early {
		t.Error("в плане нет второго приёма мастера")
	}
	// Пульт: по умолчанию только сценарии показа; остановок — все решения живой части.
	sl, err := svc.Scenarios(ctx, false)
	if err != nil || len(sl.Items) != 1 || sl.Items[0].ScenarioID != "SHOW-IS2" || sl.Items[0].Decisions != 23 {
		t.Errorf("пульт показа: %+v %v", sl.Items, err)
	}
	if all, _ := svc.Scenarios(ctx, true); len(all.Items) < 40 {
		t.Errorf("весь каталог: %d", len(all.Items))
	}
	// Остановок 23: 3 приёма + 2 допуска + 4 сварки × («Начать», «Выполнено») + 3 ЗТ-3 +
	// НС, изоляция, доп. проверка, изолятор, остановка ИС-2, причина, переделка; ранний
	// приём Ф-002 закрылся без ожидания, но остановкой в плане остался.
	if len(stops) != 23 {
		t.Errorf("остановок %d, ждали 23: %v", len(stops), stops)
	}
	for _, op := range d.auto {
		if !slices.Contains([]string{"access.operator.confirm_step", "item.presentation.record"}, op) {
			t.Errorf("demo-signer сделал решение живой партии %s — только руками", op)
		}
	}
}
