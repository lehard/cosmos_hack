package item_test

import (
	"bytes"
	"io/fs"
	"os"
	"strings"
	"testing"

	appitem "ant/internal/application/item"
	storage "ant/internal/infrastructure/storage/item"
)

// Встроенная копия нормативного слоя совпадает с репозиторием; зоны и связи
// фланца разобраны.
func TestSeedMatchesRepo(t *testing.T) {
	repo := os.DirFS("../../../../..")
	for _, p := range []string{appitem.PathItemTypes, storage.PathProcess} {
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
	env, err := storage.SeedEnv()
	if err != nil {
		t.Fatal(err)
	}
	fl := env.Types["FL-100.00.000"]
	if len(fl.Zones) != 14 || len(fl.Links) != 3 || fl.Links[1].ID != "J-1" || len(fl.Links[1].ClosesAccessTo) != 2 {
		t.Fatalf("номенклатура фланца: %+v", fl)
	}
	if v, err := storage.SeedProcessVersion(); err != nil || !strings.HasPrefix(v, "streebog256:") || len(v) != len("streebog256:")+64 {
		t.Fatalf("версия процесса: %q %v", v, err)
	}
}
