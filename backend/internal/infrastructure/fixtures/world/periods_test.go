package world

import (
	"context"
	"testing"
	"time"

	analyticsapp "ant/internal/application/analytics"
	processapp "ant/internal/application/process"
	"ant/internal/infrastructure/fixtures/loader"
)

// TestPeriodWindows — окна периодов заготовок (UI-15, FR-3): смена — по
// шаблонам смен справочника в рабочие дни, сутки, неделя с понедельника,
// месяц — по МСК; прошлый период — такое же окно назад.
func TestPeriodWindows(t *testing.T) {
	m := buildMain(t)
	sh, err := LoadShifts(repoFS())
	if err != nil {
		t.Fatal(err)
	}
	m.shifts = sh
	msk := m.clk.loc
	at := func(d, h, mi int) time.Time { return time.Date(2026, 9, d, h, mi, 0, 0, msk).UTC() }
	cases := []struct {
		name             string
		now              time.Time
		kind             string
		from, pFrom, pTo time.Time
	}{
		{"смена 1, прошлая — смена 2 накануне", at(23, 13, 45), "shift", at(23, 8, 0), at(22, 16, 30), at(22, 22, 15)},
		{"смена 2", at(23, 17, 0), "shift", at(23, 16, 30), at(23, 8, 0), at(23, 8, 30)},
		{"вне смен — последняя начавшаяся", at(23, 3, 0), "shift", at(22, 16, 30), at(22, 8, 0), at(22, 18, 30)},
		{"понедельник: прошлая смена — пятница", at(21, 9, 0), "shift", at(21, 8, 0), at(18, 16, 30), at(18, 17, 30)},
		{"сутки", at(23, 13, 45), "day", at(23, 0, 0), at(22, 0, 0), at(22, 13, 45)},
		{"неделя с понедельника", at(23, 13, 45), "week", at(21, 0, 0), at(14, 0, 0), at(16, 13, 45)},
		{"месяц", at(23, 13, 45), "month", at(1, 0, 0), time.Date(2026, 8, 1, 0, 0, 0, 0, msk).UTC(), time.Date(2026, 8, 23, 13, 45, 0, 0, msk).UTC()},
	}
	for _, tc := range cases {
		c := &Ctx{M: m, T: tc.now}
		w := c.window(tc.kind)
		if w.Kind != tc.kind || !w.From.Equal(tc.from) || !w.To.Equal(tc.now) || !w.prevFrom.Equal(tc.pFrom) || !w.prevTo.Equal(tc.pTo) {
			t.Errorf("%s: %s [%s, %s], прошлый [%s, %s]", tc.name, w.Kind, w.From, w.To, w.prevFrom, w.prevTo)
		}
	}
}

// TestPeriodResponses — ответы аналитики заготовок выбираются по period (как
// их ищет адаптер fixtures/analytics): тело за свой период, итог плитки
// штучного показателя равен сумме строк раскрытия за тот же период (AD-45).
func TestPeriodResponses(t *testing.T) {
	ctx := context.Background()
	lib, err := testLibrary()
	if err != nil {
		t.Fatal(err)
	}
	rt := loader.New(lib, loader.NewMemoryCursor())
	get := func(op string, out any, kv ...string) {
		t.Helper()
		p := map[string]string{}
		for i := 0; i+1 < len(kv); i += 2 {
			p[kv[i]] = kv[i+1]
		}
		if err := rt.Respond(ctx, op, p, nil, out); err != nil {
			t.Fatalf("%s %v: %v", op, p, err)
		}
	}
	seen := map[string]int64{}
	for _, kind := range []string{"", "shift", "day", "week", "month"} {
		want := kind
		if want == "" {
			want = "shift" // без period — смена, как у live
		}
		var tl analyticsapp.MetricTileList
		get("analytics.tile.list", &tl, "period", kind)
		if tl.Period.Kind != want {
			t.Fatalf("period=%q: плитки за %s", kind, tl.Period.Kind)
		}
		var ov analyticsapp.AnalyticsOverview
		get("analytics.overview.read", &ov, "period", kind)
		var nc analyticsapp.NodeCounterSet
		get("analytics.node_counters.read", &nc, "period", kind, "process_version_id", ProcessVersionID)
		var cc analyticsapp.ControlChart
		get("analytics.control_chart.read", &cc, "period", kind, "step_key", "welding.weld", "metric_id", "current_a")
		if ov.Period != tl.Period || nc.Period != tl.Period {
			t.Errorf("period=%q: периоды ответов разные", kind)
		}
		for _, pt := range cc.Points {
			if pt.At.Before(tl.Period.From) || pt.At.After(tl.Period.To) {
				t.Errorf("period=%q: точка карты %s вне периода", kind, pt.At)
			}
		}
		for _, tile := range tl.Items {
			if tile.Value.Unit != "pcs" {
				continue
			}
			var dd analyticsapp.MetricDrilldown
			get("analytics.metric.drilldown", &dd, "period", kind, "metric_id", tile.MetricID)
			var sum int64
			for _, r := range dd.Items {
				sum += r.Value.Value
				if r.At != nil && (r.At.Before(tl.Period.From) || r.At.After(tl.Period.To)) {
					t.Errorf("period=%q %s: строка %s вне периода", kind, tile.MetricID, r.At)
				}
			}
			if dd.Period != tl.Period || sum != tile.Value.Value {
				t.Errorf("period=%q %s: плитка %d, раскрытие %d", kind, tile.MetricID, tile.Value.Value, sum)
			}
			if tile.Previous == nil {
				t.Errorf("period=%q %s: нет прошлого периода", kind, tile.MetricID)
			}
			if tile.MetricID == "inspected_items" {
				seen[want] = tile.Value.Value
			}
		}
	}
	// Разгар (Ср 13:45): за смену и сутки проверенных впервые нет, за месяц — все 64.
	if seen["month"] != 64 || seen["month"] == seen["shift"] {
		t.Errorf("проверено изделий по периодам: %v", seen)
	}
}

// TestLiveMapPeriods — живая карта заготовок по period (UI-15): счётчики
// «прошло» и «дефекты» — за окно периода, как node_counters аналитики;
// incident_id и process_id не теряют period и сами не теряются.
func TestLiveMapPeriods(t *testing.T) {
	ctx := context.Background()
	lib, err := testLibrary()
	if err != nil {
		t.Fatal(err)
	}
	rt := loader.New(lib, loader.NewMemoryCursor())
	get := func(out any, kv ...string) {
		t.Helper()
		p := map[string]string{}
		for i := 0; i+1 < len(kv); i += 2 {
			p[kv[i]] = kv[i+1]
		}
		if err := rt.Respond(ctx, "process.live_map.read", p, nil, out); err != nil {
			t.Fatalf("%v: %v", p, err)
		}
	}
	sum := func(lm processapp.LiveMap) (passed, defects int) {
		for _, c := range lm.Counters {
			passed += c.Passed
			defects += c.Defects
		}
		return
	}
	passed := map[string]int{}
	for _, kind := range []string{"", "shift", "day", "week", "month", "custom"} {
		var lm, inc, fl processapp.LiveMap
		var nc analyticsapp.NodeCounterSet
		get(&lm, "period", kind)
		get(&inc, "period", kind, "incident_id", "RS-01")
		get(&fl, "period", kind, "process_id", FlangeProcessID)
		if err := rt.Respond(ctx, "analytics.node_counters.read", map[string]string{"period": kind, "process_version_id": ProcessVersionID}, nil, &nc); err != nil {
			t.Fatal(err)
		}
		p, d := sum(lm)
		pi, di := sum(inc)
		pf, _ := sum(fl)
		var pn, dn int
		for _, c := range nc.Counters {
			pn += c.Passed
			dn += c.Defects
		}
		if p != pi || d != di || p != pf || p != pn || d != dn {
			t.Errorf("period=%q: карта %d/%d, с инцидентом %d/%d, с процессом %d, аналитика %d/%d", kind, p, d, pi, di, pf, pn, dn)
		}
		if inc.Incident == nil || inc.Incident.IncidentID != "RS-01" {
			t.Errorf("period=%q: инцидент потерян", kind)
		}
		passed[kind] = p
	}
	if passed[""] != passed["shift"] || passed["custom"] != passed["shift"] || passed["month"] <= passed["shift"] {
		t.Errorf("прошло по периодам: %v", passed)
	}
	var br processapp.LiveMap
	get(&br, "period", "week", "process_id", BracketProcessID)
	if br.ProcessID != BracketProcessID {
		t.Errorf("карта кронштейна с period: %s", br.ProcessID)
	}
}
