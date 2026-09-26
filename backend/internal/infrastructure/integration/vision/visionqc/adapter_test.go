package visionqc_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	appvision "ant/internal/application/vision"
	"ant/internal/contracts/normative"
	"ant/internal/infrastructure/integration/vision/visionqc"
	"ant/internal/infrastructure/integration/vision/visiontest"
)

func example(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(visiontest.Contracts(), "integrations", "vision", "visionqc", "examples", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// events — эталон → адаптер → Relay → события; каждое проходит схемы конверта и data.
func events(t *testing.T, r appvision.Relay, raw []byte) []map[string]any {
	t.Helper()
	sig, err := visionqc.New(nil).Translate(raw)
	if err != nil {
		t.Fatal(err)
	}
	evs, err := r.Events(context.Background(), sig)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range evs {
		// Поля устройства ставит edge-агент (AD-7); здесь — для проверки схемой.
		full := map[string]any{"source_id": "edge-kt3", "source_seq": 1, "integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{"device-edge-kt3@1"}}}
		for k, v := range e {
			full[k] = v
		}
		b, _ := json.Marshal(full)
		visiontest.Validate(t, "events/common/envelope.v1.json", b)
		visiontest.Validate(t, "events/inspection/inspection.result.recorded.v1.json", e["data"].(json.RawMessage))
	}
	return evs
}

type data struct {
	Outcome         string            `json:"outcome"`
	ProcessingState string            `json:"processing_state"`
	UnableReason    string            `json:"unable_reason"`
	QualityBP       *int              `json:"observation_quality_bp"`
	ConfidenceBP    *int              `json:"analyzer_confidence_bp"`
	StepKey         string            `json:"step_key"`
	Point           string            `json:"inspection_point"`
	Stages          []map[string]any  `json:"stages"`
	Versions        map[string]string `json:"versions"`
	Limitations     []string          `json:"limitations"`
	Defects         []map[string]any  `json:"defects"`
	Evidence        []map[string]any  `json:"evidence_refs"`
}

func dataOf(t *testing.T, e map[string]any) data {
	var d data
	if err := json.Unmarshal(e["data"].(json.RawMessage), &d); err != nil {
		t.Fatal(err)
	}
	return d
}

// FR-38, AD-29: каждое наблюдение несёт полный вектор версий и ступени;
// «испорченный кадр» передаётся как есть (качество 0,3 при «признаков нет») —
// решает интерпретация quality; блик — «оценка невозможна», плохое изображение.
func TestTranslateMainStory(t *testing.T) {
	r := appvision.Relay{}
	for _, name := range []string{"weld_ok", "weld_pores", "weld_undercut", "weld_burnthrough", "bad_frame", "glare", "aborted"} {
		evs := events(t, r, example(t, name))
		if len(evs) != 1 {
			t.Fatalf("%s: событий %d", name, len(evs))
		}
		e := evs[0]
		d := dataOf(t, e)
		if len(d.Versions) != 8 || d.Versions["recipe_ref"] != "kt3-weld@1" || d.Versions["analyzer_version"] != "vqc-weld 2.3.1" {
			t.Fatalf("%s: вектор версий %v", name, d.Versions)
		}
		for k, v := range d.Versions {
			if v == "" || v == "unknown" {
				t.Fatalf("%s: составляющая %s пуста", name, k)
			}
		}
		if d.StepKey != "welding.kt3_camera" || d.Point != "KT-3" || e["item_ref"].(map[string]any)["carrier_type"] != "internal_id" {
			t.Fatalf("%s: точка и изделие: %+v %v", name, d, e["item_ref"])
		}
		switch name {
		case "bad_frame":
			if d.Outcome != "no_defect_indicated" || *d.QualityBP != 3000 || d.ProcessingState != "completed" {
				t.Fatalf("испорченный кадр: %+v", d)
			}
		case "glare":
			if d.Outcome != "unable_to_assess" || d.UnableReason != "poor_image" || !strings.Contains(strings.Join(d.Limitations, ";"), "блик") {
				t.Fatalf("блик: %+v", d)
			}
		case "aborted":
			if d.Outcome != "unable_to_assess" || d.ProcessingState != "aborted" || d.UnableReason != "processing_aborted" {
				t.Fatalf("прервано: %+v", d)
			}
		case "weld_burnthrough":
			if d.Outcome != "defect_indicated" || d.Defects[0]["defect_type_code"] != "W-BURNTHRU" || len(d.Stages) != 2 {
				t.Fatalf("прожог: %+v", d)
			}
		}
	}
	// Повтор системы — тот же event_id (AD-7): приём даст «дубль».
	a, b := events(t, r, example(t, "weld_ok")), events(t, r, example(t, "weld_ok"))
	if a[0]["event_id"] != b[0]["event_id"] {
		t.Fatal("event_id не детерминирован")
	}
}

// Закрытая схема: лишнее поле — отказ; неизвестная камера и неполный вектор
// версий — не потеря наблюдения, а явное unknown и пометка.
func TestProtocolStrictAndUnknowns(t *testing.T) {
	raw := example(t, "weld_ok")
	bad := []byte(strings.Replace(string(raw), "{", `{"lens_temp_c":41,`, 1))
	if _, err := visionqc.New(nil).Translate(bad); err == nil {
		t.Fatal("лишнее поле должно отвергаться")
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	m["software"] = map[string]any{"analyzer_version": "vqc-weld 3.0.0", "contract_version": "2.0"}
	b2, _ := json.Marshal(m)
	if _, err := visionqc.New(nil).Translate(b2); err == nil || !strings.Contains(err.Error(), "несовместима") {
		t.Fatalf("контракт 2.0 принят: %v", err)
	}
	m["camera_id"] = "CAM-X"
	delete(m, "configuration")
	m["software"] = map[string]any{"analyzer_version": "vqc-weld 2.3.1"}
	b, _ := json.Marshal(m)
	d := dataOf(t, events(t, appvision.Relay{}, b)[0])
	if d.Versions["calibration"] != "unknown" || d.Versions["contract_version"] != "unknown" || d.StepKey != "" {
		t.Fatalf("неполный вектор: %+v", d)
	}
	lim := strings.Join(d.Limitations, ";")
	if !strings.Contains(lim, "вектор версий неполный") || !strings.Contains(lim, "CAM-X") {
		t.Fatalf("пометки: %v", d.Limitations)
	}
}

type sink struct {
	addr string
	err  error
	got  []string
}

func (s *sink) PutIllustration(_ context.Context, b []byte, mt, note string) (string, error) {
	s.got = append(s.got, note)
	return s.addr, s.err
}

func catalog() normative.IllustrationsCatalog {
	return normative.IllustrationsCatalog{Version: 1, Datasets: []normative.IllustrationsCatalogDatasetsElem{{
		ID: "tig-al5083", Title: "TIG Aluminium 5083", URL: "https://www.kaggle.com/datasets/danielbacioiu/tig-aluminium-5083",
		License: "CC BY-SA 4.0", Author: "Daniel Bacioiu", Mark: "ИЛЛЮСТРАЦИЯ", Note: "не относится к изделию",
		Classes: []normative.IllustrationsCatalogDatasetsElemClassesElem{
			{DatasetClass: "good_weld", DefectCodes: []string{}, Sample: "good_weld/sample.png", MediaType: "image/png"},
			{DatasetClass: "burn_through", DefectCodes: []string{"W-BURNTHRU"}, Sample: "burn_through/sample.png", MediaType: "image/png"},
		}}}}
}

// FR-102: иллюстрация прикладывается с пометкой «ИЛЛЮСТРАЦИЯ», источником,
// лицензией и автором — только если файл есть офлайн и хранилище его приняло;
// иначе материала нет, в ограничениях — ссылка и метаданные.
func TestIllustrations(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "burn_through"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "burn_through", "0001.png"), []byte("png"), 0o644)
	addr := "streebog256:" + strings.Repeat("ab", 32)

	s := &sink{addr: addr}
	d := dataOf(t, events(t, appvision.Relay{Catalog: catalog(), Files: visionqc.DirIllustrations{Dir: dir}, Materials: s}, example(t, "weld_burnthrough"))[0])
	if len(d.Evidence) != 1 {
		t.Fatalf("иллюстрация не приложена: %+v", d)
	}
	ev := d.Evidence[0]
	note, _ := ev["source_note"].(string)
	if ev["is_illustration"] != true || ev["kind"] != "illustration" || ev["material_address"] != addr ||
		!strings.Contains(note, "ИЛЛЮСТРАЦИЯ") || !strings.Contains(note, "CC BY-SA 4.0") || !strings.Contains(note, "Daniel Bacioiu") || !strings.Contains(note, "kaggle") {
		t.Fatalf("пометка иллюстрации: %+v", ev)
	}

	cases := map[string]appvision.Relay{
		"нет файлов":           {Catalog: catalog(), Materials: s},
		"нет файла класса":     {Catalog: catalog(), Files: visionqc.DirIllustrations{Dir: t.TempDir()}, Materials: s},
		"хранилище не приняло": {Catalog: catalog(), Files: visionqc.DirIllustrations{Dir: dir}, Materials: &sink{err: errors.New("501")}},
		"хранилища нет вообще": {Catalog: catalog(), Files: visionqc.DirIllustrations{Dir: dir}},
	}
	for name, r := range cases {
		d := dataOf(t, events(t, r, example(t, "weld_burnthrough"))[0])
		if len(d.Evidence) != 0 || !slices.ContainsFunc(d.Limitations, func(l string) bool { return strings.Contains(l, "ИЛЛЮСТРАЦИЯ не приложена") }) {
			t.Fatalf("%s: фиктивное наличие или нет пометки: %+v", name, d)
		}
	}
	// «Оценка невозможна» не иллюстрируется: картинка годного шва к блику — ложь.
	d = dataOf(t, events(t, appvision.Relay{Catalog: catalog(), Files: visionqc.DirIllustrations{Dir: dir}, Materials: s}, example(t, "glare"))[0])
	if len(d.Evidence) != 0 {
		t.Fatalf("блик с иллюстрацией: %+v", d.Evidence)
	}
}
