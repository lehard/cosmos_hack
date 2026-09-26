package stand

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/infrastructure/integration/vision/standkit"
	"ant/internal/infrastructure/integration/vision/visionqc"
	"ant/internal/infrastructure/integration/vision/visiontest"
)

var t0 = time.Date(2026, 9, 21, 9, 30, 0, 0, time.UTC)

// examples — эталонные сообщения протокола: по одному на сцену главной истории.
func examples(t *testing.T) string {
	return filepath.Join(visiontest.Contracts(), "integrations", "vision", "visionqc", "examples")
}

func golden(t *testing.T, sc string, i int) []byte {
	r, err := Result(KT3(), int64(i+1), t0.Add(time.Duration(i)*time.Minute), standkit.Shot{Kind: sc, PartID: "ENT01:F-23" + string(rune('1'+i)), PartIDKind: "internal"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	return append(b, '\n')
}

// Эталонные сообщения: совпадают с тем, что шлёт stand, и проходят схему
// result.v1 (ANT_UPDATE_EXAMPLES=1 — перезаписать).
func TestExamplesMatchStand(t *testing.T) {
	dir := examples(t)
	for i, sc := range Scenes {
		name := filepath.Join(dir, sc+".json")
		want := golden(t, sc, i)
		if os.Getenv("ANT_UPDATE_EXAMPLES") != "" {
			if err := os.WriteFile(name, want, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		got, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s расходится с stand-ом (ANT_UPDATE_EXAMPLES=1 go test)", name)
		}
		visiontest.Validate(t, "integrations/vision/visionqc/result.v1.json", got)
	}
}

// Stand шлёт edge-агенту главную историю по кругу; сбои каркаса: пропуск
// виден разрывом номеров, «кривое» сообщение отвергает закрытая схема.
func TestStandSendsMainStory(t *testing.T) {
	var mu sync.Mutex
	var got []visionqc.Result
	edge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/vision/visionqc" {
			http.NotFound(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		res, err := visionqc.Decode(b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := visiontest.Check("integrations/vision/visionqc/result.v1.json", b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		got = append(got, res)
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer edge.Close()
	s := New(Options{Name: "visionqc-kt3", EdgeURL: edge.URL})
	s.Client = edge.Client()
	s.Now = func() time.Time { return t0 }
	ctx := context.Background()
	for range MainStory {
		if err := s.Tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	_ = s.Faults().Set(app.Fault{Kind: app.FaultDrop})
	_ = s.Tick(ctx)
	s.Faults().Clear()
	_ = s.Faults().Set(app.Fault{Kind: app.FaultCorrupt})
	if err := s.Tick(ctx); err == nil {
		t.Fatal("кривое сообщение должно быть отвергнуто")
	}
	s.Faults().Clear()
	if err := s.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(MainStory)+1 || got[len(got)-1].Seq != int64(len(MainStory)+3) {
		t.Fatalf("получено %d, последний номер %d", len(got), got[len(got)-1].Seq)
	}
	if got[2].Frame.QualityBP != 3000 || got[2].Verdict != "no_defects" || got[4].Frame.Issues[0] != "glare" {
		t.Fatalf("испорченный кадр и блик: %+v %+v", got[2], got[4])
	}
	// Сцена по заказу страницы тестовых сценариев.
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	resp, err := srv.Client().Post(srv.URL+"/shots", "application/json", strings.NewReader(`{"kind":"bad_frame","part_id":"ENT01:F-240"}`))
	if err != nil || resp.StatusCode != http.StatusAccepted {
		t.Fatalf("заказ сцены: %v %v", err, resp.Status)
	}
	if last := got[len(got)-1]; last.PartID != "ENT01:F-240" || last.Frame.QualityBP != 3000 {
		t.Fatalf("заказанная сцена: %+v", last)
	}
}
