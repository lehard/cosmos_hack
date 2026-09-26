package simulation

import (
	"fmt"
	"slices"
	"time"
)

// stream — журнал оборудования за окно обрыва связи (S04, S06, S07): шлюз
// копит записи у себя и досылает пачкой; часть записей потеряна (буфер
// переполнен), часть приходит повторно (тот же номер). Записи встают на своё
// время события, а не на время получения (кейс §4.6, FR-32).
//
// Состав записей: сводки сварок по минутам (из «истины» сварок этого
// оборудования в окне), отклонения на своё время (Deviations) и сводки
// простоя, равномерно заполняющие окно до числа Records.
func (g *gen) stream(scenario string, st *Step) {
	s := st.Stream
	from, to := g.t(s.From), g.t(s.To)
	deliver := g.t(s.Deliver)
	src, ok := g.b.World.Sources[s.Source]
	if !ok {
		g.fail("%s: поток: источник %q не описан", scenario, s.Source)
		return
	}
	_ = src
	rnd := NewRand(g.seed, "stream/"+scenario+"/"+s.Source)
	var recs []*draft
	busy := [][2]time.Time{}
	for _, w := range g.welds {
		if w.Station != s.Equipment || !w.End.After(from) || !w.Start.Before(to) {
			continue
		}
		runID := "{local:" + w.Run + "}"
		for m := w.ArcFrom; m.Before(w.ArcTo); m = m.Add(time.Minute) {
			if m.Before(from) || !m.Before(to) {
				continue
			}
			d := &draft{at: m.Add(time.Minute), source: s.Source, typ: "equipment.cycle.summarized", scenario: scenario,
				data: g.cycleData(w, m, runID, rnd), label: fmt.Sprintf("%s/weld/%s", w.Item, m.Add(-g.shift).Format("0102-1504"))}
			recs = append(recs, d)
		}
		busy = append(busy, [2]time.Time{w.Start, w.End})
	}
	for _, dv := range s.Deviations {
		at := g.t(dv.At)
		wt, found := g.weldAt(s.Equipment, at)
		r := g.b.World.Route
		data := map[string]any{"equipment_id": g.ids.Equipment(s.Equipment), "station_id": "ST-WELD", "deviation_kind": "out_of_setpoint",
			"parameter": dv.Parameter, "value": measurement(int64(dv.Value), 0, dv.Unit), "started_at": FormatTime(at),
			"setpoint": map[string]any{"nominal": measurement(int64(r.CurrentNominal), 0, dv.Unit),
				"lower": measurement(int64(r.CurrentNominal-r.CurrentTol), 0, dv.Unit), "upper": measurement(int64(r.CurrentNominal+r.CurrentTol), 0, dv.Unit)},
			"code": "I_OUT_OF_SETPOINT"}
		if found {
			data["ended_at"] = FormatTime(wt.End)
		}
		label := dv.Label
		if label == "" {
			label = fmt.Sprintf("%s/deviation/%s", s.Equipment, dv.Run)
		}
		recs = append(recs, &draft{at: at, source: s.Source, typ: "equipment.deviation.detected", scenario: scenario, data: data, label: label})
	}
	idle := s.Records - len(recs)
	if idle < 0 {
		g.fail("%s: поток %s: записей сварок и отклонений (%d) больше, чем записей окна (%d)", scenario, s.Source, len(recs), s.Records)
		return
	}
	// сводки простоя — равномерно по свободному от сварок времени окна
	free := freeIntervals(from, to, busy)
	var total time.Duration
	for _, f := range free {
		total += f[1].Sub(f[0])
	}
	if idle > 0 && total <= 0 {
		g.fail("%s: поток %s: нет времени простоя для %d сводок", scenario, s.Source, idle)
		return
	}
	for k := 0; k < idle; k++ {
		off := time.Duration(int64(total) * int64(2*k+1) / int64(2*idle))
		at := pointIn(free, off).Truncate(time.Second)
		win := time.Duration(int64(total) / int64(idle))
		if win > 10*time.Minute {
			win = 10 * time.Minute
		}
		start := at.Add(-win).Truncate(time.Second)
		recs = append(recs, &draft{at: at, source: s.Source, typ: "equipment.cycle.summarized", scenario: scenario,
			data: map[string]any{"equipment_id": g.ids.Equipment(s.Equipment), "station_id": "ST-WELD",
				"window_start": FormatTime(start), "window_end": FormatTime(at), "cycle_ref": "idle",
				"parameters": []any{map[string]any{"parameter": "arc_time", "mean": measurement(0, 0, "s")}}}})
	}
	slices.SortStableFunc(recs, func(a, b *draft) int { return a.at.Compare(b.at) })
	// потери — у источника: номер получен, запись не доставлена
	var lostFrom, lostTo time.Time
	if s.Lost != nil {
		lostFrom, lostTo = g.t(s.Lost.From), g.t(s.Lost.To)
	}
	var delivered []*draft
	for _, d := range recs {
		if s.Lost != nil && !d.at.Before(lostFrom.Add(time.Minute)) && !d.at.After(lostTo) {
			d.lost = true
			continue
		}
		delivered = append(delivered, d)
	}
	span := time.Duration(s.DeliverSpanSec) * time.Second
	if span <= 0 {
		span = 7 * time.Second
	}
	for i, d := range delivered {
		d.deliver = deliver.Add(time.Duration(int64(span) * int64(i) / int64(max(1, len(delivered)))))
	}
	if s.Duplicates > len(delivered) {
		g.fail("%s: поток %s: повторов %d больше доставленных записей %d", scenario, s.Source, s.Duplicates, len(delivered))
		return
	}
	for _, i := range rnd.Pick(len(delivered), s.Duplicates) {
		d := delivered[i]
		d.dups = append(d.dups, d.deliver.Add(time.Duration(rnd.Between(1, 900))*time.Millisecond))
	}
	for _, d := range recs {
		g.add(d)
	}
}

// weldAt — сварка оборудования, идущая в момент t.
func (g *gen) weldAt(equipment string, t time.Time) (WeldTruth, bool) {
	for _, w := range g.welds {
		if w.Station == equipment && !t.Before(w.Start) && t.Before(w.End) {
			return w, true
		}
	}
	return WeldTruth{}, false
}

// freeIntervals — окно [from, to) без занятых отрезков.
func freeIntervals(from, to time.Time, busy [][2]time.Time) [][2]time.Time {
	slices.SortFunc(busy, func(a, b [2]time.Time) int { return a[0].Compare(b[0]) })
	var out [][2]time.Time
	cur := from
	for _, b := range busy {
		s, e := b[0], b[1]
		if s.Before(from) {
			s = from
		}
		if e.After(to) {
			e = to
		}
		if s.After(cur) {
			out = append(out, [2]time.Time{cur, s})
		}
		if e.After(cur) {
			cur = e
		}
	}
	if to.After(cur) {
		out = append(out, [2]time.Time{cur, to})
	}
	return out
}

// pointIn — момент на расстоянии off от начала свободного времени.
func pointIn(free [][2]time.Time, off time.Duration) time.Time {
	for _, f := range free {
		d := f[1].Sub(f[0])
		if off < d {
			return f[0].Add(off)
		}
		off -= d
	}
	return free[len(free)-1][1]
}
