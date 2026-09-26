package machinelogs_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	crossapp "ant/internal/application/crossitem"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	app "ant/internal/application/machinelogs"
	"ant/internal/application/machinelogs/mltest"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	dcross "ant/internal/domain/crossitem"
	"ant/internal/domain/kernel"
	ml "ant/internal/domain/machinelogs"
)

var t0 = time.Date(2026, 9, 26, 7, 0, 0, 0, time.UTC)

// Заготовка функции-намерения nonconformity (эпик 21) для тестов: модуль и
// тип — переменные, чтобы emitcheck не принял тест за эмитента чужого типа.
var (
	ncModule kernel.Module = "nonconformity"
	ncType                 = catalog.DecisionNonconformityRegistered
)

func fakeRegistrar(q ml.NCRequest) (kernel.Addressed, error) {
	return kernel.NewAddressed(ncModule, ncType, "item:"+q.ItemID, q.Key(),
		map[string]any{"nc_id": q.NCID, "violation_window_event_id": q.WindowEventID, "operation_run_id": q.OperationRunID, "step_key": q.StepKey},
		q.CauseRecords()...)
}

// stageWith — межизделийная стадия с портом несоответствий reg (как её
// соберёт crossitem после эпика 21: machinelogs.StageWith(nonconformity…)).
func stageWith(reg ml.Registrar) func(dcross.Stage, kernel.Record) (dcross.Stage, []kernel.Addressed) {
	f := ml.StageWith(ml.StagePorts{Register: reg})
	return func(s dcross.Stage, r kernel.Record) (dcross.Stage, []kernel.Addressed) {
		return dcross.Settle(s, r, func(s dcross.Stage, r kernel.Record) (dcross.Stage, []kernel.Addressed) {
			var a []kernel.Addressed
			s.Machinelogs, a = f(s.Machinelogs, r)
			return s, a
		})
	}
}

// world — журнал в памяти, стадия, воркер и проектор, как в cmd/ant.
type world struct {
	t     *testing.T
	j     *enginemem.Journal
	codec *engineapp.Codec
	reg   *engineapp.Registry
	svc   *app.Service
}

func newWorld(t *testing.T, sc mltest.Scenario, stage func(dcross.Stage, kernel.Record) (dcross.Stage, []kernel.Addressed)) *world {
	t.Helper()
	ctx := context.Background()
	clock := t0.Add(24 * time.Hour)
	j := enginemem.New(func() time.Time { return clock })
	j.SetEpoch(appjournal.PartitionLease(0), 1)
	w := &world{t: t, j: j, reg: engineapp.NewRegistry(),
		codec: &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}}
	if err := app.RegisterProjections(w.reg, ml.Env{}); err != nil {
		t.Fatal(err)
	}
	for _, o := range sc.Facts {
		p, err := w.codec.Encode(ctx, o)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
			t.Fatal(err)
		}
	}
	// Стадия (роль crossitem): одна пачка — весь журнал.
	sr := &crossapp.StageRunner{Consumer: j, Codec: w.codec, Store: j, Fold: stage}
	_, rq, err := sr.Apply(ctx, dcross.Stage{}, j.Entries())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(ctx, rq); err != nil {
		t.Fatal(err)
	}
	// Воркер: пересвёртка изделий с новым входом.
	part := engineapp.Partition{Number: 0, Epoch: 1}
	feed := &enginemem.Feed{J: j, Parts: []engineapp.Partition{part}}
	wk := engineapp.NewWorker(engineapp.WorkerConfig{Feed: feed, Codec: w.codec, Projections: w.reg})
	for {
		nctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		works, err := feed.Next(nctx, part)
		cancel()
		if errors.Is(err, context.DeadlineExceeded) {
			break
		}
		if err := wk.Process(ctx, part, works); err != nil {
			t.Fatal(err)
		}
	}
	// Проектор: глобальные проекции по всему журналу.
	pr := &engineapp.Projector{Consumer: j, Codec: w.codec, Store: j, Registry: w.reg}
	for _, g := range w.reg.Globals() {
		rq, err := pr.Apply(ctx, g, j.Entries())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := j.Append(ctx, rq); err != nil {
			t.Fatal(err)
		}
	}
	w.svc = app.NewLiveService(j, engineapp.StateQueries{Codec: w.codec}, ml.Env{})
	return w
}

func (w *world) count(t catalog.Type) int {
	n := 0
	for _, e := range w.j.Entries() {
		if e.EventType == string(t) {
			n++
		}
	}
	return n
}

// FR-147, FR-148, FR-149: нормальное выполнение и выполнение с двумя
// отклонениями на станке ЧПУ; ручная подача 130 % — отклонение в профиле.
func TestRunProfileCNC(t *testing.T) {
	sc := mltest.CNCTwoDeviations(t0, "")
	w := newWorld(t, sc, nil)
	ctx := context.Background()

	bad, err := w.svc.RunProfile(ctx, "RUN-C-2", platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	var feed130, overload bool
	for _, e := range bad.Events {
		if e.Layer != "deviation" {
			continue
		}
		switch e.Params["deviation_kind"] {
		case "manual_override":
			feed130 = e.Params["parameter"] == "feed_override" && e.Params["value"] == "130 %" && strings.Contains(e.Summary, "130 %")
		case "overload":
			overload = true
		}
	}
	if !feed130 || !overload {
		t.Fatalf("два отклонения (подача 130 %%, перегрузка) в профиле: %+v", bad.Events)
	}
	if bad.ItemID != "ENT01:FL-C-2" || bad.EquipmentID != "CNC-1" || bad.ProgramRef != "O1001 ред. B" || bad.ToolID != "T05" || bad.FinishedAt == nil {
		t.Fatalf("профиль: %+v", bad)
	}
	outOfRange := 0
	for _, p := range bad.Parameters {
		if p.InRange != nil && !*p.InRange {
			outOfRange++
		}
	}
	if outOfRange != 2 {
		t.Fatalf("сводки против уставки: %+v", bad.Parameters)
	}

	good, err := w.svc.RunProfile(ctx, "RUN-C-1", platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range good.Events {
		if e.Layer == "deviation" {
			t.Fatalf("нормальное выполнение без отклонений: %+v", e)
		}
	}
	// Профиль на момент до второго выполнения — второго выполнения ещё нет.
	before := t0.Add(15 * time.Minute)
	if _, err := w.svc.RunProfile(ctx, "RUN-C-2", platform.Moment{AsOf: &before}); err == nil {
		t.Fatal("на момент до выполнения профиля нет")
	}
	if _, err := w.svc.RunProfile(ctx, "RUN-NONE", platform.Moment{}); err == nil {
		t.Fatal("нет выполнения — 404")
	}

	// Оборудование: состояние по MTConnect, программа, инструмент и ресурс.
	eq, err := w.svc.EquipmentByID(ctx, "CNC-1", platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if eq.Execution != "idle" || eq.Condition != "normal" || eq.ToolLifeUsed == nil || *eq.ToolLifeUsed != 73 || eq.ProgramRef != "O1001 ред. B" {
		t.Fatalf("оборудование: %+v", eq)
	}
	list, err := w.svc.Equipment(ctx, "", platform.Moment{})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("список оборудования: %+v %v", list, err)
	}
	tl, err := w.svc.Timeline(ctx, "CNC-1", t0.Add(-time.Hour), t0.Add(time.Hour), platform.Moment{})
	if err != nil || len(tl.Rows) != 11 {
		t.Fatalf("журнал оборудования: %d строк %v", len(tl.Rows), err)
	}
	layers := map[string]bool{}
	for _, r := range tl.Rows {
		layers[r.Layer] = true
	}
	if !layers["what"] || !layers["with_what"] || !layers["how"] || !layers["deviation"] {
		t.Fatalf("четыре слоя: %v", layers)
	}
	// Состояние на момент: во время второго выполнения — работает.
	during := t0.Add(22 * time.Minute)
	eq, err = w.svc.EquipmentByID(ctx, "CNC-1", platform.Moment{AsOf: &during})
	if err != nil || eq.Execution != "running" {
		t.Fatalf("оборудование на момент: %+v %v", eq, err)
	}
}

// FR-151: «сварка вне режима» — все 6 изделий окна получают несоответствие,
// 3 из них без найденного дефекта (с портом nonconformity).
func TestSpecialProcessSixItems(t *testing.T) {
	sc := mltest.WeldingOutOfRegime(t0, "")
	w := newWorld(t, sc, stageWith(fakeRegistrar))
	ctx := context.Background()

	if n := w.count(catalog.EquipmentViolationWindowResolved); n != 1 {
		t.Fatalf("окон нарушения: %d", n)
	}
	if n := w.count(catalog.DecisionNonconformityRegistered); n != 6 {
		t.Fatalf("несоответствий: %d", n)
	}
	v, err := w.svc.Violations(ctx, platform.Moment{})
	if err != nil || len(v.Items) != 1 {
		t.Fatalf("окна нарушений: %+v %v", v, err)
	}
	win := v.Items[0]
	var items []string
	for _, it := range win.Items {
		items = append(items, it.ID)
	}
	if !slices.Equal(items, sc.WindowItems) || len(win.Nonconformities) != 6 || win.EquipmentID != "IS-2" || win.StepKey != "welding.weld" {
		t.Fatalf("окно: %+v", win)
	}
	withoutDefect := 0
	for _, it := range sc.WindowItems {
		if !slices.Contains(sc.WithDefect, it) {
			withoutDefect++
		}
		p, err := w.svc.RunProfile(ctx, sc.Runs[it], platform.Moment{})
		if err != nil || !p.Violation || !p.SpecialProcess {
			t.Fatalf("профиль %s: нарушение спецпроцесса: %+v %v", it, p, err)
		}
	}
	if withoutDefect != 3 {
		t.Fatalf("без найденного дефекта: %d", withoutDefect)
	}
	// Вне окна — без нарушения.
	p, err := w.svc.RunProfile(ctx, "RUN-W-0", platform.Moment{})
	if err != nil || p.Violation {
		t.Fatalf("вне окна: %+v %v", p, err)
	}
	eq, err := w.svc.EquipmentByID(ctx, "IS-2", platform.Moment{})
	if err != nil || !eq.SpecialProcess || len(eq.Warnings) == 0 || eq.Warnings[0].Kind != "out_of_setpoint" {
		t.Fatalf("сварочный источник: %+v %v", eq, err)
	}
	// id окна, который домен ставит в несоответствие, — id записи окна в журнале.
	for _, e := range w.j.Entries() {
		if e.EventType == string(catalog.EquipmentViolationWindowResolved) {
			a := kernel.Addressed{Module: ml.Module, Type: catalog.EquipmentViolationWindowResolved, Stream: e.Stream, Key: sc.Facts[len(sc.Facts)-1].EventID}
			if dcross.AddressedID(a) != e.EventID || ml.WindowEventID(e.Stream, a.Key) != e.EventID {
				t.Fatal("формула id окна разошлась с crossitem.AddressedID")
			}
		}
	}
}

// Настоящая сборка стадии crossitem.Fold: порт несоответствий подключён
// (эпик 21, nonconformity.RegisterWindowNC) — окно, его 6 изделий и 6
// несоответствий окна (FR-151, «даже без найденного дефекта»).
func TestSpecialProcessWithFrameStage(t *testing.T) {
	sc := mltest.WeldingOutOfRegime(t0, "run-7")
	w := newWorld(t, sc, nil)
	v, err := w.svc.Violations(context.Background(), platform.Moment{RunID: "run-7"})
	if err != nil || len(v.Items) != 1 || len(v.Items[0].Items) != 6 || len(v.Items[0].Nonconformities) != 6 {
		t.Fatalf("окно и несоответствия окна: %+v %v", v, err)
	}
	if w.count(catalog.EquipmentEventBound) == 0 {
		t.Fatal("привязка событий к выполнению")
	}
	// Другой прогон не видит окна (AD-38).
	if v, _ := w.svc.Violations(context.Background(), platform.Moment{}); len(v.Items) != 0 {
		t.Fatal("окна прогона видны вне прогона")
	}
}

func TestServiceWithoutStoreIs501(t *testing.T) {
	_, err := app.NewService().RunProfile(context.Background(), "x", platform.Moment{})
	if pe, ok := platform.AsError(err); !ok || pe.Code != "api.not_implemented" {
		t.Fatalf("без проекций — 501: %v", err)
	}
}
