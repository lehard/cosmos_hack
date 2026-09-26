package world

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	accessdom "ant/internal/domain/access"
)

// Policy — то, что мир заготовок берёт из стартовой политики normative/policy
// (демо-персоны, роли) и столов normative/desks: сеансы и столы ролей отдаются
// заготовками access тем же форматом, что live (AD-21, AD-36).
type Policy struct {
	Roles   []PolicyRole   `yaml:"roles"`
	Persons []PolicyPerson `yaml:"persons"`
	// Grants — выдачи затравки; здесь нужны только квалификации (карточка сотрудника, UI-16).
	Grants struct {
		Qualifications []PolicyQualification `yaml:"qualifications"`
	} `yaml:"grants"`
	// Desks — столы ролей: id роли → содержимое normative/desks/‹роль›.yaml.
	Desks map[string]map[string]any `yaml:"-"`
}

// PolicyRole — роль политики.
type PolicyRole struct {
	ID       string   `yaml:"id"`
	Title    string   `yaml:"title"`
	Inherits []string `yaml:"inherits"`
	// Actions — операции роли (шаблоны access): кто вправе решить по «Запросить решение».
	Actions []string `yaml:"actions"`
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

// PolicyQualification — квалификация сотрудника из затравки (FR-80).
type PolicyQualification struct {
	Person         string `yaml:"person"`
	Qualification  string `yaml:"qualification"`
	Scope          string `yaml:"scope"`
	ValidUntil     string `yaml:"valid_until"`
	CertificateRef string `yaml:"certificate_ref"`
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

// DeskFor — стол роли; у наследника без своего файла — стол ближайшей базовой
// роли (AD-21) по одной функции наследования ролей access (accessdom.Roles.Closure).
func (p *Policy) DeskFor(role string) (map[string]any, bool) {
	h := make(accessdom.Roles, len(p.Roles))
	for _, r := range p.Roles {
		h[r.ID] = r.Inherits
	}
	for _, r := range h.Closure(role) {
		if d, ok := p.Desks[r]; ok {
			return d, true
		}
	}
	return nil, false
}

// Holders — псевдонимы сотрудников, чья роль (с наследованием) разрешает
// действие action: подписанты запроса решения на заготовках (FR-146) —
// «запрос уходит к тем, у кого есть полномочия». Области не учитываются.
func (p *Policy) Holders(action string) []string {
	h := make(accessdom.Roles, len(p.Roles))
	acts := map[string][]string{}
	for _, r := range p.Roles {
		h[r.ID] = r.Inherits
		acts[r.ID] = r.Actions
	}
	var out []string
	for _, x := range p.Persons {
		for _, pr := range x.Roles {
			if p.grants(h, acts, pr.Role, action) && !slices.Contains(out, x.ID) {
				out = append(out, x.ID)
			}
		}
	}
	return out
}

func (p *Policy) grants(h accessdom.Roles, acts map[string][]string, role, action string) bool {
	for _, r := range h.Closure(role) {
		for _, pat := range acts[r] {
			if accessdom.ActionMatches(action, pat) {
				return true
			}
		}
	}
	return false
}
