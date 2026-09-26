package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ant/cmd/internal/config"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appingest "ant/internal/application/ingest"
	"ant/internal/application/ingest/inmem"
	appjournal "ant/internal/application/journal"
	opsapp "ant/internal/application/ops"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	dom "ant/internal/domain/ops"
	"ant/internal/infrastructure/storage/journal/journaltest"
	"ant/internal/infrastructure/storage/journal/migrator"
	opsstore "ant/internal/infrastructure/storage/ops"
)

func testEnv() *environment {
	return &environment{cfg: &config.Config{}, log: slog.New(slog.DiscardHandler)}
}

// FR-41, FR-2, FR-113: /metrics процесса отдаёт метрики приёма и
// ant_event_to_sse_seconds — приём и публикатор SSE пишут в телеметрию
// процесса (адаптер prometheus), роль api выдаёт её.
func TestMetricsEndpoint(t *testing.T) {
	env := testEnv()
	tel := env.telemetry()

	// Приём (эпик 06) с телеметрией процесса: принято, дубль, карантин.
	cfg := appingest.DefaultConfig()
	cfg.Profile = "demo"
	core := inmem.NewCore(cfg, nil, func(d *appingest.Deps) { d.Telemetry = tel })
	ev := fmt.Appendf(nil, `{"event_id":"01929a2b-7c3d-7e4f-8a5b-000000000001","event_type":"operation.run.started","schema_version":1,
"source_id":"edge-weld-1","source_seq":1,"source_kind":"machine","reliability":"high","occurred_at":"%s",
"correlation_id":"01929a2b-7c3d-7e4f-8a5b-000000000001","causation_id":null,"item_id":"ENT01:FL-0007",
"integrity":{"format_version":1,"crypto_profile":"gost","signers":["device-edge-weld-1@1"]},
"data":{"operation_run_id":"run-W2-FL-0007-1","operation_code":"030","step_key":"welding.weld","operator_id":"O17","station_id":"weld-2"}}`,
		time.Now().Add(-time.Minute).UTC().Format("2006-01-02T15:04:05.000Z"))
	ctx := context.Background()
	for _, raw := range [][]byte{ev, ev, []byte(`{"event_type":"x"`)} {
		if _, err := core.Service.Ingest(ctx, raw); err != nil {
			t.Fatal(err)
		}
	}

	// Живые обновления (эпик 07): изменение по записи-триггеру → SSE.
	mem := enginemem.New(nil)
	live := engineapp.NewLiveUpdates(engineapp.LiveConfig{Log: mem, Telemetry: tel})
	lctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { _ = live.Run(lctx) }()
	sub, err := live.Subscribe(lctx, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()
	if _, err := mem.Append(ctx, appjournal.AppendRequest{Effects: []appjournal.Effect{engineapp.Notify{Changes: []engineapp.Change{
		{Entity: platform.EntityItem, ID: "ENT01:FL-0007", Seq: 1, ReceivedAt: time.Now().Add(-300 * time.Millisecond)}}}}}); err != nil {
		t.Fatal(err)
	}
	nctx, ncancel := context.WithTimeout(lctx, 2*time.Second)
	defer ncancel()
	if _, err := sub.Next(nctx); err != nil {
		t.Fatalf("SSE: %v", err)
	}

	rec := httptest.NewRecorder()
	env.metricsHandler(nil).ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body, _ := io.ReadAll(rec.Body)
	s := string(body)
	if rec.Code != http.StatusOK {
		t.Fatalf("/metrics: %d", rec.Code)
	}
	for _, want := range []string{
		appingest.MetricMessages + `{code="",outcome="accepted"} 1`,
		appingest.MetricMessages + `{code="",outcome="duplicate"} 1`,
		appingest.MetricLatency + "_count",
		appingest.MetricDeliveryDelay + "_bucket",
		appingest.MetricQuarantineOpen,
		appingest.MetricCompleteness,
		engineapp.MetricEventToSSE + `_count{entity="item"} 1`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("нет %q в /metrics", want)
		}
	}
	if t.Failed() {
		t.Log(s)
	}
}

// FR-127, AD-45: администратор через HTTP API видит остановленное изделие
// и повторяет обработку (операции ops.* отвечают вживую).
func TestOpsLiveOverHTTP(t *testing.T) {
	j := enginemem.New(nil)
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}
	item := "ENT01:FL-0099"
	fact, err := codec.Encode(context.Background(), engineapp.Out{EventID: "01929a2b-7c3d-7e4f-8a5b-000000000010", Type: catalog.InspectionResultRecorded,
		Kind: catalog.KindFact, Stream: "item:" + item, ItemID: item, OccurredAt: time.Now(), Data: map[string]string{"outcome": "defect_indicated"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(context.Background(), appjournal.AppendRequest{Batch: []appjournal.Pending{fact}}); err != nil {
		t.Fatal(err)
	}
	all := j.Entries()
	fail, err := engineapp.FailureRequest(context.Background(), codec, appjournal.WorkerConsumer, item, 1, all[0], fmt.Errorf("паника свёртки"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(context.Background(), fail); err != nil {
		t.Fatal(err)
	}
	svc := opsapp.NewLive(opsapp.Config{Journal: j, Codec: codec, Partitions: 1, Profile: "demo", Version: "test", Mode: platform.ModeLive,
		Adapters: map[string]string{"telemetry": "prometheus"}, Enabled: []string{"onec"}})
	mux := http.NewServeMux()
	buildAPI(mux, apiOptions{mode: platform.ModeLive, ops: svc})

	get := func(path string, out any) {
		t.Helper()
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1"+path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %s", path, rec.Code, rec.Body)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatal(err)
		}
	}
	var list opsapp.StoppedItemList
	get("/ops/stopped-items", &list)
	if len(list.Items) != 1 || list.Items[0].ItemID != item {
		t.Fatalf("остановленные: %+v", list)
	}
	var health opsapp.OpsHealth
	get("/ops/health", &health)
	if health.StoppedItems != 1 || health.Profile != "demo" || len(health.Integrations) < 3 {
		t.Fatalf("здоровье: %+v", health)
	}
	var settings opsapp.SettingList
	get("/ops/settings", &settings)
	if len(settings.Ports) != len(platform.PortKeys) || len(settings.Modules) == 0 {
		t.Fatalf("настройки: %+v", settings)
	}

	body, _ := json.Marshal(map[string]any{"command_id": "01929a2b-7c3d-7e4f-8a5b-0000000000b1", "basis_seq": 0, "policy_seq": 0,
		"failure_event_id": list.Items[0].FailureEventID, "reason": map[string]string{"text": "правило исправлено"}})
	req := httptest.NewRequest("POST", "/api/v1/ops/stopped-items/"+item+"/retry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("повтор: %d %s", rec.Code, rec.Body)
	}
	get("/ops/stopped-items", &list)
	if len(list.Items) != 0 {
		t.Fatalf("после повтора: %+v", list.Items)
	}

	// Отключение и включение источника — решения в журнале, видны в настройках.
	for i, op := range []string{"disable", "enable", "disable"} {
		body, _ := json.Marshal(map[string]any{"command_id": fmt.Sprintf("01929a2b-7c3d-7e4f-8a5b-0000000000c%d", i), "basis_seq": 0, "policy_seq": 0, "reason": map[string]string{"text": "обслуживание " + op}})
		req := httptest.NewRequest("POST", "/api/v1/ops/sources/edge-weld-1/"+op, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", op, rec.Code, rec.Body)
		}
	}
	get("/ops/settings", &settings)
	if len(settings.Sources) != 1 || settings.Sources[0].Enabled || settings.Sources[0].Reason != "обслуживание disable" {
		t.Fatalf("источники: %+v", settings.Sources)
	}
}

// FR-109: самопроверка на своей БД (make dev-db) находит неприменённые
// миграции и отсутствие обязательного генезиса; после migrate — «без
// критических ошибок». Роли БД и права на журнал — настоящие.
func TestSelfCheckOnDB(t *testing.T) {
	db := journaltest.NewDB(t) // миграции только journal и engine
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool := db.AppPool(t)
	admin := adminPool(t, db)
	store := journaltestStore(pool)
	database := opsDatabase{Probe: opsstore.Probe{Pool: admin}, pool: admin}
	sc := &opsapp.SelfCheck{Journal: store, Database: database, RequireGenesis: true, Roles: []string{"api"}}
	v := sc.Run(ctx)
	if v.OK || !strings.Contains(v.Summary, "генезиса нет") || !strings.Contains(v.Summary, "модуль ingest") {
		t.Fatalf("находки: %+v", v)
	}
	for _, f := range v.Findings {
		if f.Check == dom.CheckDBRoles {
			t.Fatalf("роли БД в порядке, а нашлось: %+v", f)
		}
	}
	if _, err := migrator.Up(ctx, db.Admin.ConnConfig, nil, migrationSets...); err != nil {
		t.Fatal(err)
	}
	sc.RequireGenesis = false
	v = sc.Run(ctx)
	if !v.OK || v.Summary != dom.MsgOK {
		t.Fatalf("после migrate: %+v", v)
	}
	// Состояние ролей и очередей по настоящим арендам и курсорам.
	rt := opsRuntime{store: store, leases: journaltestLeases(pool)}
	if _, _, err := rt.leases.Acquire(ctx, "projector", "h:1", time.Minute); err != nil {
		t.Fatal(err)
	}
	ls, err := rt.Leases(ctx)
	if err != nil || len(ls) != 1 || ls[0].Name != "projector" {
		t.Fatalf("аренды: %+v %v", ls, err)
	}
	bl, err := rt.Backlogs(ctx, 4)
	if err != nil || len(bl) != 4 {
		t.Fatalf("курсоры: %+v %v", bl, err)
	}
}
