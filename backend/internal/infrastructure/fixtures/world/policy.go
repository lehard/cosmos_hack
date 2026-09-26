package world

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Policy — то, что мир заготовок берёт из стартовой политики normative/policy
// (демо-персоны, роли) и столов normative/desks: сеансы и столы ролей отдаются
// заготовками access тем же форматом, что live (AD-21, AD-36).
type Policy struct {
	Roles   []PolicyRole   `yaml:"roles"`
	Persons []PolicyPerson `yaml:"persons"`
	// Desks — столы ролей: id роли → содержимое normative/desks/‹роль›.yaml.
	Desks map[string]map[string]any `yaml:"-"`
}

// PolicyRole — роль политики.
type PolicyRole struct {
	ID       string   `yaml:"id"`
	Title    string   `yaml:"title"`
	Inherits []string `yaml:"inherits"`
}

// PolicyPerson — сотрудник (псевдоним) с ролями в областях.
type PolicyPerson struct {
	ID    string `yaml:"id"`
	Name  string `yaml:"name"`
	Roles []struct {
		Role  string `yaml:"role"`
		Scope string `yaml:"scope"`
	} `yaml:"roles"`
}

// LoadPolicy читает normative/policy/policy.v1.yaml и normative/desks/*.yaml из fsys.
func LoadPolicy(fsys fs.FS) (*Policy, error) {
	p := &Policy{Desks: map[string]map[string]any{}}
	b, err := fs.ReadFile(fsys, "normative/policy/policy.v1.yaml")
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, p); err != nil {
		return nil, fmt.Errorf("policy.v1.yaml: %w", err)
	}
	names, err := fs.Glob(fsys, "normative/desks/*.yaml")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	for _, n := range names {
		b, err := fs.ReadFile(fsys, n)
		if err != nil {
			return nil, err
		}
		var d map[string]any
		if err := yaml.Unmarshal(b, &d); err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		p.Desks[strings.TrimSuffix(path.Base(n), ".yaml")] = d
	}
	return p, nil
}

// Person — сотрудник по псевдониму.
func (p *Policy) Person(id string) (PolicyPerson, bool) {
	for _, x := range p.Persons {
		if x.ID == id {
			return x, true
		}
	}
	return PolicyPerson{}, false
}

// Role — роль по id.
func (p *Policy) Role(id string) (PolicyRole, bool) {
	for _, r := range p.Roles {
		if r.ID == id {
			return r, true
		}
	}
	return PolicyRole{}, false
}

// DeskFor — стол роли; у наследника без своего файла — стол ближайшей базовой роли (AD-21).
func (p *Policy) DeskFor(role string) (map[string]any, bool) {
	seen := map[string]bool{}
	queue := []string{role}
	for len(queue) > 0 {
		r := queue[0]
		queue = queue[1:]
		if seen[r] {
			continue
		}
		seen[r] = true
		if d, ok := p.Desks[r]; ok {
			return d, true
		}
		if pr, ok := p.Role(r); ok {
			queue = append(queue, pr.Inherits...)
		}
	}
	return nil, false
}
