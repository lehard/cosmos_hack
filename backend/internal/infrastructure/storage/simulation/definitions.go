package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	sim "ant/internal/domain/simulation"
)

// Files — определения сценариев из каталога scenarios/ (AD-26): мир,
// прогоны и карточки — scenarios/definitions (YAML), ожидания —
// scenarios/expected (отдельно от входа, кейс §5.1). В контейнере каталог
// подключён томом только для чтения.
//
// Разбор строгий: неизвестное поле определения — ошибка (опечатка в карточке
// не должна тихо превратиться в «шага нет»).
type Files struct {
	// Root — каталог scenarios (в нём definitions/ и expected/).
	Root string
}

// NewFiles — определения из каталога root.
func NewFiles(root string) *Files { return &Files{Root: root} }

// ErrNotFound — определения нет.
var ErrNotFound = errors.New("определение не найдено")

// Catalog — scenarios/definitions/index.yaml.
func (f *Files) Catalog(_ context.Context) (sim.Catalog, error) {
	var c sim.Catalog
	err := f.load(filepath.Join("definitions", "index.yaml"), &c)
	return c, err
}

// Bundle — мир, прогон и его карточки для запуска генератора.
func (f *Files) Bundle(_ context.Context, run string) (sim.Bundle, error) {
	var b sim.Bundle
	if err := f.load(filepath.Join("definitions", "runs", run+".yaml"), &b.Run); err != nil {
		return b, err
	}
	if b.Run.ID != run {
		return b, fmt.Errorf("runs/%s.yaml: id = %q", run, b.Run.ID)
	}
	if err := f.load(filepath.Join("definitions", "worlds", b.Run.World+".yaml"), &b.World); err != nil {
		return b, err
	}
	for _, id := range b.Run.Scenarios {
		var sc sim.ScenarioDef
		if err := f.load(filepath.Join("definitions", "scenarios", id+".yaml"), &sc); err != nil {
			return b, err
		}
		if sc.ID != id {
			return b, fmt.Errorf("scenarios/%s.yaml: id = %q", id, sc.ID)
		}
		b.Scenarios = append(b.Scenarios, sc)
	}
	return b, nil
}

// Expected — scenarios/expected/‹сценарий›.yaml; нет файла — ok = false.
func (f *Files) Expected(_ context.Context, scenario string) (sim.Expected, bool, error) {
	var e sim.Expected
	err := f.load(filepath.Join("expected", scenario+".yaml"), &e)
	if errors.Is(err, ErrNotFound) {
		return e, false, nil
	}
	return e, err == nil, err
}

// Scenario — карточка сценария.
func (f *Files) Scenario(_ context.Context, id string) (sim.ScenarioDef, error) {
	var sc sim.ScenarioDef
	err := f.load(filepath.Join("definitions", "scenarios", id+".yaml"), &sc)
	return sc, err
}

// load читает YAML в структуру через JSON: ключи и значения — как в YAML,
// моменты времени остаются строками, неизвестные поля — ошибка.
func (f *Files) load(rel string, into any) error {
	path := filepath.Join(f.Root, rel)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, rel)
		}
		return err
	}
	v, err := DecodeYAML(b)
	if err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	j, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	dec := json.NewDecoder(bytes.NewReader(j))
	dec.DisallowUnknownFields()
	dec.UseNumber()
	if err := dec.Decode(into); err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	return nil
}

// DecodeYAML — YAML в значения JSON: объекты, списки, строки, целые (json.Number),
// логические, null. Моменты времени и даты остаются строками как в тексте
// (иначе YAML превратит их в time.Time и потеряет зону).
func DecodeYAML(b []byte) (any, error) {
	var n yaml.Node
	if err := yaml.Unmarshal(b, &n); err != nil {
		return nil, err
	}
	if n.Kind == 0 {
		return nil, nil
	}
	return nodeValue(&n)
}

func nodeValue(n *yaml.Node) (any, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, nil
		}
		return nodeValue(n.Content[0])
	case yaml.AliasNode:
		return nodeValue(n.Alias)
	case yaml.MappingNode:
		m := make(map[string]any, len(n.Content)/2)
		var merged []map[string]any
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i]
			v, err := nodeValue(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			if k.Tag == "!!merge" {
				if mm, ok := v.(map[string]any); ok {
					merged = append(merged, mm)
				}
				continue
			}
			if _, dup := m[k.Value]; dup {
				return nil, fmt.Errorf("строка %d: ключ %q повторяется", k.Line, k.Value)
			}
			m[k.Value] = v
		}
		// «<<: *якорь» — явные ключи сильнее вставленных
		for _, mm := range merged {
			for _, kk := range slices.Sorted(maps.Keys(mm)) {
				if _, exists := m[kk]; !exists {
					m[kk] = mm[kk]
				}
			}
		}
		return m, nil
	case yaml.SequenceNode:
		out := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := nodeValue(c)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.ScalarNode:
		switch n.ShortTag() {
		case "!!null":
			return nil, nil
		case "!!bool":
			var b bool
			if err := n.Decode(&b); err != nil {
				return nil, err
			}
			return b, nil
		case "!!int":
			var i int64
			if err := n.Decode(&i); err != nil {
				return nil, err
			}
			return json.Number(fmt.Sprint(i)), nil
		case "!!float":
			return json.Number(strings.TrimPrefix(n.Value, "+")), nil
		default:
			return n.Value, nil
		}
	}
	return nil, fmt.Errorf("строка %d: неизвестный вид узла YAML", n.Line)
}

// Runs — определения прогонов в каталоге runs/.
func (f *Files) Runs() ([]string, error) {
	ents, err := os.ReadDir(filepath.Join(f.Root, "definitions", "runs"))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if name, ok := strings.CutSuffix(e.Name(), ".yaml"); ok && !e.IsDir() {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out, nil
}
