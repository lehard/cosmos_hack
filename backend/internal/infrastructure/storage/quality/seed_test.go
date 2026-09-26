package quality_test

import (
	"bytes"
	"io/fs"
	"os"
	"testing"

	appquality "ant/internal/application/quality"
	storage "ant/internal/infrastructure/storage/quality"
)

// Встроенная копия нормативного слоя совпадает с normative/ репозитория.
func TestSeedMatchesRepo(t *testing.T) {
	repo := os.DirFS("../../../../..")
	for _, p := range []string{appquality.PathClassifier, appquality.PathReactionMap, appquality.PathProcess, appquality.PathItemTypes} {
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
	env, err := storage.SeedEnv("seed")
	if err != nil || len(env.Steps) == 0 || len(env.ReactionMap.Rules) == 0 {
		t.Fatalf("Env по встроенной копии: %v", err)
	}
}
