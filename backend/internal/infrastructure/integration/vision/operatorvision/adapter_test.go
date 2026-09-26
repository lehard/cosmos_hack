package operatorvision_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appvision "ant/internal/application/vision"
	"ant/internal/infrastructure/integration/vision/operatorvision"
	"ant/internal/infrastructure/integration/vision/operatorvision/stand"
	"ant/internal/infrastructure/integration/vision/standkit"
	"ant/internal/infrastructure/integration/vision/visiontest"
)

var t0 = time.Date(2026, 9, 21, 11, 0, 0, 0, time.UTC)

// Эталонные гипотезы совпадают с stand-ом и проходят схему action.v1;
// перевод даёт operator.action.observed по схеме контракта (FR-126).
func TestExamplesAndTranslate(t *testing.T) {
	dir := filepath.Join(visiontest.Contracts(), "integrations", "vision", "operatorvision", "examples")
	for i, sc := range []string{"step_missed", "sequence_broken", "tool_missing"} {
		a, err := stand.Action(stand.Assembly(), int64(i+1), t0.Add(time.Duration(i)*time.Minute), standkit.Shot{Kind: sc, PartID: "ENT01:F-231"})
		if err != nil {
			t.Fatal(err)
		}
		want, _ := json.MarshalIndent(a, "", "  ")
		want = append(want, '\n')
		name := filepath.Join(dir, sc+".json")
		if os.Getenv("ANT_UPDATE_EXAMPLES") != "" {
			_ = os.WriteFile(name, want, 0o644)
		}
		got, err := os.ReadFile(name)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("%s расходится с stand-ом (ANT_UPDATE_EXAMPLES=1 go test): %v", name, err)
		}
		visiontest.Validate(t, "integrations/vision/operatorvision/action.v1.json", got)
		sig, err := operatorvision.New().Translate(got)
		if err != nil {
			t.Fatal(err)
		}
		evs, err := appvision.Relay{}.Events(context.Background(), sig)
		if err != nil || len(evs) != 1 || evs[0]["event_type"] != "operator.action.observed" {
			t.Fatalf("%v %+v", err, evs)
		}
		raw := evs[0]["data"].(json.RawMessage)
		visiontest.Validate(t, "events/operator/operator.action.observed.v1.json", raw)
		var d struct {
			Observation string            `json:"observation"`
			Workplace   string            `json:"workplace_id"`
			Versions    map[string]string `json:"versions"`
		}
		_ = json.Unmarshal(raw, &d)
		if d.Observation != sc || d.Workplace != "WP-ASM-1" || len(d.Versions) != 8 || d.Versions["recipe_ref"] != "ov-asm@1" {
			t.Fatalf("гипотеза: %+v", d)
		}
		for _, bad := range []string{"person", "face", "operator_id"} {
			if strings.Contains(string(raw), bad) {
				t.Fatalf("в событии сведения о человеке: %s", raw)
			}
		}
	}
}

// FR-126: распознавания лиц нет — сообщение со сведениями о человеке
// отвергается закрытой схемой протокола, а не передаётся дальше.
func TestNoFaces(t *testing.T) {
	a, _ := stand.Action(stand.Assembly(), 1, t0, standkit.Shot{Kind: "step_missed"})
	var m map[string]any
	b, _ := json.Marshal(a)
	_ = json.Unmarshal(b, &m)
	for _, f := range []string{"person_id", "face_embedding", "operator_name"} {
		m2 := map[string]any{f: "x"}
		for k, v := range m {
			m2[k] = v
		}
		raw, _ := json.Marshal(m2)
		_, err := operatorvision.New().Translate(raw)
		if err == nil || !strings.Contains(err.Error(), "исполнитель — по входу на рабочее место") {
			t.Fatalf("%s: %v", f, err)
		}
		if visiontest.Check("integrations/vision/operatorvision/action.v1.json", raw) == nil {
			t.Fatalf("%s: схема протокола пропускает сведения о человеке", f)
		}
	}
}
