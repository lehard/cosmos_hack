package world

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"go.yaml.in/yaml/v3"

	"ant/internal/infrastructure/fixtures/loader"
)

// Тела ответов заготовок проверяются схемами contracts/openapi.yaml (AD-36:
// «тела валидируются схемами openapi.yaml в make check»): операция по
// operationId, схема ответа 200/201 application/json, режим «ответ сервера».

// nullableTypes: OpenAPI 3.1 `type: [T, "null"]` → `type: T` + x-nullable (Schema.Type в Huma — строка).
func nullableTypes(n *yaml.Node) {
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if k.Value == "type" && v.Kind == yaml.SequenceNode {
				t := ""
				for _, x := range v.Content {
					if x.Value != "null" {
						t = x.Value
					}
				}
				*v = yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: t}
				n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x-nullable"}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
			}
		}
	}
	for _, c := range n.Content {
		nullableTypes(c)
	}
}

func prepare(s *huma.Schema, seen map[*huma.Schema]bool) {
	if s == nil || seen[s] {
		return
	}
	seen[s] = true
	if v, ok := s.Extensions["x-nullable"].(bool); ok && v {
		s.Nullable = true
	}
	prepare(s.Items, seen)
	for _, p := range s.Properties {
		prepare(p, seen)
	}
	if ap, ok := s.AdditionalProperties.(map[string]any); ok {
		b, _ := yaml.Marshal(ap)
		var sub huma.Schema
		if yaml.Unmarshal(b, &sub) == nil {
			prepare(&sub, seen)
			s.AdditionalProperties = &sub
		}
	}
	for _, l := range [][]*huma.Schema{s.OneOf, s.AnyOf, s.AllOf} {
		for _, x := range l {
			prepare(x, seen)
		}
	}
	s.PrecomputeMessages()
}

type openAPI struct {
	Paths      map[string]map[string]yaml.Node `yaml:"paths"`
	Components struct {
		Schemas map[string]*huma.Schema `yaml:"schemas"`
	} `yaml:"components"`
}

type operation struct {
	OperationID string `yaml:"operationId"`
	Responses   map[string]struct {
		Content map[string]struct {
			Schema *huma.Schema `yaml:"schema"`
		} `yaml:"content"`
	} `yaml:"responses"`
}

func loadOpenAPI(t *testing.T) (huma.Registry, map[string]*huma.Schema) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "contracts", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		t.Fatal(err)
	}
	nullableTypes(&root)
	var doc openAPI
	if err := root.Decode(&doc); err != nil {
		t.Fatal(err)
	}
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	seen := map[*huma.Schema]bool{}
	for name, s := range doc.Components.Schemas {
		prepare(s, seen)
		reg.Map()[name] = s
	}
	ops := map[string]*huma.Schema{}
	for _, methods := range doc.Paths {
		for _, n := range methods {
			var op operation
			if n.Decode(&op) != nil || op.OperationID == "" {
				continue
			}
			for _, code := range []string{"200", "201"} {
				if r, ok := op.Responses[code]; ok {
					if c, ok := r.Content["application/json"]; ok && c.Schema != nil {
						prepare(c.Schema, seen)
						ops[op.OperationID] = c.Schema
					}
				}
			}
			if _, ok := ops[op.OperationID]; !ok {
				ops[op.OperationID] = nil
			}
		}
	}
	return reg, ops
}

// TestBodiesMatchOpenAPI — каждое тело заготовок проходит схему своей операции.
func TestBodiesMatchOpenAPI(t *testing.T) {
	reg, ops := loadOpenAPI(t)
	// Проверка самой проверки: заведомо плохое тело схема отвергает.
	for _, bad := range []string{`{"item_id":1}`, `{"item_id":"ENT01:F-001","label":"Ф-001","status":{"position":"flying"}}`} {
		var v any
		_ = json.Unmarshal([]byte(bad), &v)
		res := &huma.ValidateResult{}
		huma.Validate(reg, ops["item.passport.read"], huma.NewPathBuffer([]byte(""), 0), huma.ModeReadFromServer, v, res)
		if len(res.Errors) == 0 {
			t.Fatalf("валидатор пропустил заведомо плохое тело %s", bad)
		}
	}
	lib, err := testLibrary()
	if err != nil {
		t.Fatal(err)
	}
	bad := map[string]int{}
	checked := map[string]bool{}
	err = lib.Walk(func(where string, r loader.Response, body []byte) error {
		s, ok := ops[r.Op]
		if !ok {
			return fmt.Errorf("%s: операции %s нет в openapi.yaml", where, r.Op)
		}
		if s == nil {
			return fmt.Errorf("%s: у операции %s нет тела ответа 200/201", where, r.Op)
		}
		var v any
		if err := json.Unmarshal(body, &v); err != nil {
			return err
		}
		res := &huma.ValidateResult{}
		huma.Validate(reg, s, huma.NewPathBuffer([]byte(""), 0), huma.ModeReadFromServer, v, res)
		checked[r.Op] = true
		for _, e := range res.Errors {
			if bad[r.Op] < 3 {
				t.Errorf("%s %s %v: %s", where, r.Op, r.Params, e.Error())
			}
			bad[r.Op]++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Какие операции чтения ещё не наполнены (команды отвечают квитанцией Decide).
	var missing []string
	for op, s := range ops {
		if s != nil && !checked[op] && !strings.Contains(op, "stream") {
			missing = append(missing, op)
		}
	}
	sort.Strings(missing)
	t.Logf("проверено операций: %d; без заготовок (не чтение или не наполнено): %v", len(checked), missing)
}
