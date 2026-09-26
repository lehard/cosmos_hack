package simulation

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	appingest "ant/internal/application/ingest"
	"ant/internal/application/ingest/inmem"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	app "ant/internal/application/simulation"
	"ant/internal/application/simulation/simfake"
	sim "ant/internal/domain/simulation"
	ingeststore "ant/internal/infrastructure/storage/ingest"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
	"ant/internal/infrastructure/storage/journal/migrator"
)

// Цифровой стенд на своей БД (make dev-db; без ANT_DB_HOST пропускается):
// прогон F01 идёт через настоящий приём (эпик 06) в журнал Postgres (эпик 04);
// каждая кнопка → запись в журнале (или ответ приёма «повтор», разрыв номеров
// в реестре источников) → строки табло, которые проверяет приём, совпали;
// строки модулей без движка ждут («операция пока не отвечает»), не «не совпало».
func TestDigitalStandOnPostgres(t *testing.T) {
	db := journaltest.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	set := migrator.Set{Module: "ingest", FS: ingeststore.Migrations, Dir: ingeststore.MigrationsDir}
	if _, err := migrator.Up(ctx, db.Admin.ConnConfig, nil, set); err != nil {
		t.Fatal(err)
	}
	pool := db.AppPool(t)
	js := journalstore.NewStore(pool, journaltest.SysClock{})
	ist := ingeststore.NewStore(pool)
	ing := appingest.NewService(appingest.WithConfig(appingest.DefaultConfig()), appingest.WithDeps(appingest.Deps{Journal: js, Registry: ist,
		Quarantine: ist, DomainClock: inmem.SystemClock{}, InfraClock: inmem.SystemInfra{}}))
	fk := simfake.NewIngest() // регистрация изделий демо-персоной (без модуля item)
	probe := dbProbe{ing: ing, fake: fk}
	files := NewFiles(filepath.Join(repo, "scenarios"))
	store := NewMemoryRuns()
	rec := &simfake.Recorder{}
	clock := &simfake.Clock{T: time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)}
	svc := app.NewServiceWith(app.Deps{Definitions: files, Gateway: runGateway{app.IngestGateway{Ingest: ing, SentAt: true}}, Probe: probe,
		Actor: simfake.Actor{In: fk}, Recorder: rec, Store: store, Infra: clock, Profile: "demo"})

	started, err := svc.StartRun(ctx, "F01", app.StartRun{Mode: app.ModeInteractive, Speed: 1000})
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := store.Load(ctx, started.RunID)
	p, err := sim.Generate(mustBundle(t, files, "F01"), sim.Params{RunID: st.RunID, Seed: st.Seed, Now: st.GenNow})
	if err != nil {
		t.Fatal(err)
	}
	kt3 := p.IDs.Labels["F-501/kt3"]
	for i := 0; i < 500 && (st.Cursor.Emissions < len(p.Emissions)/2 || !delivered(p, st, kt3)); i++ {
		clock.Advance(2 * time.Second)
		if err := svc.Step(ctx, st.RunID); err != nil {
			t.Fatal(err)
		}
		st, _, _ = store.Load(ctx, st.RunID)
	}
	if st.State != app.StateRunning || !delivered(p, st, kt3) {
		t.Fatalf("прогон не дошёл до КТ-3: %s, курсор %d", st.State, st.Cursor.Emissions)
	}
	if st.Delivered[app.DeliveryAccepted] == 0 {
		t.Fatalf("события прогона не приняты приёмом: %v", st.Delivered)
	}

	entries := func() map[string]map[string]any {
		out := map[string]map[string]any{}
		for _, e := range journaltest.ReadAll(t, js, "main") {
			env, err := js.Open(ctx, e)
			if err != nil {
				t.Fatal(err)
			}
			var dsse struct {
				Payload string `json:"payload"`
			}
			_ = json.Unmarshal(env.Raw, &dsse)
			raw, _ := base64.StdEncoding.DecodeString(dsse.Payload)
			var ev map[string]any
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.UseNumber()
			_ = dec.Decode(&ev)
			if ev != nil {
				out[e.EventID] = ev
			}
		}
		return out
	}
	press := func(kind, target string) (app.InjectionState, map[string]string) {
		t.Helper()
		if _, err := svc.ApplyInjection(ctx, st.RunID, app.ApplyInjection{Injection: kind, TargetEventID: target}); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		cur, _, _ := store.Load(ctx, st.RunID)
		x := cur.Injections[len(cur.Injections)-1]
		b, err := svc.Board(ctx, st.RunID, platform.Moment{})
		if err != nil {
			t.Fatal(err)
		}
		rows := map[string]string{}
		for _, r := range b.Rows {
			if id, n, ok := strings.Cut(r.AssertionID, "#"); ok && n == strconv.Itoa(x.N) {
				rows[id] = r.Status
				if r.Status == "failed" {
					t.Errorf("%s %s «%s»: не совпало (%s)", kind, id, r.Title, r.Detail)
				}
			}
		}
		return x, rows
	}
	want := func(kind string, rows map[string]string, ids ...string) {
		t.Helper()
		for _, id := range ids {
			if rows[id] != "passed" {
				t.Errorf("%s %s: %s, ждали passed (все строки: %v)", kind, id, rows[id], rows)
			}
		}
	}

	// Повтор: приём отвечает «повтор», в журнале новой записи нет.
	head0, _ := js.Head(ctx)
	_, rows := press("duplicate_event", kt3)
	want("повтор", rows, "STAND-DUP-01", "STAND-DUP-02", "STAND-DUP-03")
	if h, _ := js.Head(ctx); h.MainSeq != head0.MainSeq {
		t.Fatalf("повтор записан в журнал: %d → %d", head0.MainSeq, h.MainSeq)
	}

	// Опоздавшее: запись в журнале со временем события, а не получения.
	x, rows := press("late_event", kt3)
	want("опоздавшее", rows, "STAND-LATE-01")
	ev := entries()[x.EventIDs[0]]
	target := entries()[kt3]
	tOcc, _ := time.Parse(time.RFC3339Nano, target["occurred_at"].(string))
	lOcc, _ := time.Parse(time.RFC3339Nano, ev["occurred_at"].(string))
	if ev == nil || !lOcc.Equal(tOcc.Add(-sim.LateBy)) || ev["source_id"] != st.RunID+"/"+sim.StandLate {
		t.Fatalf("опоздавшее в журнале: %v", ev)
	}

	// Испорченный кадр: в журнале исходный результат с качеством 0,30.
	x, rows = press("corrupt_frame", "")
	want("кадр", rows, "STAND-FRAME-00")
	d := entries()[x.EventIDs[0]]["data"].(map[string]any)
	if d["observation_quality_bp"] != json.Number("3000") || d["method"] != "camera" {
		t.Fatalf("кадр в журнале: %v", d)
	}

	// Ток вне уставки: запись оборудования в журнале.
	x, rows = press("machine_fault", "")
	want("ток", rows, "STAND-MACH-01")
	if e := entries()[x.EventIDs[0]]; e["event_type"] != "equipment.deviation.detected" ||
		e["data"].(map[string]any)["value"].(map[string]any)["value"] != json.Number("176") {
		t.Fatalf("ток в журнале: %v", e)
	}

	// Потеря данных: реестр источников приёма видит разрыв номеров.
	x, rows = press("data_loss", "")
	want("потеря", rows, "STAND-LOSS-00", "STAND-LOSS-01")
	if len(x.EventIDs) != 2 {
		t.Fatalf("потеря: в журнале записи до и после разрыва: %v", x.EventIDs)
	}

	// Подделка: без cmd/tamper (эпик 29) — заглушка, журнал не тронут, строки ждут.
	head1, _ := js.Head(ctx)
	_, rows = press("tamper_outside", "")
	if rows["STAND-TAMP-01"] != "pending" || rows["STAND-TAMP-02"] != "pending" {
		t.Fatalf("подделка без демо-инструмента: %v", rows)
	}
	if h, _ := js.Head(ctx); h.MainSeq != head1.MainSeq {
		t.Fatal("заглушка подделки не пишет в журнал")
	}
	var applied int
	for _, r := range rec.Records {
		if r.Type == "simulation.injection.applied" {
			applied++
		}
	}
	if applied != 6 {
		t.Fatalf("записей simulation.injection.applied: %d, ждали 6", applied)
	}
}

func delivered(p *sim.Plan, st *app.RunState, id string) bool {
	for _, e := range p.Emissions[:st.Cursor.Emissions] {
		if e.EventID == id {
			return true
		}
	}
	return false
}

// runGateway — события прогона в приём с прогоном в контексте (как cmd/ant, эпик 16).
type runGateway struct{ g app.IngestGateway }

func (r runGateway) Deliver(ctx context.Context, runID string, batch []sim.Emission) ([]app.Delivered, error) {
	return r.g.Deliver(appjournal.WithRun(ctx, runID), runID, batch)
}

// dbProbe — чтение приёма на Postgres; регистрация изделий — заготовка; прочее «пока не отвечает».
type dbProbe struct {
	ing  *appingest.Service
	fake *simfake.Ingest
}

func (p dbProbe) Read(ctx context.Context, op string, params map[string]string, runID string) (any, error) {
	var v any
	var err error
	switch op {
	case "ingest.metrics.read":
		v, err = p.ing.Metrics(ctx)
	case "ingest.source.list":
		v, err = p.ing.Sources(ctx, platform.Page{Limit: 500})
	case "journal.entry.read":
		return p.fake.Read(ctx, op, params, runID)
	default:
		return nil, app.ErrUnavailable
	}
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(v)
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var doc any
	return doc, dec.Decode(&doc)
}
