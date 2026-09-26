package vision_test

import (
	"bytes"
	"io/fs"
	"os"
	"testing"

	appquality "ant/internal/application/quality"
	appvision "ant/internal/application/vision"
	storage "ant/internal/infrastructure/storage/vision"
)

// Встроенная копия затравки совпадает с normative/ репозитория и читается.
func TestSeedMatchesRepo(t *testing.T) {
	repo := os.DirFS("../../../../..")
	for _, p := range []string{appvision.PassportsSeedFile, appvision.IllustrationsFile} {
		want, err := fs.ReadFile(repo, p)
		if err != nil {
			t.Fatal(err)
		}
		got, err := fs.ReadFile(storage.Seed(), p)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: встроенная копия расходится с репозиторием — скопируйте файл в seed/", p)
		}
	}
	ps, err := storage.PassportsSeed()
	if err != nil || len(ps.Passports) < 6 {
		t.Fatalf("паспорта: %d, %v", len(ps.Passports), err)
	}
	for _, p := range ps.Passports {
		if p.AnalyzerKind == "visionqc" && p.TrustLevel < 3 {
			t.Errorf("%s: камера демо ниже уровня 3", p.PassportID)
		}
	}
	il, err := storage.Illustrations()
	if err != nil || len(il.Datasets) == 0 || il.Datasets[0].Author == "" || il.Datasets[0].License == "" {
		t.Fatalf("иллюстрации: %+v, %v", il, err)
	}
}

// У каждой карты контроля камеры в процессе есть паспорт затравки уровня ≥ 3:
// иначе камера демо работает на уровне 0 (только запись, AD-29).
func TestEveryCameraRecipeHasPassport(t *testing.T) {
	b, err := os.ReadFile("../../../../../normative/process/flange-process.bpmn")
	if err != nil {
		t.Fatal(err)
	}
	steps, err := appquality.StepsFromBPMN(b)
	if err != nil {
		t.Fatal(err)
	}
	ps, _ := storage.PassportsSeed()
	have := map[string]int{}
	for _, p := range ps.Passports {
		have[p.RecipeRef] = p.TrustLevel
	}
	n := 0
	for _, s := range steps {
		for _, in := range s.Inspections {
			if in.Method != "camera" || in.RecipeRef == "" {
				continue
			}
			n++
			if lvl, ok := have[in.RecipeRef]; !ok || lvl < 3 {
				t.Errorf("карта контроля %s (%s): нет паспорта уровня ≥ 3", in.RecipeRef, s.StepKey)
			}
		}
	}
	if n < 6 {
		t.Fatalf("камер в процессе: %d", n)
	}
}
