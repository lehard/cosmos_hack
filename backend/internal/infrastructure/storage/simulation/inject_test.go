package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"ant/internal/application/platform"
	app "ant/internal/application/simulation"
	"ant/internal/application/simulation/simfake"
	sim "ant/internal/domain/simulation"
)

// Цифровой стенд на заготовках (FR-152, эпик 36): каждая кнопка поверх
// идущего прогона F01 → событие в обычный приём (или подделка демо-инструментом)
// → запись simulation.injection.applied в журнале прогона → строки табло
// «ожидалось → получилось» из scenarios/expected/stand совпали. Модули, которые
// отвечают на чтение строк, здесь — заготовка «системы» (fakeSystem): приём —
// simfake, паспорт, контроль, окна нарушений, инциденты, целостность — по
// принятым событиям так, как их показывают живые модули.
func TestDigitalStandOnFakes(t *testing.T) {
	ctx := context.Background()
	w := newStandWorld(t, "demo", true)
	st, kt3 := w.runUntil(t, "F-501/kt3")

	list, err := w.svc.Injections(ctx, st.RunID)
	if err != nil || len(list.Items) != len(sim.InjectionKinds) {
		t.Fatalf("кнопки стенда: %+v %v", list, err)
	}
	for _, it := range list.Items {
		if !it.Available || it.NeedsTarget || it.Title == "" || it.Description == "" {
			t.Fatalf("кнопка %s: доступна идущему прогону, цель необязательна: %+v", it.Injection, it)
		}
	}

	press := func(kind, target string) (*app.RunState, []app.BoardRow) {
		t.Helper()
		rc, err := w.svc.ApplyInjection(ctx, st.RunID, app.ApplyInjection{Injection: kind, TargetEventID: target})
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		cur, _, _ := w.store.Load(ctx, st.RunID)
		x := cur.Injections[len(cur.Injections)-1]
		if x.Injection != kind || rc.Seq != x.Seq || rc.Seq == 0 {
			t.Fatalf("%s: квитанция %+v, нажатие %+v", kind, rc, x)
		}
		rec := w.rec.Records[len(w.rec.Records)-1]
		if rec.Type != "simulation.injection.applied" || rec.Data["injection"] != kind || rec.RunID != st.RunID || rec.Data["press"] != x.N {
			t.Fatalf("%s: нет записи simulation.injection.applied в журнале прогона: %+v", kind, rec)
		}
		b, err := w.svc.Board(ctx, st.RunID, platformMoment())
		if err != nil {
			t.Fatal(err)
		}
		var rows []app.BoardRow
		for _, r := range b.Rows {
			if strings.HasSuffix(r.AssertionID, "#"+strconv.Itoa(x.N)) {
				rows = append(rows, r)
			}
		}
		if len(rows) == 0 {
			t.Fatalf("%s: строк табло нет", kind)
		}
		return cur, rows
	}
	passed := func(kind string, rows []app.BoardRow, want ...string) {
		t.Helper()
		got := map[string]bool{}
		for _, r := range rows {
			if r.Status != "passed" {
				actual := "—"
				if r.Actual != nil {
					actual = *r.Actual
				}
				t.Errorf("%s %s «%s»: %s, ожидалось %s, получено %s (%s)", kind, r.AssertionID, r.Title, r.Status, r.Expected, actual, r.Detail)
			}
			got[strings.SplitN(r.AssertionID, "#", 2)[0]] = true
		}
		for _, id := range want {
			if !got[id] {
				t.Errorf("%s: нет строки %s", kind, id)
			}
		}
	}

	// Повтор события: приём — «повтор», ничего не задвоилось.
	_, acc0, dup0 := w.ing.Deliveries()
	cur, rows := press("duplicate_event", kt3)
	passed("повтор", rows, "STAND-DUP-01", "STAND-DUP-02", "STAND-DUP-03", "STAND-DUP-04")
	if _, acc, dups := w.ing.Deliveries(); acc != acc0 || dups != dup0+1 {
		t.Fatalf("приём: принято %d → %d, повторов %d → %d", acc0, acc, dup0, dups)
	}
	if x := cur.Injections[0]; len(x.EventIDs) != 0 || x.Target != kt3 {
		t.Fatalf("повтор не вносит записей: %+v", x)
	}

	// Повтор нажатия с тем же command_id — прежний ответ (AD-7).
	cmd := "0b7d6c1e-8f4a-4d2b-9c3e-1a2b3c4d5e6f"
	r1, err := w.svc.ApplyInjection(ctx, st.RunID, app.ApplyInjection{Injection: "duplicate_event", TargetEventID: kt3, CommandHeader: platform.CommandHeader{CommandID: cmd}})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := w.svc.ApplyInjection(ctx, st.RunID, app.ApplyInjection{Injection: "duplicate_event", TargetEventID: kt3, CommandHeader: platform.CommandHeader{CommandID: cmd}})
	if err != nil || !r2.Replayed || r2.Seq != r1.Seq {
		t.Fatalf("повтор команды: %+v %+v %v", r1, r2, err)
	}

	// Опоздавшее событие: встало на своё время, история перестроена.
	cur, rows = press("late_event", kt3)
	passed("опоздавшее", rows, "STAND-LATE-01", "STAND-LATE-02", "STAND-LATE-03", "STAND-LATE-04")
	late := cur.Injections[len(cur.Injections)-1]
	if len(late.EventIDs) != 1 || w.sys.event(late.EventIDs[0]).SourceID != st.RunID+"/"+sim.StandLate {
		t.Fatalf("опоздавшее: %+v", late)
	}

	// Испорченный кадр (цель по умолчанию — последний кадр камеры).
	cur, rows = press("corrupt_frame", "")
	passed("кадр", rows, "STAND-FRAME-01", "STAND-FRAME-02", "STAND-FRAME-03")
	if x := cur.Injections[len(cur.Injections)-1]; x.Target == "" || len(x.EventIDs) != 1 {
		t.Fatalf("кадр: %+v", x)
	}

	// Ток вне уставки: окно нарушения, область риска изменилась.
	_, rows = press("machine_fault", "")
	passed("ток", rows, "STAND-MACH-01", "STAND-MACH-02", "STAND-MACH-03")

	// Потеря данных: разрыв номеров учтён; служебный порт stand-ов вызван.
	_, rows = press("data_loss", "")
	passed("потеря", rows, "STAND-LOSS-01", "STAND-LOSS-02")
	if len(w.stands.calls) != 1 || !strings.HasPrefix(w.stands.calls[0], "equipment:") || !strings.HasSuffix(w.stands.calls[0], ":drop") {
		t.Fatalf("служебный порт stand-ов: %v", w.stands.calls)
	}
	_, rows = press("data_loss", "")
	passed("потеря-2", rows, "STAND-LOSS-01")

	// Подделка в обход системы: демо-инструмент, подделку поймали.
	cur, rows = press("tamper_outside", "")
	passed("подделка", rows, "STAND-TAMP-01", "STAND-TAMP-02")
	if len(w.tamper.calls) != 1 || w.tamper.calls[0].t.Kind != "update_in_place" || w.tamper.calls[0].run != st.RunID {
		t.Fatalf("демо-инструмент: %+v", w.tamper.calls)
	}

	// Цель, которой нет в прогоне, — отказ с причиной, а не выдумка.
	if _, err := w.svc.ApplyInjection(ctx, st.RunID, app.ApplyInjection{Injection: "late_event", TargetEventID: "00000000-0000-5000-8000-000000000000"}); err == nil ||
		!strings.Contains(err.Error(), "нет в прогоне") {
		t.Fatal("цель не из прогона — отказ")
	}

	// Прогон идёт дальше без ошибок: время журнала не убывает, дублей нет.
	for i := 0; i < 200; i++ {
		w.clock.Advance(10 * time.Second)
		if err := w.svc.Step(ctx, st.RunID); err != nil {
			t.Fatal(err)
		}
		cur, _, _ = w.store.Load(ctx, st.RunID)
		if cur.State != app.StateRunning {
			break
		}
	}
	if cur.State != app.StateComplete {
		t.Fatalf("прогон после кнопок: %s %s", cur.State, cur.Error)
	}
	v, _ := w.svc.Run(ctx, st.RunID, platformMoment())
	b, _ := w.svc.Board(ctx, st.RunID, platformMoment())
	if v.BoardTotal != len(b.Rows) {
		t.Fatalf("табло: всего %d строк, в пульте %d", len(b.Rows), v.BoardTotal)
	}
	for _, r := range b.Rows {
		switch {
		case strings.HasPrefix(r.AssertionID, "STAND-") && r.Status != "passed":
			t.Errorf("строка кнопки %s после конца прогона: %s (%s)", r.AssertionID, r.Status, r.Detail)
		case r.Status == "failed" && !strings.Contains(r.Detail, "кнопками стенда"):
			// кнопки меняют ход прогона: расхождение строки карточки — с пояснением
			t.Errorf("строка карточки %s не совпала без пояснения: %s", r.AssertionID, r.Detail)
		}
	}
	if _, err := w.svc.ApplyInjection(ctx, st.RunID, app.ApplyInjection{Injection: "duplicate_event"}); err == nil {
		t.Fatal("законченный прогон — кнопки не работают")
	}
	if l, _ := w.svc.Injections(ctx, st.RunID); slices.ContainsFunc(l.Items, func(i app.Injection) bool { return i.Available }) {
		t.Fatal("законченный прогон — кнопки выключены")
	}
}

// Подделка: в профиле prod демо-инструмента нет; без реализации эпика 29 —
// заглушка за портом: шаг «пропущен», строки ждут, а не «не совпало».
func TestDigitalStandTamperGuards(t *testing.T) {
	ctx := context.Background()
	w := newStandWorld(t, "prod", false)
	st, _ := w.runUntil(t, "F-501/kt3")
	l, _ := w.svc.Injections(ctx, st.RunID)
	for _, it := range l.Items {
		if it.Injection == "tamper_outside" && it.Available {
			t.Fatal("prod: подделка недоступна")
		}
	}
	if _, err := w.svc.ApplyInjection(ctx, st.RunID, app.ApplyInjection{Injection: "tamper_outside"}); err == nil || !strings.Contains(err.Error(), "prod") {
		t.Fatal("prod: подделка — отказ")
	}

	d := newStandWorld(t, "demo", false)
	st, _ = d.runUntil(t, "F-501/kt3")
	if _, err := d.svc.ApplyInjection(ctx, st.RunID, app.ApplyInjection{Injection: "tamper_outside"}); err != nil {
		t.Fatal(err)
	}
	cur, _, _ := d.store.Load(ctx, st.RunID)
	step := cur.Steps["inject:1"]
	if step.Status != "skipped" || !strings.Contains(step.Detail, "эпик 29") {
		t.Fatalf("заглушка подделки: %+v", step)
	}
	b, _ := d.svc.Board(ctx, st.RunID, platformMoment())
	for _, r := range b.Rows {
		if strings.HasPrefix(r.AssertionID, "STAND-TAMP-") && r.Status != "pending" {
			t.Fatalf("без демо-инструмента строки ждут: %+v", r)
		}
	}
}

// Ожидания кнопок: операции есть в контракте, сравнения допустимы, «до» — у delta/unchanged/changed.
func TestStandExpectedFiles(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repo, "contracts", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var oa struct {
		Paths map[string]map[string]struct {
			OperationID string `yaml:"operationId"`
		} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &oa); err != nil {
		t.Fatal(err)
	}
	ops := map[string]bool{"step": true}
	for _, m := range oa.Paths {
		for _, o := range m {
			ops[o.OperationID] = true
		}
	}
	f := NewFiles(filepath.Join(repo, "scenarios"))
	for _, k := range sim.InjectionKinds {
		ex, ok, err := f.Expected(context.Background(), "stand/"+string(k))
		if err != nil || !ok {
			t.Fatalf("expected/stand/%s.yaml: %v", k, err)
		}
		ids := map[string]bool{}
		for _, cp := range ex.Checkpoints {
			for _, a := range cp.Assertions {
				if ids[a.ID] || !strings.HasPrefix(a.ID, "STAND-") {
					t.Errorf("%s: id %s", k, a.ID)
				}
				ids[a.ID] = true
				if !ops[a.Check.Operation] || !slices.Contains(sim.Ops, a.Check.Op) {
					t.Errorf("%s %s: %s %s", k, a.ID, a.Check.Operation, a.Check.Op)
				}
				if slices.Contains([]string{"delta", "unchanged", "changed"}, a.Check.Op) && a.Baseline != "apply" {
					t.Errorf("%s %s: %s без «до» (baseline: apply)", k, a.ID, a.Check.Op)
				}
			}
		}
		if len(ids) == 0 {
			t.Errorf("%s: нет строк", k)
		}
	}
}

// ── мир теста ──

type standWorld struct {
	svc    *app.Service
	store  *MemoryRuns
	ing    *simfake.Ingest
	sys    *fakeSystem
	rec    *simfake.Recorder
	clock  *simfake.Clock
	stands *fakeStands
	tamper *fakeTamper
	files  *Files
}

func newStandWorld(t *testing.T, profile string, withTamper bool) *standWorld {
	t.Helper()
	w := &standWorld{files: NewFiles(filepath.Join(repo, "scenarios")), store: NewMemoryRuns(), ing: simfake.NewIngest(), rec: &simfake.Recorder{},
		clock: &simfake.Clock{T: time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)}, stands: &fakeStands{}, tamper: &fakeTamper{}}
	w.sys = &fakeSystem{ing: w.ing, store: w.store, tamper: w.tamper}
	d := app.Deps{Definitions: w.files, Gateway: w.sys, Probe: w.sys, Actor: simfake.Actor{In: w.ing}, Stands: w.stands,
		Recorder: w.rec, Store: w.store, Infra: w.clock, Profile: profile}
	if withTamper {
		d.Tamper = w.tamper
	}
	w.svc = app.NewServiceWith(d)
	return w
}

// runUntil — интерактивный прогон F01 ×1000, пока событие с меткой не доставлено.
func (w *standWorld) runUntil(t *testing.T, label string) (*app.RunState, string) {
	t.Helper()
	ctx := context.Background()
	started, err := w.svc.StartRun(ctx, "F01", app.StartRun{Mode: app.ModeInteractive, Speed: 1000})
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := w.store.Load(ctx, started.RunID)
	p, err := sim.Generate(mustBundle(t, w.files, "F01"), sim.Params{RunID: st.RunID, Seed: st.Seed, Now: st.GenNow})
	if err != nil {
		t.Fatal(err)
	}
	id := p.IDs.Labels[label]
	idx := slices.IndexFunc(p.Emissions, func(e sim.Emission) bool { return e.EventID == id })
	for i := 0; i < 500 && st.Cursor.Emissions <= idx+3; i++ {
		w.clock.Advance(2 * time.Second)
		if err := w.svc.Step(ctx, st.RunID); err != nil {
			t.Fatal(err)
		}
		st, _, _ = w.store.Load(ctx, st.RunID)
	}
	if st.Cursor.Emissions <= idx || st.State != app.StateRunning || id == "" {
		t.Fatalf("прогон не дошёл до %s: курсор %d, состояние %s", label, st.Cursor.Emissions, st.State)
	}
	return st, id
}

// fakeSystem — приём simfake и ответы модулей по принятым событиям.
type fakeSystem struct {
	mu       sync.Mutex
	ing      *simfake.Ingest
	store    *MemoryRuns
	tamper   *fakeTamper
	accepted []sim.Emission
}

func (f *fakeSystem) Deliver(ctx context.Context, runID string, batch []sim.Emission) ([]app.Delivered, error) {
	res, err := f.ing.Deliver(ctx, runID, batch)
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, r := range res {
		if r.Status == app.DeliveryAccepted {
			f.accepted = append(f.accepted, batch[i])
		}
	}
	return res, err
}

func (f *fakeSystem) event(id string) sim.Emission {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, e := range f.accepted {
		if e.EventID == id {
			return e
		}
	}
	return sim.Emission{}
}

func (f *fakeSystem) Read(ctx context.Context, op string, params map[string]string, runID string) (any, error) {
	switch op {
	case "item.passport.read", "quality.inspection.list", "machinelogs.violation.list", "analysis.incident.list", "security.integrity.read":
	default:
		return f.ing.Read(ctx, op, params, runID)
	}
	st, _, _ := f.store.Load(ctx, runID)
	item := ""
	for def, id := range st.Items {
		if id == params["item_id"] {
			item = def
		}
	}
	f.mu.Lock()
	var own []sim.Emission
	for _, e := range f.accepted {
		if item != "" && e.Item == item {
			own = append(own, e)
		}
	}
	all := slices.Clone(f.accepted)
	f.mu.Unlock()
	slices.SortStableFunc(own, func(a, b sim.Emission) int { return a.OccurredAt.Compare(b.OccurredAt) })
	data := func(e sim.Emission) map[string]any {
		var ev map[string]any
		dec := json.NewDecoder(bytes.NewReader(e.Event))
		dec.UseNumber()
		_ = dec.Decode(&ev)
		d, _ := ev["data"].(map[string]any)
		return d
	}
	poor := func(d map[string]any) bool {
		q, ok := d["observation_quality_bp"].(json.Number)
		n, _ := q.Int64()
		return ok && n < 5000
	}
	var v any
	switch op {
	case "item.passport.read":
		if item == "" {
			return nil, app.ErrNotFound
		}
		quality := "not_inspected" // «годно» ставит только человек (Д-43)
		var entries []any
		for _, e := range own {
			entries = append(entries, map[string]any{"event_id": e.EventID, "event_type": e.EventType, "occurred_at": sim.FormatTime(e.OccurredAt),
				"recorded_at": sim.FormatTime(e.DeliverAt)})
			if e.EventType == "inspection.result.recorded" {
				quality = "signal"
				if poor(data(e)) {
					quality = "unable_to_assess"
				}
			}
		}
		v = map[string]any{"item_id": params["item_id"], "entries": entries, "status": map[string]any{"quality": quality}}
	case "quality.inspection.list":
		var items []any
		for _, e := range own {
			if e.EventType != "inspection.result.recorded" {
				continue
			}
			d := data(e)
			outcome := d["outcome"]
			if poor(d) {
				outcome = "unable_to_assess" // reinterpreted: poor_observation
			}
			items = append(items, map[string]any{"event_id": e.EventID, "outcome": outcome, "observation_quality_bp": d["observation_quality_bp"]})
		}
		v = map[string]any{"items": items}
	case "machinelogs.violation.list", "analysis.incident.list":
		byEq := map[string][]any{}
		var eqs []string
		for _, e := range all {
			if e.EventType != "equipment.deviation.detected" {
				continue
			}
			eq, _ := data(e)["equipment_id"].(string)
			if _, ok := byEq[eq]; !ok {
				eqs = append(eqs, eq)
			}
			byEq[eq] = append(byEq[eq], e.EventID)
		}
		items := []any{}
		for _, eq := range eqs {
			if op == "machinelogs.violation.list" {
				items = append(items, map[string]any{"equipment_id": eq, "deviation_event_ids": byEq[eq]})
			} else {
				items = append(items, map[string]any{"incident_id": "RS-" + eq, "size": len(byEq[eq]), "status": "open"})
			}
		}
		v = map[string]any{"items": items}
	case "security.integrity.read":
		status := "intact"
		if len(f.tamper.calls) > 0 {
			status = "violated"
		}
		v = map[string]any{"status": status}
	}
	b, _ := json.Marshal(v)
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var doc any
	err := dec.Decode(&doc)
	return doc, err
}

type fakeStands struct{ calls []string }

func (f *fakeStands) SetFault(_ context.Context, stand string, a sim.StandAction, _ time.Time) error {
	f.calls = append(f.calls, stand+":"+a.Fault)
	return nil
}
func (f *fakeStands) ClearFaults(context.Context, string) error { return nil }

type tamperCall struct {
	run, event string
	t          sim.Tamper
}

type fakeTamper struct{ calls []tamperCall }

func (f *fakeTamper) Apply(_ context.Context, runID string, t sim.Tamper, eventID string) error {
	if eventID == "" {
		return errors.New("нет цели подделки")
	}
	f.calls = append(f.calls, tamperCall{run: runID, event: eventID, t: t})
	return nil
}
