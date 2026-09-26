package stands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/contracts/procs"
)

// EquipmentStand — stand оборудования (FR-149): сварочный источник или станок
// ЧПУ отдаёт телеметрию edge-агенту (POST ‹EdgeURL›/v1/telemetry) по контракту
// contracts/internal/stands/telemetry.v1.json.
//
// Эмулирует: смену режима работы по классификации MTConnect (ACTIVE / READY),
// режим управления и исправность, отсчёт параметра (ток сварки) на каждом шаге.
// Граница эмуляции: нет реального контроллера и полевой шины — значения
// задаёт программа шагов; протокол (сообщение stand → edge-агент) — настоящий.
type EquipmentStand struct {
	Name        string
	EquipmentID string
	EdgeURL     string
	Interval    time.Duration
	// Program — циклическая программа шагов: значения execution по шагам.
	Program []string
	// Parameter, Unit — отсчёт на каждом шаге (целое + масштаб, AD-4).
	Parameter string
	Unit      string
	Client    *http.Client
	Now       func() time.Time

	once   sync.Once
	faults *FaultSwitch
	mu     sync.Mutex
	seq    int64
	step   int
}

func (s *EquipmentStand) init() {
	s.once.Do(func() {
		if s.Now == nil {
			s.Now = time.Now
		}
		if s.Client == nil {
			s.Client = &http.Client{Timeout: 10 * time.Second}
		}
		if s.Interval <= 0 {
			s.Interval = 5 * time.Second
		}
		if len(s.Program) == 0 {
			s.Program = []string{"ACTIVE", "ACTIVE", "READY"}
		}
		if s.Parameter == "" {
			s.Parameter, s.Unit = "current", "A"
		}
		s.faults = NewFaultSwitch(nil, s.Now)
	})
}

// Info — описание stand-а.
func (s *EquipmentStand) Info() app.StandInfo {
	s.init()
	return app.StandInfo{Name: s.Name, Emulates: "оборудование " + s.EquipmentID + ": телеметрия MTConnect → edge-агент",
		Boundary: "нет реального контроллера и полевой шины; значения — программа шагов; протокол stand → edge-агент настоящий"}
}

// Faults — сбои stand-а.
func (s *EquipmentStand) Faults() *FaultSwitch { s.init(); return s.faults }

// Handler — входящего протокола нет: stand только отправляет.
func (s *EquipmentStand) Handler() http.Handler { return nil }

// Message — следующее сообщение телеметрии (номер растёт и при сбоях drop:
// пропуск виден получателю как разрыв).
func (s *EquipmentStand) Message() procs.StandTelemetryV1 {
	s.init()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	at := s.Now().Add(s.faults.ClockSkew()).UTC().Format("2006-01-02T15:04:05.000Z")
	exec := s.Program[s.step%len(s.Program)]
	s.step++
	value := 1600 + (s.seq%5)*10 // 160,0 … 164,0 А, масштаб 1
	return procs.StandTelemetryV1{StandID: s.Name, EquipmentID: s.EquipmentID, Seq: int(s.seq), SentAt: at,
		Samples: []procs.StandTelemetryV1SamplesElem{{Parameter: s.Parameter, At: at, Value: int(value), Scale: 1, Unit: s.Unit}},
		Events: []procs.StandTelemetryV1EventsElem{
			{Category: "execution", Value: exec, At: at},
			{Category: "controller_mode", Value: "AUTOMATIC", At: at},
			{Category: "condition", Value: "NORMAL", At: at},
		}}
}

// corrupt — «кривое» сообщение: поле вне закрытой схемы.
func corrupt(b []byte) []byte {
	return []byte(strings.Replace(string(b), "{", `{"firmware_note":"2.4-beta",`, 1))
}

// Tick — один шаг: сообщение со сбоями отправляется edge-агенту.
func (s *EquipmentStand) Tick(ctx context.Context) error {
	b, err := json.Marshal(s.Message())
	if err != nil {
		return err
	}
	for _, m := range s.Faults().Outgoing(b, corrupt) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.EdgeURL, "/")+"/v1/telemetry", bytes.NewReader(m))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.Client.Do(req)
		if err != nil {
			return fmt.Errorf("stand %s → edge-агент: %w", s.Name, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("stand %s → edge-агент: HTTP %d", s.Name, resp.StatusCode)
		}
	}
	return nil
}

// Run — отправка по интервалу; ошибки связи не останавливают stand (повтор на следующем шаге).
func (s *EquipmentStand) Run(ctx context.Context) error {
	s.init()
	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			_ = s.Tick(ctx)
		}
	}
}
