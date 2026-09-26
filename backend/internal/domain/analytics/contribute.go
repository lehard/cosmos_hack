package analytics

import (
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// run — выполнение операции (FR-47: повтор — новый operation_run_id + rework_of).
type run struct {
	id, step, code, line, station, eq, perf, reworkOf string
	start                                             time.Time
	srcStart, srcFinish                               *time.Time
	finish                                            *time.Time
	completion                                        string
	reported                                          *durationData
	pauses                                            []pause
	events                                            []string
}

type pause struct {
	from time.Time
	to   *time.Time
}

// Состояния физического дефекта.
const (
	defOpen = iota
	defConfirmed
	defRejected
)

// phys — физический дефект (соглашение «Дефект», FR-37): ключ — изделие, зона
// и место в пределах выполнения операции или до следующей операции, без вида;
// повторные наблюдения добавляют записи, вид уточняется последним наблюдением.
type phys struct {
	key, zone, loc, dtype, step string
	run                         *run
	incoming                    bool
	first                       time.Time
	events                      []string
	state                       int
	nc                          string
	confirmedAt                 time.Time
}

// nc — несоответствие изделия.
type nc struct {
	id         string
	at         time.Time
	confirmed  bool
	events     []string
	step       string
	run        *run
	defects    []*phys
	closed     *time.Time
	incoming   bool
	registered bool
}

// openQ — незакрытый интервал ожидания.
type openQ struct {
	step string
	at   time.Time
	ev   string
}

type verdict struct {
	pass bool
	at   time.Time
	ev   string
}

// item — разбор входа изделия.
type item struct {
	id, label, run, itemType string
	registered               *time.Time
	regEv                    string

	runs    map[string]*run
	runList []*run
	defects map[string]*phys
	defList []*phys
	ncs     map[string]*nc
	ncList  []*nc

	kinds map[string]string // event_id → вид источника

	obsSeen  map[string]bool
	presSeen map[string]bool
	resSeen  map[string]bool

	rows []Row

	presQ    []*openQ
	moveQ    *openQ
	exitAt   *time.Time
	exitEv   string
	verdicts map[string]verdict
	vSteps   []string
	verdEvs  []string
	firstV   *time.Time
	released bool
	scrapped bool
}

// sourceKind — вид источника записи для раскрытия (FR-140).
func sourceKind(r kernel.Record) string {
	switch r.Kind {
	case catalog.KindFact:
		return r.SourceKind
	case catalog.KindDecision:
		return SourceManual
	}
	return SourceSystem
}

// Prepare — вход изделия для показателей: порядок изделия (occurred_at →
// received_at → event_id, AD-5), без повторов event_id и без исправленных
// записей (исправление — новая запись corrects_event, FR-122: берётся «стало»).
func Prepare(input []kernel.Record) []kernel.Record {
	corrected := map[string]bool{}
	for _, r := range input {
		if r.Corrects != "" {
			corrected[r.Corrects] = true
		}
	}
	in := slices.Clone(input)
	slices.SortStableFunc(in, func(a, b kernel.Record) int {
		switch {
		case kernel.Less(a, b):
			return -1
		case kernel.Less(b, a):
			return 1
		}
		return 0
	})
	seen := map[string]bool{}
	out := make([]kernel.Record, 0, len(in))
	for _, r := range in {
		if corrected[r.EventID] || seen[r.EventID] {
			continue
		}
		seen[r.EventID] = true
		out = append(out, r)
	}
	return out
}

// Contribute — функция вклада изделия (AD-45, FR-86…FR-89): весь вход
// изделия → строки вклада. Чистая и детерминированная (AD-4): один вход —
// одни строки при любом числе воркеров, при повторе и при воспроизведении.
//
// Защиты от двойного счёта (кейс §5.2):
//   - повтор записи (тот же event_id) и повтор факта с тем же смыслом
//     (тот же operation_run_id, observation_id, номер предъявления, nc_id)
//     не дают второй строки;
//   - повторные наблюдения одного физического дефекта — одна строка (FR-37);
//   - позднее событие встаёт на своё occurred_at: воркер пересворачивает
//     изделие целиком и заменяет строки (инкрементов нет).
func Contribute(itemID string, input []kernel.Record) []Row {
	in := Prepare(input)
	it := &item{
		id: itemID, label: label(itemID),
		runs: map[string]*run{}, defects: map[string]*phys{}, ncs: map[string]*nc{},
		kinds: map[string]string{}, obsSeen: map[string]bool{}, presSeen: map[string]bool{}, resSeen: map[string]bool{},
		verdicts: map[string]verdict{},
	}
	for _, r := range in {
		if it.run == "" {
			it.run = r.RunID
		}
		it.kinds[r.EventID] = sourceKind(r)
		it.apply(r)
	}
	it.finish()
	SortRows(it.rows)
	return it.rows
}

func (it *item) base() Dims { return Dims{Run: it.run, ItemType: it.itemType, Label: it.label} }

func (it *item) runDims(r *run) Dims {
	d := it.base()
	if r != nil {
		d.Step, d.Station, d.Line, d.Equipment, d.Performer, d.Operation = r.step, r.station, r.line, r.eq, r.perf, r.code
	}
	return d
}

// emit — строка с видами источников по исходным записям.
func (it *item) emit(row Row) {
	if row.Unit == "" {
		row.Unit = UnitPcs
	}
	row.Sources = addUnique(nil, row.Sources...)
	for _, id := range row.Sources {
		row.Kinds = addUnique(row.Kinds, it.kinds[id])
	}
	it.rows = append(it.rows, row)
}

// runAt — выполнение, к которому относится запись в момент t: указанное явно
// или последнее начатое до t («до следующей операции», FR-37).
func (it *item) runAt(id string, t time.Time) *run {
	if r, ok := it.runs[id]; ok && id != "" {
		return r
	}
	var out *run
	for _, r := range it.runList {
		if !r.start.After(t) {
			out = r
		}
	}
	return out
}

func (it *item) apply(r kernel.Record) {
	switch r.Type {
	case catalog.ItemItemRegistered:
		d, _ := decode[registeredData](r.Data)
		if it.registered == nil {
			t := r.OccurredAt
			it.registered, it.regEv, it.itemType = &t, r.EventID, d.ItemType
		}
	case catalog.OperationRunStarted:
		it.runStarted(r)
	case catalog.OperationRunPaused, catalog.OperationRunResumed:
		d, _ := decode[runRefData](r.Data)
		ru, ok := it.runs[d.RunID]
		if !ok {
			return
		}
		ru.events = append(ru.events, r.EventID)
		if r.Type == catalog.OperationRunPaused {
			ru.pauses = append(ru.pauses, pause{from: r.OccurredAt})
		} else if n := len(ru.pauses); n > 0 && ru.pauses[n-1].to == nil {
			t := r.OccurredAt
			ru.pauses[n-1].to = &t
		}
	case catalog.OperationRunFinished:
		d, _ := decode[runFinishedData](r.Data)
		ru, ok := it.runs[d.RunID]
		if !ok || ru.finish != nil {
			return // завершение без начала или повтор завершения
		}
		t := r.OccurredAt
		ru.finish, ru.completion, ru.reported = &t, d.Completion, d.Reported
		if d.StartedAt != nil {
			ru.srcStart = d.StartedAt
		}
		ru.srcFinish = d.FinishedAt
		ru.events = append(ru.events, r.EventID)
		it.exitAt, it.exitEv = &t, r.EventID
	case catalog.InspectionResultRecorded:
		it.inspection(r)
	case catalog.ItemPresentationRecorded:
		d, _ := decode[presentationData](r.Data)
		k := d.Step + "#" + itoa(d.No)
		if it.presSeen[k] {
			return
		}
		it.presSeen[k] = true
		dims := it.base()
		dims.Step = d.Step
		it.emit(Row{Metric: RowPresentations, At: r.OccurredAt, Value: 1, Dims: dims, Sources: []string{r.EventID}})
		if d.No > 1 {
			it.emit(Row{Metric: RowRepresentations, At: r.OccurredAt, Value: 1, Dims: dims, Sources: []string{r.EventID}})
		}
		it.presQ = append(it.presQ, &openQ{step: d.Step, at: r.OccurredAt, ev: r.EventID})
	case catalog.DecisionPresentationResolved:
		it.resolved(r)
	case catalog.OperationMovementReceived:
		d, _ := decode[movementReceivedData](r.Data)
		if it.moveQ != nil {
			it.closeQueue(it.moveQ, r.OccurredAt, r.EventID)
			it.moveQ = nil
		}
		if d.Step != "" {
			it.moveQ = &openQ{step: d.Step, at: r.OccurredAt, ev: r.EventID}
			it.exitAt = nil
		}
		if d.Inspection == "damage_found" {
			// Повреждение при приёмке — входной признак дефекта (FR-87).
			p := it.physAt("receipt|"+d.To, "", d.To, d.Step, nil, r.OccurredAt)
			p.incoming = true
			p.events = addUnique(p.events, r.EventID)
		}
	case catalog.DecisionSignalRejected:
		for _, p := range it.defList {
			if p.state == defOpen && !p.first.After(r.OccurredAt) {
				p.state = defRejected
				p.events = addUnique(p.events, r.EventID)
			}
		}
	case catalog.DecisionNonconformityConfirmed:
		it.confirmed(r)
	case catalog.DecisionNonconformityRegistered:
		d, _ := decode[ncData](r.Data)
		if d.NC == "" || it.ncs[d.NC] != nil {
			return
		}
		ru := it.runAt(d.RunID, r.OccurredAt)
		n := &nc{id: d.NC, at: r.OccurredAt, events: []string{r.EventID}, step: d.Step, run: ru, registered: true}
		if n.step == "" && ru != nil {
			n.step = ru.step
		}
		it.ncs[d.NC] = n
		it.ncList = append(it.ncList, n)
	case catalog.DecisionNonconformityClosed:
		d, _ := decode[ncData](r.Data)
		if n := it.ncs[d.NC]; n != nil && n.closed == nil {
			t := r.OccurredAt
			n.closed = &t
			n.events = addUnique(n.events, r.EventID)
		}
	case catalog.DecisionDispositionSet:
		d, _ := decode[dispositionData](r.Data)
		if d.Disposition != "scrap" || it.scrapped {
			return
		}
		it.scrapped = true
		dims := it.base()
		srcs := []string{r.EventID}
		if n := it.ncs[d.NC]; n != nil {
			dims.Step, dims.NC, dims.Origin = n.step, n.id, originOf(n.incoming)
			srcs = append(srcs, n.events...)
		}
		it.emit(Row{Metric: RowScrappedItems, At: r.OccurredAt, Value: 1, Dims: dims, Sources: srcs})
	case catalog.ItemReleaseRecorded:
		if it.released {
			return
		}
		it.released = true
		start, startEv := r.OccurredAt, ""
		if it.registered != nil {
			start, startEv = *it.registered, it.regEv
		} else if len(it.runList) > 0 {
			start, startEv = it.runList[0].start, it.runList[0].events[0]
		}
		dims := it.base()
		dims.Meaning, dims.DurationOrigin = MeaningOther, OriginSystem
		it.emit(Row{Metric: RowLeadTime, At: r.OccurredAt, Value: secs(r.OccurredAt.Sub(start)), Unit: UnitSec, Dims: dims, Sources: []string{startEv, r.EventID}})
	}
}

func (it *item) runStarted(r kernel.Record) {
	d, _ := decode[runStartedData](r.Data)
	if d.RunID == "" || it.runs[d.RunID] != nil {
		return // повтор начала того же выполнения — не новое выполнение (FR-47)
	}
	ru := &run{id: d.RunID, step: d.Step, code: d.Code, line: d.Line, station: d.Station, eq: d.Equipment,
		perf: deref(d.Operator), reworkOf: d.ReworkOf, start: r.OccurredAt, srcStart: d.StartedAt, events: []string{r.EventID}}
	it.runs[d.RunID] = ru
	it.runList = append(it.runList, ru)
	// Ожидание перед операцией: от выхода из прежнего узла (или приёма на
	// участке) до начала (кейс §5.1, FR-89).
	switch {
	case it.moveQ != nil:
		it.closeQueue(it.moveQ, r.OccurredAt, r.EventID)
		it.moveQ = nil
	case it.exitAt != nil:
		it.closeQueue(&openQ{step: ru.step, at: *it.exitAt, ev: it.exitEv}, r.OccurredAt, r.EventID)
	}
	it.exitAt = nil
	if ru.reworkOf != "" {
		it.emit(Row{Metric: RowReworkRuns, At: ru.start, Value: 1, Dims: it.runDims(ru), Sources: []string{r.EventID}})
	}
}

// closeQueue — закрытый интервал ожидания в узле q.step.
func (it *item) closeQueue(q *openQ, until time.Time, ev string) {
	if q.step == "" || until.Before(q.at) {
		return
	}
	u := until
	dims := it.base()
	dims.Step, dims.Meaning, dims.DurationOrigin = q.step, MeaningOther, OriginSystem
	it.emit(Row{Metric: RowQueue, At: q.at, Interval: true, Until: &u, Value: 1, Dims: dims, Sources: []string{q.ev, ev}})
}

// physAt — физический дефект по ключу (создаётся при первом наблюдении).
func (it *item) physAt(key, zone, loc, step string, ru *run, at time.Time) *phys {
	if p, ok := it.defects[key]; ok {
		return p
	}
	p := &phys{key: key, zone: zone, loc: loc, step: step, run: ru, first: at}
	it.defects[key] = p
	it.defList = append(it.defList, p)
	return p
}

func (it *item) inspection(r kernel.Record) {
	d, _ := decode[inspectionData](r.Data)
	if d.ObservationID != "" {
		if it.obsSeen[d.ObservationID] {
			return // повтор того же наблюдения источника
		}
		it.obsSeen[d.ObservationID] = true
	}
	ru := it.runAt(d.RunID, r.OccurredAt)
	step := d.Step
	if step == "" && ru != nil {
		step = ru.step
	}
	dims := it.base()
	dims.Step, dims.Equipment, dims.Performer = step, d.Equipment, deref(d.Inspector)
	src := []string{r.EventID}
	if d.Outcome == "unable_to_assess" || d.Processing == "aborted" || d.Processing == "failed" {
		// «Оценка невозможна» ≠ «годно» — отдельная корзина (NFR-UI-4).
		ud := dims
		ud.Ref = d.UnableReason
		it.emit(Row{Metric: RowUnableToAssess, At: r.OccurredAt, Value: 1, Dims: ud, Sources: src})
		return
	}
	it.emit(Row{Metric: RowInspections, At: r.OccurredAt, Value: 1, Dims: dims, Sources: src})
	if step != "" {
		it.emit(Row{Metric: RowPassed, At: r.OccurredAt, Value: 1, Dims: dims, Sources: src})
	}
	defect := d.Outcome == "defect_indicated"
	it.verdict(step, !defect, r)
	if !defect {
		return
	}
	it.emit(Row{Metric: RowInspectionsWithDefect, At: r.OccurredAt, Value: 1, Dims: dims, Sources: src})
	defects := d.Defects
	if len(defects) == 0 {
		defects = []defectData{{}}
	}
	runRef := "pre"
	if ru != nil {
		runRef = ru.id
	}
	for _, df := range defects {
		zone := df.Zone
		if zone == "" && len(d.ZoneIDs) == 1 {
			zone = d.ZoneIDs[0]
		}
		p := it.physAt(runRef+"|"+zone+"|"+df.Location, zone, df.Location, step, ru, r.OccurredAt)
		p.events = addUnique(p.events, r.EventID)
		if df.Type != "" {
			p.dtype = df.Type // уточнение вида обновляет вид, а не создаёт дефект (FR-37)
		}
		if d.Phase == "incoming" || ru == nil {
			p.incoming = true
		}
	}
}

func (it *item) verdict(step string, pass bool, r kernel.Record) {
	it.verdEvs = append(it.verdEvs, r.EventID)
	if it.firstV == nil {
		t := r.OccurredAt
		it.firstV = &t
	}
	if step == "" {
		step = "?"
	}
	if _, ok := it.verdicts[step]; ok {
		return
	}
	it.verdicts[step] = verdict{pass: pass, at: r.OccurredAt, ev: r.EventID}
	it.vSteps = append(it.vSteps, step)
}

func (it *item) resolved(r kernel.Record) {
	d, _ := decode[presentationData](r.Data)
	k := d.Step + "#" + itoa(d.No)
	if it.resSeen[k] {
		return
	}
	it.resSeen[k] = true
	for i, q := range it.presQ {
		if q.step == d.Step {
			it.closeQueue(q, r.OccurredAt, r.EventID)
			it.presQ = slices.Delete(it.presQ, i, i+1)
			break
		}
	}
	t := r.OccurredAt
	it.exitAt, it.exitEv = &t, r.EventID
	if d.Resolution == "insufficient_data" {
		return
	}
	dims := it.base()
	dims.Step = d.Step
	it.emit(Row{Metric: RowPassed, At: r.OccurredAt, Value: 1, Dims: dims, Sources: []string{r.EventID}})
	it.verdict(d.Step, d.Resolution == "accept" || d.Resolution == "accept_with_concession", r)
}

func (it *item) confirmed(r kernel.Record) {
	d, _ := decode[ncData](r.Data)
	if d.NC == "" {
		return
	}
	if old := it.ncs[d.NC]; old != nil && old.confirmed {
		return // повтор подтверждения того же несоответствия
	}
	var open, typed []*phys
	for _, p := range it.defList {
		if p.state == defOpen && !p.first.After(r.OccurredAt) {
			open = append(open, p)
			if dt := deref(d.DefectType); dt != "" && p.dtype == dt {
				typed = append(typed, p)
			}
		}
	}
	if len(typed) > 0 {
		open = typed
	}
	n := it.ncs[d.NC]
	if n == nil {
		n = &nc{id: d.NC}
		it.ncs[d.NC] = n
		it.ncList = append(it.ncList, n)
	}
	n.confirmed, n.at = true, r.OccurredAt
	n.events = addUnique(n.events, r.EventID)
	n.incoming = len(open) > 0
	for _, p := range open {
		p.state, p.nc, p.confirmedAt = defConfirmed, d.NC, r.OccurredAt
		if deref(d.DefectType) != "" && p.dtype == "" {
			p.dtype = deref(d.DefectType)
		}
		n.defects = append(n.defects, p)
		n.incoming = n.incoming && p.incoming
		if n.step == "" {
			n.step, n.run = p.step, p.run
		}
	}
	if n.step == "" {
		if ru := it.runAt("", r.OccurredAt); ru != nil {
			n.step, n.run = ru.step, ru
		}
	}
}

func originOf(incoming bool) string {
	if incoming {
		return DefectIncoming
	}
	return DefectProduction
}

// finish — строки по итогу входа: выполнения, дефекты, несоответствия,
// «с первого раза», открытые ожидания.
func (it *item) finish() {
	for _, ru := range it.runList {
		it.runRows(ru)
	}
	for _, p := range it.defList {
		if p.state == defRejected {
			continue
		}
		dims := it.runDims(p.run)
		dims.Step, dims.DefectType, dims.Origin, dims.Ref = p.step, p.dtype, originOf(p.incoming), p.key
		it.emit(Row{Metric: RowDefectsDetected, At: p.first, Value: 1, Dims: dims, Sources: p.events})
		if p.state != defConfirmed {
			continue
		}
		cd := dims
		cd.NC = p.nc
		n := it.ncs[p.nc]
		it.emit(Row{Metric: RowConfirmedDefects, At: p.confirmedAt, Value: 1, Dims: cd, Sources: append(slices.Clone(p.events), n.events...)})
		if p.run != nil && p.run.finish != nil && !p.first.Before(*p.run.finish) {
			dd := cd
			dd.Meaning, dd.DurationOrigin = MeaningOther, OriginSystem
			it.emit(Row{Metric: RowDetectionDelay, At: p.first, Value: secs(p.first.Sub(*p.run.finish)), Unit: UnitSec, Dims: dd,
				Sources: append(slices.Clone(p.run.events), p.events[0])})
		}
	}
	var firstNC *nc
	var ncEvents []string
	allIncoming := true
	for _, n := range it.ncList {
		dims := it.runDims(n.run)
		dims.Step, dims.NC, dims.Origin = n.step, n.id, originOf(n.incoming)
		it.emit(Row{Metric: RowNCOpen, At: n.at, Interval: true, Until: n.closed, Value: 1, Dims: dims, Sources: n.events})
		if !n.confirmed {
			continue
		}
		srcs := slices.Clone(n.events)
		for _, p := range n.defects {
			srcs = append(srcs, p.events...)
		}
		it.emit(Row{Metric: RowNCConfirmed, At: n.at, Value: 1, Dims: dims, Sources: srcs})
		if firstNC == nil || n.at.Before(firstNC.at) {
			firstNC = n
		}
		ncEvents = append(ncEvents, n.events...)
		allIncoming = allIncoming && n.incoming
	}
	if firstNC != nil {
		dims := it.base()
		dims.Step, dims.NC, dims.Origin = firstNC.step, firstNC.id, originOf(allIncoming)
		it.emit(Row{Metric: RowItemsWithConfirmedNC, At: firstNC.at, Value: 1, Dims: dims, Sources: ncEvents})
	}
	if it.firstV != nil {
		dims := it.base()
		it.emit(Row{Metric: RowInspectedItems, At: *it.firstV, Value: 1, Dims: dims, Sources: it.verdEvs})
		// «С первого раза» (Д-11): строка на момент первого вердикта; Until —
		// момент, когда стало известно, что не с первого раза (первый вердикт
		// «не годно», повтор операции, подтверждённое несоответствие). Так
		// ответ на момент T не знает о провале, случившемся позже T (AD-22).
		var failAt *time.Time
		var fpySrc []string
		fail := func(t time.Time, ev string) {
			if failAt == nil || t.Before(*failAt) {
				tt := t
				failAt = &tt
			}
			fpySrc = append(fpySrc, ev)
		}
		for _, s := range it.vSteps {
			v := it.verdicts[s]
			fpySrc = append(fpySrc, v.ev)
			sd := it.base()
			if s != "?" {
				sd.Step = s
			}
			row := Row{Metric: RowFPYStepTotal, At: v.at, Value: 1, Dims: sd, Sources: []string{v.ev}}
			if !v.pass {
				t := v.at
				row.Until = &t
				fail(v.at, v.ev)
			}
			it.emit(row)
		}
		for _, ru := range it.runList {
			if ru.reworkOf != "" {
				fail(ru.start, ru.events[0])
			}
		}
		if firstNC != nil {
			fail(firstNC.at, firstNC.events[0])
		}
		it.emit(Row{Metric: RowFPYTotal, At: *it.firstV, Until: failAt, Value: 1, Dims: dims, Sources: fpySrc})
	}
	// Незакрытые ожидания: предъявление без решения («специалист заснул») и
	// приём на участке без начала операции — очередь узла сейчас (FR-5).
	for _, q := range it.presQ {
		dims := it.base()
		dims.Step, dims.Meaning, dims.DurationOrigin = q.step, MeaningOther, OriginSystem
		it.emit(Row{Metric: RowQueue, At: q.at, Interval: true, Value: 1, Dims: dims, Sources: []string{q.ev}})
	}
	if q := it.moveQ; q != nil {
		dims := it.base()
		dims.Step, dims.Meaning, dims.DurationOrigin = q.step, MeaningOther, OriginSystem
		it.emit(Row{Metric: RowQueue, At: q.at, Interval: true, Value: 1, Dims: dims, Sources: []string{q.ev}})
	}
}

// runRows — строки выполнения операции: интервал выполнения, прохождение
// узла, длительность с происхождением (соглашение «Длительности»).
func (it *item) runRows(ru *run) {
	dims := it.runDims(ru)
	dims.Ref = ru.id
	it.emit(Row{Metric: RowInProgress, At: ru.start, Interval: true, Until: ru.finish, Value: 1, Dims: dims, Sources: ru.events})
	if ru.finish == nil {
		return
	}
	if ru.completion == "interrupted" {
		it.emit(Row{Metric: RowInterrupted, At: *ru.finish, Value: 1, Dims: dims, Sources: ru.events})
		return
	}
	it.emit(Row{Metric: RowPassed, At: *ru.finish, Value: 1, Dims: dims, Sources: ru.events})
	it.emit(Row{Metric: RowComparableRuns, At: *ru.finish, Value: 1, Dims: dims, Sources: ru.events})
	v, meaning, origin := runDuration(ru)
	dd := dims
	dd.Meaning, dd.DurationOrigin = meaning, origin
	it.emit(Row{Metric: RowOperationDuration, At: *ru.finish, Value: v, Unit: UnitSec, Dims: dd, Sources: ru.events})
	if ru.reworkOf != "" {
		it.emit(Row{Metric: RowReworkTime, At: *ru.finish, Value: v, Unit: UnitSec, Dims: dd, Sources: ru.events})
	}
}

// runDuration — длительность выполнения и её смысл и происхождение: передана
// источником (reported_duration или границы источника) либо вычислена
// системой; паузы внутри выполнения вычитаются — тогда это активная обработка.
func runDuration(ru *run) (int64, string, string) {
	if d := ru.reported; d != nil {
		meaning := d.Meaning
		if meaning == "" {
			meaning = MeaningOther
		}
		return d.seconds(), meaning, OriginSource
	}
	if ru.srcStart != nil && ru.srcFinish != nil && !ru.srcFinish.Before(*ru.srcStart) {
		return secs(ru.srcFinish.Sub(*ru.srcStart)), MeaningStation, OriginSource
	}
	total := ru.finish.Sub(ru.start)
	if len(ru.pauses) == 0 {
		return secs(total), MeaningStation, OriginSystem
	}
	var paused time.Duration
	for _, p := range ru.pauses {
		end := *ru.finish
		if p.to != nil && p.to.Before(end) {
			end = *p.to
		}
		if end.After(p.from) {
			paused += end.Sub(p.from)
		}
	}
	if paused > total {
		paused = total
	}
	return secs(total - paused), MeaningActive, OriginSystem
}

func secs(d time.Duration) int64 {
	if d < 0 {
		return 0
	}
	return int64(d / time.Second)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
