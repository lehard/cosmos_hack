package world

import (
	"bytes"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"ant/internal/infrastructure/fixtures/loader"
)

// go test ./internal/infrastructure/fixtures/world -update — пересобрать
// scenarios/fixtures и встроенную копию входов из world.yaml и normative/.
var update = flag.Bool("update", false, "перезаписать сгенерированные заготовки и копию входов")

const repoRoot = "../../../../.."

func repoFS() fs.FS { return os.DirFS(repoRoot) }

// testLibrary — библиотека заготовок из репозитория, одна на тесты пакета.
var testLibrary = sync.OnceValues(func() (*loader.Library, error) { return LibraryFrom(repoFS()) })

// inputNames — входы генератора от корня репозитория.
func inputNames(t *testing.T) []string {
	t.Helper()
	names := append([]string{}, InputFiles...)
	for _, g := range InputGlobs {
		m, err := fs.Glob(repoFS(), g)
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, m...)
	}
	sort.Strings(names)
	return names
}

// TestGenerated — заготовки scenarios/fixtures и встроенная копия входов совпадают с генератором (AD-36).
func TestGenerated(t *testing.T) {
	files, err := Generate(repoFS())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]byte{}
	for name, b := range files {
		want[filepath.Join(repoRoot, FixturesDir, name)] = b
	}
	for _, n := range inputNames(t) {
		b, err := fs.ReadFile(repoFS(), n)
		if err != nil {
			t.Fatal(err)
		}
		want[filepath.Join("input", n)] = b
	}
	// Лишние файлы: сгенерированные прежде шаги и копии, которых генератор больше не даёт.
	var stale []string
	for _, root := range []string{filepath.Join(repoRoot, FixturesDir), "input"} {
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || strings.HasSuffix(p, "world.yaml") && strings.HasPrefix(p, repoRoot) || strings.HasSuffix(p, "README.md") || strings.HasSuffix(p, ".keep") {
				return nil
			}
			if _, ok := want[p]; !ok {
				stale = append(stale, p)
			}
			return nil
		})
	}
	if *update {
		for _, p := range stale {
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
		}
		for p, b := range want {
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, b, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	for _, p := range stale {
		t.Errorf("лишний файл %s — go test ./internal/infrastructure/fixtures/world -update", p)
	}
	for p, b := range want {
		got, err := os.ReadFile(p)
		if err != nil || !bytes.Equal(got, b) {
			t.Errorf("%s устарел — go test ./internal/infrastructure/fixtures/world -update", p)
		}
	}
}
