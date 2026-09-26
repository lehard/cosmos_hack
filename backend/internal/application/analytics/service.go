package analytics

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	domain "ant/internal/domain/analytics"
)

// Service — реализация live ведущих портов модуля analytics (AD-36, AD-45):
// показатели — агрегаты строк вклада изделий и глобальных проекций,
// прочитанных через ведомый порт Store. Ничего не пишет: строки пишет
// движок эффектами в транзакции journal.Append.
type Service struct {
	store Store
	clock Clock
	norms Norms
	// shifts — график смен (эпик 19); nil — смены по 8 ч.
	shifts Shifts
	// Location — часовой пояс границ смен и суток.
	Location *time.Location
}

// NewService создаёт реализацию live. Без Store (выгрузка OpenAPI, тесты
// сборки API) операции отвечают 501.
func NewService(opts ...Option) *Service {
	s := &Service{norms: DefaultNorms{}, Location: Moscow}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Option — настройка Service.
type Option func(*Service)

// WithStore — хранилище строк вклада и проекций.
func WithStore(st Store) Option { return func(s *Service) { s.store = st } }

// WithClock — доменные часы (AD-37).
func WithClock(c Clock) Option { return func(s *Service) { s.clock = c } }

// WithShifts — график смен справочника (FR-81, эпик 19).
func WithShifts(sh Shifts) Option { return func(s *Service) { s.shifts = sh } }

// WithNorms — нормы узлов (FR-5, FR-12).
func WithNorms(n Norms) Option { return func(s *Service) { s.norms = n } }

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// data — всё, что нужно запросу: строки прогона, простои, инциденты, момент.
type data struct {
	rows      []domain.Row
	down      []domain.Row
	incidents []domain.Incident
	now       time.Time
	win       window
	// shiftOf — смена строки для среза «смена»; nil — графика нет.
	shiftOf ShiftLookup
}

// load читает строки и проекции и выбирает период на момент m (AD-22: «как
// было» — строки с моментом не позже as_of; ось recorded в MVP читается так
// же — см. отчёт эпика 25).
func (s *Service) load(ctx context.Context, op string, p PeriodQuery, m platform.Moment) (*data, error) {
	if s.store == nil {
		return nil, platform.NotImplemented(op)
	}
	var now time.Time
	if m.AsOf != nil {
		now = m.AsOf.UTC()
	} else if s.clock != nil {
		t, err := s.clock.Now(appjournal.WithRun(ctx, m.RunID))
		if err != nil {
			// Часы сценария без тика — «сейчас» системное (прогон ещё не начал время).
			t = time.Now()
		}
		now = t.UTC()
	} else {
		now = time.Now().UTC()
	}
	win, err := resolvePeriod(p, now, s.Location)
	if err != nil {
		return nil, err
	}
	var shiftOf ShiftLookup
	if s.shifts != nil {
		// Смены — по графику справочника (FR-81); без графика — по 8 ч.
		if shiftOf, err = s.shifts.Schedule(ctx, m.RunID); err != nil {
			return nil, err
		}
		win = shiftWindow(win, p, now, shiftOf)
	}
	rows, err := s.store.Rows(ctx)
	if err != nil {
		return nil, err
	}
	eqs, err := s.store.Equipment(ctx)
	if err != nil {
		return nil, err
	}
	incs, err := s.store.Incidents(ctx)
	if err != nil {
		return nil, err
	}
	d := &data{now: now, win: win, shiftOf: shiftOf}
	for _, r := range rows {
		if inRun(r.Dims.Run, m.RunID) {
			d.rows = append(d.rows, r)
		}
	}
	for _, e := range eqs {
		if !inRun(e.RunID, m.RunID) {
			continue
		}
		d.down = append(d.down, e.Downtime()...)
	}
	for _, inc := range incs {
		if inRun(inc.RunID, m.RunID) {
			d.incidents = append(d.incidents, inc)
		}
	}
	return d, nil
}

// inRun — строка относится к запрошенному прогону (AD-38); без прогона — все.
func inRun(rowRun, want string) bool { return want == "" || rowRun == want }

// compute — итог показателя и его срезы за [from, to].
func (d *data) compute(def metricDef, from, to time.Time) MetricRow {
	v := newView(d.rows, d.down, d.incidents, from, to)
	es := def.entries(v)
	row := MetricRow{MetricID: def.id, Title: def.title, Group: def.group, Slices: []MetricSlice{}}
	if def.counts != "" {
		row.Counts = ptr(def.counts)
	}
	if def.account != "" {
		row.Account = ptr(def.account)
	}
	row.Total, row.Unknown = aggregate(def, es)
	base := es
	if def.sliceEntries != nil {
		base = def.sliceEntries(v)
	}
	for _, dim := range def.dims {
		groups := map[string][]entry{}
		labels := map[string]string{}
		var dimension string
		for _, e := range base {
			dn, key, label := dimValue(dim, e.row)
			if dim == "shift" {
				dn, key, label = d.shiftDim(e.row)
			}
			if key == "" {
				continue
			}
			dimension = dn
			groups[key] = append(groups[key], e)
			if label == "" {
				label = key
			}
			labels[key] = label
		}
		for _, k := range slices.Sorted(maps.Keys(groups)) {
			val, _ := aggregate(def, groups[k])
			row.Slices = append(row.Slices, MetricSlice{Dimension: dimension, Key: sliceKey(dimension, k), Label: labels[k], Value: val})
		}
	}
	return row
}

// aggregate — итог показателя по вкладам: штуки, доля (б. п.), среднее или
// сумма времени (мин) с происхождением и смыслом интервала.
func aggregate(def metricDef, es []entry) (MetricValue, bool) {
	switch def.agg {
	case aggShare:
		var n, num int64
		for _, e := range es {
			n++
			if e.num {
				num++
			}
		}
		if n == 0 {
			return MetricValue{Unit: "bp"}, true
		}
		return MetricValue{Value: num * 10000 / n, Unit: "bp"}, false
	case aggMean, aggTime:
		var sum, n int64
		for _, e := range es {
			sum += e.value
			n++
		}
		v := MetricValue{Unit: "min"}
		durationMeta(&v, def, es)
		if n == 0 {
			return v, def.agg == aggMean
		}
		if def.agg == aggMean {
			v.Value = (sum/n + 30) / 60
		} else {
			v.Value = (sum + 30) / 60
		}
		return v, false
	}
	var sum int64
	for _, e := range es {
		sum += e.value
	}
	return MetricValue{Value: sum, Unit: "pcs"}, false
}

// durationMeta — происхождение и смысл интервала длительности (соглашение
// «Длительности», FR-88): одинаковые у всех строк — их значение; разные —
// «смешанное» происхождение и «иной» смысл с пояснением.
func durationMeta(v *MetricValue, def metricDef, es []entry) {
	origins, meanings := map[string]bool{}, map[string]bool{}
	for _, e := range es {
		origins[e.row.Dims.DurationOrigin] = true
		meanings[e.row.Dims.Meaning] = true
	}
	switch {
	case len(origins) == 1 && origins[domain.OriginSource]:
		v.Origin = ptr("reported_by_source")
	case len(origins) > 1:
		v.Origin = ptr("mixed")
	default:
		v.Origin = ptr("computed_by_system")
	}
	meaning := def.meaning
	if len(meanings) == 1 {
		for m := range meanings {
			if m != "" {
				meaning = m
			}
		}
	} else if len(meanings) > 1 {
		meaning = domain.MeaningOther
		v.MeaningNote = ptr("разные смыслы интервала у строк: активная обработка и полное время на участке — раскройте показатель")
	}
	if meaning == "" {
		meaning = domain.MeaningOther
	}
	v.Meaning = ptr(meaning)
	if def.note != "" && v.MeaningNote == nil {
		v.MeaningNote = ptr(def.note)
	}
}

func ptr[T any](v T) *T { return &v }

// Tiles — плитки стола руководителя (analytics.tile.list) со значением за
// прошлый такой же период.
func (s *Service) Tiles(ctx context.Context, p PeriodQuery, m platform.Moment) (MetricTileList, error) {
	d, err := s.load(ctx, "analytics.tile.list", p, m)
	if err != nil {
		return MetricTileList{}, err
	}
	out := MetricTileList{Period: d.win.Period, Items: []MetricTile{}}
	for _, id := range tileIDs {
		def, _ := lookup(id)
		cur := d.compute(def, d.win.From, d.win.To)
		prev := d.compute(def, d.win.prevFrom, d.win.prevTo)
		t := MetricTile{MetricID: id, Title: def.title, Value: cur.Total, Unknown: cur.Unknown}
		if !prev.Unknown {
			pv := prev.Total
			t.Previous = &pv
		}
		out.Items = append(out.Items, t)
	}
	return out, nil
}

// Overview — полный набор показателей кейса (analytics.overview.read).
func (s *Service) Overview(ctx context.Context, p PeriodQuery, m platform.Moment) (AnalyticsOverview, error) {
	d, err := s.load(ctx, "analytics.overview.read", p, m)
	if err != nil {
		return AnalyticsOverview{}, err
	}
	out := AnalyticsOverview{Period: d.win.Period, Items: []MetricRow{}, BasisSeq: 0}
	for _, def := range catalog {
		out.Items = append(out.Items, d.compute(def, d.win.From, d.win.To))
	}
	return out, nil
}

// Drilldown — раскрытие показателя до строк вклада изделий и записей
// (analytics.metric.drilldown): строки группируются по изделию (объекту) и
// срезу; для штучных показателей их сумма равна итогу.
func (s *Service) Drilldown(ctx context.Context, metricID, slice string, p PeriodQuery, m platform.Moment, pg platform.Page) (MetricDrilldown, error) {
	if s.store == nil {
		return MetricDrilldown{}, platform.NotImplemented("analytics.metric.drilldown")
	}
	def, ok := lookup(metricID)
	if !ok {
		e := platform.Fail(errcodes.ApiNotFound, "object", "показатель", "id", metricID)
		e.Detail = "Показателя " + metricID + " нет в каталоге аналитики"
		return MetricDrilldown{}, e
	}
	d, err := s.load(ctx, "analytics.metric.drilldown", p, m)
	if err != nil {
		return MetricDrilldown{}, err
	}
	v := newView(d.rows, d.down, d.incidents, d.win.From, d.win.To)
	es := def.entries(v)
	if slice != "" && def.sliceEntries != nil {
		es = def.sliceEntries(v)
	}
	var sel []entry
	for _, e := range es {
		if slice == "" || d.matches(def, e, slice) {
			sel = append(sel, e)
		}
	}
	total, _ := aggregate(def, sel)
	rows := contributionRows(def, sel, slice)
	start, err := cursorOf(pg.Cursor)
	if err != nil {
		return MetricDrilldown{}, err
	}
	limit := pg.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	out := MetricDrilldown{MetricID: metricID, Period: d.win.Period, Total: total, Items: []ContributionRow{}}
	if start < len(rows) {
		end := min(len(rows), start+limit)
		out.Items = rows[start:end]
		if end < len(rows) {
			out.NextCursor = strconv.Itoa(end)
		}
	}
	return out, nil
}

func cursorOf(c string) (int, error) {
	if c == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(c)
	if err != nil || n < 0 {
		return 0, platform.Fail(errcodes.ApiValidationFailed, "field", "cursor", "reason", "неизвестный курсор")
	}
	return n, nil
}

// matches — вклад относится к срезу slice («измерение:значение»).
func (d *data) matches(def metricDef, e entry, slice string) bool {
	for _, dim := range def.dims {
		dn, key, _ := dimValue(dim, e.row)
		if dim == "shift" {
			dn, key, _ = d.shiftDim(e.row)
		}
		if key != "" && sliceKey(dn, key) == slice {
			return true
		}
	}
	return false
}

// contributionRows — строки раскрытия: по изделию (объекту) — сумма вкладов,
// исходные записи и виды их источников (FR-140).
func contributionRows(def metricDef, es []entry, slice string) []ContributionRow {
	type acc struct {
		row  ContributionRow
		sum  int64
		n    int64
		num  int64
		kind map[string]bool
	}
	by := map[string]*acc{}
	var order []string
	for _, e := range es {
		id := e.row.Item
		sk := slice
		if sk == "" && len(def.dims) > 0 {
			if dn, key, _ := dimValue(def.dims[0], e.row); key != "" {
				sk = sliceKey(dn, key)
			}
		}
		k := id + "\x1f" + sk
		a := by[k]
		if a == nil {
			lbl := e.row.Dims.Label
			if lbl == "" {
				lbl = id
			}
			at := e.row.At
			a = &acc{row: ContributionRow{ItemID: id, Label: lbl, SliceKey: sk, SourceEventIDs: []string{}, SourceKinds: []string{}, At: &at, Ref: e.ref},
				kind: map[string]bool{}}
			by[k] = a
			order = append(order, k)
		}
		a.sum += e.value
		a.n++
		if e.num {
			a.num++
		}
		for _, src := range append(slices.Clone(e.row.Sources), e.extra...) {
			if src != "" && !slices.Contains(a.row.SourceEventIDs, src) {
				a.row.SourceEventIDs = append(a.row.SourceEventIDs, src)
			}
		}
		for _, kd := range e.row.Kinds {
			if kd != "" && !slices.Contains(a.row.SourceKinds, kd) {
				a.row.SourceKinds = append(a.row.SourceKinds, kd)
			}
		}
		if e.row.At.Before(*a.row.At) {
			t := e.row.At
			a.row.At = &t
		}
	}
	out := make([]ContributionRow, 0, len(order))
	for _, k := range order {
		a := by[k]
		switch def.agg {
		case aggShare:
			// Для доли строка — изделие знаменателя: 1 — в числителе, 0 — нет.
			a.row.Value = MetricValue{Value: a.num, Unit: "pcs"}
		case aggMean, aggTime:
			v := MetricValue{Unit: "min"}
			if def.agg == aggMean && a.n > 0 {
				v.Value = (a.sum/a.n + 30) / 60
			} else {
				v.Value = (a.sum + 30) / 60
			}
			durationMeta(&v, def, filterItem(es, a.row.ItemID))
			a.row.Value = v
		default:
			a.row.Value = MetricValue{Value: a.sum, Unit: "pcs"}
		}
		out = append(out, a.row)
	}
	slices.SortStableFunc(out, func(x, y ContributionRow) int {
		if c := x.At.Compare(*y.At); c != 0 {
			return c
		}
		if x.ItemID != y.ItemID {
			return cmp(x.ItemID, y.ItemID)
		}
		return cmp(x.SliceKey, y.SliceKey)
	})
	return out
}

func filterItem(es []entry, item string) []entry {
	var out []entry
	for _, e := range es {
		if e.row.Item == item {
			out = append(out, e)
		}
	}
	return out
}

func cmp(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// NodeCounters — счётчики узлов, ограничение линии и аномалии
// (analytics.node_counters.read; FR-2, FR-3, FR-5). Счётчики — по step_key
// в той же форме, что process.MapNodeCounters живой карты.
func (s *Service) NodeCounters(ctx context.Context, processVersionID string, p PeriodQuery, m platform.Moment) (NodeCounterSet, error) {
	d, err := s.load(ctx, "analytics.node_counters.read", p, m)
	if err != nil {
		return NodeCounterSet{}, err
	}
	nodes := domain.Nodes(d.rows, d.win.From, d.win.To)
	out := NodeCounterSet{Period: d.win.Period, ProcessVersionID: processVersionID, Counters: []NodeCounters{}, Anomalies: []NodeAnomaly{}, DataGaps: []string{}}
	for _, n := range nodes {
		nc := n.OpenNC
		out.Counters = append(out.Counters, NodeCounters{StepKey: n.Step, Queue: n.Queue, InProgress: n.InProgress, Passed: n.Passed, Defects: n.Defects, Nonconformities: &nc})
	}
	if b, ok := domain.Bottleneck(nodes); ok {
		out.Bottleneck = &Bottleneck{StepKey: b.Step, Wait: ptr(minutesText(b.WaitSum / int64(max(b.Queue, 1))))}
	}
	norm := func(step string) domain.Norm { return s.norms.Norm(ctx, step) }
	for _, a := range domain.Anomalies(nodes, d.rows, d.down, d.win.From, d.win.To, bucket(d.win.Period), norm) {
		out.Anomalies = append(out.Anomalies, NodeAnomaly{StepKey: a.Step, Kind: a.Kind, Threshold: ptr(limitText(a))})
	}
	return out, nil
}

// minutesText — «37 мин», «2 ч 5 мин».
func minutesText(sec int64) string {
	m := (sec + 30) / 60
	if m < 60 {
		return fmt.Sprintf("%d мин", m)
	}
	if m%60 == 0 {
		return fmt.Sprintf("%d ч", m/60)
	}
	return fmt.Sprintf("%d ч %d мин", m/60, m%60)
}

func limitText(a domain.Anomaly) string {
	switch a.Unit {
	case domain.UnitSec:
		return "норма " + minutesText(a.Limit)
	case "bp":
		return fmt.Sprintf("верхняя граница %d,%02d %%", a.Limit/100, a.Limit%100)
	}
	return fmt.Sprintf("норма %d шт.", a.Limit)
}

// Метрики контрольных карт.
const (
	ChartDefectRate = "defect_rate"
	ChartDuration   = "operation_duration"
)

// ControlChart — контрольная карта узла (analytics.control_chart.read, FR-5):
// defect_rate — карта p доли результатов контроля с признаком дефекта по
// подгруппам времени; operation_duration — карта XmR длительности операций.
func (s *Service) ControlChart(ctx context.Context, stepKey, metricID string, p PeriodQuery, m platform.Moment) (ControlChart, error) {
	if s.store == nil {
		return ControlChart{}, platform.NotImplemented("analytics.control_chart.read")
	}
	if metricID == "" {
		metricID = ChartDefectRate
	}
	if metricID != ChartDefectRate && metricID != ChartDuration {
		e := platform.Fail(errcodes.ApiNotFound, "object", "контрольная карта", "id", metricID)
		e.Detail = "Карты " + metricID + " нет: доступны defect_rate и operation_duration"
		return ControlChart{}, e
	}
	d, err := s.load(ctx, "analytics.control_chart.read", p, m)
	if err != nil {
		return ControlChart{}, err
	}
	out := ControlChart{StepKey: stepKey, MetricID: metricID, Points: []ControlChartPoint{}}
	var ch domain.Chart
	unit := "bp"
	if metricID == ChartDefectRate {
		out.Title, out.ChartKind = "Доля результатов контроля с признаком дефекта", ptr("p")
		ch = domain.StepPChart(d.rows, stepKey, d.win.From, d.win.To, bucket(d.win.Period))
	} else {
		out.Title, out.ChartKind = "Длительность операции", ptr("xmr")
		ch = domain.StepDurationChart(d.rows, stepKey, d.win.From, d.win.To)
		unit = "s"
	}
	val := func(x int64) MetricValue {
		v := MetricValue{Value: x, Unit: unit}
		if unit == "s" {
			v.Origin = ptr("computed_by_system")
		}
		return v
	}
	out.Center, out.Upper, out.Lower = val(ch.Center), val(ch.Upper), val(ch.Lower)
	for _, pt := range ch.Points {
		cp := ControlChartPoint{At: pt.At, Value: val(pt.Value), OutOfControl: pt.Out}
		if pt.Item != "" {
			cp.Ref = &platform.DrillRef{Entity: platform.EntityItem, ID: pt.Item}
		}
		out.Points = append(out.Points, cp)
	}
	return out, nil
}
