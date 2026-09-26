package analytics

import (
	"slices"
	"strings"
	"time"

	"ant/internal/application/platform"
	domain "ant/internal/domain/analytics"
)

// Показатели экрана (FR-86…FR-89; кейс §2.4, §5.2) — агрегаты строк вклада:
// сумма, доля или среднее. Раскрытие (Drilldown) строится из тех же строк,
// поэтому каждое число раскрывается до изделий и записей журнала, а для
// штучных показателей сумма строк раскрытия равна итогу (AD-45).

// agg — вид агрегата.
type agg int

const (
	aggSum   agg = iota // сумма штук
	aggShare            // доля числителя, б. п.
	aggMean             // среднее время, мин
	aggTime             // суммарное время, мин
)

// entry — вклад строки в показатель.
type entry struct {
	row   domain.Row
	value int64 // штуки или секунды
	num   bool  // для долей — в числителе
	// extra — дополнительные исходные записи (вывод о причине, ошибка исполнителя).
	extra []string
	// ref — объект строки вне изделия.
	ref *platform.DrillRef
}

// metricDef — определение показателя.
type metricDef struct {
	id, title, group, counts, account string
	agg                               agg
	// dims — измерения срезов: step, location, equipment, performer,
	// defect_type, origin, cause_category, comparable.
	dims []string
	// entries — вклады за период.
	entries func(v *view) []entry
	// sliceEntries — вклады для срезов, если они иные (доля по узлам).
	sliceEntries func(v *view) []entry
	// meaning, note — смысл интервала для длительностей.
	meaning, note string
}

// view — строки и проекции, отобранные для запроса: прогон, период, момент.
type view struct {
	rows      []domain.Row
	down      []domain.Row
	incidents []domain.Incident
	from, to  time.Time
	causes    map[string]domain.IncidentEntry
	causeInc  map[string]string
	errs      map[string]domain.IncidentEntry
	errInc    map[string]string
}

func newView(rows, down []domain.Row, incs []domain.Incident, from, to time.Time) *view {
	v := &view{rows: rows, down: down, incidents: incs, from: from, to: to,
		causes: map[string]domain.IncidentEntry{}, causeInc: map[string]string{},
		errs: map[string]domain.IncidentEntry{}, errInc: map[string]string{}}
	for _, inc := range incs {
		for _, c := range inc.Causes {
			if c.At.After(to) {
				continue
			}
			for _, id := range c.NCIDs {
				v.causes[id], v.causeInc[id] = c, inc.IncidentID // последний вывод — по порядку проекции
			}
		}
		for _, e := range inc.Errors {
			if e.At.After(to) {
				continue
			}
			for _, id := range e.NCIDs {
				v.errs[id], v.errInc[id] = e, inc.IncidentID
			}
		}
	}
	return v
}

// points — точечные строки показателя в периоде.
func (v *view) points(metric string) []entry {
	var out []entry
	for _, r := range v.rows {
		if r.Metric == metric && r.In(v.from, v.to) {
			out = append(out, entry{row: r, value: r.Value})
		}
	}
	return out
}

// active — строки-интервалы показателя, активные в конце периода.
func (v *view) active(metric string) []entry {
	var out []entry
	for _, r := range v.rows {
		if r.Metric == metric && r.ActiveAt(v.to) {
			out = append(out, entry{row: r, value: r.Value})
		}
	}
	return out
}

// overlap — время интервалов в пределах периода, с.
func overlap(rows []domain.Row, metric string, from, to time.Time) []entry {
	var out []entry
	for _, r := range rows {
		if r.Metric != metric || !r.Interval {
			continue
		}
		if s := r.Overlap(from, to); s > 0 {
			out = append(out, entry{row: r, value: s})
		}
	}
	return out
}

// origin — происхождение дефекта с учётом вывода о причине: причина
// «входной брак» переводит дефект во входной (FR-87).
func (v *view) origin(r domain.Row) string {
	if c, ok := v.causes[r.Dims.NC]; ok && r.Dims.NC != "" && c.Conclusion == "confirmed" && c.Category == "incoming" {
		return domain.DefectIncoming
	}
	return r.Dims.Origin
}

// withOrigin — строки дефектов с происхождением origin (с учётом причины).
func (v *view) withOrigin(es []entry, origin string) []entry {
	var out []entry
	for _, e := range es {
		o := v.origin(e.row)
		if o == origin {
			e.row.Dims.Origin = o
			if c, ok := v.causes[e.row.Dims.NC]; ok && e.row.Dims.NC != "" {
				e.extra = append(e.extra, c.EventID)
			}
			out = append(out, e)
		}
	}
	return out
}

// causeOf — категория причины несоответствия на конец периода: категория
// подтверждённой причины, not_established или pending (вывода ещё нет).
func (v *view) causeOf(nc string) (string, bool, string) {
	c, ok := v.causes[nc]
	switch {
	case !ok:
		return "pending", false, ""
	case c.Conclusion == "confirmed" && c.Category != "" && c.Category != "not_established":
		return c.Category, true, c.EventID
	}
	return "not_established", false, c.EventID
}

// ncWithCause — подтверждённые несоответствия с категорией причины в срезе.
func (v *view) ncWithCause(filter func(cat string, ok bool) bool) []entry {
	var out []entry
	for _, e := range v.points(domain.RowNCConfirmed) {
		cat, ok, ev := v.causeOf(e.row.Dims.NC)
		if filter != nil && !filter(cat, ok) {
			continue
		}
		e.num = ok
		e.row.Dims.Ref = cat
		if ev != "" {
			e.extra = append(e.extra, ev)
		}
		out = append(out, e)
	}
	return out
}

// recurrence — повторяемость проблем (кейс §1.2): повторы одного вида
// дефекта на одной операции за период — все подтверждённые дефекты группы,
// кроме первого.
func (v *view) recurrence() []entry {
	es := v.withOrigin(v.points(domain.RowConfirmedDefects), domain.DefectProduction)
	slices.SortStableFunc(es, func(a, b entry) int { return a.row.At.Compare(b.row.At) })
	seen := map[string]bool{}
	var out []entry
	for _, e := range es {
		k := e.row.Dims.DefectType + "|" + e.row.Dims.Step
		if e.row.Dims.DefectType == "" {
			continue
		}
		if seen[k] {
			out = append(out, e)
		}
		seen[k] = true
	}
	return out
}

// fpy — «Прохождение контроля с первого раза» (Д-11): доля изделий, у
// которых первый вердикт каждого узла — «годно», без повторных выполнений и
// подтверждённых несоответствий, известных к концу периода.
func (v *view) fpy(total string) []entry {
	var out []entry
	for _, e := range v.points(total) {
		e.num = !e.row.FailedBy(v.to)
		out = append(out, e)
	}
	return out
}

// hypotheses — гипотезы причин, записанные людьми, за период (FR-87).
func (v *view) hypotheses() []entry {
	var out []entry
	for _, inc := range v.incidents {
		for _, h := range inc.Hypotheses {
			if h.At.Before(v.from) || h.At.After(v.to) {
				continue
			}
			r := domain.Row{Metric: "hypotheses", Item: inc.IncidentID, At: h.At, Value: 1,
				Dims: domain.Dims{Label: inc.IncidentID, Ref: h.Category}, Sources: []string{h.EventID}, Kinds: []string{h.SourceKind}}
			out = append(out, entry{row: r, value: 1, ref: &platform.DrillRef{Entity: platform.EntityIncident, ID: inc.IncidentID}})
		}
	}
	return out
}

// performerErrors — подтверждённые ошибки исполнителей (FR-87, ТК РФ ст. 247:
// только решением уполномоченного) на сопоставимые работы исполнителя.
func (v *view) performerErrors() []entry {
	var out []entry
	for _, e := range v.points(domain.RowNCConfirmed) {
		er, ok := v.errs[e.row.Dims.NC]
		if !ok {
			continue
		}
		e.row.Dims.Performer = er.Operator
		e.extra = append(e.extra, er.EventID)
		out = append(out, e)
	}
	return out
}

// downtime — простой оборудования в периоде (FR-89).
func (v *view) downtime() []entry {
	out := overlap(v.down, domain.RowEquipmentDowntime, v.from, v.to)
	for i := range out {
		id := out[i].row.Dims.Equipment
		if id == "" {
			id = out[i].row.Dims.Step
		}
		out[i].row.Item = id
		out[i].row.Dims.Label = id
		if out[i].row.Dims.Equipment != "" {
			out[i].ref = &platform.DrillRef{Entity: platform.EntityEquipment, ID: out[i].row.Dims.Equipment}
		}
	}
	return out
}

func points(metric string) func(*view) []entry {
	return func(v *view) []entry { return v.points(metric) }
}

// catalog — показатели раздела «Аналитика» в порядке экрана.
var catalog = []metricDef{
	{id: "inspected_items", title: "Проверено изделий", group: "inspection", counts: "items", agg: aggSum,
		entries: points(domain.RowInspectedItems)},
	{id: "first_pass_yield", title: "Прохождение контроля с первого раза", group: "inspection", counts: "items", agg: aggShare, dims: []string{"step"},
		entries:      func(v *view) []entry { return v.fpy(domain.RowFPYTotal) },
		sliceEntries: func(v *view) []entry { return v.fpy(domain.RowFPYStepTotal) }},
	{id: "unable_to_assess", title: "Оценка невозможна (отдельная корзина)", group: "inspection", counts: "observations", agg: aggSum, dims: []string{"step"},
		entries: points(domain.RowUnableToAssess)},
	{id: "representations", title: "Повторные предъявления", group: "inspection", counts: "presentations", agg: aggSum, dims: []string{"step"},
		entries: points(domain.RowRepresentations)},
	{id: "items_with_confirmed_nc", title: "Изделия с подтверждёнными несоответствиями", group: "defects", counts: "items", agg: aggSum, dims: []string{"step", "origin"},
		entries: func(v *view) []entry {
			out := v.points(domain.RowItemsWithConfirmedNC)
			for i := range out {
				out[i].row.Dims.Origin = v.origin(out[i].row)
			}
			return out
		}},
	{id: "confirmed_defects", title: "Подтверждённые дефекты (производственные)", group: "defects", counts: "defects", agg: aggSum,
		dims: []string{"defect_type", "step", "location"},
		entries: func(v *view) []entry {
			return v.withOrigin(v.points(domain.RowConfirmedDefects), domain.DefectProduction)
		}},
	{id: "recurrence_rate", title: "Повторяемость проблем: повторы вида дефекта на операции", group: "defects", counts: "defects", agg: aggSum,
		dims: []string{"defect_type", "step"}, entries: func(v *view) []entry { return v.recurrence() }},
	{id: "rework_runs", title: "Повторные выполнения операций", group: "defects", counts: "operations", agg: aggSum, dims: []string{"step", "performer"},
		entries: points(domain.RowReworkRuns)},
	{id: "unfinished_operations", title: "Незавершённые операции", group: "defects", counts: "operations", agg: aggSum, dims: []string{"step"},
		entries: func(v *view) []entry { return v.active(domain.RowInProgress) }},
	{id: "interrupted_operations", title: "Прерванные операции", group: "defects", counts: "operations", agg: aggSum, dims: []string{"step"},
		entries: points(domain.RowInterrupted)},
	{id: "scrap_losses", title: "Потери от брака: списано изделий", group: "defects", counts: "items", agg: aggSum, dims: []string{"step", "origin"},
		entries: points(domain.RowScrappedItems)},
	{id: "incoming_defects", title: "Входной брак (отдельно от производственных ошибок)", group: "causes", counts: "defects", account: "incoming", agg: aggSum,
		dims: []string{"defect_type", "step"},
		entries: func(v *view) []entry {
			return v.withOrigin(v.points(domain.RowConfirmedDefects), domain.DefectIncoming)
		}},
	{id: "cause_established", title: "Причина установлена", group: "causes", counts: "nonconformities", agg: aggShare, dims: []string{"cause_category", "step"},
		entries: func(v *view) []entry { return v.ncWithCause(nil) }},
	{id: "cause_not_established", title: "Причина не установлена или разбор не завершён", group: "causes", counts: "nonconformities", agg: aggSum,
		dims:    []string{"cause_category", "step"},
		entries: func(v *view) []entry { return v.ncWithCause(func(_ string, ok bool) bool { return !ok }) }},
	{id: "hypotheses", title: "Гипотезы причин (не выводы)", group: "causes", counts: "hypotheses", account: "hypotheses", agg: aggSum,
		dims: []string{"cause_category"}, entries: func(v *view) []entry { return v.hypotheses() }},
	{id: "equipment_caused_nc", title: "Несоответствия по причине оборудования", group: "equipment", counts: "nonconformities", account: "equipment", agg: aggSum,
		dims: []string{"equipment", "step"},
		entries: func(v *view) []entry {
			return v.ncWithCause(func(cat string, ok bool) bool { return ok && cat == "equipment" })
		}},
	{id: "equipment_downtime", title: "Простой оборудования и остановки точек процесса", group: "equipment", counts: "time", account: "equipment", agg: aggTime,
		dims: []string{"equipment", "step"}, meaning: domain.MeaningOther, note: "простой: остановлено, прервано, неисправность или остановка точки процесса — от перехода источника до возврата в работу",
		entries: func(v *view) []entry { return v.downtime() }},
	{id: "comparable_runs", title: "Сопоставимые работы: выполнения по исполнителям", group: "comparison", counts: "operations", agg: aggSum,
		dims: []string{"comparable", "equipment"}, entries: points(domain.RowComparableRuns)},
	{id: "confirmed_performer_errors", title: "Подтверждённые ошибки исполнителей", group: "people", counts: "nonconformities", account: "performer", agg: aggSum,
		dims: []string{"comparable"}, entries: func(v *view) []entry { return v.performerErrors() }},
	{id: "operation_duration", title: "Длительность операций", group: "time", counts: "time", agg: aggMean,
		dims: []string{"step", "location", "performer", "equipment", "shift"}, entries: points(domain.RowOperationDuration)},
	{id: "waiting_time", title: "Ожидание изделий в очередях", group: "time", counts: "time", agg: aggTime, dims: []string{"step"},
		meaning: domain.MeaningOther, note: "от выхода из прежнего узла (или предъявления) до начала операции (или решения)",
		entries: func(v *view) []entry { return overlap(v.rows, domain.RowQueue, v.from, v.to) }},
	{id: "rework_time", title: "Потери от брака: время повторных выполнений", group: "time", counts: "time", agg: aggTime, dims: []string{"step"},
		entries: func(v *view) []entry { return v.points(domain.RowReworkTime) }},
	{id: "lead_time", title: "Время детали в системе (выпущенные)", group: "time", counts: "time", agg: aggMean,
		meaning: domain.MeaningOther, note: "от запуска изделия до сдачи на склад готовой продукции",
		entries: points(domain.RowLeadTime)},
	{id: "detection_delay", title: "Задержка обнаружения дефектов", group: "time", counts: "time", agg: aggMean, dims: []string{"step", "defect_type"},
		meaning: domain.MeaningOther, note: "от конца выполнения операции до первого наблюдения дефекта",
		entries: points(domain.RowDetectionDelay)},
}

// aliases — показатели плиток и заготовок, которые считаются как другие.
var aliases = map[string]metricDef{
	"defects_by_type": {id: "defects_by_type", title: "Дефекты по видам (подтверждённые, производственные)", group: "defects", counts: "defects", agg: aggSum,
		dims: []string{"defect_type"},
		entries: func(v *view) []entry {
			return v.withOrigin(v.points(domain.RowConfirmedDefects), domain.DefectProduction)
		}},
}

// lookup — определение показателя по id.
func lookup(id string) (metricDef, bool) {
	for _, d := range catalog {
		if d.id == id {
			return d, true
		}
	}
	d, ok := aliases[id]
	return d, ok
}

// tileIDs — плитки стола руководителя (PRD §3a).
var tileIDs = []string{"inspected_items", "items_with_confirmed_nc", "first_pass_yield", "defects_by_type", "cause_established", "lead_time", "waiting_time"}

// dimValue — измерение среза строки: (измерение контракта, ключ, подпись).
func dimValue(dim string, r domain.Row) (string, string, string) {
	d := r.Dims
	switch dim {
	case "step":
		return "step", d.Step, d.Step
	case "location":
		if d.Station != "" {
			return "location", d.Station, d.Station
		}
		return "location", d.Line, d.Line
	case "equipment":
		return "equipment", d.Equipment, d.Equipment
	case "performer":
		return "performer", d.Performer, d.Performer
	case "defect_type":
		return "defect_type", d.DefectType, d.DefectType
	case "origin":
		return "origin", d.Origin, map[string]string{domain.DefectIncoming: "Входной брак", domain.DefectProduction: "Производственные"}[d.Origin]
	case "cause_category":
		// Категория причины — «откуда брак» (измерение origin контракта v1).
		return "origin", d.Ref, causeTitle[d.Ref]
	case "comparable":
		// Сопоставимые работы: тип операции × тип изделия × исполнитель.
		if d.Performer == "" {
			return "performer", "", ""
		}
		key := strings.Join([]string{d.Operation, d.ItemType, d.Performer}, "/")
		lbl := d.Performer
		if d.Operation != "" || d.ItemType != "" {
			lbl += " — " + strings.Trim(d.Operation+" · "+d.ItemType, " ·")
		}
		return "performer", key, lbl
	}
	return "", "", ""
}

var causeTitle = map[string]string{
	"incoming": "Входной брак", "equipment": "Оборудование", "performer": "Исполнитель", "handling": "Обращение с изделием",
	"assembly": "Сборка", "documentation": "Документация", "not_established": "Не установлена", "pending": "Разбор не завершён",
}

// sliceKey — ключ среза в ответе: «измерение:значение» (уникален в пределах показателя).
func sliceKey(dim, key string) string { return dim + ":" + key }
