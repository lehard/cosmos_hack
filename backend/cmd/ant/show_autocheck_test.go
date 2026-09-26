package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestShowIS2Autocheck — автопроверка показа SHOW-IS2 на живом движке (весь
// ant профиля demo на своей БД): прогон autocheck до конца, табло «ожидалось →
// получилось» — строго все строки совпали; не совпавшие — с расшифровкой
// (что проверяется, операция и путь, ожидалось, получилось, пояснение).
//
//	ANT_SHOW_AUTOCHECK=1 go test -run TestShowIS2Autocheck -v ./cmd/ant/
//
// (со своей БД: make dev-db; рецепт — agent-brief.md, «Сквозной тест показа»).
func TestShowIS2Autocheck(t *testing.T) {
	if os.Getenv("ANT_SHOW_AUTOCHECK") == "" || os.Getenv("ANT_DB_HOST") == "" {
		t.Skip("автопроверка SHOW-IS2 на живом движке — только по ANT_SHOW_AUTOCHECK=1 и со своей БД (make dev-db)")
	}
	s := startShowSystem(t)
	code, out := s.call("ADM-01", "simulation.run.start", map[string]string{"scenario_id": "SHOW-IS2"},
		map[string]any{"command_id": newID(), "basis_seq": 0, "policy_seq": 0, "mode": "autocheck", "speed": 1000})
	if code != http.StatusOK || str(out, "run_id") == "" {
		t.Fatalf("запуск автопроверки SHOW-IS2: HTTP %d %v", code, out)
	}
	s.runID = str(out, "run_id")
	t.Logf("прогон %s", s.runID)
	var last map[string]any
	s.until("автопроверка не дошла до конца", 30*time.Minute, func() bool {
		last = s.run()
		switch str(last, "state") {
		case "failed", "stopped":
			t.Fatalf("прогон остановился: %v", last)
		}
		return str(last, "state") == "completed"
	})
	// Невыполненные шаги прогона — в журнале процесса (ANT_SHOW_LOG=файл): «прогон: шаг не выполнен».
	b := s.read("ADM-01", "simulation.board.read", map[string]string{"run_id": s.runID})
	rows := list(b, "rows")
	var bad []string
	passed := 0
	for _, r := range rows {
		if str(r, "status") == "passed" {
			passed++
			continue
		}
		// Целостность пишет отдельный процесс verifier (cmd/verifier): в этом
		// тесте его нет — индикатор «unknown»; на стенде verifier работает.
		if str(r, "assertion_id") == "SHOW-13" && str(r, "actual") == `"unknown"` {
			t.Logf("строка SHOW-13 — нет процесса verifier в тесте: %s", str(r, "actual"))
			continue
		}
		// «Видно на экране» (mapping manual) табло не проверяет по определению.
		if str(r, "mapping") == "manual" && str(r, "status") == "pending" {
			t.Logf("строка %s — проверяется глазами: %s", str(r, "assertion_id"), str(r, "title"))
			continue
		}
		bad = append(bad, fmt.Sprintf("  %s [%s] «%s»\n    %s %s\n    ожидалось %s, получилось %s\n    %s",
			str(r, "assertion_id"), str(r, "status"), str(r, "title"), str(r, "operation_id"), str(r, "path"),
			str(r, "expected"), str(r, "actual"), str(r, "detail")))
	}
	t.Logf("табло: %d из %d", passed, len(rows))
	if len(bad) > 0 {
		// Расшифровка: что записано в журнал прогона по спорным строкам.
		for _, et := range []string{"decision.presentation.resolved", "incident.cause.concluded", "decision.disposition.set", "operation.run.started"} {
			l := s.read("ADM-01", "journal.entry.list", map[string]string{"event_type": et, "limit": "500"})
			for _, e := range list(l, "items") {
				d, _ := e["data"].(map[string]any)
				t.Logf("журнал %s seq %s item %s: %v", et, str(e, "seq"), str(e, "item_id"), d)
			}
		}
		for _, inc := range list(s.read("TEC-01", "analysis.incident.list", nil), "items") {
			t.Logf("инцидент %s: %s общий фактор %v, НС %v", str(inc, "incident_id"), str(inc, "label"), inc["common_factor"], inc["nc_ids"])
		}
		t.Fatalf("табло SHOW-IS2 — не совпало %d из %d:\n%s", len(bad), len(rows), strings.Join(bad, "\n"))
	}
}
