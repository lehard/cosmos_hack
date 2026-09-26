// Пакет scripted — общий stand оборудования модуля machinelogs (FR-149,
// AD-18, NFR-TEST-2): «эмулятор» кейса, который по программе выполнений
// отдаёт edge-агенту телеметрию по контракту
// contracts/internal/stands/telemetry.v1.json (AD-46) — маркеры цикла,
// уставки программы, события и условия MTConnect, отсчёты. Станок ЧПУ
// (cnc/stand) и сварочный источник (welder/stand) — программы этого stand-а.
//
// Эмулирует: одно нормальное выполнение и одно с двумя отклонениями на
// каждом виде оборудования (FR-149), по очереди или по запросу.
// Граница эмуляции: нет реального контроллера и полевой шины, значения — из
// программы; протокол stand → edge-агент настоящий. Состояние — в памяти
// процесса роли stands (одна копия, AD-6); сбои — каркас stand-ов эпика 06
// (служебный порт /stand/_control/).
//
// Слой: infrastructure/integration. Владелец: эпик 23 (MachineLogs).
package scripted

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
	"ant/internal/infrastructure/integration/ingest/stands"
)

// Sample — отсчёт SAMPLE: целое + масштаб (AD-4).
type Sample struct {
	Parameter string
	Value     int
	Scale     int
	Unit      string
}

// Event — событие или условие MTConnect (категория контракта телеметрии).
type Event struct {
	Category  string
	Value     string
	Code      string
	ToolUsed  *int
	ToolLimit *int
}

// Step — одно сообщение программы: маркер цикла, события, отсчёты.
type Step struct {
	Cycle   string // "start" | "end" | ""
	Events  []Event
	Samples []Sample
}

// Execution — программа одного выполнения.
type Execution struct {
	Kind  string // "normal" | "deviations"
	Steps []Step
}

// Stand — stand оборудования по программе выполнений.
type Stand struct {
	Name        string
	EquipmentID string
	Emulates    string
	EdgeURL     string
	Interval    time.Duration
	// Setpoints — уставки программы; отдаются в сообщении начала цикла.
	Setpoints []procs.StandTelemetryV1SetpointsElem
	// Executions — программы выполнений по видам; Order — очередность по умолчанию.
	Executions map[string]Execution
	Order      []string
	// Prepare — правка шага перед отправкой (номер выполнения, шаг): ресурс
	// инструмента, номер цикла.
	Prepare func(n int, s Step) Step
	RunID   string
	Client  *http.Client
	Now     func() time.Time

	once   sync.Once
	faults *stands.FaultSwitch
	mu     sync.Mutex
	seq    int
	n      int      // номер текущего выполнения
	cur    []Step   // оставшиеся шаги текущего выполнения
	kind   string   // вид текущего выполнения
	queue  []string // заказанные выполнения
	done   []string // завершённые выполнения (виды)
}

func (s *Stand) init() {
	s.once.Do(func() {
		if s.Now == nil {
			s.Now = time.Now
		}
		if s.Client == nil {
			s.Client = &http.Client{Timeout: 10 * time.Second}
		}
		if s.Interval <= 0 {
			s.Interval = time.Second
		}
		if len(s.Order) == 0 {
			s.Order = []string{"normal", "deviations"}
		}
		s.faults = stands.NewFaultSwitch(nil, s.Now)
	})
}

// Info — описание stand-а (NFR-TEST-2).
func (s *Stand) Info() app.StandInfo {
	s.init()
	return app.StandInfo{Name: s.Name, Emulates: s.Emulates,
		Boundary: "нет реального контроллера и полевой шины; значения — программа выполнений (нормальное и с двумя отклонениями); протокол stand → edge-агент (telemetry.v1) настоящий"}
}

// Faults — сбои stand-а (каркас эпика 06).
func (s *Stand) Faults() *stands.FaultSwitch { s.init(); return s.faults }

// Enqueue заказывает выполнение вида kind (оно пойдёт следующим после текущего).
func (s *Stand) Enqueue(kind string) error {
	s.init()
	if _, ok := s.Executions[kind]; !ok {
		return fmt.Errorf("stand %s: нет выполнения %q", s.Name, kind)
	}
	s.mu.Lock()
	s.queue = append(s.queue, kind)
	s.mu.Unlock()
	return nil
}

// Done — виды завершённых выполнений по порядку.
func (s *Stand) Done() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.done...)
}

// Handler — протокол stand-а для страницы тестовых сценариев:
//
//	GET  /            — вид текущего выполнения, очередь, завершённые;
//	POST /executions  — заказать выполнение {"kind": "normal" | "deviations"}.
func (s *Stand) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		v := map[string]any{"equipment_id": s.EquipmentID, "current": s.kind, "queue": s.queue, "done": s.done}
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	})
	mux.HandleFunc("POST /executions", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Kind string `json:"kind"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.Enqueue(in.Kind); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
	return mux
}

// next — следующий шаг программы (начинает новое выполнение при необходимости).
func (s *Stand) next() (Step, int) {
	if len(s.cur) == 0 {
		kind := s.Order[s.n%len(s.Order)]
		if len(s.queue) > 0 {
			kind, s.queue = s.queue[0], s.queue[1:]
		}
		s.kind = kind
		s.cur = append([]Step(nil), s.Executions[kind].Steps...)
		s.n++
	}
	st := s.cur[0]
	s.cur = s.cur[1:]
	if len(s.cur) == 0 {
		s.done = append(s.done, s.kind)
	}
	if s.Prepare != nil {
		st = s.Prepare(s.n, st)
	}
	return st, s.n
}

// Message — следующее сообщение телеметрии (номер растёт и при сбое drop:
// пропуск виден получателю как разрыв).
func (s *Stand) Message() procs.StandTelemetryV1 {
	s.init()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	at := s.Now().Add(s.faults.ClockSkew()).UTC().Format("2006-01-02T15:04:05.000Z")
	st, n := s.next()
	m := procs.StandTelemetryV1{StandID: s.Name, EquipmentID: s.EquipmentID, Seq: s.seq, SentAt: at}
	if s.RunID != "" {
		run := s.RunID
		m.RunID = &run
	}
	if st.Cycle != "" {
		m.Cycle = &procs.StandTelemetryV1Cycle{CycleRef: fmt.Sprintf("%s-%04d", s.EquipmentID, n), Phase: procs.StandTelemetryV1CyclePhase(st.Cycle)}
		if st.Cycle == "start" {
			m.Setpoints = s.Setpoints
		}
	}
	for _, e := range st.Events {
		x := procs.StandTelemetryV1EventsElem{Category: procs.StandTelemetryV1EventsElemCategory(e.Category), Value: e.Value, At: at,
			ToolLifeUsed: e.ToolUsed, ToolLifeLimit: e.ToolLimit}
		if e.Code != "" {
			c := e.Code
			x.Code = &c
		}
		m.Events = append(m.Events, x)
	}
	for _, x := range st.Samples {
		m.Samples = append(m.Samples, procs.StandTelemetryV1SamplesElem{Parameter: x.Parameter, At: at, Value: x.Value, Scale: x.Scale, Unit: x.Unit})
	}
	return m
}

// corrupt — «кривое» сообщение: поле вне закрытой схемы.
func corrupt(b []byte) []byte {
	return []byte(strings.Replace(string(b), "{", `{"firmware_note":"2.4-beta",`, 1))
}

// Tick — один шаг: сообщение со сбоями отправляется edge-агенту.
func (s *Stand) Tick(ctx context.Context) error {
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

// Run — отправка по интервалу; ошибки связи не останавливают stand.
func (s *Stand) Run(ctx context.Context) error {
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

var _ stands.Stand = (*Stand)(nil)

// Ptr — указатель на целое (ресурс инструмента в программах).
func Ptr(v int) *int { return &v }
