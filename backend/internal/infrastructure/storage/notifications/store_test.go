package notifications_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	notifapp "ant/internal/application/notifications"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	notif "ant/internal/domain/notifications"
	enginestore "ant/internal/infrastructure/storage/engine"
	store "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/feed"
	jt "ant/internal/infrastructure/storage/journal/journaltest"
	storage "ant/internal/infrastructure/storage/notifications"
)

// Сроки, эскалации и тревоги на своей БД (make dev-db): журнал на Postgres,
// настоящий воркер над WorkFeed (свёртка изделия ставит сроки по состоянию
// nonconformity), проектор ведёт проекции notifications.*, планировщик пишет
// «наступил срок» по проекции сроков, live-сервис отвечает лентой тревог и
// блоком «требует вашего внимания». Без БД тест пропускается.

var t0 = time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)

type clock struct{ t time.Time }

func (c *clock) Now(context.Context) (time.Time, error) { return c.t, nil }

type env struct {
	t      *testing.T
	ctx    context.Context
	j      *store.Store
	wf     *feed.WorkFeed
	part   engineapp.Partition
	codec  *engineapp.Codec
	worker *engineapp.WorkerService
	proj   *engineapp.Projector
	reg    *engineapp.Registry
	seq    int64
	clock  *clock
	sched  *notifapp.Scheduler
	svc    *notifapp.Service
}

func newEnv(t *testing.T) *env {
	t.Helper()
	d := jt.NewDB(t)
	pool := d.AppPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	j := store.NewStore(pool, jt.SysClock{})
	leases := store.NewLeases(pool, jt.SysClock{})
	sig := store.NewListener(pool, nil)
	go func() { _ = sig.Run(ctx) }()
	wf := feed.NewWorkFeed(j, leases, sig, 1, feed.Options{Holder: "t", TTL: 30 * time.Second})
	parts, err := wf.Partitions(ctx)
	if err != nil || len(parts) != 1 {
		t.Fatalf("партиции: %v %v", parts, err)
	}
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: dj.ZeroLink.String(), Partitions: 1}
	reg := engineapp.NewRegistry()
	if err := notifapp.Register(reg); err != nil {
		t.Fatal(err)
	}
	c := &clock{t: t0}
	st := storage.New(pool)
	e := &env{t: t, ctx: ctx, j: j, wf: wf, part: parts[0], codec: codec, reg: reg, clock: c,
		worker: engineapp.NewWorker(engineapp.WorkerConfig{Feed: wf, Codec: codec, Projections: reg}),
		proj:   &engineapp.Projector{Codec: codec, Store: &enginestore.Store{Pool: pool}, Registry: reg},
		sched:  &notifapp.Scheduler{Projections: st, Codec: codec, Clock: c}}
	e.svc = notifapp.NewLive(notifapp.Config{Projections: st, Clock: c,
		Decisions: notifapp.JournalDecisions{Journal: j, DomainBuild: dj.ZeroLink.String(), Partitions: 1}})
	return e
}

func uid(name string) string { return kernel.UUIDv5(constants.NsAnt, "notifications-test/"+name) }

func (e *env) fact(itemID, id string, typ catalog.Type, at time.Duration, data string) {
	e.t.Helper()
	info, _ := catalog.Lookup(typ)
	p, err := e.codec.Encode(e.ctx, engineapp.Out{EventID: uid(id), Type: typ, Kind: info.Kind, Stream: "item:" + itemID, ItemID: itemID,
		OccurredAt: t0.Add(at), Data: json.RawMessage(data)})
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := e.j.Append(e.ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
		e.t.Fatal(err)
	}
}

// settle — воркер по новой работе и проектор по новым записям.
func (e *env) settle() {
	e.t.Helper()
	for {
		ctx, cancel := context.WithTimeout(e.ctx, 500*time.Millisecond)
		works, err := e.wf.Next(ctx, e.part)
		cancel()
		if err != nil || len(works) == 0 {
			break
		}
		if err := e.worker.Process(e.ctx, e.part, works); err != nil {
			e.t.Fatal(err)
		}
	}
	entries, err := e.j.Read(e.ctx, appjournal.ReadQuery{AfterSeq: e.seq, Limit: 1000})
	if err != nil {
		e.t.Fatal(err)
	}
	if len(entries) == 0 {
		return
	}
	e.seq = int64(entries[len(entries)-1].Seq)
	for _, g := range e.reg.Globals() {
		rq, err := e.proj.Apply(e.ctx, g, append([]jc.JournalEntry(nil), entries...))
		if err != nil {
			e.t.Fatal(err)
		}
		if len(rq.Effects) > 0 {
			if _, err := e.j.Append(e.ctx, rq); err != nil {
				e.t.Fatal(err)
			}
		}
	}
}

func (e *env) count(t catalog.Type) int {
	e.t.Helper()
	es, err := e.j.Read(e.ctx, appjournal.ReadQuery{EventType: string(t), Limit: 1000})
	if err != nil {
		e.t.Fatal(err)
	}
	return len(es)
}

// isolate — изделие начало операцию step на месте loc и изолировано со
// сроком решения через час (FR-55).
func (e *env) isolate(itemID, step, loc string, incident bool) {
	e.fact(itemID, itemID+"/start", catalog.OperationRunStarted, time.Minute,
		`{"operation_run_id":"RUN-`+itemID+`","operation_code":"op","step_key":"`+step+`","operator_id":"W21","workplace_id":"`+loc+`"}`)
	if incident {
		e.fact(itemID, itemID+"/scope", catalog.IncidentMembershipChanged, 2*time.Minute,
			`{"incident_id":"RS-1","scope_version":1,"status":"suspect","action":"block"}`)
	}
	e.fact(itemID, itemID+"/iso", catalog.DecisionItemIsolated, 3*time.Minute,
		`{"decision_due_at":"`+notif.FormatTime(t0.Add(time.Hour))+`","reason":{"text":"прожог"}}`)
}

// FR-8, FR-55, FR-57, AD-4: просроченная изоляция даёт тревогу с числом
// стоящих изделий и операций; повторный запуск планировщика не дублирует
// «наступил срок».
func TestOverdueIsolationOnPostgres(t *testing.T) {
	e := newEnv(t)
	// Два изделия в области риска RS-1 на блоке — на разных операциях; третье
	// изолировано само по себе.
	e.isolate("ENT01:FL-1", "welding.weld", "WP-W1", true)
	e.isolate("ENT01:FL-2", "machining.turn", "WP-M1", true)
	e.isolate("ENT01:FL-3", "assembly.fit", "WP-A1", false)
	e.settle()
	if n := e.count(catalog.ObligationDueSet); n < 3 {
		t.Fatalf("сроки не поставлены: %d", n)
	}

	// До срока тревог нет.
	e.clock.t = t0.Add(59 * time.Minute)
	if rep, err := e.sched.Tick(e.ctx, nil); err != nil || rep.Written != 0 {
		t.Fatalf("до срока: %+v %v", rep, err)
	}

	// Просрочено на 37 минут.
	e.clock.t = t0.Add(time.Hour + 37*time.Minute)
	rep, err := e.sched.Tick(e.ctx, nil)
	if err != nil || rep.Written != 3 {
		t.Fatalf("наступил срок: %+v %v", rep, err)
	}
	// Повторный запуск до того, как проекция догнала, — дублей нет.
	rep, err = e.sched.Tick(e.ctx, nil)
	if err != nil || rep.Written != 0 {
		t.Fatalf("повтор до проекции: %+v %v", rep, err)
	}
	e.settle()
	// И после: проекция знает, что срок отмечен.
	rep, err = e.sched.Tick(e.ctx, nil)
	if err != nil || rep.Written != 0 {
		t.Fatalf("повтор после проекции: %+v %v", rep, err)
	}
	if n := e.count(catalog.ObligationDueReached); n != 3 {
		t.Fatalf("«наступил срок» записан %d раз, ожидалось 3", n)
	}
	for _, it := range []string{"ENT01:FL-1", "ENT01:FL-2", "ENT01:FL-3"} {
		es, _ := e.j.Read(e.ctx, appjournal.ReadQuery{Stream: "item:" + it, EventType: string(catalog.ObligationDueReached)})
		for _, x := range es {
			if x.OccurredAt != notif.FormatTime(t0.Add(time.Hour)) {
				t.Fatalf("occurred_at «наступил срок» — срок: %s", x.OccurredAt)
			}
		}
	}
	if n := e.count(catalog.ObligationEscalationRaised); n != 3 {
		t.Fatalf("эскалаций: %d", n)
	}

	al, err := e.svc.Alerts(e.ctx, platform.Moment{}, platform.Page{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][3]int{}
	for _, a := range al.Items {
		if a.Kind != "overdue_isolation" || a.Item == nil {
			t.Fatalf("тревога: %+v", a)
		}
		got[*a.Item] = [3]int{*a.OverdueMinutes, *a.Items, *a.Operations}
	}
	want := map[string][3]int{"ENT01:FL-1": {37, 2, 2}, "ENT01:FL-2": {37, 2, 2}, "ENT01:FL-3": {37, 1, 1}}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("тревога %s: просрочено/изделий/операций = %v, ожидалось %v", k, got[k], w)
		}
	}

	att, err := e.svc.Attention(e.ctx, platform.Moment{})
	if err != nil || len(att.Items) != 2 {
		t.Fatalf("требует внимания: %+v %v", att, err)
	}
	top := att.Items[0]
	if top.Kind != "overdue_decision" || *top.Target != "Инцидент RS-1" || *top.OverdueMinutes != 37 || *top.Items != 2 || *top.Operations != 2 ||
		top.Ref.Entity != platform.EntityIncident || top.Ref.ID != "RS-1" {
		t.Fatalf("«просрочено 37 мин — стоят 2 изделия, 2 операции»: %+v", top)
	}

	// Адресно: мастер видит задачу перемещения на своём месте, технолог — запрос решения.
	foreman := platform.WithPrincipal(e.ctx, platform.Principal{PersonID: "FOR-WC", Role: "site_foreman"})
	tl, err := e.svc.Tasks(foreman, notifapp.TaskFilter{LocationID: "WP-W1"}, platform.Moment{}, platform.Page{Limit: 50})
	if err != nil || len(tl.Items) == 0 {
		t.Fatalf("задачи мастера: %+v %v", tl, err)
	}
	for _, x := range tl.Items {
		if x.AssigneeRole != "site_foreman" {
			t.Fatalf("чужая задача мастеру: %+v", x)
		}
	}
	sum, err := e.svc.Summary(platform.WithPrincipal(e.ctx, platform.Principal{PersonID: "TEC-01", Role: "technologist"}), platform.Moment{})
	if err != nil || sum.ByKind.DecisionRequest != 3 || sum.ByKind.Alarm != 3 {
		t.Fatalf("сводка технолога: %+v %v", sum.ByKind, err)
	}

	// Отметка задачи — решение в поток изделия; проекция закрывает задачу.
	task := tl.Items[0]
	if _, err := e.svc.AcknowledgeTask(foreman, task.TaskID, notifapp.AcknowledgeTask{Outcome: "done", Note: "перемещено"}); err != nil {
		t.Fatal(err)
	}
	e.settle()
	tl, _ = e.svc.Tasks(foreman, notifapp.TaskFilter{State: "done"}, platform.Moment{}, platform.Page{Limit: 50})
	if len(tl.Items) != 1 || tl.Items[0].TaskID != task.TaskID {
		t.Fatalf("задача отмечена: %+v", tl.Items)
	}
	if _, err := e.svc.AcknowledgeTask(foreman, task.TaskID, notifapp.AcknowledgeTask{Outcome: "done"}); err == nil {
		t.Fatal("повторная отметка закрытой задачи")
	}

	// Решение по несоответствию изделия 3 снимает срок и тревогу.
	e.fact("ENT01:FL-3", "FL-3/nc", catalog.DecisionNonconformityRegistered, 100*time.Minute,
		`{"nc_id":"NC-3","violation_window_event_id":"w-1","step_key":"assembly.fit","operation_run_id":"RUN-ENT01:FL-3"}`)
	e.fact("ENT01:FL-3", "FL-3/disp", catalog.DecisionDispositionSet, 101*time.Minute, `{"nc_id":"NC-3","disposition":"scrap","reason":{"text":"брак"}}`)
	e.settle()
	al, _ = e.svc.Alerts(e.ctx, platform.Moment{}, platform.Page{Limit: 50})
	for _, a := range al.Items {
		if a.Item != nil && *a.Item == "ENT01:FL-3" && a.Kind == "overdue_isolation" {
			t.Fatalf("тревога после решения: %+v", a)
		}
	}
}

// Задачи по решениям в потоке инцидента (эпик 22: запрос измерения — задачу
// ставит notifications) и проверка целостности цепочки планировщиком.
func TestObjectTasksAndIntegrityOnPostgres(t *testing.T) {
	e := newEnv(t)
	info, _ := catalog.Lookup(catalog.IncidentMeasurementRequested)
	p, err := e.codec.Encode(e.ctx, engineapp.Out{EventID: uid("measure"), Type: catalog.IncidentMeasurementRequested, Kind: info.Kind,
		Stream: "incident:RS-1", OccurredAt: t0, Data: json.RawMessage(`{"incident_id":"RS-1","hypothesis_id":"H-1","what":"Твёрдость шва","assignee_id":"NDT-61"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.j.Append(e.ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		entries, err := e.j.Read(e.ctx, appjournal.ReadQuery{Limit: 1000})
		if err != nil {
			t.Fatal(err)
		}
		rq, err := e.sched.ObjectTasks(e.ctx, entries)
		if err != nil {
			t.Fatal(err)
		}
		if len(rq.Batch) > 0 {
			if _, err := e.j.Append(e.ctx, rq); err != nil {
				t.Fatal(err)
			}
		}
	}
	if n := e.count(catalog.TaskTaskCreated); n != 1 {
		t.Fatalf("задача измерения записана %d раз", n)
	}
	e.settle()
	tl, err := e.svc.Tasks(platform.WithPrincipal(e.ctx, platform.Principal{PersonID: "NDT-61", Role: "ndt_specialist"}), notifapp.TaskFilter{}, platform.Moment{}, platform.Page{Limit: 10})
	if err != nil || len(tl.Items) != 1 || tl.Items[0].Ref.Entity != platform.EntityIncident {
		t.Fatalf("задача исполнителю измерения: %+v %v", tl, err)
	}
	integrity := &notifapp.Integrity{Journal: e.j}
	if err := integrity.Check(e.ctx); err != nil {
		t.Fatalf("целостность: %v", err)
	}
}
