package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	crossapp "ant/internal/application/crossitem"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	app "ant/internal/application/ingest"
	"ant/internal/application/ingest/inmem"
	appjournal "ant/internal/application/journal"
	mlapp "ant/internal/application/machinelogs"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/procs"
	dcross "ant/internal/domain/crossitem"
	"ant/internal/domain/kernel"
	ml "ant/internal/domain/machinelogs"
	cncstand "ant/internal/infrastructure/integration/machinelogs/cnc/stand"
	"ant/internal/infrastructure/integration/machinelogs/scripted"
	weldstand "ant/internal/infrastructure/integration/machinelogs/welder/stand"
)

var st0 = time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)

// clockStand — stand с часами, которые идут на step за сообщение.
func clockStand(s *scripted.Stand, step time.Duration) *scripted.Stand {
	now := st0
	s.Now = func() time.Time { t := now; now = now.Add(step); return t }
	return s
}

// messages — сообщения n выполнений stand-а (нормальное, с отклонениями, …).
func messages(s *scripted.Stand, executions int) []procs.StandTelemetryV1 {
	var out []procs.StandTelemetryV1
	for len(s.Done()) < executions {
		out = append(out, s.Message())
	}
	return out
}

type extracted struct {
	events []map[string]any
}

func extract(msgs []procs.StandTelemetryV1) extracted {
	x := NewExtractor()
	var e extracted
	for _, m := range msgs {
		e.events = append(e.events, x.Extract(m)...)
	}
	return e
}

func (e extracted) of(t string) []map[string]any {
	var out []map[string]any
	for _, ev := range e.events {
		if ev["event_type"] == t {
			out = append(out, ev)
		}
	}
	return out
}

func data(ev map[string]any) map[string]any { return ev["data"].(map[string]any) }

// FR-147, FR-149: станок ЧПУ — нормальное выполнение без отклонений,
// выполнение с двумя: ручная подача 130 % и перегрузка шпинделя; сводки на
// окно цикла вместо отсчётов.
func TestSummariesCNC(t *testing.T) {
	e := extract(messages(clockStand(cncstand.New("cnc-1", "CNC-1", "", 0), 20*time.Second), 2))
	cycles := e.of("equipment.cycle.summarized")
	if len(cycles) != 2 {
		t.Fatalf("сводок цикла: %d", len(cycles))
	}
	devs := e.of("equipment.deviation.detected")
	if len(devs) != 2 {
		t.Fatalf("отклонений: %d %v", len(devs), devs)
	}
	kinds := map[string]map[string]any{}
	for _, d := range devs {
		kinds[data(d)["deviation_kind"].(string)] = data(d)
	}
	feed := kinds["manual_override"]
	if feed == nil || feed["parameter"] != "feed_override" || feed["value"].(map[string]any)["value"] != int64(130) {
		t.Fatalf("ручная подача 130 %%: %v", feed)
	}
	load := kinds["overload"]
	if load == nil || load["value"].(map[string]any)["value"] != int64(118) {
		t.Fatalf("перегрузка шпинделя: %v", load)
	}
	// Отклонения — во втором цикле (после сводки первого).
	firstEnd := data(cycles[0])["window_end"].(string)
	for _, d := range devs {
		if data(d)["started_at"].(string) <= firstEnd {
			t.Fatal("в нормальном выполнении отклонений нет")
		}
	}
	if len(e.of("equipment.program.changed")) != 1 || len(e.of("equipment.tool.changed")) != 2 {
		t.Fatalf("программа и инструмент: %d %d", len(e.of("equipment.program.changed")), len(e.of("equipment.tool.changed")))
	}
	// Отсчёты не уходят в ant: только события и сводки.
	for _, ev := range e.events {
		if strings.Contains(ev["event_type"].(string), "sample") {
			t.Fatal("сырой отсчёт ушёл с края")
		}
	}
	// Повтор тех же сообщений даёт те же event_id (идемпотентность).
	again := extract(messages(clockStand(cncstand.New("cnc-1", "CNC-1", "", 0), 20*time.Second), 2))
	if len(again.events) != len(e.events) || again.events[len(again.events)-1]["event_id"] != e.events[len(e.events)-1]["event_id"] {
		t.Fatal("event_id сводок недетерминированы")
	}
}

// FR-149, FR-153: сварочный источник — ток выше уставки и ручное изменение режима.
func TestSummariesWelder(t *testing.T) {
	e := extract(messages(clockStand(weldstand.New("weld-is-2", "IS-2", "", 0), 20*time.Second), 2))
	devs := e.of("equipment.deviation.detected")
	if len(devs) != 2 {
		t.Fatalf("отклонений: %v", devs)
	}
	var current, manual bool
	for _, d := range devs {
		x := data(d)
		switch x["deviation_kind"] {
		case "out_of_setpoint":
			current = x["parameter"] == "current" && x["value"].(map[string]any)["value"] == int64(1820) && x["ended_at"] != nil
		case "manual_override":
			manual = x["parameter"] == "controller_mode"
		}
	}
	if !current || !manual {
		t.Fatalf("ток вне уставки и ручной режим: %v", devs)
	}
	var modes []string
	for _, s := range e.of("equipment.state.changed") {
		modes = append(modes, data(s)["controller_mode"].(string))
	}
	if !strings.Contains(strings.Join(modes, ","), "manual") {
		t.Fatalf("смена режима управления — equipment.state.changed: %v", modes)
	}
}

// Выделенные события проходят приём по контракту (схемы, повторы): stand →
// локальный вход агента → буфер → ядро; ни одного карантина.
func TestSummariesPassIngest(t *testing.T) {
	core := inmem.NewCore(app.DefaultConfig(), nil, nil)
	srv := httptest.NewServer(coreHandler(core.Service))
	defer srv.Close()
	a, _ := NewAgent(Config{SourceID: "edge-shop-1", CoreURL: srv.URL, StateDir: t.TempDir()}, Unsigned{Ref: "device-edge-shop-1@1"}, srv.Client(), nil, nil)
	h := LocalHandler(a, NewExtractor())
	var total int
	for _, s := range []*scripted.Stand{cncstand.New("cnc-1", "CNC-1", "", 0), weldstand.New("weld-is-2", "IS-2", "", 0)} {
		for _, m := range messages(clockStand(s, 20*time.Second), 2) {
			b, _ := json.Marshal(m)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/telemetry", bytes.NewReader(b)))
			if rec.Code != http.StatusAccepted {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
			var enq []Enqueued
			_ = json.Unmarshal(rec.Body.Bytes(), &enq)
			total += len(enq)
		}
	}
	if n, err := a.Drain(context.Background()); err != nil || n != total {
		t.Fatalf("досылка %d из %d: %v", n, total, err)
	}
	st, _ := core.Service.Stats(context.Background())
	if st.Accepted != int64(total) || st.Quarantined != 0 {
		t.Fatalf("приём: %+v (ждали %d принятых)", st, total)
	}
}

// Сквозной путь FR-147 → FR-148: stand станка → выделитель edge-агента →
// журнал → межизделийная стадия (привязка) → свёртка изделия → профиль
// выполнения операции: ручная подача 130 % видна как отклонение.
func TestStandToRunProfile(t *testing.T) {
	ctx := context.Background()
	s := clockStand(cncstand.New("cnc-1", "CNC-1", "", 0), 20*time.Second)
	e := extract(messages(s, 2))
	cycles := e.of("equipment.cycle.summarized")

	j := enginemem.New(func() time.Time { return st0.Add(time.Hour) })
	j.SetEpoch(appjournal.PartitionLease(0), 1)
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}
	add := func(o engineapp.Out) {
		p, err := codec.Encode(ctx, o)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
			t.Fatal(err)
		}
	}
	at := func(s string) time.Time { v, _ := time.Parse(time.RFC3339, s); return v }
	for i, ev := range e.events {
		d := data(ev)
		id, _ := ev["event_id"].(string)
		if id == "" { // событие базового выделителя: event_id ставит агент
			id = kernel.UUIDv5(constants.NsAnt, fmt.Sprint("edge-test-", i))
		}
		add(engineapp.Out{EventID: id, Type: catalog.Type(ev["event_type"].(string)), Kind: catalog.KindFact,
			Stream: "equipment:CNC-1", OccurredAt: at(ev["occurred_at"].(string)), Data: d})
	}
	// Терминал: выполнения операции по окнам циклов (как их сообщил MES).
	for i, c := range cycles {
		run, item := []string{"RUN-1", "RUN-2"}[i], []string{"ENT01:FL-1", "ENT01:FL-2"}[i]
		ws, we := at(data(c)["window_start"].(string)), at(data(c)["window_end"].(string))
		add(engineapp.Out{EventID: "start-" + run, Type: catalog.OperationRunStarted, Kind: catalog.KindFact, Stream: "item:" + item, ItemID: item,
			OccurredAt: ws, Data: map[string]any{"operation_run_id": run, "operation_code": "010", "step_key": "machining.turn", "equipment_id": "CNC-1", "operator_id": "T-03"}})
		add(engineapp.Out{EventID: "finish-" + run, Type: catalog.OperationRunFinished, Kind: catalog.KindFact, Stream: "item:" + item, ItemID: item,
			OccurredAt: we, Data: map[string]any{"operation_run_id": run, "completion": "completed"}})
	}
	sr := &crossapp.StageRunner{Consumer: j, Codec: codec, Store: j}
	_, rq, err := sr.Apply(ctx, dcross.Stage{}, j.Entries())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(ctx, rq); err != nil {
		t.Fatal(err)
	}
	reg := engineapp.NewRegistry()
	if err := mlapp.RegisterProjections(reg, ml.Env{}); err != nil {
		t.Fatal(err)
	}
	part := engineapp.Partition{Number: 0, Epoch: 1}
	feed := &enginemem.Feed{J: j, Parts: []engineapp.Partition{part}}
	wk := engineapp.NewWorker(engineapp.WorkerConfig{Feed: feed, Codec: codec, Projections: reg})
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
	pr := &engineapp.Projector{Consumer: j, Codec: codec, Store: j, Registry: reg}
	for _, g := range reg.Globals() {
		rq, err := pr.Apply(ctx, g, j.Entries())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := j.Append(ctx, rq); err != nil {
			t.Fatal(err)
		}
	}
	svc := mlapp.NewLiveService(j, engineapp.StateQueries{Codec: codec}, ml.Env{})
	bad, err := svc.RunProfile(ctx, "RUN-2", platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	var feed130 bool
	var devs int
	for _, r := range bad.Events {
		if r.Layer == "deviation" {
			devs++
			feed130 = feed130 || (r.Params["deviation_kind"] == "manual_override" && r.Params["value"] == "130 %")
		}
	}
	if !feed130 || devs != 2 {
		t.Fatalf("профиль с отклонениями: %+v", bad.Events)
	}
	good, err := svc.RunProfile(ctx, "RUN-1", platform.Moment{})
	if err != nil || len(good.Parameters) == 0 {
		t.Fatalf("нормальный профиль: %+v %v", good, err)
	}
	for _, r := range good.Events {
		if r.Layer == "deviation" {
			t.Fatalf("нормальное выполнение с отклонением: %+v", r)
		}
	}
}
