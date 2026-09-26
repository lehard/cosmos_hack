package analytics_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	analytics "ant/internal/application/analytics"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
)

// Сквозная проверка на фейках (журнал движка в памяти): факты → воркер
// заменяет вклады изделия целиком (AD-45) → проектор ведёт глобальные
// проекции → показатели читаются live-сервисом.

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC) // 11:00 МСК — первая смена дня

type fixedClock struct{ t time.Time }

func (c *fixedClock) Now(context.Context) (time.Time, error) { return c.t, nil }

type pipe struct {
	t      *testing.T
	j      *enginemem.Journal
	codec  *engineapp.Codec
	feed   *enginemem.Feed
	worker *engineapp.WorkerService
	reg    *engineapp.Registry
	part   engineapp.Partition
	items  []string
	clock  *fixedClock
	svc    *analytics.Service
	projAt int
}

func newPipe(t *testing.T) *pipe {
	t.Helper()
	j := enginemem.New(nil)
	part := engineapp.Partition{Number: 0, Epoch: 1}
	j.SetEpoch(appjournal.PartitionLease(0), 1)
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}
	reg := engineapp.NewRegistry()
	if err := analytics.Register(reg); err != nil {
		t.Fatal(err)
	}
	feed := &enginemem.Feed{J: j, Parts: []engineapp.Partition{part}}
	p := &pipe{t: t, j: j, codec: codec, feed: feed, reg: reg, part: part, clock: &fixedClock{t: t0.Add(4 * time.Hour)},
		worker: engineapp.NewWorker(engineapp.WorkerConfig{Feed: feed, Codec: codec, Projections: reg})}
	p.svc = analytics.NewService(analytics.WithStore(analytics.MemStore{Src: j, Items: func() []string { return p.items }}), analytics.WithClock(p.clock))
	return p
}

// fact — запись входа (факт с видом источника, решение) через тот же конверт, что у приёма.
func (p *pipe) fact(itemID, id string, typ catalog.Type, at time.Duration, data string, sourceKind string) {
	p.t.Helper()
	info, _ := catalog.Lookup(typ)
	stream := "item:" + itemID
	if itemID == "" {
		stream = "global"
	}
	pend, err := p.codec.Encode(context.Background(), engineapp.Out{EventID: id, Type: typ, Kind: info.Kind, Stream: stream, ItemID: itemID,
		OccurredAt: t0.Add(at), Data: json.RawMessage(data)})
	if err != nil {
		p.t.Fatal(err)
	}
	if sourceKind != "" {
		pend.Envelope = patch(p.t, pend.Envelope, sourceKind)
	}
	if _, err := p.j.Append(context.Background(), appjournal.AppendRequest{Batch: []appjournal.Pending{pend}}); err != nil {
		p.t.Fatal(err)
	}
	if itemID != "" && !slices.Contains(p.items, itemID) {
		p.items = append(p.items, itemID)
	}
}

func patch(t *testing.T, raw []byte, kind string) []byte {
	t.Helper()
	var d map[string]any
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	payload, _ := base64.StdEncoding.DecodeString(d["payload"].(string))
	var env map[string]any
	if err := json.Unmarshal(payload, &env); err != nil {
		t.Fatal(err)
	}
	env["source_kind"] = kind
	b, _ := json.Marshal(env)
	d["payload"] = base64.StdEncoding.EncodeToString(b)
	out, _ := json.Marshal(d)
	return out
}

// settle — воркер сворачивает всё новое, проектор применяет глобальные проекции.
func (p *pipe) settle() {
	p.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		works, err := p.feed.Next(ctx, p.part)
		if err != nil {
			for _, e := range p.j.Entries() {
				it := ""
				if e.ItemID != nil {
					it = *e.ItemID
				}
				p.t.Logf("%d %s %s part=%d", e.Seq, e.EventType, it, e.Partition)
			}
			p.t.Fatalf("воркер: %v, курсор %d", err, p.j.Cursor(appjournal.WorkerConsumer, 0))
		}
		if len(works) == 0 {
			break
		}
		if err := p.worker.Process(ctx, p.part, works); err != nil {
			p.t.Fatal(err)
		}
		if p.feed.J.Cursor(appjournal.WorkerConsumer, 0) >= works[len(works)-1].UpToSeq {
			if w, _ := p.pending(); !w {
				break
			}
		}
	}
	entries := p.j.Entries()
	batch := entries[p.projAt:]
	p.projAt = len(entries)
	pr := &engineapp.Projector{Codec: p.codec, Store: p.j, Registry: p.reg}
	for _, g := range p.reg.Globals() {
		rq, err := pr.Apply(ctx, g, batch)
		if err != nil {
			p.t.Fatal(err)
		}
		if _, err := p.j.Append(ctx, rq); err != nil {
			p.t.Fatal(err)
		}
	}
}

func (p *pipe) pending() (bool, error) {
	cur := p.j.Cursor(appjournal.WorkerConsumer, 0)
	for _, e := range p.j.Entries() {
		if e.EventType == string(catalog.OpsProcessingFailed) {
			env, _ := p.j.Open(context.Background(), e)
			p.t.Fatalf("обработка остановлена: %s", env.Raw)
		}
		if info, ok := catalog.Lookup(catalog.Type(e.EventType)); ok && info.Role == engineapp.RoleWorker {
			continue // выход воркера — не триггер (AD-5)
		}
		if int64(e.Seq) > cur && e.ItemID != nil {
			return true, nil
		}
	}
	return false, nil
}

// weld — изделие проходит сварку, контроль и предъявление на ЗТ-3.
func (p *pipe) weld(n int, at time.Duration, performer string, resolve bool) string {
	id := fmt.Sprintf("ENT01:FL-%03d", n)
	e := func(s string) string { return fmt.Sprintf("%s-%d", s, n) }
	p.fact(id, e("reg"), catalog.ItemItemRegistered, at, fmt.Sprintf(`{"item_id":%q,"item_type_id":"FL-100","item_revision":"A","process_version_hash":"x","normative_rev":"r1"}`, id), "external_system")
	p.fact(id, e("start"), catalog.OperationRunStarted, at+5*time.Minute, fmt.Sprintf(`{"operation_run_id":"RUN-%d","operation_code":"welding","step_key":"welding.weld","station_id":"ST-2","line_id":"LINE-FL-1","equipment_id":"IS-2","operator_id":%q}`, n, performer), "machine")
	p.fact(id, e("fin"), catalog.OperationRunFinished, at+35*time.Minute, fmt.Sprintf(`{"operation_run_id":"RUN-%d","completion":"completed"}`, n), "machine")
	p.fact(id, e("kt3"), catalog.InspectionResultRecorded, at+40*time.Minute, fmt.Sprintf(`{"observation_id":"OBS-%d","method":"visual","phase":"after_operation","step_key":"welding.kt3_camera","operation_run_id":"RUN-%d","outcome":"no_defect_indicated","processing_state":"completed"}`, n, n), "camera")
	p.fact(id, e("pres"), catalog.ItemPresentationRecorded, at+45*time.Minute, `{"step_key":"welding.zt3_acceptance","presentation_no":1,"presented_to":"qc","presented_by":"M01"}`, "manual_entry")
	if resolve {
		p.fact(id, e("zt3"), catalog.DecisionPresentationResolved, at+55*time.Minute, `{"step_key":"welding.zt3_acceptance","closing_point":"ZT-3","resolution":"accept","presentation_no":1,"method_event_ids":[]}`, "")
	}
	return id
}

func (p *pipe) overview(m platform.Moment) analytics.AnalyticsOverview {
	p.t.Helper()
	ov, err := p.svc.Overview(context.Background(), analytics.PeriodQuery{Kind: "shift"}, m)
	if err != nil {
		p.t.Fatal(err)
	}
	return ov
}

func row(ov analytics.AnalyticsOverview, id string) analytics.MetricRow {
	for _, r := range ov.Items {
		if r.MetricID == id {
			return r
		}
	}
	return analytics.MetricRow{}
}

// world — пять изделий, одно с подтверждённым дефектом, повтор операции,
// простой оборудования, вывод о причине и гипотеза.
func world(t *testing.T) *pipe {
	p := worldFacts(t)
	p.settle()
	return p
}

// worldFacts — факты world без ожидания воркера (тест дописывает свои).
func worldFacts(t *testing.T) *pipe {
	p := newPipe(t)
	for i := range 5 {
		perf := "W21"
		if i%2 == 1 {
			perf = "W22"
		}
		p.weld(i, time.Duration(i)*10*time.Minute, perf, true)
	}
	id := "ENT01:FL-002"
	p.fact(id, "rt-2", catalog.InspectionResultRecorded, 90*time.Minute, `{"observation_id":"RT-2","method":"radiography","phase":"after_operation","step_key":"welding.kt3_radiography","operation_run_id":"RUN-2","outcome":"defect_indicated","processing_state":"completed","defects":[{"defect_type_code":"POR","zone_id":"W1","location":"12h","severity":"major"}]}`, "sensor")
	p.fact(id, "nc-2", catalog.DecisionNonconformityConfirmed, 100*time.Minute, `{"nc_id":"NC-2","signal_ids":["S-2"],"severity":"major","defect_type_code":"POR","reason":{"text":"пора"}}`, "")
	p.fact(id, "rw-2", catalog.OperationRunStarted, 110*time.Minute, `{"operation_run_id":"RUN-2R","operation_code":"welding","step_key":"welding.weld","equipment_id":"IS-2","operator_id":"W22","rework_of":"RUN-2"}`, "machine")
	p.fact(id, "rwf-2", catalog.OperationRunFinished, 130*time.Minute, `{"operation_run_id":"RUN-2R","completion":"completed","reported_duration":{"value":18,"unit":"min","meaning":"active_processing","origin":"source_reported"}}`, "machine")
	p.fact("", "eq-1", catalog.EquipmentStateChanged, 60*time.Minute, `{"equipment_id":"IS-2","execution":"stopped","controller_mode":"automatic","condition":"fault"}`, "machine")
	p.fact("", "eq-2", catalog.EquipmentStateChanged, 80*time.Minute, `{"equipment_id":"IS-2","execution":"running","controller_mode":"automatic","condition":"normal"}`, "machine")
	p.fact("", "cause", catalog.IncidentCauseConcluded, 150*time.Minute, `{"incident_id":"INC-1","nc_ids":["NC-2"],"conclusion":"confirmed","category":"equipment","verification":"осциллограмма тока","reason":{"text":"ток вне уставки"}}`, "")
	p.fact("", "hyp", catalog.IncidentHypothesisRecorded, 140*time.Minute, `{"incident_id":"INC-1","nc_ids":["NC-2"],"branch":"why_made","category":"equipment","statement":"просадка тока"}`, "")
	return p
}

func TestOverviewOnFakes(t *testing.T) {
	p := world(t)
	ov := p.overview(platform.Moment{})
	for id, want := range map[string]int64{
		"inspected_items": 5, "items_with_confirmed_nc": 1, "confirmed_defects": 1, "incoming_defects": 0, "rework_runs": 1,
		"comparable_runs": 6, "equipment_caused_nc": 1, "hypotheses": 1, "cause_established": 10000, "first_pass_yield": 8000,
		"equipment_downtime": 20, "unable_to_assess": 0,
	} {
		if got := row(ov, id).Total.Value; got != want {
			t.Errorf("%s = %d, ожидалось %d", id, got, want)
		}
	}
	if acc := row(ov, "equipment_downtime").Account; acc == nil || *acc != "equipment" {
		t.Error("простой — в графе «оборудование»")
	}
	d := row(ov, "operation_duration").Total
	if d.Origin == nil || *d.Origin != "mixed" || d.Meaning == nil || *d.Meaning != "other" || d.MeaningNote == nil {
		t.Errorf("длительность операций: смешанное происхождение и смысл: %+v", d)
	}
	lt := row(ov, "comparable_runs")
	if len(lt.Slices) == 0 || lt.Slices[0].Dimension != "performer" {
		t.Errorf("сопоставимые работы по исполнителям: %+v", lt.Slices)
	}
}

// Б-30 (кейс §2.4, I2 и I6): дефекты по линиям и доля подтверждённых ошибок
// исполнителя на сопоставимых работах — числитель только по решению
// уполномоченного, знаменатель — выполнения группы.
func TestDefectsByLineAndPerformerErrorRate(t *testing.T) {
	p := worldFacts(t)
	p.fact("", "err-2", catalog.IncidentOperatorErrorConfirmed, 160*time.Minute, `{"incident_id":"INC-1","nc_ids":["NC-2"],"operator_id":"W21","reason":{"text":"нарушен режим"}}`, "")
	p.settle()
	ov := p.overview(platform.Moment{})

	byLine := row(ov, "defects_by_line")
	if byLine.Total.Value != 1 || len(byLine.Slices) != 1 || byLine.Slices[0].Key != "location:line/LINE-FL-1" || byLine.Slices[0].Value.Value != 1 {
		t.Errorf("дефекты по линиям: %+v / %+v", byLine.Total, byLine.Slices)
	}

	rate := row(ov, "performer_error_rate")
	if rate.Total.Value <= 0 {
		t.Fatalf("доля ошибок исполнителя: %+v", rate.Total)
	}
	var w21, w22 int64 = -1, -1
	for _, s := range rate.Slices {
		switch {
		case strings.HasPrefix(s.Label, "W21"):
			w21 = s.Value.Value
		case strings.HasPrefix(s.Label, "W22"):
			w22 = s.Value.Value
		}
	}
	if w21 <= 0 || w22 != 0 {
		t.Errorf("ошибка — только у W21 среди его сопоставимых работ: W21=%d W22=%d (%+v)", w21, w22, rate.Slices)
	}
}

// Каждое число раскрывается до записей (FR-7, AD-45): для каждого
// показателя и каждого среза раскрытие даёт тот же итог; строки штучных
// показателей складываются в итог; у каждой строки есть исходные записи и
// вид источника.
func TestEveryNumberDrillsDown(t *testing.T) {
	p := world(t)
	ov := p.overview(platform.Moment{})
	ctx := context.Background()
	check := func(metric, slice string, want analytics.MetricValue) {
		t.Helper()
		dd, err := p.svc.Drilldown(ctx, metric, slice, analytics.PeriodQuery{Kind: "shift"}, platform.Moment{}, platform.Page{Limit: 500})
		if err != nil {
			t.Fatalf("%s/%s: %v", metric, slice, err)
		}
		if dd.Total.Value != want.Value || dd.Total.Unit != want.Unit {
			t.Fatalf("%s/%s: раскрытие %d %s, показатель %d %s", metric, slice, dd.Total.Value, dd.Total.Unit, want.Value, want.Unit)
		}
		var sum int64
		for _, r := range dd.Items {
			sum += r.Value.Value
			if len(r.SourceEventIDs) == 0 {
				t.Fatalf("%s/%s: строка %s без исходных записей", metric, slice, r.ItemID)
			}
			if len(r.SourceKinds) == 0 {
				t.Fatalf("%s/%s: строка %s без вида источника", metric, slice, r.ItemID)
			}
			for _, k := range r.SourceKinds {
				if k == "fact" || k == "decision" || k == "reaction" {
					t.Fatalf("%s: в source_kinds вид записи %q, а не вид источника", metric, k)
				}
			}
		}
		if want.Unit == "pcs" && sum != want.Value {
			t.Fatalf("%s/%s: сумма строк %d ≠ итог %d", metric, slice, sum, want.Value)
		}
		if want.Value > 0 && len(dd.Items) == 0 {
			t.Fatalf("%s/%s: ненулевое число без строк", metric, slice)
		}
	}
	for _, r := range ov.Items {
		check(r.MetricID, "", r.Total)
		for _, s := range r.Slices {
			check(r.MetricID, s.Key, s.Value)
		}
	}
	tiles, err := p.svc.Tiles(ctx, analytics.PeriodQuery{Kind: "shift"}, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range tiles.Items {
		check(tl.MetricID, "", tl.Value)
	}
}

// Повтор не меняет показатели; позднее событие меняет ровно то, что должно
// (кейс §5.2, FR-88).
func TestRepeatAndLateEvent(t *testing.T) {
	p := world(t)
	before := p.overview(platform.Moment{})
	// Повтор: те же факты от источника ещё раз (новые event_id, тот же смысл).
	id := "ENT01:FL-001"
	p.fact(id, "start-1-again", catalog.OperationRunStarted, 15*time.Minute, `{"operation_run_id":"RUN-1","operation_code":"welding","step_key":"welding.weld","station_id":"ST-2","line_id":"LINE-FL-1","equipment_id":"IS-2","operator_id":"W22"}`, "machine")
	p.fact(id, "kt3-1-again", catalog.InspectionResultRecorded, 50*time.Minute, `{"observation_id":"OBS-1","method":"visual","phase":"after_operation","step_key":"welding.kt3_camera","operation_run_id":"RUN-1","outcome":"no_defect_indicated","processing_state":"completed"}`, "camera")
	p.fact("", "eq-1-again", catalog.EquipmentStateChanged, 60*time.Minute, `{"equipment_id":"IS-2","execution":"stopped","controller_mode":"automatic","condition":"fault"}`, "machine")
	p.settle()
	after := p.overview(platform.Moment{})
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(after)
	if string(a) != string(b) {
		t.Fatalf("повтор изменил показатели:\nдо    %s\nпосле %s", a, b)
	}
	// Позднее событие: дефект на изделии 3, случившийся до выпуска, пришёл сейчас.
	id = "ENT01:FL-003"
	p.fact(id, "rt-3", catalog.InspectionResultRecorded, 70*time.Minute, `{"observation_id":"RT-3","method":"radiography","phase":"after_operation","step_key":"welding.kt3_radiography","operation_run_id":"RUN-3","outcome":"defect_indicated","processing_state":"completed","defects":[{"defect_type_code":"POR","zone_id":"W2","location":"3h","severity":"major"},{"defect_type_code":"POR","zone_id":"W2","location":"3h","severity":"major"}]}`, "sensor")
	p.fact(id, "nc-3", catalog.DecisionNonconformityConfirmed, 160*time.Minute, `{"nc_id":"NC-3","signal_ids":["S-3"],"severity":"major","reason":{"text":"пора"}}`, "")
	p.settle()
	late := p.overview(platform.Moment{})
	for metric, delta := range map[string]int64{"confirmed_defects": 1, "items_with_confirmed_nc": 1, "inspected_items": 0, "comparable_runs": 0, "rework_runs": 0, "recurrence_rate": 1} {
		if got := row(late, metric).Total.Value - row(after, metric).Total.Value; got != delta {
			t.Errorf("позднее событие: %s изменился на %d, ожидалось %d", metric, got, delta)
		}
	}
	if row(late, "first_pass_yield").Total.Value != 6000 {
		t.Errorf("«с первого раза» после позднего дефекта: %d", row(late, "first_pass_yield").Total.Value)
	}
	// Момент «как было» до позднего дефекта: показатели прежние (AD-22).
	asOf := t0.Add(65 * time.Minute)
	was := p.overview(platform.Moment{AsOf: &asOf})
	if row(was, "confirmed_defects").Total.Value != 0 || row(was, "inspected_items").Total.Value != 3 || row(was, "first_pass_yield").Total.Value != 10000 {
		t.Errorf("на момент %s: %+v", asOf, was.Items[:3])
	}
}

// «Специалист заснул»: изделия предъявлены на ЗТ-3, решения нет — узел точки
// предъявления — ограничение линии и аномалия (FR-5).
func TestSpecialistAsleepOnFakes(t *testing.T) {
	p := newPipe(t)
	for i := range 8 {
		p.weld(i, time.Duration(i)*10*time.Minute, "W21", i < 2)
	}
	p.settle()
	nc, err := p.svc.NodeCounters(context.Background(), "", analytics.PeriodQuery{Kind: "shift"}, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if nc.Bottleneck == nil || nc.Bottleneck.StepKey != "welding.zt3_acceptance" {
		t.Fatalf("ограничение линии: %+v", nc.Bottleneck)
	}
	kinds := map[string]bool{}
	for _, a := range nc.Anomalies {
		if a.StepKey == "welding.zt3_acceptance" {
			kinds[a.Kind] = true
		}
	}
	if !kinds["queue_above_norm"] || !kinds["wait_above_norm"] {
		t.Fatalf("аномалии узла точки предъявления: %+v", nc.Anomalies)
	}
	for _, c := range nc.Counters {
		if c.StepKey == "welding.zt3_acceptance" && (c.Queue != 6 || c.Passed != 2) {
			t.Fatalf("счётчики ЗТ-3: %+v", c)
		}
		if c.StepKey == "welding.weld" && c.Passed != 8 {
			t.Fatalf("счётчики сварки: %+v", c)
		}
	}
	// Контрольная карта узла: название и вид.
	ch, err := p.svc.ControlChart(context.Background(), "welding.weld", "operation_duration", analytics.PeriodQuery{Kind: "shift"}, platform.Moment{})
	if err != nil || ch.Title == "" || len(ch.Points) != 8 || ch.Points[0].Ref == nil {
		t.Fatalf("карта длительности: %+v %v", ch, err)
	}
}
