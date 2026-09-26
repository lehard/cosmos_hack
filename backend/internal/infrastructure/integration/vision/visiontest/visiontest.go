// Пакет visiontest — опора контрактных тестов адаптеров видеофиксации
// (AD-20: контрактные тесты адаптеров на эталонных сообщениях): проверка
// JSON схемами contracts/ репозитория тем же валидатором, что у приёма
// (santhosh-tekuri/jsonschema). В сборку ролей не входит.
//
// Слой: infrastructure/integration (только тесты). Владелец: эпик 33.
package visiontest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const base = "https://ant.invalid/contracts/"

// Contracts — каталог contracts/ репозитория (поиск вверх от рабочего каталога теста).
func Contracts() string {
	dir, _ := os.Getwd()
	for range 10 {
		c := filepath.Join(dir, "contracts")
		if st, err := os.Stat(filepath.Join(c, "events", "catalog.yaml")); err == nil && !st.IsDir() {
			return c
		}
		dir = filepath.Dir(dir)
	}
	return "contracts"
}

type loader struct{ root string }

func (l loader) Load(url string) (any, error) {
	p, ok := strings.CutPrefix(url, base)
	if !ok {
		return nil, fmt.Errorf("схема %s вне contracts/", url)
	}
	b, err := os.ReadFile(filepath.Join(l.root, p))
	if err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(b))
}

var (
	mu    sync.Mutex
	comp  *jsonschema.Compiler
	cache = map[string]*jsonschema.Schema{}
)

// Validate — doc соответствует схеме path (путь от contracts/).
func Validate(t testing.TB, path string, doc []byte) {
	t.Helper()
	if err := Check(path, doc); err != nil {
		t.Fatalf("%s: %v\n%s", path, err, doc)
	}
}

// Check — ошибка несоответствия схеме path.
func Check(path string, doc []byte) error {
	mu.Lock()
	if comp == nil {
		comp = jsonschema.NewCompiler()
		comp.DefaultDraft(jsonschema.Draft7)
		comp.UseLoader(loader{root: Contracts()})
	}
	sch, ok := cache[path]
	if !ok {
		var err error
		sch, err = comp.Compile(base + path)
		if err != nil {
			mu.Unlock()
			return err
		}
		cache[path] = sch
	}
	mu.Unlock()
	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(doc))
	if err != nil {
		return err
	}
	return sch.Validate(v)
}
