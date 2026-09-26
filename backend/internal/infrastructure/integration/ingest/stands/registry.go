package stands

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"

	app "ant/internal/application/ingest"
)

// Stand — stand внешней системы или оборудования (AD-18).
type Stand interface {
	// Info — имя, что эмулирует и где граница эмуляции (NFR-TEST-2).
	Info() app.StandInfo
	// Faults — сбои stand-а.
	Faults() *FaultSwitch
	// Handler — протокол реальной системы (nil — stand только отправляет).
	Handler() http.Handler
	// Run — фоновая работа stand-а (отправка по сценарию); возвращается при остановке.
	Run(ctx context.Context) error
}

// Registry — stand-ы роли stands (одна копия, AD-6) и служебный порт сбоев.
type Registry struct {
	mu     sync.Mutex
	stands map[string]Stand
}

// NewRegistry — реестр stand-ов.
func NewRegistry(ss ...Stand) *Registry {
	r := &Registry{stands: map[string]Stand{}}
	for _, s := range ss {
		r.Add(s)
	}
	return r
}

// Add регистрирует stand.
func (r *Registry) Add(s Stand) {
	r.mu.Lock()
	r.stands[s.Info().Name] = s
	r.mu.Unlock()
}

func (r *Registry) get(name string) (Stand, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.stands[name]
	return s, ok
}

func (r *Registry) names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.stands))
	for n := range r.stands {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}

// ErrUnknownStand — stand не зарегистрирован.
var ErrUnknownStand = errors.New("stand не зарегистрирован")

// Stands — stand-ы и их сбои (StandControl).
func (r *Registry) Stands(context.Context) ([]app.StandInfo, error) {
	var out []app.StandInfo
	for _, n := range r.names() {
		s, _ := r.get(n)
		i := s.Info()
		i.Supported = s.Faults().Supported()
		i.Active = s.Faults().Active()
		out = append(out, i)
	}
	return out, nil
}

// SetFault включает сбой (StandControl).
func (r *Registry) SetFault(_ context.Context, name string, f app.Fault) error {
	s, ok := r.get(name)
	if !ok {
		return ErrUnknownStand
	}
	return s.Faults().Set(f)
}

// ClearFaults снимает сбои (StandControl).
func (r *Registry) ClearFaults(_ context.Context, name string) error {
	if name == "" {
		for _, n := range r.names() {
			s, _ := r.get(n)
			s.Faults().Clear()
		}
		return nil
	}
	s, ok := r.get(name)
	if !ok {
		return ErrUnknownStand
	}
	s.Faults().Clear()
	return nil
}

// Run запускает фоновую работу всех stand-ов; первая ошибка останавливает остальные.
func (r *Registry) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errc := make(chan error, len(r.names()))
	var wg sync.WaitGroup
	for _, n := range r.names() {
		s, _ := r.get(n)
		wg.Go(func() {
			if err := s.Run(ctx); err != nil {
				errc <- err
				cancel()
			}
		})
	}
	wg.Wait()
	close(errc)
	return <-errc
}

// Handler — HTTP роли stands:
//
//	/stand/‹имя›/…                           — протокол stand-а (со сбоями входящих);
//	GET    /stand/_control/                  — stand-ы, что эмулируют, граница, сбои;
//	POST   /stand/_control/‹имя›/faults      — включить сбой {kind, param, until};
//	DELETE /stand/_control/‹имя›/faults      — снять сбои (‹имя› = _all — у всех).
//
// Служебный порт вызывает страница тестовых сценариев (эпики 14, 32, 36) через
// операцию API своего модуля; прав здесь нет — роль stands слушает только
// внутреннюю сеть контура (AD-18).
func (r *Registry) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /stand/_control/", func(w http.ResponseWriter, req *http.Request) {
		ss, _ := r.Stands(req.Context())
		writeJSON(w, http.StatusOK, ss)
	})
	mux.HandleFunc("POST /stand/_control/{name}/faults", func(w http.ResponseWriter, req *http.Request) {
		var f app.Fault
		if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 1<<16)).Decode(&f); err != nil {
			http.Error(w, "сбой не разобран: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := r.SetFault(req.Context(), req.PathValue("name"), f); err != nil {
			code := http.StatusBadRequest
			if errors.Is(err, ErrUnknownStand) {
				code = http.StatusNotFound
			}
			http.Error(w, err.Error(), code)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /stand/_control/{name}/faults", func(w http.ResponseWriter, req *http.Request) {
		name := req.PathValue("name")
		if name == "_all" {
			name = ""
		}
		if err := r.ClearFaults(req.Context(), name); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/stand/", func(w http.ResponseWriter, req *http.Request) {
		rest := strings.TrimPrefix(req.URL.Path, "/stand/")
		name, _, _ := strings.Cut(rest, "/")
		s, ok := r.get(name)
		if !ok || s.Handler() == nil {
			http.NotFound(w, req)
			return
		}
		http.StripPrefix("/stand/"+name, s.Faults().Middleware(s.Handler())).ServeHTTP(w, req)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

var _ app.StandControl = (*Registry)(nil)
