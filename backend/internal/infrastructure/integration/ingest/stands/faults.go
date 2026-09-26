package stands

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	app "ant/internal/application/ingest"
)

// FaultSwitch — включённые сбои одного stand-а; время — now (InfraClock, AD-37).
type FaultSwitch struct {
	mu        sync.Mutex
	supported []app.FaultKind
	active    map[app.FaultKind]app.Fault
	now       func() time.Time
	held      [][]byte // придержанные сообщения для нарушения порядка
	// manual — stand применяет сбои сам (свой протокол, сбой по образцу
	// сообщения, служебная страница без сбоев): Middleware их пропускает.
	manual bool
}

// Manual — сбои применяет сам stand (эпик 30, stand 1С): общий Middleware
// пропускает запросы без изменений.
func (f *FaultSwitch) Manual() *FaultSwitch {
	f.manual = true
	return f
}

// NewFaultSwitch — сбои stand-а с поддерживаемыми видами (nil — все).
func NewFaultSwitch(supported []app.FaultKind, now func() time.Time) *FaultSwitch {
	if supported == nil {
		supported = app.FaultKinds
	}
	if now == nil {
		now = time.Now
	}
	return &FaultSwitch{supported: supported, active: map[app.FaultKind]app.Fault{}, now: now}
}

// Supported — поддерживаемые виды сбоев.
func (f *FaultSwitch) Supported() []app.FaultKind { return slices.Clone(f.supported) }

// Set включает сбой.
func (f *FaultSwitch) Set(x app.Fault) error {
	if !slices.Contains(f.supported, x.Kind) {
		return fmt.Errorf("сбой %q stand-ом не поддерживается", x.Kind)
	}
	f.mu.Lock()
	f.active[x.Kind] = x
	f.mu.Unlock()
	return nil
}

// Clear снимает все сбои.
func (f *FaultSwitch) Clear() {
	f.mu.Lock()
	f.active = map[app.FaultKind]app.Fault{}
	f.mu.Unlock()
}

// Active — действующие сбои (истёкшие снимаются).
func (f *FaultSwitch) Active() []app.Fault {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.now()
	var out []app.Fault
	for _, k := range f.supported {
		x, ok := f.active[k]
		if !ok {
			continue
		}
		if !x.Until.IsZero() && !now.Before(x.Until) {
			delete(f.active, k)
			continue
		}
		out = append(out, x)
	}
	return out
}

// Get — действующий сбой вида k.
func (f *FaultSwitch) Get(k app.FaultKind) (app.Fault, bool) {
	for _, x := range f.Active() {
		if x.Kind == k {
			return x, true
		}
	}
	return app.Fault{}, false
}

// Middleware — сбои входящих запросов протокола stand-а: offline — соединение
// рвётся, error — 503, delay — ответ через Param мс.
func (f *FaultSwitch) Middleware(next http.Handler) http.Handler {
	if f.manual {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := f.Get(app.FaultOffline); ok {
			if hj, ok := w.(http.Hijacker); ok {
				if c, _, err := hj.Hijack(); err == nil {
					_ = c.Close()
					return
				}
			}
			http.Error(w, "stand недоступен", http.StatusServiceUnavailable)
			return
		}
		if x, ok := f.Get(app.FaultDelay); ok && x.Param > 0 {
			if err := wait(r.Context(), time.Duration(x.Param)*time.Millisecond); err != nil {
				return
			}
		}
		if _, ok := f.Get(app.FaultError); ok {
			http.Error(w, "сбой stand-а", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Outgoing — сбои исходящего сообщения stand-а: что отправить вместо msg.
// offline и drop — ничего (drop — номер пропущен, получатель увидит разрыв);
// duplicate — дважды; reorder — сообщение придерживается и уходит после
// следующего; corrupt — порча через corrupt(msg).
func (f *FaultSwitch) Outgoing(msg []byte, corrupt func([]byte) []byte) [][]byte {
	if _, ok := f.Get(app.FaultOffline); ok {
		return nil
	}
	if _, ok := f.Get(app.FaultDrop); ok {
		return nil
	}
	if _, ok := f.Get(app.FaultCorrupt); ok && corrupt != nil {
		msg = corrupt(msg)
	}
	out := [][]byte{msg}
	if _, ok := f.Get(app.FaultDuplicate); ok {
		out = append(out, msg)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.active[app.FaultReorder]; ok {
		if len(f.held) == 0 {
			f.held = append(f.held, msg)
			return nil
		}
		out = append(out, f.held...)
		f.held = nil
	}
	return out
}

// ClockSkew — сдвиг часов stand-а (FaultClockSkew, Param мс).
func (f *FaultSwitch) ClockSkew() time.Duration {
	if x, ok := f.Get(app.FaultClockSkew); ok {
		return time.Duration(x.Param) * time.Millisecond
	}
	return 0
}

func wait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
