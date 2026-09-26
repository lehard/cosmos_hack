package world

import (
	"slices"
	"time"

	analyticsapp "ant/internal/application/analytics"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
)

// Показатели заготовок (эпик 25): из данных мира, в той же форме, что live
// (application/analytics). Каждый показатель — список вкладов (изделие или
// объект, срезы, значение, записи журнала мира); из одного списка строятся
// итог, срезы и раскрытие итога и каждого среза, поэтому сумма строк
// раскрытия штучного показателя равна итогу (AD-45). Дефекты и изделия с
// дефектами — раздельно; входной брак — отдельно от производственных;
// «под подозрением» браком не считается (кейс §2.4, §5.2).

// aggKind — вид агрегата показателя.
type aggKind int

const (
	aggSum   aggKind = iota // штуки
	aggShare                // доля числителя, б. п.
	aggMean                 // среднее время, мин
	aggTime                 // суммарное время, мин
)

// aSlice — срез вклада: измерение, значение, подпись.
type aSlice struct{ dim, key, label string }

// aEntry — вклад в показатель заготовок.
type aEntry struct {
	item, label string
	ref         *platform.DrillRef
	at          time.Time
	value       int64
	num         bool
	slices      []aSlice
	events      []*Event
}

// aMetric — показатель заготовок.
type aMetric struct {
	id, title, group, counts, account string
	agg                               aggKind
	origin, meaning, note             string
	// fixed — срезы, которые показываются и с нулём (исполнители в сравнении).
	fixed   []aSlice
	entries []aEntry
}

func pcs(n int) analyticsapp.MetricValue {
	return analyticsapp.MetricValue{Value: int64(n), Scale: 0, Unit: "pcs"}
}
func bp(n int) analyticsapp.MetricValue {
	return analyticsapp.MetricValue{Value: int64(n), Scale: 0, Unit: "bp"}
}

// value — агрегат вкладов.
func (m aMetric) value(es []aEntry) (analyticsapp.MetricValue, bool) {
	switch m.agg {
	case aggShare:
		num := 0
		for _, e := range es {
			if e.num {
				num++
			}
		}
		if len(es) == 0 {
			return bp(0), true
		}
		return bp(num * 10000 / len(es)), false
	case aggMean, aggTime:
		var sum int64
		for _, e := range es {
			sum += e.value
		}
		v := analyticsapp.MetricValue{Unit: "min", Origin: ptr(m.origin), Meaning: ptr(m.meaning)}
		if m.note != "" {
			v.MeaningNote = ptr(m.note)
		}
		if len(es) == 0 {
			return v, m.agg == aggMean
		}
		if m.agg == aggMean {
			sum /= int64(len(es))
		}
		v.Value = sum
		return v, false
	}
	var sum int64
	for _, e := range es {
		sum += e.value
	}
	return analyticsapp.MetricValue{Value: sum, Unit: "pcs"}, false
}

func sliceKeyOf(s aSlice) string { return s.dim + ":" + s.key }

// row — строка раздела «Аналитика».
func (m aMetric) row() analyticsapp.MetricRow {
	total, unknown := m.value(m.entries)
	r := analyticsapp.MetricRow{MetricID: m.id, Title: m.title, Group: m.group, Total: total, Unknown: unknown, Slices: []analyticsapp.MetricSlice{}}
	if m.counts != "" {
		r.Counts = ptr(m.counts)
	}
	if m.account != "" {
		r.Account = ptr(m.account)
	}
	for _, s := range m.sliceList() {
		v, _ := m.value(m.inSlice(sliceKeyOf(s)))
		r.Slices = append(r.Slices, analyticsapp.MetricSlice{Dimension: s.dim, Key: sliceKeyOf(s), Label: s.label, Value: v})
	}
	return r
}

// sliceList — срезы показателя: заданные и встреченные во вкладах, по порядку.
func (m aMetric) sliceList() []aSlice {
	out := slices.Clone(m.fixed)
	for _, e := range m.entries {
		for _, s := range e.slices {
			if s.key != "" && !slices.ContainsFunc(out, func(x aSlice) bool { return sliceKeyOf(x) == sliceKeyOf(s) }) {
				out = append(out, s)
			}
		}
	}
	return out
}

func (m aMetric) inSlice(key string) []aEntry {
	if key == "" {
		return m.entries
	}
	var out []aEntry
	for _, e := range m.entries {
		if slices.ContainsFunc(e.slices, func(s aSlice) bool { return sliceKeyOf(s) == key }) {
			out = append(out, e)
		}
	}
	return out
}

// srcKind — вид источника записи для раскрытия (FR-140): факт — его
// source_kind; решение человека — ручной ввод; вывод системы — system.
func srcKind(e *Event) string {
	if k := sourceKindOf(e); k != "" {
		return k
	}
	if e.Kind == "decision" {
		return "manual_entry"
	}
	if e.Kind == "fact" {
		return "manual_entry"
	}
	return "system"
}

// drilldown — раскрытие итога (key пусто) или среза.
func (m aMetric) drilldown(c *Ctx, key string) analyticsapp.MetricDrilldown {
	es := m.inSlice(key)
	total, _ := m.value(es)
	dd := analyticsapp.MetricDrilldown{MetricID: m.id, Period: c.period(), Total: total, Items: []analyticsapp.ContributionRow{}}
	for _, e := range es {
		sk := key
		if sk == "" && len(e.slices) > 0 {
			sk = sliceKeyOf(e.slices[0])
		}
		v, _ := m.value([]aEntry{e})
		if m.agg == aggShare {
			v = pcs(0)
			if e.num {
				v = pcs(1)
			}
		}
		at := e.at
		row := analyticsapp.ContributionRow{ItemID: e.item, Label: e.label, SliceKey: sk, Value: v, SourceEventIDs: []string{}, SourceKinds: []string{}, Ref: e.ref}
		if !at.IsZero() {
			row.At = &at
		}
		for _, ev := range e.events {
			if !slices.Contains(row.SourceEventIDs, ev.ID) {
				row.SourceEventIDs = append(row.SourceEventIDs, ev.ID)
			}
			if k := srcKind(ev); !slices.Contains(row.SourceKinds, k) {
				row.SourceKinds = append(row.SourceKinds, k)
			}
		}
		dd.Items = append(dd.Items, row)
	}
	return dd
}

func (c *Ctx) period() analyticsapp.Period {
	return analyticsapp.Period{Kind: "day", From: c.dayStart(), To: c.T}
}

// visible — запись мира видна на шаге (поздние — с шага записи).
func (c *Ctx) visible(e *Event) bool { return e.Step <= c.N && !e.Occurred.After(c.T) }

// eventsWhere — видимые записи мира по условию.
func (c *Ctx) eventsWhere(f func(e *Event) bool) []*Event {
	var out []*Event
	for _, e := range c.M.Events {
		if c.visible(e) && f(e) {
			out = append(out, e)
		}
	}
	return out
}

func (c *Ctx) ncEventsOf(n *NC) []*Event {
	return c.eventsWhere(func(e *Event) bool { return e.Params["nc_id"] == n.ID })
}

func itemEntry(it *Item) aEntry {
	return aEntry{item: FullID(it.ID), label: it.Label, value: 1}
}

// analyticsMetrics — показатели шага в порядке раздела «Аналитика».
func (c *Ctx) analyticsMetrics() []aMetric {
	lines := map[string]string{"LINE-FL-1": "Линия ФЛ-100 № 1", "LINE-FL-2": "Линия ФЛ-100 № 2"}
	var (
		inspected = aMetric{id: "inspected_items", title: "Проверено изделий", group: "inspection", counts: "items"}
		fpy       = aMetric{id: "first_pass_yield", title: "Прохождение контроля с первого раза (ЗТ-3)", group: "inspection", counts: "items", agg: aggShare}
		unable    = aMetric{id: "unable_to_assess", title: "Оценка невозможна (отдельная корзина)", group: "inspection", counts: "observations"}
		withNC    = aMetric{id: "items_with_confirmed_nc", title: "Изделия с подтверждёнными несоответствиями", group: "defects", counts: "items"}
		defects   = aMetric{id: "confirmed_defects", title: "Подтверждённые дефекты (производственные)", group: "defects", counts: "defects"}
		byType    = aMetric{id: "defects_by_type", title: "Дефекты сварки (подтверждённые)", group: "defects", counts: "defects"}
		rework    = aMetric{id: "rework_runs", title: "Повторные выполнения операций", group: "defects", counts: "operations"}
		incoming  = aMetric{id: "incoming_defects", title: "Входной брак (отдельно от производственных ошибок)", group: "causes", counts: "defects", account: "incoming"}
		cause     = aMetric{id: "cause_established", title: "Причина установлена", group: "causes", counts: "nonconformities", agg: aggShare}
		welds     = aMetric{id: "comparable_welds", title: "Сопоставимые сварки по исполнителям", group: "comparison", counts: "operations"}
		errs      = aMetric{id: "confirmed_performer_errors", title: "Подтверждённые ошибки исполнителей", group: "people", counts: "nonconformities", account: "performer"}
		downtime  = aMetric{id: "equipment_downtime", title: "Простой оборудования: остановка по качеству", group: "equipment", counts: "time", account: "equipment", agg: aggTime,
			origin: "computed_by_system", meaning: "other", note: "от остановки точки процесса до конца периода (остановка не снята)"}
		lead = aMetric{id: "lead_time", title: "Время детали в системе (выпущенные)", group: "time", counts: "time", agg: aggMean,
			origin: "computed_by_system", meaning: "other", note: "от запуска изделия до сдачи на склад готовой продукции"}
		wait = aMetric{id: "waiting_time", title: "Ожидание изделий в очередях", group: "time", counts: "time", agg: aggTime,
			origin: "computed_by_system", meaning: "other", note: "время изделий в очереди узла до начала операции или решения, в пределах периода"}
	)
	for _, w := range []string{"W21", "W22"} {
		s := aSlice{dim: "performer", key: w, label: c.M.personName(w)}
		welds.fixed = append(welds.fixed, s)
		errs.fixed = append(errs.fixed, s)
	}
	for _, l := range []string{"LINE-FL-1", "LINE-FL-2"} {
		defects.fixed = append(defects.fixed, aSlice{dim: "location", key: l, label: lines[l]})
	}
	ncOf := map[*Item]*NC{}
	for _, n := range c.M.NCs {
		if !n.ConfirmedAt.After(c.T) && len(n.Items) > 0 {
			if _, ok := ncOf[n.Items[0]]; !ok {
				ncOf[n.Items[0]] = n
			}
		}
	}
	for _, it := range c.Existing() {
		evs := c.itemEvents(it)
		var resolved, firstZT3 []*Event
		for _, e := range evs {
			if e.Type == "decision.presentation.resolved" {
				resolved = append(resolved, e)
				if e.StepKey == "welding.zt3_acceptance" && len(firstZT3) == 0 {
					firstZT3 = append(firstZT3, e)
				}
			}
			if e.Type == "inspection.result.recorded" && e.Params["outcome"] == "unable_to_assess" {
				ue := itemEntry(it)
				ue.at, ue.events, ue.slices = e.Occurred, []*Event{e}, []aSlice{{dim: "step", key: e.StepKey, label: stepName(e.StepKey)}}
				unable.entries = append(unable.entries, ue)
			}
		}
		nc := false
		var ncAt time.Time
		for _, mk := range it.marks {
			if mk.axis == "quality" && mk.value == "nonconforming" && !mk.at.After(c.T) {
				nc, ncAt = true, mk.at
			}
		}
		if len(resolved) > 0 {
			e := itemEntry(it)
			e.at, e.events = resolved[0].Occurred, resolved
			inspected.entries = append(inspected.entries, e)
		}
		if nc {
			e := itemEntry(it)
			e.at = ncAt
			e.events = c.eventsWhere(func(x *Event) bool {
				return x.Item == it && len(x.Type) > 23 && x.Type[:23] == "decision.nonconformity."
			})
			if n := ncOf[it]; n != nil {
				origin := "production"
				if n.Spec.Cause != nil && n.Spec.Cause.Category == "incoming" && !n.Spec.Cause.At.Time().After(c.T) {
					origin = "incoming"
				}
				e.slices = []aSlice{{dim: "step", key: n.StepKey, label: stepName(n.StepKey)}, {dim: "origin", key: origin, label: originTitle(origin)}}
			}
			withNC.entries = append(withNC.entries, e)
		}
		if len(firstZT3) > 0 || nc {
			e := itemEntry(it)
			e.num = len(firstZT3) > 0 && firstZT3[0].Params["outcome"] != "reject" && !nc && !c.hasRework(it)
			e.events = firstZT3
			if len(firstZT3) > 0 {
				e.at = firstZT3[0].Occurred
			} else {
				e.at = ncAt
				e.events = c.eventsWhere(func(x *Event) bool { return x.Item == it && x.Type == "decision.nonconformity.confirmed" })
			}
			e.slices = []aSlice{{dim: "step", key: "welding.zt3_acceptance", label: stepName("welding.zt3_acceptance")}}
			fpy.entries = append(fpy.entries, e)
		}
		for _, r := range it.Runs {
			if r.From.After(c.T) {
				continue
			}
			runEvs := c.eventsWhere(func(x *Event) bool { return x.Item == it && x.Params["operation_run_id"] == r.ID })
			if len(runEvs) == 0 {
				runEvs = c.eventsWhere(func(x *Event) bool {
					return x.Item == it && x.StepKey == r.StepKey && x.Type == "operation.run.started"
				})
			}
			if r.ReworkOf != "" {
				e := itemEntry(it)
				e.at, e.events = r.From, runEvs
				e.slices = []aSlice{{dim: "step", key: r.StepKey, label: stepName(r.StepKey)}, {dim: "performer", key: r.Performer, label: c.M.personName(r.Performer)}}
				rework.entries = append(rework.entries, e)
			}
			if r.Kind == "welding" {
				e := itemEntry(it)
				e.at, e.events = r.From, runEvs
				e.slices = []aSlice{{dim: "performer", key: r.Performer, label: c.M.personName(r.Performer)}}
				welds.entries = append(welds.entries, e)
			}
		}
		st := c.S(it)
		if st.Position == "completed" {
			for _, mv := range it.moves {
				if mv.step == "final.released" {
					e := itemEntry(it)
					e.at, e.value = mv.at, int64(mv.at.Sub(it.Launch).Minutes())
					e.events = c.eventsWhere(func(x *Event) bool {
						return x.Item == it && (x.Type == "item.item.registered" || x.Type == "item.release.recorded" || x.StepKey == "final.released")
					})
					lead.entries = append(lead.entries, e)
				}
			}
		}
		if st.Position != "in_progress" && st.Position != "at_inspection" && st.Position != "completed" && st.Exists {
			var since time.Time
			for _, mv := range it.moves {
				if !mv.at.After(c.T) && mv.step == st.Step {
					if since.IsZero() {
						since = mv.at
					}
				} else if !mv.at.After(c.T) {
					since = time.Time{}
				}
			}
			if !since.IsZero() {
				// Время ожидания в пределах периода (сутки на часах шага), как у live.
				from := since
				if day := c.dayStart(); from.Before(day) {
					from = day
				}
				e := itemEntry(it)
				e.at, e.value = since, int64(c.T.Sub(from).Minutes())
				e.slices = []aSlice{{dim: "step", key: st.Step, label: stepName(st.Step)}}
				e.events = c.eventsWhere(func(x *Event) bool { return x.Item == it && x.StepKey == st.Step })
				if len(e.events) == 0 {
					e.events = lastEvents(evs, 1)
				}
				wait.entries = append(wait.entries, e)
			}
		}
	}
	for _, n := range c.M.NCs {
		if n.ConfirmedAt.After(c.T) || len(n.Spec.Items) > 0 {
			continue
		}
		nevs := c.ncEventsOf(n)
		var it *Item
		if len(n.Items) > 0 {
			it = n.Items[0]
		}
		ce := aEntry{item: n.ID, label: n.Number, ref: &platform.DrillRef{Entity: platform.EntityNonconformity, ID: n.ID}, at: n.ConfirmedAt, value: 1, events: nevs}
		if it != nil {
			ce.item, ce.label = FullID(it.ID), it.Label+" · "+n.Number
		}
		cat := "pending"
		if n.Spec.Cause != nil && !n.Spec.Cause.At.Time().After(c.T) {
			cat, ce.num = n.Spec.Cause.Category, true
		}
		ce.slices = []aSlice{{dim: "cause_category", key: cat, label: causeTitle(cat)}}
		cause.entries = append(cause.entries, ce)
		isIncoming := n.Spec.Cause != nil && n.Spec.Cause.Category == "incoming"
		for _, d := range n.Spec.Defects {
			e := ce
			e.num = false
			e.ref = nil
			if isIncoming {
				e.slices = []aSlice{{dim: "defect_type", key: d.Kind, label: defectTitle(d.Kind)}, {dim: "origin", key: "incoming", label: "Поставщик-3, партия П-117"}}
				incoming.entries = append(incoming.entries, e)
				continue
			}
			e.slices = []aSlice{{dim: "defect_type", key: d.Kind, label: defectTitle(d.Kind)}}
			byType.entries = append(byType.entries, e)
			if it != nil {
				if w := c.weldBefore(it, n.SignalAt); w != nil {
					e.slices = append(e.slices, aSlice{dim: "location", key: w.Line, label: lines[w.Line]})
				}
			}
			defects.entries = append(defects.entries, e)
		}
	}
	for _, h := range c.M.Spec.ProcessHolds {
		if h.Set.Time().After(c.T) {
			continue
		}
		e := aEntry{item: h.Equipment, label: "Сварочный источник " + h.Equipment, ref: &platform.DrillRef{Entity: platform.EntityEquipment, ID: h.Equipment},
			at: h.Set.Time(), value: int64(c.T.Sub(h.Set.Time()).Minutes())}
		e.events = c.eventsWhere(func(x *Event) bool {
			return (x.Type == "decision.process_hold.set" && x.Params["equipment_id"] == h.Equipment) || (x.Type == "equipment.state.changed" && x.Entity.ID == h.Equipment && !x.Occurred.Before(h.Set.Time()))
		})
		e.slices = []aSlice{{dim: "equipment", key: h.Equipment, label: "Сварочный источник " + h.Equipment}}
		downtime.entries = append(downtime.entries, e)
	}
	return []aMetric{inspected, withNC, defects, incoming, unable, fpy, cause, rework, welds, errs, downtime, lead, wait, byType}
}

// metricTotal — итог штучного показателя заготовок (для табло автосверки).
func (c *Ctx) metricTotal(id string) int {
	for _, m := range c.analyticsMetrics() {
		if m.id == id {
			v, _ := m.value(m.entries)
			return int(v.Value)
		}
	}
	return 0
}

func lastEvents(evs []*Event, n int) []*Event {
	if len(evs) <= n {
		return evs
	}
	return evs[len(evs)-n:]
}

func stepName(k string) string {
	if n, ok := stepTitle[k]; ok {
		return n
	}
	return k
}

func originTitle(o string) string {
	return map[string]string{"incoming": "Входной брак", "production": "Производственные"}[o]
}

func causeTitle(c string) string {
	if t, ok := map[string]string{"incoming": "Входной брак", "equipment": "Оборудование", "performer": "Исполнитель", "pending": "Разбор не завершён", "not_established": "Не установлена"}[c]; ok {
		return t
	}
	return c
}

func renderAnalytics(c *Ctx) []loader.Response {
	ms := c.analyticsMetrics()
	byID := map[string]aMetric{}
	for _, m := range ms {
		byID[m.id] = m
	}
	tiles := analyticsapp.MetricTileList{Period: c.period(), Items: []analyticsapp.MetricTile{}}
	for _, id := range []string{"inspected_items", "items_with_confirmed_nc", "first_pass_yield", "defects_by_type", "cause_established", "lead_time", "waiting_time"} {
		m := byID[id]
		v, unknown := m.value(m.entries)
		tiles.Items = append(tiles.Items, analyticsapp.MetricTile{MetricID: id, Title: m.title, Value: v, Unknown: unknown})
	}
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
	ov := analyticsapp.AnalyticsOverview{Period: c.period(), BasisSeq: c.Seq(), Items: []analyticsapp.MetricRow{}}
	for _, m := range ms {
		if m.id == "defects_by_type" {
			continue // плитка; в разделе — «Подтверждённые дефекты» со срезом по видам
		}
		ov.Items = append(ov.Items, m.row())
	}
	out := []loader.Response{
		resp("analytics.tile.list", tiles),
		resp("analytics.node_counters.read", ncs),
		resp("analytics.overview.read", ov),
	}
	// Раскрытие каждого числа: итог и каждый срез каждого показателя (FR-7, AD-45).
	for _, m := range ms {
		out = append(out, resp("analytics.metric.drilldown", m.drilldown(c, ""), "metric_id", m.id))
		for _, s := range m.sliceList() {
			k := sliceKeyOf(s)
			out = append(out, resp("analytics.metric.drilldown", m.drilldown(c, k), "metric_id", m.id, "slice", k))
		}
	}
	// Контрольная карта тока сварки по выполнениям (FR-5): центр 160 А, границы ±10 А.
	cc := analyticsapp.ControlChart{StepKey: "welding.weld", MetricID: "current_a", Title: "Ток сварки (максимум за выполнение)", ChartKind: ptr("xmr"),
		Center: analyticsapp.MetricValue{Value: 160, Unit: "A"}, Upper: analyticsapp.MetricValue{Value: 170, Unit: "A"}, Lower: analyticsapp.MetricValue{Value: 150, Unit: "A"}, Points: []analyticsapp.ControlChartPoint{}}
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
