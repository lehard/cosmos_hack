package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/application/ingest/inmem"
	ingesthttp "ant/internal/infrastructure/transport/ingest"
)

// DemoSummary — итог самопоказа (для теста и защиты).
type DemoSummary struct {
	Cases          map[string]string // файл примера → «итог код»
	RepeatSeqSame  bool
	JournalBefore  int
	JournalAfter   int
	Duplicates     int64
	ConflictCA     int
	ConflictSecEvt int
	Buffered       int
	Facts          int
	Accepted       int64
	DupAfterLoss   int64
	Completeness   int64
}

// link — вход ядра с выключателем связи: up — как есть; down — соединение
// рвётся (ядро недоступно); lose — ядро обработало, но ответ потерян (502).
type link struct {
	mode  atomic.Value
	inner http.Handler
}

func (l *link) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch l.mode.Load() {
	case "down":
		if hj, ok := w.(http.Hijacker); ok {
			if c, _, err := hj.Hijack(); err == nil {
				_ = c.Close()
				return
			}
		}
		http.Error(w, "ядро недоступно", http.StatusServiceUnavailable)
	case "lose":
		rec := httptest.NewRecorder()
		l.inner.ServeHTTP(rec, r)
		http.Error(w, "ответ потерян", http.StatusBadGateway)
	default:
		l.inner.ServeHTTP(w, r)
	}
}

// findExamples — каталог примеров пяти случаев FR-29.
func findExamples(dir string) (string, error) {
	cands := []string{dir, "../contracts/events/examples/contract-change", "contracts/events/examples/contract-change", "../../contracts/events/examples/contract-change", "../../../contracts/events/examples/contract-change"}
	for _, c := range cands {
		if c == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("не найден каталог примеров contracts/events/examples/contract-change (флаг -examples)")
}

// runDemo — самопоказ приёма и edge-агента на ядре в памяти через настоящий
// HTTP-вход: пять случаев FR-29, повтор, конфликт содержимого, недоступность и
// восстановление (FR-39), метрики (FR-41).
func runDemo(ctx context.Context, out io.Writer, examplesDir, stateDir string) (DemoSummary, error) {
	sum := DemoSummary{Cases: map[string]string{}}
	cfg := app.DefaultConfig()
	cfg.Profile = "demo"
	core := inmem.NewCore(cfg, nil, nil)
	l := &link{inner: ingesthttp.RawHandler(core.Service)}
	l.mode.Store("up")
	srv := httptest.NewServer(l)
	defer srv.Close()
	p := func(format string, a ...any) { _, _ = fmt.Fprintf(out, format+"\n", a...) }
	post := func(body []byte) (int, map[string]any, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+ingesthttp.PathEvents, bytes.NewReader(body))
		resp, err := srv.Client().Do(req)
		if err != nil {
			return 0, nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		var m map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&m)
		return resp.StatusCode, m, nil
	}

	dir, err := findExamples(examplesDir)
	if err != nil {
		return sum, err
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	slices.Sort(files)
	p("== Пять случаев изменения контракта (FR-29, кейс §4.7) — примеры %s", dir)
	var accepted []byte
	var acceptedSeq any
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return sum, err
		}
		var ex struct {
			Case    string          `json:"case"`
			Message json.RawMessage `json:"message"`
		}
		if err := json.Unmarshal(b, &ex); err != nil {
			return sum, err
		}
		st, m, err := post(ex.Message)
		if err != nil {
			return sum, err
		}
		outcome, code := str(m["outcome"]), str(m["code"])
		if outcome == "" { // problem+json
			outcome = "quarantined"
			if code == "ingest.duplicate_conflict" {
				outcome = "conflict"
			}
		}
		extra := ""
		if q := str(m["quarantine_id"]); q != "" {
			extra = " → карантин " + q
		}
		if fl, ok := m["flags"].([]any); ok && len(fl) > 0 {
			extra = fmt.Sprintf(" → флаг %v", fl[0].(map[string]any)["raw_value"])
		}
		if n := str(m["signature_note"]); n != "" {
			extra += " (" + n + ")"
		}
		p("  %-36s %-58s HTTP %d  %s %s%s", filepath.Base(f), ex.Case, st, outcome, code, extra)
		sum.Cases[filepath.Base(f)] = strings.TrimSpace(outcome + " " + code)
		if outcome == "accepted" {
			accepted, acceptedSeq = ex.Message, m["seq"]
		}
	}

	p("\n== Повтор (FR-31, кейс §4.5): то же сообщение ещё 3 раза")
	sum.JournalBefore = len(core.Journal.Main())
	seqs := []any{acceptedSeq}
	for range 3 {
		st, m, err := post(accepted)
		if err != nil {
			return sum, err
		}
		seqs = append(seqs, m["seq"])
		p("  HTTP %d  %s  seq=%v  replayed=%v", st, str(m["outcome"]), m["seq"], m["replayed"])
	}
	sum.JournalAfter = len(core.Journal.Main())
	sum.RepeatSeqSame = len(slices.Compact(slices.Clone(seqs))) == 1
	st, _ := core.Service.Stats(ctx)
	sum.Duplicates = st.Duplicates
	p("  записей в журнале: до %d, после %d; дублей учтено: %d — показатели не изменились", sum.JournalBefore, sum.JournalAfter, st.Duplicates)

	p("\n== Тот же event_id, другое содержимое (FR-31, AD-7)")
	var ev map[string]any
	_ = json.Unmarshal(accepted, &ev)
	ev["data"].(map[string]any)["station_id"] = "weld-3"
	bad, _ := json.Marshal(ev)
	code, m, err := post(bad)
	if err != nil {
		return sum, err
	}
	sum.ConflictCA, sum.ConflictSecEvt = len(core.Journal.CA()), core.Journal.Count("security.idempotency.conflict")
	p("  HTTP %d  %s  %s", code, str(m["code"]), str(m["detail"]))
	p("  событий безопасности: %d; записей журнала критических действий: %d; исходное не перезаписано", sum.ConflictSecEvt, sum.ConflictCA)

	p("\n== Недоступность и восстановление (FR-39): edge-агент edge-weld-7")
	a, err := NewAgent(Config{SourceID: "edge-weld-7", CoreURL: srv.URL, StateDir: stateDir, BatchSize: 3,
		SourceKind: "machine", Reliability: "high"}, Unsigned{Ref: "device-edge-weld-7@1"}, srv.Client(), nil, nil)
	if err != nil {
		return sum, err
	}
	facts0 := core.Journal.Count("equipment.state.changed")
	l.mode.Store("down")
	base := time.Now().UTC().Add(-10 * time.Minute)
	for i := range 5 {
		e := map[string]any{"event_type": "equipment.state.changed", "schema_version": 2,
			"occurred_at": base.Add(time.Duration(i) * time.Minute).Format("2006-01-02T15:04:05.000Z"),
			"data": map[string]any{"equipment_id": "IS-2", "station_id": "weld-2", "execution": []string{"running", "idle"}[i%2],
				"controller_mode": "automatic", "condition": "normal"}}
		if _, err := a.Enqueue(e); err != nil {
			return sum, err
		}
	}
	_, derr := a.Drain(ctx)
	sum.Buffered = a.Status().Buffered
	p("  ядро недоступно: досылка — %v; в буфере агента %d событий (source_seq 1–5, исходное occurred_at)", short(derr), sum.Buffered)
	l.mode.Store("lose")
	_, lerr := a.Flush(ctx)
	p("  связь вернулась, но ответ на первую пачку потерян: %v; в буфере %d", short(lerr), a.Status().Buffered)
	l.mode.Store("up")
	n, err := a.Drain(ctx)
	if err != nil {
		return sum, err
	}
	st, _ = core.Service.Stats(ctx)
	sum.Facts = core.Journal.Count("equipment.state.changed") - facts0
	sum.Accepted = st.Accepted
	sum.DupAfterLoss = st.Duplicates - sum.Duplicates
	for _, s := range st.Sources {
		if s.SourceID == "edge-weld-7" {
			sum.Completeness = s.CompletenessBP
		}
	}
	p("  досылка: отправлено %d; в буфере %d; фактов в журнале %d из 5; повторов (подтверждены как дубли) %d; полнота источника %d.%02d %%",
		n, a.Status().Buffered, sum.Facts, sum.DupAfterLoss, sum.Completeness/100, sum.Completeness%100)

	p("\n== Метрики приёма (FR-41)")
	js, _ := json.MarshalIndent(st, "  ", "  ")
	p("  %s", js)
	return sum, nil
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func short(err error) string {
	if err == nil {
		return "успешно"
	}
	s := err.Error()
	if len(s) > 90 {
		s = s[:90] + "…"
	}
	return s
}
