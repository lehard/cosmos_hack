package casbin

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/security/identity"
)

// Решения людей в тестовых сценариях (scenarios/definitions/streams, AD-26)
// идут через тот же декоратор прав: каждое разрешено стартовой политикой, а
// ожидаемый отказ по правам (refusal: access.*) — отклоняется. Так демо не
// ломается политикой, а политика не подгоняется под демо молча.
func TestScenarioDecisionsMatchPolicy(t *testing.T) {
	ac := New(access.StaticPolicy(seed(t)))
	places, err := identity.LoadPlaces(os.DirFS(repo))
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(repo, "scenarios/definitions/streams/*.jsonl"))
	if err != nil || len(files) == 0 {
		t.Fatal("нет потоков сценариев", err)
	}
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	seen := map[string]bool{}
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			t.Fatal(err)
		}
		sc := bufio.NewScanner(fh)
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		for sc.Scan() {
			var d struct {
				Kind      string            `json:"kind"`
				Label     string            `json:"label"`
				Operation string            `json:"operation"`
				Actor     string            `json:"actor"`
				Refusal   string            `json:"refusal"`
				Params    map[string]string `json:"params"`
			}
			if err := json.Unmarshal(sc.Bytes(), &d); err != nil || d.Kind != "decision" || d.Operation == "" {
				continue
			}
			scope := ""
			if wp := d.Params["workplace_id"]; wp != "" {
				scope = places[wp]
			}
			key := d.Actor + " " + d.Operation + " " + scope + " " + d.Refusal
			if seen[key] {
				continue
			}
			seen[key] = true
			ok := allowed(t, ac, d.Actor, scope, platform.Action{ID: d.Operation, Class: platform.ClassRecord}, at)
			wantDenied := strings.HasPrefix(d.Refusal, "access.")
			if ok == wantDenied {
				t.Errorf("%s (%s): %s %s — разрешено %v, ожидался отказ по правам %v", filepath.Base(f), d.Label, d.Actor, d.Operation, ok, wantDenied)
			}
		}
		_ = fh.Close()
	}
	if len(seen) < 20 {
		t.Fatalf("мало решений в сценариях: %d", len(seen))
	}
}
