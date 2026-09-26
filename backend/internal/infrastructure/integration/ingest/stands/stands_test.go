package stands

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/contracts/procs"
)

// Сбои исходящих сообщений: пропуск, повтор, порядок, порча, сдвиг часов, истечение.
func TestFaultSwitchOutgoing(t *testing.T) {
	now := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	f := NewFaultSwitch(nil, func() time.Time { return now })
	m := []byte(`{"a":1}`)
	if out := f.Outgoing(m, nil); len(out) != 1 {
		t.Fatal(out)
	}
	_ = f.Set(app.Fault{Kind: app.FaultDuplicate})
	if out := f.Outgoing(m, nil); len(out) != 2 {
		t.Fatal("повтор")
	}
	f.Clear()
	_ = f.Set(app.Fault{Kind: app.FaultReorder})
	if out := f.Outgoing([]byte("1"), nil); len(out) != 0 {
		t.Fatal("придержано")
	}
	if out := f.Outgoing([]byte("2"), nil); len(out) != 2 || string(out[0]) != "2" || string(out[1]) != "1" {
		t.Fatalf("порядок: %q", out)
	}
	f.Clear()
	_ = f.Set(app.Fault{Kind: app.FaultDrop, Until: now.Add(time.Minute)})
	if out := f.Outgoing(m, nil); len(out) != 0 {
		t.Fatal("пропуск")
	}
	now = now.Add(2 * time.Minute)
	if len(f.Active()) != 0 {
		t.Fatal("сбой не истёк")
	}
	_ = f.Set(app.Fault{Kind: app.FaultCorrupt})
	if out := f.Outgoing(m, corrupt); !strings.Contains(string(out[0]), "firmware_note") {
		t.Fatal("порча")
	}
	_ = f.Set(app.Fault{Kind: app.FaultClockSkew, Param: 300000})
	if f.ClockSkew() != 5*time.Minute {
		t.Fatal("часы")
	}
	if err := NewFaultSwitch([]app.FaultKind{app.FaultOffline}, nil).Set(app.Fault{Kind: app.FaultDrop}); err == nil {
		t.Fatal("неподдерживаемый сбой")
	}
}

type echoStand struct {
	f *FaultSwitch
}

func (e *echoStand) Info() app.StandInfo {
	return app.StandInfo{Name: "echo", Emulates: "проверка каркаса", Boundary: "эхо"}
}
func (e *echoStand) Faults() *FaultSwitch { return e.f }
func (e *echoStand) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "ok "+r.URL.Path) })
}
func (e *echoStand) Run(context.Context) error { return nil }

// Служебный порт сбоев через HTTP роли stands: включить, увидеть, снять.
func TestRegistryControl(t *testing.T) {
	reg := NewRegistry(&echoStand{f: NewFaultSwitch(nil, nil)})
	srv := httptest.NewServer(reg.Handler())
	defer srv.Close()
	get := func(p string) (int, string) {
		resp, err := srv.Client().Get(srv.URL + p)
		if err != nil {
			return 0, err.Error()
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	if c, b := get("/stand/echo/ping"); c != 200 || b != "ok /ping" {
		t.Fatalf("%d %s", c, b)
	}
	resp, _ := srv.Client().Post(srv.URL+"/stand/_control/echo/faults", "application/json", strings.NewReader(`{"kind":"error"}`))
	if resp.StatusCode != http.StatusNoContent {
		t.Fatal(resp.Status)
	}
	if c, _ := get("/stand/echo/ping"); c != http.StatusServiceUnavailable {
		t.Fatalf("сбой error: %d", c)
	}
	_, b := get("/stand/_control/")
	var ss []app.StandInfo
	_ = json.Unmarshal([]byte(b), &ss)
	if len(ss) != 1 || len(ss[0].Active) != 1 || ss[0].Active[0].Kind != app.FaultError {
		t.Fatalf("%s", b)
	}
	_ = reg.SetFault(context.Background(), "echo", app.Fault{Kind: app.FaultOffline})
	if c, _ := get("/stand/echo/ping"); c != 0 {
		t.Fatalf("offline: %d", c)
	}
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/stand/_control/_all/faults", nil)
	if r, _ := srv.Client().Do(req); r.StatusCode != http.StatusNoContent {
		t.Fatal(r.Status)
	}
	if c, _ := get("/stand/echo/ping"); c != 200 {
		t.Fatal("сбои не сняты")
	}
	if err := reg.SetFault(context.Background(), "nope", app.Fault{Kind: app.FaultError}); err != ErrUnknownStand {
		t.Fatal(err)
	}
}

// Stand оборудования отдаёт телеметрию по контракту; пропуск виден разрывом номеров.
func TestEquipmentStand(t *testing.T) {
	var mu sync.Mutex
	var got []procs.StandTelemetryV1
	edge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m procs.StandTelemetryV1
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&m); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		got = append(got, m)
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer edge.Close()
	s := &EquipmentStand{Name: "weld-is-1", EquipmentID: "IS-1", EdgeURL: edge.URL, Client: edge.Client()}
	ctx := context.Background()
	if err := s.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	_ = s.Faults().Set(app.Fault{Kind: app.FaultDrop})
	_ = s.Tick(ctx)
	s.Faults().Clear()
	_ = s.Faults().Set(app.Fault{Kind: app.FaultCorrupt})
	if err := s.Tick(ctx); err == nil {
		t.Fatal("кривое сообщение должно быть отвергнуто получателем")
	}
	s.Faults().Clear()
	if err := s.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Seq != 1 || got[1].Seq != 4 {
		t.Fatalf("%+v", got)
	}
}
