package reference_test

import (
	"bytes"
	"io/fs"
	"os"
	"testing"

	app "ant/internal/application/reference"
	storage "ant/internal/infrastructure/storage/reference"
)

// Встроенная копия справочников совпадает с репозиторием и разбирается.
func TestSeedMatchesRepo(t *testing.T) {
	repo := os.DirFS("../../../../..")
	for _, p := range app.SeedFiles {
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
	recs, err := storage.SeedRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) < 60 {
		t.Fatalf("записей затравки: %d", len(recs))
	}
}
