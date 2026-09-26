package process_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	app "ant/internal/application/process"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	jc "ant/internal/contracts/journal"
	dp "ant/internal/domain/process"
)

// Сквозной путь на фейках в памяти, как в cmd/ant: приём фактов в журнал →
// воркер с Bundles (свёртка, реакции, проекции process.item) → проектор
// (process.index, process.runs) → операции чтения process (живая карта,
// карточка узла, версии).
type liveWorld struct {
	t       *testing.T
	j       *enginemem.Journal
	codec   *engineapp.Codec
	reg     *engineapp.Registry
	store   *app.MemVersions
	bundles *app.Bundles
	svc     *app.LiveService
	hash    string
	n       int
	lastSeq int
}

func newLiveWorld(t *testing.T) *liveWorld {
	t.Helper()
	ctx := context.Background()
	clock := t0.Add(48 * time.Hour)
	j := enginemem.New(func() time.Time { return clock })
	j.SetEpoch(appjournal.PartitionLease(0), 1)
	w := &liveWorld{t: t, j: j, reg: engineapp.NewRegistry(), store: &app.MemVersions{},
		codec: &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}}
	if err := app.RegisterProjections(w.reg); err != nil {
		t.Fatal(err)
	}
	seed, err := app.EnsureSeed(ctx, w.store, flangeXML(t), t0.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	w.hash = seed.Hash
	w.bundles = &app.Bundles{Store: w.store, TTL: time.Nanosecond}
	w.svc = &app.LiveService{Store: j, States: engineapp.StateQueries{Codec: w.codec, Bundles: w.bundles}, Library: w.store, Bundles: w.bundles,
		Clock: func(context.Context) (time.Time, error) { return clock, nil }}
	return w
}

// fact — запись приёма (решения человека — с происхождением personal).
func (w *liveWorld) fact(item string, tp catalog.Type, h float64, data map[string]any) string {
	w.t.Helper()
	w.n++
	id := fmt.Sprintf("0190a000-0000-7000-8000-%012d", w.n)
	info, _ := catalog.Lookup(tp)
	p, err := w.codec.Encode(context.Background(), engineapp.Out{EventID: id, Type: tp, Kind: info.Kind, Stream: "item:" + item, ItemID: item,
		OccurredAt: at(h), Data: data})
	if err != nil {
		w.t.Fatal(err)
	}
	p.Entry.SourceID = "terminal-1"
	if info.Kind == catalog.KindDecision {
		p.Entry.ProvenanceClass = jc.JournalEntryProvenanceClassPersonal
	} else {
		p.Entry.ProvenanceClass = jc.JournalEntryProvenanceClassDevice
	}
	if _, err := w.j.Append(context.Background(), appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
		w.t.Fatal(err)
	}
	return id
}

// sync — воркер по всем изделиям с новым входом, затем проектор.
func (w *liveWorld) sync() {
	w.t.Helper()
	ctx := context.Background()
	part := engineapp.Partition{Number: 0, Epoch: 1}
	feed := &enginemem.Feed{J: w.j, Parts: []engineapp.Partition{part}}
	wk := engineapp.NewWorker(engineapp.WorkerConfig{Feed: feed, Codec: w.codec, Projections: w.reg, Bundles: w.bundles})
	for {
		nctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		works, err := feed.Next(nctx, part)
		cancel()
		if errors.Is(err, context.DeadlineExceeded) {
			break
		}
		if err := wk.Process(ctx, part, works); err != nil {
			w.t.Fatal(err)
		}
	}
	var fresh []jc.JournalEntry
	for _, e := range w.j.Entries() {
		if e.Seq > w.lastSeq {
			fresh = append(fresh, e)
			w.lastSeq = e.Seq
		}
	}
	pr := &engineapp.Projector{Consumer: w.j, Codec: w.codec, Store: w.j, Registry: w.reg}
	for _, g := range w.reg.Globals() {
		rq, err := pr.Apply(ctx, g, fresh)
		if err != nil {
			w.t.Fatal(err)
		}
		if _, err := w.j.Append(ctx, rq); err != nil {
			w.t.Fatal(err)
		}
	}
}

func (w *liveWorld) register(item string, h float64) {
	w.fact(item, catalog.ItemItemRegistered, h, map[string]any{"item_id": item, "item_type_id": "FL-100.00.000", "item_revision": "Б",
		"process_version_hash": w.hash, "normative_rev": "flange-1", "lot_ids": []string{"LOT-1"}})
}

func (w *liveWorld) count(t catalog.Type) int {
	n := 0
	for _, e := range w.j.Entries() {
		if e.EventType == string(t) {
			n++
		}
	}
	return n
}

func counter(lm app.LiveMap, step string) app.MapNodeCounters {
	for _, c := range lm.Counters {
		if c.StepKey == step {
			return c
		}
	}
	return app.MapNodeCounters{}
}

func TestLiveMapOverWorker(t *testing.T) {
	w := newLiveWorld(t)
	ctx := context.Background()
	a, b := "ent01:FL-0001", "ent01:FL-0002"
	w.register(a, 0)
	w.register(b, 0.5)
	w.fact(a, catalog.DecisionPresentationResolved, 1, map[string]any{"step_key": "incoming.zt1_lot_acceptance", "closing_point": "ZT-1",
		"resolution": "accept", "presentation_no": 1, "method_event_ids": []string{}})
	w.fact(a, catalog.OperationRunStarted, 2, map[string]any{"operation_run_id": "RUN-A-1", "operation_code": "010", "step_key": "machining.cnc", "operator_id": "OP-1"})
	w.sync()

	lm, err := w.svc.LiveMap(ctx, app.LiveMapQuery{Period: "day"}, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if lm.ProcessVersion.ProcessVersionID != app.SeedVersionID || !lm.ProcessVersion.IsCurrent || lm.ProcessVersion.Items != 2 || !strings.Contains(lm.BpmnXML, "Process_Flange") {
		t.Fatalf("версия на карте: %+v", lm.ProcessVersion)
	}
	pos := map[string]app.MapItem{}
	for _, it := range lm.Items {
		pos[it.ItemID] = it
	}
	if pos[a].StepKey != "machining.cnc" || pos[a].Position != dp.PosInProgress || pos[a].Label != "FL-0001" {
		t.Fatalf("изделие A: %+v", pos[a])
	}
	if pos[b].StepKey != "incoming.lot_registration" || pos[b].Position != dp.PosInQueue {
		t.Fatalf("изделие B: %+v", pos[b])
	}
	if c := counter(lm, "machining.cnc"); c.InProgress != 1 {
		t.Fatalf("счётчик мехобработки: %+v", c)
	}
	if c := counter(lm, "incoming.zt1_lot_acceptance"); c.Passed != 1 {
		t.Fatalf("выполнено на ЗТ-1: %+v", c)
	}
	if len(lm.DataGaps) == 0 || lm.BasisSeq == 0 {
		t.Fatalf("пропуски данных и basis_seq: %+v %d", lm.DataGaps, lm.BasisSeq)
	}
	// Интервал выполнения для machinelogs (эпик 23) записан реакцией.
	if w.count(catalog.OperationRunIntervalResolved) != 1 {
		t.Fatalf("operation.run.interval_resolved: %d", w.count(catalog.OperationRunIntervalResolved))
	}
	// Карта на момент до запуска второго изделия — та же свёртка на префиксе (AD-22).
	early := at(0.25)
	lm2, err := w.svc.LiveMap(ctx, app.LiveMapQuery{}, platform.Moment{AsOf: &early})
	if err != nil {
		t.Fatal(err)
	}
	if len(lm2.Items) != 1 || lm2.Items[0].ItemID != a || lm2.Items[0].StepKey != "incoming.lot_registration" {
		t.Fatalf("карта на момент: %+v", lm2.Items)
	}

	// Карточка узла (FR-154, FR-156).
	card, err := w.svc.Node(ctx, "", "welding.weld", platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if card.ElementID != "W2" || card.Documentation == "" || card.Properties["specialProcess"] != true || len(card.NormRefs) == 0 || card.Lane == "" {
		t.Fatalf("карточка сварки: %+v", card)
	}
	cnc, _ := w.svc.Node(ctx, app.SeedVersionID, "machining.cnc", platform.Moment{})
	if len(cnc.Items) != 1 || cnc.Items[0].ID != a {
		t.Fatalf("изделия узла: %+v", cnc.Items)
	}

	// Версии (FR-22…FR-24).
	vl, err := w.svc.Versions(ctx, platform.Moment{})
	if err != nil || len(vl.Items) != 1 || vl.Items[0].Quorum.Have != 3 || vl.Items[0].ItemsInWork != 2 || vl.Items[0].Status != dp.StatusActive {
		t.Fatalf("версии: %+v %v", vl, err)
	}
	v, err := w.svc.Version(ctx, app.SeedVersionID, platform.Moment{})
	if err != nil || len(v.Elements) != 106 || v.Hash != w.hash {
		t.Fatalf("версия в читаемом виде: %d элементов, %v", len(v.Elements), err)
	}
	bp, _ := w.svc.Bpmn(ctx, app.SeedVersionID)
	if dp.VersionHash([]byte(bp.BpmnXML)) != bp.Hash {
		t.Fatal("BPMN версии не как загружен")
	}

	// Черновик: проверка при загрузке (FR-13) — отказ с кодом и id элемента.
	bad := strings.Replace(string(flangeXML(t)), `<bpmn:task id="W1"`, `<bpmn:scriptTask id="SCRIPT1"/><bpmn:task id="W1"`, 1)
	_, err = w.svc.DraftVersion(ctx, app.DraftVersion{Label: "v2", BpmnXML: bad})
	pe, ok := platform.AsError(err)
	if !ok || pe.Code != errcodes.ProcessUnsupportedElement || pe.Params["element"] != "SCRIPT1" {
		t.Fatalf("черновик с неподдерживаемым элементом: %v", err)
	}
	good := strings.Replace(string(flangeXML(t)), `waitLimitMinutes="60" repeatAuthority`, `waitLimitMinutes="45" repeatAuthority`, 1)
	rc, err := w.svc.DraftVersion(ctx, app.DraftVersion{Label: "v2", BpmnXML: good})
	if err != nil {
		t.Fatal(err)
	}
	diff, err := w.svc.Diff(ctx, rc.EventIDs[0], "", platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Entries) != 0 {
		// Порог ожидания — не наше свойство ant:properties: разницы свойств нет.
		t.Logf("разница: %+v", diff.Entries)
	}
}

// FR-23: подделка XML в хранилище в обход системы — изделие не исполняется,
// на карте видна причина; гард команды отказывает.
func TestTamperedVersionStopsExecution(t *testing.T) {
	w := newLiveWorld(t)
	ctx := context.Background()
	a := "ent01:FL-0001"
	w.register(a, 0)
	w.sync()
	w.store.Tamper(app.SeedVersionID, []byte(strings.Replace(string(flangeXML(t)), "Сварка", "Сворка", 1)))
	w.fact(a, catalog.DecisionPresentationResolved, 1, map[string]any{"step_key": "incoming.zt1_lot_acceptance", "closing_point": "ZT-1",
		"resolution": "accept", "presentation_no": 1, "method_event_ids": []string{}})
	w.sync()
	raw, ok, err := w.j.Get(ctx, app.ProjectionItem, a)
	if err != nil || !ok {
		t.Fatal("нет проекции изделия", err)
	}
	if !strings.Contains(string(raw), `"refused":"process.version_tampered"`) {
		t.Fatalf("проекция изделия без отказа версии: %s", raw)
	}
	_, err = w.svc.StartOperation(ctx, a, app.StartOperation{OperationRunID: "RUN-1", OperationCode: "010", StepKey: "incoming.marking"})
	pe, ok := platform.AsError(err)
	if !ok || pe.Code != errcodes.ProcessVersionTampered {
		t.Fatalf("команда по изменённой версии: %v", err)
	}
}
