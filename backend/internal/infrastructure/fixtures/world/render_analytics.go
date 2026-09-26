package world

import (
	"slices"
	"strings"

	analyticsapp "ant/internal/application/analytics"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
)

// Показатели — из данных мира, каждое число раскрывается до изделий и записей
// (AD-45); дефекты и изделия с дефектами — раздельно; входной брак — отдельно
// от производственных; «под подозрением» браком не считается (кейс §2.4, §5.2).

func pcs(n int) analyticsapp.MetricValue {
	return analyticsapp.MetricValue{Value: int64(n), Scale: 0, Unit: "pcs"}
}
func bp(n int) analyticsapp.MetricValue {
	return analyticsapp.MetricValue{Value: int64(n), Scale: 0, Unit: "bp"}
}
func mins(n int, origin string) analyticsapp.MetricValue {
	return analyticsapp.MetricValue{Value: int64(n), Scale: 0, Unit: "min", Origin: ptr(origin)}
}

// metrics — показатели шага.
type metrics struct {
	inspected, withNC, zt3First, zt3Total, causeOK, ncTotal, unable, rework int
	defects                                                                 map[string]int // вид → число (производственные)
	byLine                                                                  map[string]int
	incoming                                                                int
	ncItems, inspItems                                                      []*Item
	leadSum, leadN                                                          int
}

func (c *Ctx) metrics() metrics {
	mt := metrics{defects: map[string]int{}, byLine: map[string]int{"LINE-FL-1": 0, "LINE-FL-2": 0}}
	for _, it := range c.Existing() {
		insp, nc := false, false
		firstZT3 := ""
		for _, e := range c.itemEvents(it) {
			if e.Type == "decision.presentation.resolved" {
				insp = true
				if e.StepKey == "welding.zt3_acceptance" && firstZT3 == "" {
					firstZT3 = e.Params["outcome"]
				}
			}
			if e.Type == "inspection.result.recorded" && e.Params["outcome"] == "unable_to_assess" {
				mt.unable++
			}
		}
		for _, mk := range it.marks {
			if mk.axis == "quality" && mk.value == "nonconforming" && !mk.at.After(c.T) {
				nc = true
			}
		}
		for _, r := range it.Runs {
			if r.ReworkOf != "" && !r.From.After(c.T) {
				mt.rework++
			}
		}
		if firstZT3 != "" || nc {
			mt.zt3Total++
			if firstZT3 != "" && !nc && !c.hasRework(it) {
				mt.zt3First++
			}
		}
		if insp {
			mt.inspected++
			mt.inspItems = append(mt.inspItems, it)
		}
		if nc {
			mt.withNC++
			mt.ncItems = append(mt.ncItems, it)
		}
		st := c.S(it)
		if st.Position == "completed" {
			for _, mv := range it.moves {
				if mv.step == "final.released" {
					mt.leadSum += int(mv.at.Sub(it.Launch).Minutes())
					mt.leadN++
				}
			}
		}
	}
	for _, n := range c.M.NCs {
		if n.ConfirmedAt.After(c.T) || len(n.Spec.Items) > 0 {
			continue
		}
		mt.ncTotal++
		if n.Spec.Cause != nil && !n.Spec.Cause.At.Time().After(c.T) {
			mt.causeOK++
		}
		if n.Spec.Cause != nil && n.Spec.Cause.Category == "incoming" {
			mt.incoming += len(n.Spec.Defects)
			continue
		}
		for _, d := range n.Spec.Defects {
			mt.defects[d.Kind]++
			if len(n.Items) > 0 {
				if w := c.weldBefore(n.Items[0], n.SignalAt); w != nil {
					mt.byLine[w.Line]++
				}
			}
		}
	}
	return mt
}

func sum(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func (c *Ctx) period() analyticsapp.Period {
	return analyticsapp.Period{Kind: "day", From: c.dayStart(), To: c.T}
}

func renderAnalytics(c *Ctx) []loader.Response {
	mt := c.metrics()
	fpy := 10000
	if mt.zt3Total > 0 {
		fpy = mt.zt3First * 10000 / mt.zt3Total
	}
	causeBP := 0
	if mt.ncTotal > 0 {
		causeBP = mt.causeOK * 10000 / mt.ncTotal
	}
	lead := 0
	if mt.leadN > 0 {
		lead = mt.leadSum / mt.leadN
	}
	wait := 0
	for _, v := range c.Counters() {
		wait += int(v.WaitSum.Minutes())
	}
	tiles := analyticsapp.MetricTileList{Period: c.period(), Items: []analyticsapp.MetricTile{
		{MetricID: "inspected_items", Title: "Проверено изделий", Value: pcs(mt.inspected)},
		{MetricID: "items_with_confirmed_nc", Title: "Изделия с подтверждёнными несоответствиями", Value: pcs(mt.withNC)},
		{MetricID: "first_pass_yield", Title: "Прохождение контроля с первого раза (ЗТ-3)", Value: bp(fpy), Unknown: mt.zt3Total == 0},
		{MetricID: "defects_by_type", Title: "Дефекты сварки (подтверждённые)", Value: pcs(sum(mt.defects))},
		{MetricID: "cause_established", Title: "Причина установлена", Value: bp(causeBP), Unknown: mt.ncTotal == 0},
		{MetricID: "lead_time", Title: "Время детали в системе (выпущенные)", Value: mins(lead, "computed_by_system"), Unknown: mt.leadN == 0},
		{MetricID: "waiting_time", Title: "Ожидание изделий в очередях", Value: mins(wait, "computed_by_system")},
	}}
	cs := c.Counters()
	ncs := analyticsapp.NodeCounterSet{Period: c.period(), ProcessVersionID: ProcessVersionID, Counters: []analyticsapp.NodeCounters{}, Anomalies: []analyticsapp.NodeAnomaly{}, DataGaps: c.dataGaps(), BasisSeq: c.Seq()}
	for _, mc := range c.mapCounters(cs) {
		ncs.Counters = append(ncs.Counters, analyticsapp.NodeCounters{StepKey: mc.StepKey, Queue: mc.Queue, InProgress: mc.InProgress, Passed: mc.Passed, Defects: mc.Defects, Nonconformities: mc.Nonconformities})
	}
	for _, a := range c.anomalies(cs) {
		ncs.Anomalies = append(ncs.Anomalies, analyticsapp.NodeAnomaly{StepKey: a.StepKey, Kind: a.Kind, Threshold: ptr(a.Threshold)})
	}
	if k, w := c.bottleneck(cs); k != "" {
		ncs.Bottleneck = &analyticsapp.Bottleneck{StepKey: k, Wait: ptr(w)}
	}
	// Полный набор показателей (FR-86…FR-89).
	defSlices := []analyticsapp.MetricSlice{}
	for _, k := range sortedKeys(mt.defects) {
		defSlices = append(defSlices, analyticsapp.MetricSlice{Dimension: "defect_type", Key: k, Label: defectTitle(k), Value: pcs(mt.defects[k])})
	}
	for _, k := range sortedKeys(mt.byLine) {
		defSlices = append(defSlices, analyticsapp.MetricSlice{Dimension: "location", Key: k, Label: map[string]string{"LINE-FL-1": "Линия ФЛ-100 № 1", "LINE-FL-2": "Линия ФЛ-100 № 2"}[k], Value: pcs(mt.byLine[k])})
	}
	welds := map[string]int{}
	for _, it := range c.M.Items {
		for _, r := range it.Runs {
			if r.Kind == "welding" && !r.From.After(c.T) {
				welds[r.Performer]++
			}
		}
	}
	people := []analyticsapp.MetricSlice{}
	errs := []analyticsapp.MetricSlice{}
	for _, w := range []string{"W21", "W22"} {
		people = append(people, analyticsapp.MetricSlice{Dimension: "performer", Key: w, Label: c.M.personName(w), Value: pcs(welds[w])})
		errs = append(errs, analyticsapp.MetricSlice{Dimension: "performer", Key: w, Label: c.M.personName(w), Value: pcs(0)})
	}
	downtime := 0
	for _, h := range c.M.Spec.ProcessHolds {
		if !h.Set.Time().After(c.T) {
			downtime = int(c.T.Sub(h.Set.Time()).Minutes())
		}
	}
	ov := analyticsapp.AnalyticsOverview{Period: c.period(), BasisSeq: c.Seq(), Items: []analyticsapp.MetricRow{
		{MetricID: "inspected_items", Title: "Проверено изделий", Group: "inspection", Total: pcs(mt.inspected), Slices: []analyticsapp.MetricSlice{}},
		{MetricID: "items_with_confirmed_nc", Title: "Изделия с подтверждёнными несоответствиями", Group: "defects", Total: pcs(mt.withNC), Slices: []analyticsapp.MetricSlice{}},
		{MetricID: "confirmed_defects", Title: "Подтверждённые дефекты (производственные)", Group: "defects", Total: pcs(sum(mt.defects)), Slices: defSlices},
		{MetricID: "incoming_defects", Title: "Входной брак (отдельно от производственных ошибок)", Group: "causes", Total: pcs(mt.incoming),
			Slices: []analyticsapp.MetricSlice{{Dimension: "origin", Key: "incoming", Label: "Поставщик-3, партия П-117", Value: pcs(mt.incoming)}}},
		{MetricID: "unable_to_assess", Title: "Оценка невозможна (отдельная корзина)", Group: "inspection", Total: pcs(mt.unable), Slices: []analyticsapp.MetricSlice{}},
		{MetricID: "first_pass_yield", Title: "Прохождение контроля с первого раза (ЗТ-3)", Group: "inspection", Total: bp(fpy), Unknown: mt.zt3Total == 0, Slices: []analyticsapp.MetricSlice{}},
		{MetricID: "cause_established", Title: "Причина установлена", Group: "causes", Total: bp(causeBP), Slices: []analyticsapp.MetricSlice{}},
		{MetricID: "rework_runs", Title: "Повторные выполнения операций", Group: "defects", Total: pcs(mt.rework), Slices: []analyticsapp.MetricSlice{}},
		{MetricID: "comparable_welds", Title: "Сопоставимые сварки по исполнителям", Group: "comparison", Total: pcs(sum(welds)), Slices: people},
		{MetricID: "confirmed_performer_errors", Title: "Подтверждённые ошибки исполнителей", Group: "people", Total: pcs(0), Slices: errs},
		{MetricID: "equipment_downtime", Title: "Простой оборудования: остановка по качеству", Group: "equipment", Total: mins(downtime, "computed_by_system"),
			Slices: []analyticsapp.MetricSlice{{Dimension: "equipment", Key: "IS-2", Label: "Сварочный источник ИС-2", Value: mins(downtime, "computed_by_system")}}},
		{MetricID: "lead_time", Title: "Время детали в системе", Group: "time", Total: mins(lead, "computed_by_system"), Unknown: mt.leadN == 0, Slices: []analyticsapp.MetricSlice{}},
	}}
	out := []loader.Response{
		resp("analytics.tile.list", tiles),
		resp("analytics.node_counters.read", ncs),
		resp("analytics.overview.read", ov),
	}
	for _, d := range []struct {
		id    string
		items []*Item
	}{{"items_with_confirmed_nc", mt.ncItems}, {"inspected_items", mt.inspItems}} {
		dd := analyticsapp.MetricDrilldown{MetricID: d.id, Period: c.period(), Total: pcs(len(d.items)), Items: []analyticsapp.ContributionRow{}}
		for _, it := range d.items {
			row := analyticsapp.ContributionRow{ItemID: FullID(it.ID), Label: it.Label, SliceKey: c.S(it).Line, Value: pcs(1), SourceEventIDs: []string{}, SourceKinds: []string{}}
			for _, e := range c.itemEvents(it) {
				if (d.id == "items_with_confirmed_nc" && strings.HasPrefix(e.Type, "decision.nonconformity")) || (d.id == "inspected_items" && e.Type == "decision.presentation.resolved") {
					row.SourceEventIDs = append(row.SourceEventIDs, e.ID)
					if !slices.Contains(row.SourceKinds, e.Kind) {
						row.SourceKinds = append(row.SourceKinds, e.Kind)
					}
				}
			}
			dd.Items = append(dd.Items, row)
		}
		out = append(out, resp("analytics.metric.drilldown", dd, "metric_id", d.id))
	}
	// Контрольная карта тока сварки по выполнениям (FR-5): центр 160 А, границы ±10 А.
	cc := analyticsapp.ControlChart{StepKey: "welding.weld", MetricID: "current_a", Center: analyticsapp.MetricValue{Value: 160, Unit: "A"}, Upper: analyticsapp.MetricValue{Value: 170, Unit: "A"}, Lower: analyticsapp.MetricValue{Value: 150, Unit: "A"}, Points: []analyticsapp.ControlChartPoint{}}
	var runs []*OpRun
	for _, it := range c.M.Items {
		for _, r := range it.Runs {
			if r.Kind == "welding" && !r.LogLost && !r.LogReceived.IsZero() && !r.LogReceived.After(c.T) && r.To.After(c.M.Steps[0]) {
				runs = append(runs, r)
			}
		}
	}
	slices.SortFunc(runs, func(a, b *OpRun) int { return a.To.Compare(b.To) })
	for _, r := range runs {
		cc.Points = append(cc.Points, analyticsapp.ControlChartPoint{At: r.To, Value: analyticsapp.MetricValue{Value: int64(r.CurrentA[1]), Unit: "A"}, OutOfControl: r.CurrentA[1] > 170,
			Ref: &platform.DrillRef{Entity: platform.EntityItem, ID: FullID(r.Item.ID)}})
	}
	out = append(out, resp("analytics.control_chart.read", cc), resp("analytics.control_chart.read", cc, "step_key", "welding.weld", "metric_id", "current_a"))
	return out
}
