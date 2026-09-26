package documents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"

	"go.yaml.in/yaml/v3"

	engineapp "ant/internal/application/engine"
	"ant/internal/contracts/normative"
	dom "ant/internal/domain/documents"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// Пути нормативного слоя модуля documents от корня fsys (корень репозитория
// или встроенная копия storage/documents.Seed).
const (
	// PathTemplates — шаблоны документов (AD-12, FR-65).
	PathTemplates = "normative/documents/templates.v1.yaml"
	// PathPolicy — стартовая политика: роли, полномочия, клейма (срез для
	// проверки подписей до проекции политики эпика 26).
	PathPolicy = "normative/policy/policy.v1.yaml"
)

// EnvFromFS — Env модуля documents: шаблоны и срез политики. verification —
// full | demo (демо-профиль: подписи без агента токена, Д-30).
func EnvFromFS(fsys fs.FS, verification string) (dom.Env, error) {
	tb, err := fs.ReadFile(fsys, PathTemplates)
	if err != nil {
		return dom.Env{}, err
	}
	ts, err := TemplatesFromYAML(tb)
	if err != nil {
		return dom.Env{}, fmt.Errorf("%s: %w", PathTemplates, err)
	}
	env := dom.Env{Templates: ts, Verification: verification}
	if pb, err := fs.ReadFile(fsys, PathPolicy); err == nil {
		if env.People, err = PeopleFromYAML(pb); err != nil {
			return dom.Env{}, fmt.Errorf("%s: %w", PathPolicy, err)
		}
	}
	return env, nil
}

// yamlJSON — YAML как JSON (схемы контрактов описаны на JSON Schema).
func yamlJSON(b []byte) ([]byte, error) {
	var raw any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	return json.Marshal(raw)
}

// TemplatesFromYAML — шаблоны документов: проверка по схеме
// contracts/normative/templates.schema.json (сгенерированный тип) и разбор в
// доменные шаблоны.
func TemplatesFromYAML(b []byte) (dom.Templates, error) {
	j, err := yamlJSON(b)
	if err != nil {
		return nil, err
	}
	var seed normative.DocumentTemplatesSeed
	if err := json.NewDecoder(bytes.NewReader(j)).Decode(&seed); err != nil {
		return nil, err
	}
	var file struct {
		Templates dom.Templates `json:"templates"`
	}
	if err := json.Unmarshal(j, &file); err != nil {
		return nil, err
	}
	return file.Templates, nil
}

// PeopleFromYAML — срез стартовой политики для проверки подписей: роли
// сотрудников с наследованием, полномочия и цифровые клейма (AD-15, FR-145).
func PeopleFromYAML(b []byte) (dom.People, error) {
	var pf struct {
		Roles []struct {
			ID       string   `yaml:"id"`
			Inherits []string `yaml:"inherits"`
		} `yaml:"roles"`
		Persons []struct {
			ID    string `yaml:"id"`
			Roles []struct {
				Role string `yaml:"role"`
			} `yaml:"roles"`
		} `yaml:"persons"`
		Grants struct {
			Authorities []struct {
				Person    string `yaml:"person"`
				Authority string `yaml:"authority"`
			} `yaml:"authorities"`
			Stamps []struct {
				Person  string `yaml:"person"`
				StampID string `yaml:"stamp_id"`
				Kind    string `yaml:"kind"`
			} `yaml:"stamps"`
		} `yaml:"grants"`
	}
	if err := yaml.Unmarshal(b, &pf); err != nil {
		return nil, err
	}
	inherits := map[string][]string{}
	for _, r := range pf.Roles {
		inherits[r.ID] = r.Inherits
	}
	expand := func(roles []string) []string {
		var out []string
		queue := slices.Clone(roles)
		for len(queue) > 0 {
			r := queue[0]
			queue = queue[1:]
			if slices.Contains(out, r) {
				continue
			}
			out = append(out, r)
			queue = append(queue, inherits[r]...)
		}
		slices.Sort(out)
		return out
	}
	var people dom.People
	for _, p := range pf.Persons {
		var roles []string
		for _, r := range p.Roles {
			roles = append(roles, r.Role)
		}
		person := dom.Person{ID: p.ID, Roles: expand(roles), Authorities: []string{}}
		for _, g := range pf.Grants.Authorities {
			if g.Person == p.ID && !slices.Contains(person.Authorities, g.Authority) {
				person.Authorities = append(person.Authorities, g.Authority)
			}
		}
		for _, s := range pf.Grants.Stamps {
			if s.Person == p.ID {
				person.Stamps = append(person.Stamps, dom.Stamp{ID: s.StampID, Kind: s.Kind})
			}
		}
		slices.Sort(person.Authorities)
		people = append(people, person)
	}
	return people, nil
}

// Bundles — BundleSource движка, дополняющий нормативный слой изделия
// частью модуля documents (AD-17): шаблоны, срез политики и названия шагов
// процесса закреплённой версии (для строк сопроводительной карты).
// Декорирует Next.
type Bundles struct {
	// Next — источник версии; nil — пустой нормативный слой.
	Next engineapp.BundleSource
	// Env — шаблоны и политика (EnvFromFS).
	Env dom.Env
}

var _ engineapp.BundleSource = Bundles{}

// Bundle — нормативный слой изделия с частью documents.
func (b Bundles) Bundle(ctx context.Context, itemID string, input []kernel.Record) (engine.Bundle, string, error) {
	next := b.Next
	if next == nil {
		next = engineapp.EmptyBundles{}
	}
	bd, rev, err := next.Bundle(ctx, itemID, input)
	if err != nil {
		return bd, rev, err
	}
	env := bd.Documents
	if !env.Active() {
		env = b.Env
	}
	if env.Steps == nil && bd.Process.Def != nil {
		def := bd.Process.Def
		env.Steps = map[string]dom.Step{}
		for _, id := range def.Order {
			n := def.Nodes[id]
			if n == nil || n.StepKey() == "" || n.Name == "" {
				continue
			}
			if _, ok := env.Steps[n.StepKey()]; !ok {
				env.Steps[n.StepKey()] = dom.Step{Title: n.Name, Workshop: def.Workshop(n)}
			}
		}
	}
	bd.Documents = env
	return bd, rev, nil
}
