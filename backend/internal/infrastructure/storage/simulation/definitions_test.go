package simulation

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	app "ant/internal/application/simulation"
	"ant/internal/application/simulation/simfake"
	sim "ant/internal/domain/simulation"
)

// TestMainStoryTruth — мир главной истории сходится с числами кейса
// (process/scenarios-flange.md §2.6, S04, S06, S07, S15; FR-104).
func TestMainStoryTruth(t *testing.T) {
	f := NewFiles(filepath.Join(repo, "scenarios"))
	b, err := f.Bundle(context.Background(), "MS-1")
	if err != nil {
		t.Fatal(err)
	}
	p, err := sim.Generate(b, sim.Params{RunID: "ms-1-check"})
	if err != nil {
		t.Fatal(err)
	}
	msk := time.FixedZone("MSK", 3*3600)
	at := func(s string) time.Time {
		v, _ := time.ParseInLocation("2006-01-02 15:04", s, msk)
		return v
	}
	var anchor time.Time
	for _, w := range p.Truth.Welds {
		if w.Item == "F-006" {
			anchor = w.End
		}
	}
	if !anchor.Equal(at("2026-09-21 14:55")) {
		t.Fatalf("отсчёт круга — конец сварки Ф-006 в Пн 14:55, а не %v", anchor)
	}
	circle, ws2, early := []string{}, 0, []string{}
	for _, w := range p.Truth.Welds {
		if w.Start.After(anchor) && w.Start.Before(at("2026-09-23 11:10")) && w.Rework == "" {
			circle = append(circle, w.Item)
			if w.Station == "WS-2" {
				ws2++
				if !w.End.After(at("2026-09-22 10:20")) {
					early = append(early, w.Item)
				}
			}
		}
	}
	slices.Sort(early)
	if len(circle) != 34 || ws2 != 13 || len(circle)-ws2 != 21 {
		t.Fatalf("круг: %d сварок (ИС-2 %d, ИС-1 %d), ждали 34 = 13 + 21", len(circle), ws2, len(circle)-ws2)
	}
	if want := []string{"F-008", "F-010", "F-012", "F-014", "F-016", "F-221", "F-222"}; !slices.Equal(early, want) {
		t.Fatalf("сварки ИС-2, законченные до Вт 10:20: %v, ждали %v (34 → 13 → 6)", early, want)
	}
	// где детали круга в Ср 11:10 (S05-03): 21 — участок, 8 — кладовая, 4 — сборка, 1 — стенд
	where := map[string]int{}
	for _, it := range p.Truth.Items {
		if !slices.Contains(circle, it.ID) {
			continue
		}
		last := ""
		for _, s := range sim.Stages {
			if t0, ok := it.Stages[s]; ok && !t0.After(at("2026-09-23 11:10")) {
				last = s
			}
		}
		switch last {
		case "sent_to_assembly":
			where["storage"]++
		case "received_assembly", "assembly_started", "assembly", "zt4_presented", "zt4":
			where["assembly"]++
		case "leak_test":
			where["leak_test"]++
		default:
			where["welding"]++
		}
	}
	if where["welding"] != 21 || where["storage"] != 8 || where["assembly"] != 4 || where["leak_test"] != 1 {
		t.Fatalf("где детали в Ср 11:10: %v, ждали 21 / 8 / 4 / 1", where)
	}
	// не меньше 10 сопоставимых сварок на сварщика, источник и смену (FR-104)
	per := map[string]int{}
	for _, w := range p.Truth.Welds {
		per["welder "+w.Welder]++
		per["source "+w.Station]++
		per["shift "+w.Shift]++
	}
	for k, n := range per {
		if n < 10 {
			t.Errorf("%s: %d сварок, нужно не меньше 10 (FR-104)", k, n)
		}
	}
	for _, s := range p.Truth.Sources {
		switch s.Key {
		case "is-2":
			c := s.Counters
			if c.Late != 928 || c.Duplicates != 312 || c.Lost != 35 || s.LostSeqs[0] != 4711 || s.LostSeqs[len(s.LostSeqs)-1] != 4745 {
				t.Errorf("журнал ИС-2: опоздало %d, повторов %d, потеряно %d (%d…%d); ждали 928, 312, 35 (4711…4745)",
					c.Late, c.Duplicates, c.Lost, s.LostSeqs[0], s.LostSeqs[len(s.LostSeqs)-1])
			}
		case "cmm-1":
			if s.Counters.Skewed != 5 || s.SkewSec != 420 {
				t.Errorf("часы КИМ-1: %d записей со сдвигом %d с, ждали 5 и 420", s.Counters.Skewed, s.SkewSec)
			}
		}
	}
}

// TestExpectedFiles — ожидания разбираются, операции есть в contracts/openapi.yaml,
// сравнения из допустимых, моменты — время определения, id уникальны.
func TestExpectedFiles(t *testing.T) {
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
	cat, err := f.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, e := range cat.Entries {
		for _, c := range e.Cards {
			if seen[c] {
				continue
			}
			seen[c] = true
			ex, ok, err := f.Expected(context.Background(), c)
			if err != nil || !ok {
				t.Fatalf("expected/%s.yaml: %v", c, err)
			}
			ids := map[string]bool{}
			for _, cp := range ex.Checkpoints {
				if cp.At != "end" && cp.At != "standalone" {
					if _, err := sim.ParseTime(cp.At); err != nil {
						t.Errorf("%s: момент %q", c, cp.At)
					}
				}
				for _, a := range cp.Assertions {
					if ids[a.ID] {
						t.Errorf("%s: id %s повторяется", c, a.ID)
					}
					ids[a.ID] = true
					if !ops[a.Check.Operation] {
						t.Errorf("%s %s: операции %s нет в contracts/openapi.yaml", c, a.ID, a.Check.Operation)
					}
					if !slices.Contains(sim.Ops, a.Check.Op) {
						t.Errorf("%s %s: сравнение %s", c, a.ID, a.Check.Op)
					}
					if (a.Check.Op == "delta" || a.Check.Op == "unchanged") && a.Mapping == "exact" && a.Baseline == "" {
						t.Errorf("%s %s: %s без момента «до»", c, a.ID, a.Check.Op)
					}
					if !slices.Contains([]string{"exact", "draft", "manual"}, a.Mapping) {
						t.Errorf("%s %s: mapping %q", c, a.ID, a.Mapping)
					}
				}
			}
		}
	}
}

// TestCatalogCoverage — каталог закрывает кейс: 8 ситуаций §4.2, 9 проверок
// §5.1, демо-сценарии PRD, 25 строк таблицы сбоев; ссылки — на сценарии пульта.
func TestCatalogCoverage(t *testing.T) {
	f := NewFiles(filepath.Join(repo, "scenarios"))
	cat, err := f.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, e := range cat.Entries {
		ids[e.ID] = true
		if _, err := f.Bundle(context.Background(), e.Run); err != nil {
			t.Errorf("%s: прогон %s: %v", e.ID, e.Run, err)
		}
	}
	want := map[string]int{"s4_2": 8, "s5_1": 9, "demo": 5, "failures": 25}
	for k, n := range want {
		if len(cat.Coverage[k]) != n {
			t.Errorf("покрытие %s: %d строк, ждали %d", k, len(cat.Coverage[k]), n)
		}
	}
	for k, rows := range cat.Coverage {
		for _, r := range rows {
			for _, s := range r.Scenarios {
				if !ids[s] {
					t.Errorf("покрытие %s «%s»: сценария %s нет в каталоге", k, r.Case, s)
				}
			}
		}
	}
	demo := strings.Join(func() []string {
		var out []string
		for _, r := range cat.Coverage["demo"] {
			out = append(out, r.Case)
		}
		return out
	}(), "; ")
	for _, s := range []string{"§1.5", "станок сломался", "на вход завезли брак", "специалист заснул"} {
		if !strings.Contains(demo, s) {
			t.Errorf("демо-сценарий PRD FR-105 «%s» не покрыт", s)
		}
	}
}

// TestInteractivePauseResume — интерактивный прогон (FR-129): часы и
// скорость, остановка на решении человека, пауза в любой момент, продолжение
// без потерь и дублей доставки.
func TestInteractivePauseResume(t *testing.T) {
	ctx := context.Background()
	files := NewFiles(filepath.Join(repo, "scenarios"))
	ing := simfake.NewIngest()
	clock := &simfake.Clock{T: time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)}
	actor := &simfake.Actor{In: ing}
	store := NewMemoryRuns()
	svc := app.NewServiceWith(app.Deps{Definitions: files, Gateway: ing, Probe: ing, Actor: actorProxy{actor}, Recorder: &simfake.Recorder{},
		Store: store, Infra: clock, Profile: "demo"})
	started, err := svc.StartRun(ctx, "MS-1", app.StartRun{Mode: app.ModeInteractive, Speed: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartRun(ctx, "S01", app.StartRun{Mode: app.ModeInteractive}); err == nil {
		t.Fatal("второй прогон при идущем первом (AD-37: время журнала не убывает)")
	}
	step := func(n int, d time.Duration) *app.RunState {
		for i := 0; i < n; i++ {
			clock.Advance(d)
			if err := svc.Step(ctx, started.RunID); err != nil {
				t.Fatal(err)
			}
		}
		st, _, _ := store.Load(ctx, started.RunID)
		return st
	}
	st := step(10, 10*time.Second) // 10 с × 1000 = 2,8 ч виртуального времени на шаг
	if st.Cursor.Emissions == 0 {
		t.Fatal("часы идут — события должны уходить в приём")
	}
	if _, err := svc.PauseRun(ctx, started.RunID, app.RunControl{}); err != nil {
		t.Fatal(err)
	}
	paused := step(5, time.Hour)
	if paused.Cursor != st.Cursor && paused.Cursor.Emissions != st.Cursor.Emissions {
		t.Fatalf("на паузе поток идёт: %+v → %+v", st.Cursor, paused.Cursor)
	}
	if _, err := svc.ResumeRun(ctx, started.RunID, app.RunControl{}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 400; i++ {
		st = step(1, 30*time.Second)
		if st.State == app.StateWaiting {
			break
		}
	}
	if st.State != app.StateWaiting || st.Waiting == nil {
		t.Fatalf("прогон должен остановиться на решении человека, состояние %s", st.State)
	}
	v, _ := svc.Run(ctx, started.RunID, platformMoment())
	if v.WaitingFor == nil || v.WaitingFor.Role == "" || v.WaitingFor.Title == "" {
		t.Fatalf("пульт должен показать, чьё решение ждём: %+v", v.WaitingFor)
	}
	before := st.Cursor
	st = step(3, time.Minute)
	if st.Cursor.Emissions != before.Emissions || st.State != app.StateWaiting {
		t.Fatal("пока решения нет — сценарий стоит")
	}
	actor.OnDesk = true
	st = step(1, time.Second)
	if st.State == app.StateWaiting && st.Cursor.Actions <= before.Actions {
		t.Fatal("решение на столе принято — сценарий идёт дальше")
	}
	received, accepted, dups := ing.Deliveries()
	p, _ := sim.Generate(mustBundle(t, files, "MS-1"), sim.Params{RunID: started.RunID, Seed: st.Seed, Now: st.GenNow})
	sent, uniq := 0, map[string]bool{}
	for _, e := range p.Emissions[:st.Cursor.Emissions] {
		sent++
		if e.Delivery != sim.DeliveryDuplicate && e.Delivery != sim.DeliveryConflict && e.Quarantine == false {
			uniq[e.EventID] = true
		}
	}
	if received != sent || accepted != len(uniq) || received != accepted+dups+(received-accepted-dups) {
		t.Fatalf("без потерь и дублей: отправлено по плану %d, получено %d, принято %d (уникальных по плану %d), повторов %d",
			sent, received, accepted, len(uniq), dups)
	}
}

// actorProxy — Actor с изменяемым флагом решения на столе.
type actorProxy struct{ a *simfake.Actor }

func (p actorProxy) Act(ctx context.Context, persona, op string, params map[string]string, body map[string]any) (app.ActResult, error) {
	return p.a.Act(ctx, persona, op, params, body)
}
func (p actorProxy) Decided(ctx context.Context, run, op, object string, since int64) ([]int64, error) {
	return p.a.Decided(ctx, run, op, object, since)
}

func mustBundle(t *testing.T, f *Files, run string) sim.Bundle {
	b, err := f.Bundle(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
