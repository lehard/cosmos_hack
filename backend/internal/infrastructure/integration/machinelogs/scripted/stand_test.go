package scripted_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/contracts/procs"
	cncstand "ant/internal/infrastructure/integration/machinelogs/cnc/stand"
	weldstand "ant/internal/infrastructure/integration/machinelogs/welder/stand"
)

// Программа по умолчанию: нормальное выполнение, затем с отклонениями;
// заказ через протокол stand-а меняет очередь; уставки — в начале цикла.
func TestScriptOrderAndEnqueue(t *testing.T) {
	s := cncstand.New("cnc-1", "CNC-1", "", time.Second)
	var starts int
	for len(s.Done()) < 2 {
		m := s.Message()
		if m.Cycle != nil && m.Cycle.Phase == procs.StandTelemetryV1CyclePhaseStart {
			starts++
			if len(m.Setpoints) != 2 {
				t.Fatal("уставки программы — в сообщении начала цикла")
			}
		}
	}
	if d := s.Done(); d[0] != "normal" || d[1] != "deviations" || starts != 2 {
		t.Fatalf("очерёдность: %v", d)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/executions", strings.NewReader(`{"kind":"deviations"}`)))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("заказ выполнения: %d", rec.Code)
	}
	for len(s.Done()) < 3 {
		s.Message()
	}
	if s.Done()[2] != "deviations" {
		t.Fatalf("заказанное выполнение: %v", s.Done())
	}
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/executions", strings.NewReader(`{"kind":"nope"}`)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("неизвестное выполнение: %d", rec.Code)
	}
}

// Сообщение уходит edge-агенту по контракту; сбой «повтор» каркаса удваивает отправку.
func TestTickWithFault(t *testing.T) {
	var got []procs.StandTelemetryV1
	edge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var m procs.StandTelemetryV1
		dec := json.NewDecoder(strings.NewReader(string(b)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&m); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		got = append(got, m)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer edge.Close()
	s := weldstand.New("weld-is-2", "IS-2", edge.URL, time.Second)
	if err := s.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Faults().Set(app.Fault{Kind: app.FaultDuplicate}); err != nil {
		t.Fatal(err)
	}
	if err := s.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[1].Seq != got[2].Seq || got[0].EquipmentID != "IS-2" {
		t.Fatalf("отправлено: %d", len(got))
	}
	if !strings.Contains(s.Info().Emulates, "сварочный источник") {
		t.Fatal("описание эмуляции")
	}
}
