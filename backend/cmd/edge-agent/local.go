package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"ant/internal/contracts/procs"
)

// Extractor — выделитель событий из телеметрии stand-а (AD-18, FR-149):
// сырые отсчёты остаются на краю, в ant уходят события. Базовый выделитель
// переводит смену режима работы, режима управления и исправности в
// equipment.state.changed v2 (классификация MTConnect); сводки циклов и
// значимые отклонения — эпик 23 (MachineLogs, FR-147) поверх того же места.
type Extractor struct {
	mu    sync.Mutex
	state map[string]map[string]string // equipment_id → категория → значение
	// sum — сводки на окно цикла и отклонения (эпик 23, summary.go).
	sum *Summarizer
}

// NewExtractor создаёт выделитель.
func NewExtractor() *Extractor {
	return &Extractor{state: map[string]map[string]string{}, sum: NewSummarizer()}
}

// mtconnect — значения MTConnect → значения контракта; неизвестное проходит
// как есть (ядро решит: неизвестное в поле безопасности — карантин).
var mtconnect = map[string]map[string]string{
	"execution": {"active": "running", "ready": "idle", "stopped": "stopped", "program_stopped": "stopped",
		"optional_stop": "stopped", "interrupted": "interrupted", "feed_hold": "interrupted", "setup": "setup", "unavailable": "unknown"},
	"controller_mode": {"automatic": "automatic", "manual": "manual", "manual_data_input": "manual_data_input", "unavailable": "unknown"},
	"condition":       {"normal": "normal", "warning": "warning", "fault": "fault", "unavailable": "unknown"},
}

func norm(cat, v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if m, ok := mtconnect[cat][v]; ok {
		return m
	}
	return v
}

// Extract — события из сообщения stand-а, по одному на смену состояния.
func (x *Extractor) Extract(t procs.StandTelemetryV1) []map[string]any {
	x.mu.Lock()
	defer x.mu.Unlock()
	st := x.state[t.EquipmentID]
	if st == nil {
		st = map[string]string{"execution": "unknown", "controller_mode": "unknown", "condition": "unknown"}
		x.state[t.EquipmentID] = st
	}
	var out []map[string]any
	for _, e := range t.Events {
		cat := string(e.Category)
		if _, ok := mtconnect[cat]; !ok {
			continue
		}
		v := norm(cat, e.Value)
		if st[cat] == v {
			continue
		}
		st[cat] = v
		data := map[string]any{"equipment_id": t.EquipmentID, "execution": st["execution"],
			"controller_mode": st["controller_mode"], "condition": st["condition"]}
		if e.Code != nil && *e.Code != "" {
			data["code"] = *e.Code
		}
		ev := map[string]any{"event_type": "equipment.state.changed", "schema_version": 2, "occurred_at": e.At,
			"source_kind": "machine", "reliability": "high", "data": data}
		if t.RunID != nil && *t.RunID != "" {
			ev["run_id"] = *t.RunID
		}
		out = append(out, ev)
	}
	if x.sum != nil {
		out = append(out, x.sum.Process(t)...)
	}
	return out
}

// LocalHandler — локальный вход агента на краю:
//
//	POST /v1/events    — событие источника или массив событий (JSON) → буфер;
//	POST /v1/telemetry — сообщение stand-а оборудования (contracts/internal/stands/telemetry.v1.json);
//	POST /v1/vision/‹система› — сообщение системы видеофиксации (эпик 33, vision.go);
//	GET  /v1/status    — состояние агента: номер, буфер, последняя ошибка.
func LocalHandler(a *Agent, x *Extractor, vision ...VisionRoute) http.Handler {
	mux := http.NewServeMux()
	registerVision(mux, a, vision)
	mux.HandleFunc("POST /v1/events", func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var evs []map[string]any
		if len(b) > 0 && strings.TrimSpace(string(b))[0] == '[' {
			err = json.Unmarshal(b, &evs)
		} else {
			var one map[string]any
			err = json.Unmarshal(b, &one)
			evs = []map[string]any{one}
		}
		if err != nil {
			http.Error(w, "не JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		enqueue(w, a, evs)
	})
	mux.HandleFunc("POST /v1/telemetry", func(w http.ResponseWriter, r *http.Request) {
		var t procs.StandTelemetryV1
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<20))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&t); err != nil {
			http.Error(w, "телеметрия не по контракту: "+err.Error(), http.StatusBadRequest)
			return
		}
		enqueue(w, a, x.Extract(t))
	})
	mux.HandleFunc("GET /v1/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(a.Status())
	})
	return mux
}

func enqueue(w http.ResponseWriter, a *Agent, evs []map[string]any) {
	out := make([]Enqueued, 0, len(evs))
	for _, ev := range evs {
		e, err := a.Enqueue(ev)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, e)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(out)
}
