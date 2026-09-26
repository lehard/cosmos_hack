// Пакет standkit — общий каркас stand-ов видеофиксации модуля vision (VisionQC
// и OperatorVision; AD-18, NFR-TEST-2): «эмулятор» кейса поверх каркаса
// stand-ов эпика 06 (integration/ingest/stands). Stand ведёт программу
// срабатываний камеры (главная история по кругу) и срабатывания по заказу
// страницы тестовых сценариев, отдаёт сообщения протокола внешней системы edge-агенту
// (POST ‹edge›/v1/vision/‹система›) со сбоями каркаса (пропуск, повтор,
// порядок, порча, сдвиг часов, недоступность) и хранит историю отправленного
// «глазами системы».
//
// Граница эмуляции: нет камеры, света и нейросети — результаты ступеней
// заданы программой; протокол (сообщение системы → edge-агент) настоящий,
// по contracts/integrations/vision. Состояние — в памяти роли stands (одна
// копия, AD-6).
//
// Слой: infrastructure/integration. Владелец: эпик 33.
package standkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/infrastructure/integration/ingest/stands"
)

// Shot — срабатывание камеры: вид сцены программы, деталь, прогон.
type Shot struct {
	Kind       string `json:"kind"`
	PartID     string `json:"part_id,omitempty"`
	PartIDKind string `json:"part_id_kind,omitempty"`
	RunID      string `json:"run_id,omitempty"`
}

// Builder — сообщение протокола системы для срабатывания: номер, время по
// часам системы (со сдвигом часов из сбоя).
type Builder func(seq int64, at time.Time, s Shot) (any, error)

// Stand — stand видеофиксации по программе срабатываний.
type Stand struct {
	Name     string
	Emulates string
	Boundary string
	// Path — путь локального входа edge-агента: /v1/vision/‹система›.
	Path     string
	EdgeURL  string
	Interval time.Duration
	// Kinds — виды сцен, которые stand умеет; Program — главная история по кругу.
	Kinds   []string
	Program []Shot
	Build   Builder
	Client  *http.Client
	Now     func() time.Time

	once    sync.Once
	faults  *stands.FaultSwitch
	mu      sync.Mutex
	seq     int64
	step    int
	history []json.RawMessage
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
			s.Interval = 15 * time.Second
		}
		s.faults = stands.NewFaultSwitch(nil, s.Now)
	})
}

// Info — что эмулирует stand и где граница эмуляции (NFR-TEST-2).
func (s *Stand) Info() app.StandInfo {
	s.init()
	return app.StandInfo{Name: s.Name, Emulates: s.Emulates, Boundary: s.Boundary}
}

// Faults — сбои stand-а (служебный порт /stand/_control/).
func (s *Stand) Faults() *stands.FaultSwitch { s.init(); return s.faults }

// next — следующее срабатывание программы по кругу.
func (s *Stand) next() (Shot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Program) == 0 {
		return Shot{}, false
	}
	sh := s.Program[s.step%len(s.Program)]
	s.step++
	return sh, true
}

// Message — сообщение системы для срабатывания (номер растёт и при сбое
// «пропуск»: разрыв виден получателю).
func (s *Stand) Message(sh Shot) ([]byte, error) {
	s.init()
	if !slices.Contains(s.Kinds, sh.Kind) {
		return nil, fmt.Errorf("stand %s: вид сцены %q не поддерживается (%s)", s.Name, sh.Kind, strings.Join(s.Kinds, ", "))
	}
	s.mu.Lock()
	s.seq++
	seq := s.seq
	s.mu.Unlock()
	m, err := s.Build(seq, s.Now().Add(s.faults.ClockSkew()).UTC(), sh)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.history = append(s.history, b)
	if len(s.history) > 500 {
		s.history = s.history[len(s.history)-500:]
	}
	s.mu.Unlock()
	return b, nil
}

// corrupt — «кривое» сообщение: поле вне закрытой схемы протокола.
func corrupt(b []byte) []byte {
	return []byte(strings.Replace(string(b), "{", `{"lens_temp_c":41,`, 1))
}

// Send — сообщение срабатывания edge-агенту со сбоями исходящих.
func (s *Stand) Send(ctx context.Context, sh Shot) ([]byte, error) {
	b, err := s.Message(sh)
	if err != nil {
		return nil, err
	}
	for _, m := range s.Faults().Outgoing(b, corrupt) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.EdgeURL, "/")+s.Path, bytes.NewReader(m))
		if err != nil {
			return b, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.Client.Do(req)
		if err != nil {
			return b, fmt.Errorf("stand %s → edge-агент: %w", s.Name, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			return b, fmt.Errorf("stand %s → edge-агент: HTTP %d", s.Name, resp.StatusCode)
		}
	}
	return b, nil
}

// Tick — одно срабатывание по программе.
func (s *Stand) Tick(ctx context.Context) error {
	sh, ok := s.next()
	if !ok {
		return nil
	}
	_, err := s.Send(ctx, sh)
	return err
}

// Run — срабатывания по интервалу; ошибки связи не останавливают stand.
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

// History — отправленные сообщения «глазами системы» (последние 500).
func (s *Stand) History() []json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.history)
}

// Handler — протокол stand-а под /stand/‹имя›/:
//
//	GET  /           — что эмулирует, виды сцен, программа;
//	GET  /results    — отправленные сообщения «глазами системы»;
//	POST /shots      — заказать срабатывание {kind, part_id, part_id_kind, run_id}:
//	                   сообщение уходит edge-агенту сразу, ответ — само сообщение.
func (s *Stand) Handler() http.Handler {
	s.init()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"info": s.Info(), "kinds": s.Kinds, "program": s.Program})
	})
	mux.HandleFunc("GET /results", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.History())
	})
	mux.HandleFunc("POST /shots", func(w http.ResponseWriter, r *http.Request) {
		var sh Shot
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&sh); err != nil {
			http.Error(w, "срабатывание не разобрано: "+err.Error(), http.StatusBadRequest)
			return
		}
		if !slices.Contains(s.Kinds, sh.Kind) {
			http.Error(w, "вид сцены: "+strings.Join(s.Kinds, ", "), http.StatusBadRequest)
			return
		}
		b, err := s.Send(r.Context(), sh)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(b)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
