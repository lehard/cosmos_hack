package simulation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRandDeterministic(t *testing.T) {
	a, b := NewRand(42, "шум"), NewRand(42, "шум")
	for i := 0; i < 100; i++ {
		if a.Uint64() != b.Uint64() {
			t.Fatal("один seed и поток — разные числа")
		}
	}
	if NewRand(42, "шум").Uint64() == NewRand(42, "фон").Uint64() {
		t.Fatal("разные потоки должны давать разные числа")
	}
	p := NewRand(7, "pick").Pick(100, 10)
	if len(p) != 10 {
		t.Fatalf("Pick: %d", len(p))
	}
	for i := 1; i < len(p); i++ {
		if p[i] <= p[i-1] {
			t.Fatalf("Pick не по возрастанию или с повтором: %v", p)
		}
	}
}

func TestWeekShift(t *testing.T) {
	anchor := time.Date(2026, 9, 14, 5, 0, 0, 0, time.UTC)
	if WeekShift(anchor, time.Time{}) != 0 || WeekShift(anchor, anchor.Add(-time.Hour)) != 0 {
		t.Fatal("«сейчас» раньше якоря — без сдвига")
	}
	if got := WeekShift(anchor, anchor.Add(time.Hour)); got != Week {
		t.Fatalf("сдвиг %v, ждали неделю", got)
	}
	if got := WeekShift(anchor, anchor.Add(15*24*time.Hour)); got != 3*Week {
		t.Fatalf("сдвиг %v, ждали три недели", got)
	}
	v := ShiftValue(map[string]any{"at": "2026-09-23T11:08:00+03:00", "due": "2026-09-28", "x": []any{"F-017"}}, Week).(map[string]any)
	if v["at"] != "2026-09-30T08:08:00.000Z" || v["due"] != "2026-10-05" || v["x"].([]any)[0] != "F-017" {
		t.Fatalf("ShiftValue: %v", v)
	}
}

func TestExtractEvaluate(t *testing.T) {
	var doc any
	dec := json.NewDecoder(strings.NewReader(`{"versions":[{"scope_version":1,"size":34},{"scope_version":3,"size":6}],
		"items":[{"item_id":"A","known":"suspect"},{"item_id":"B","known":"excluded"},{"item_id":"C","known":"suspect"}]}`))
	dec.UseNumber()
	_ = dec.Decode(&doc)
	cases := []struct {
		path, op string
		want     any
		ok       bool
	}{
		{"/versions[scope_version=3]/0/size", "eq", json.Number("6"), true},
		{"/items[known=suspect]/#", "eq", json.Number("2"), true},
		{"/items[known!=suspect]/*/item_id", "eq", "B", true},
		{"/items[known!=suspect,item_id!=B]/#", "eq", json.Number("0"), true},
		{"/items/*/item_id", "set_eq", []any{"C", "A", "B"}, true},
		{"/items/*/item_id", "contains", []any{"B"}, true},
		{"/items/*/item_id", "not_contains", []any{"Z"}, true},
		{"/items/*/item_id", "before", []any{"A", "C"}, true},
		{"/items/*/item_id", "before", []any{"C", "A"}, false},
		{"/versions/0/size", "gte", json.Number("30"), true},
		{"/nothing", "not_exists", nil, true},
		{"/items/0", "eq", map[string]any{"item_id": "A"}, true},
	}
	for _, c := range cases {
		vals, err := Extract(doc, c.path)
		if err != nil {
			t.Fatal(err)
		}
		r := Evaluate(Check{Op: c.op, Value: c.want}, Single(vals), len(vals) > 0, nil)
		if (r.Status == StatusPassed) != c.ok {
			t.Errorf("%s %s %v: %s %s", c.path, c.op, c.want, r.Status, r.Detail)
		}
	}
	r := Evaluate(Check{Op: "delta", Value: json.Number("313")}, json.Number("400"), true, json.Number("87"))
	if r.Status != StatusPassed {
		t.Fatalf("delta: %s", r.Detail)
	}
}

func TestClock(t *testing.T) {
	v0 := time.Date(2026, 9, 21, 5, 0, 0, 0, time.UTC)
	r0 := time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)
	c := Clock{VirtualAt: v0, RealAt: r0, Speed: 1000}
	if got := c.Now(r0.Add(time.Second)); !got.Equal(v0.Add(1000 * time.Second)) {
		t.Fatalf("×1000: %v", got)
	}
	p := c.Pause(r0.Add(time.Second))
	if !p.Now(r0.Add(time.Hour)).Equal(v0.Add(1000 * time.Second)) {
		t.Fatal("на паузе часы стоят")
	}
	q := p.Resume(r0.Add(time.Hour))
	if !q.Now(r0.Add(time.Hour + time.Second)).Equal(v0.Add(2000 * time.Second)) {
		t.Fatal("продолжение — с того же момента, без скачка")
	}
	s := q.WithSpeed(r0.Add(time.Hour+time.Second), 1)
	if !s.Now(r0.Add(time.Hour + 2*time.Second)).Equal(v0.Add(2001 * time.Second)) {
		t.Fatal("смена скорости — без скачка времени")
	}
}

// bundle — маленький мир для проверки генератора без файлов.
func bundle() Bundle {
	w := World{ID: "w", Enterprise: "ENT01", ItemType: "FL-100.00.000", ItemRevision: "Б",
		Aliases: Aliases{People: map[string]string{"WLD-02": "W21"}, Equipment: map[string]string{"WS-1": "IS-1", "CNC-1": "CNC-1"},
			Zones: map[string]string{"U1": "W-1.U1"}},
		Sources: map[string]Source{"onec": {ID: "onec", Kind: "external_system", Reliability: "high"}, "term-vk": {ID: "t", Kind: "manual_entry", Reliability: "high"},
			"cam-kt1": {ID: "k1", Kind: "camera", Reliability: "medium"}, "cnc-1": {ID: "cnc", Kind: "machine", Reliability: "high"},
			"marker": {ID: "m", Kind: "machine", Reliability: "high"}, "cam-kt2": {ID: "k2", Kind: "camera", Reliability: "medium"},
			"cmm-1": {ID: "cmm", Kind: "machine", Reliability: "high"}, "is-1": {ID: "w1", Kind: "machine", Reliability: "high"},
			"cam-kt3-a": {ID: "k3", Kind: "camera", Reliability: "medium"}},
		Lines:  map[string]Line{"L-1": {WeldingSource: "is-1", Workplace: "WP-WELD-1", Equipment: "WS-1"}},
		Shifts: map[string]Shift{"A": {Starts: "08:00", Ends: "16:30"}},
		Route:  Route{MachiningMin: 50, CMMMin: 30, EdgePrepMin: 45, WeldMin: 10, XRayAfterMin: 60, ZT3AfterMin: 30, CurrentNominal: 160, CurrentTol: 10}}
	r := RunDef{ID: "R", Seed: 5, Anchor: "2026-09-21T07:00:00+03:00", End: "2026-09-21T16:00:00+03:00",
		Orders: []OrderPlan{{ID: "ORD-1", At: "2026-09-21T07:00:00+03:00", Quantity: 1, Lots: map[string]string{"blank": "LOT-B", "ring": "LOT-R"}}},
		Lots:   []LotPlan{{ID: "LOT-B", Component: "FL-100.01.001", Supplier: "S", Quantity: 1, Arrived: "2026-09-21T07:05:00+03:00", Accepted: "2026-09-21T07:40:00+03:00"}},
		Items: []ItemPlan{{ID: "F-501", Order: "ORD-1", Launch: "2026-09-21T08:00:00+03:00", Weld: &WeldPlan{At: "2026-09-21T12:00:00+03:00", Station: "WS-1", Welder: "WLD-02"},
			Until: "kt3"}},
		Noise: Noise{DuplicateBP: 3000, LossBP: 1000}}
	return Bundle{World: w, Run: r}
}

func TestGenerateDeterministic(t *testing.T) {
	b := bundle()
	p1, err := Generate(b, Params{RunID: "r-1"})
	if err != nil {
		t.Fatal(err)
	}
	p2, _ := Generate(b, Params{RunID: "r-1"})
	j1, _ := json.Marshal(p1.Emissions)
	j2, _ := json.Marshal(p2.Emissions)
	if string(j1) != string(j2) {
		t.Fatal("тот же вход — другой план (AD-4)")
	}
	p3, _ := Generate(b, Params{RunID: "r-2"})
	if p3.Emissions[0].EventID == p1.Emissions[0].EventID {
		t.Fatal("другой прогон — те же event_id (AD-38)")
	}
	if !strings.HasPrefix(p1.Emissions[0].SourceID, "r-1/") {
		t.Fatalf("source_id без префикса прогона: %s", p1.Emissions[0].SourceID)
	}
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	p4, _ := Generate(b, Params{RunID: "r-3", Now: now})
	if p4.Shift != 2*Week || p4.Start.Before(now) {
		t.Fatalf("сдвиг к «сейчас»: %v, начало %v", p4.Shift, p4.Start)
	}
	tr := p1.Truth.Totals
	if tr.Duplicates == 0 || tr.Lost == 0 || tr.Deliveries != tr.Unique+tr.Duplicates {
		t.Fatalf("шум фона по seed: %+v", tr)
	}
	for i := 1; i < len(p1.Emissions); i++ {
		if p1.Emissions[i].DeliverAt.Before(p1.Emissions[i-1].DeliverAt) {
			t.Fatal("события не в порядке доставки")
		}
	}
}
