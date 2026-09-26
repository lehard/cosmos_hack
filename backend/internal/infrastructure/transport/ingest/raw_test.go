package ingest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	app "ant/internal/application/ingest"
	"ant/internal/application/ingest/inmem"
)

// Тонкий обработчик: problem+json с кодом и карантином, 202 и 200 при повторе,
// пачка — итог по каждому сообщению, 413 — слишком большая пачка.
func TestRawHandler(t *testing.T) {
	cfg := app.DefaultConfig()
	cfg.MaxBatch = 2
	h := RawHandler(inmem.NewCore(cfg, nil, nil).Service)
	do := func(path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		return rec
	}
	ev := `{"event_id":"01929a2b-7c3d-7e4f-8a5b-6c7d8e9f1004","event_type":"operation.run.paused","schema_version":1,
"source_id":"terminal-weld-2","source_seq":44,"source_kind":"manual_entry","reliability":"high","occurred_at":"2026-09-25T10:15:30.123Z",
"correlation_id":"01929a2b-7c3d-7e4f-8a5b-000000000001","causation_id":null,"item_id":"ENT01:FL-0007",
"integrity":{"format_version":1,"crypto_profile":"gost","signers":["person-welder-o17@1"]},"data":{"operation_run_id":"run-W2-FL-0007-1","pause_reason":"break"}}`
	if r := do(PathEvents, ev); r.Code != http.StatusAccepted {
		t.Fatalf("%d %s", r.Code, r.Body)
	}
	if r := do(PathEvents, ev); r.Code != http.StatusOK || !strings.Contains(r.Body.String(), `"duplicate"`) {
		t.Fatalf("повтор: %d %s", r.Code, r.Body)
	}
	bad := strings.Replace(ev, `"schema_version":1`, `"schema_version":7`, 1)
	r := do(PathEvents, bad)
	var p map[string]any
	_ = json.Unmarshal(r.Body.Bytes(), &p)
	if r.Code != 422 || r.Header().Get("Content-Type") != "application/problem+json" || p["code"] != "ingest.unknown_schema_version" || p["quarantine_id"] == "" {
		t.Fatalf("карантин: %d %s", r.Code, r.Body)
	}
	b := `{"source_id":"terminal-weld-2","sent_at":"2026-09-25T10:16:00.000Z","messages":[` + ev + `,` + bad + `]}`
	r = do(PathBatches, b)
	var br app.BatchResult
	_ = json.Unmarshal(r.Body.Bytes(), &br)
	if r.Code != 200 || len(br.Results) != 2 || br.Results[0].Outcome != app.OutcomeDuplicate || br.Results[1].Outcome != app.OutcomeQuarantined {
		t.Fatalf("пачка: %d %s", r.Code, r.Body)
	}
	b = `{"messages":[` + ev + `,` + ev + `,` + ev + `]}`
	if r = do(PathBatches, b); r.Code != http.StatusRequestEntityTooLarge || !strings.Contains(r.Body.String(), "ingest.batch_too_large") {
		t.Fatalf("большая пачка: %d %s", r.Code, r.Body)
	}
	// Без подключённых портов — 501 api.not_implemented.
	rec := httptest.NewRecorder()
	RawHandler(app.NewService()).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, PathEvents, strings.NewReader(ev)))
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("не подключён: %d", rec.Code)
	}
}
