// Пакет identity — адаптер порта IdentityProvider для демо-трека эпика 08
// (ключ identity_provider: demo): вход демо-персоной без пароля и сеанс
// подписанным токеном в cookie; каталог политики (роли, демо-персоны, столы)
// из normative/policy и normative/desks.
//
// Слой: infrastructure/security — технический механизм (AD-1); реализует порт
// application/access.IdentityProvider и собирает access.Directory.
//
// Вход по логину и паролю (argon2id), сеансы scs в Postgres, блокировка после
// неудач — эпик 08 во втором слое; этот адаптер тогда остаётся только для
// профилей fixtures и demo.
package identity

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"ant/internal/application/access"
)

// PolicyFile, DesksGlob — пути нормативного слоя от корня fsys (AD-21).
const (
	PolicyFile = "normative/policy/policy.v1.yaml"
	DesksGlob  = "normative/desks/*.yaml"
)

// policyFile — то, что нужно входу из стартовой политики.
type policyFile struct {
	Roles []struct {
		ID       string   `yaml:"id"`
		Title    string   `yaml:"title"`
		Inherits []string `yaml:"inherits"`
	} `yaml:"roles"`
	Persons []struct {
		ID    string `yaml:"id"`
		Name  string `yaml:"name"`
		Roles []struct {
			Role  string `yaml:"role"`
			Scope string `yaml:"scope"`
		} `yaml:"roles"`
	} `yaml:"persons"`
}

// LoadDirectory читает стартовую политику и столы ролей из fsys (корень
// репозитория или встроенная копия нормативного слоя).
func LoadDirectory(fsys fs.FS) (*access.Directory, error) {
	b, err := fs.ReadFile(fsys, PolicyFile)
	if err != nil {
		return nil, fmt.Errorf("каталог политики: %w", err)
	}
	var pf policyFile
	if err := yaml.Unmarshal(b, &pf); err != nil {
		return nil, fmt.Errorf("%s: %w", PolicyFile, err)
	}
	d := &access.Directory{Desks: map[string]access.Desk{}}
	for _, r := range pf.Roles {
		d.Roles = append(d.Roles, access.RoleRef{ID: r.ID, Title: r.Title, Inherits: r.Inherits})
	}
	for _, p := range pf.Persons {
		if len(p.Roles) == 0 {
			continue
		}
		role, ok := d.Role(p.Roles[0].Role)
		if !ok {
			return nil, fmt.Errorf("%s: у сотрудника %s неизвестная роль %q", PolicyFile, p.ID, p.Roles[0].Role)
		}
		d.Personas = append(d.Personas, access.DemoPersona{ID: p.ID, Name: p.Name, Role: role, Scope: p.Roles[0].Scope})
	}
	names, err := fs.Glob(fsys, DesksGlob)
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	for _, n := range names {
		desk, err := loadDesk(fsys, n)
		if err != nil {
			return nil, err
		}
		d.Desks[strings.TrimSuffix(path.Base(n), ".yaml")] = desk
	}
	return d, nil
}

// loadDesk — стол роли: YAML → JSON → access.Desk (те же имена полей, что у
// схемы normative/desks/desk.schema.json и операции access.desk.read).
func loadDesk(fsys fs.FS, name string) (access.Desk, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return access.Desk{}, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return access.Desk{}, fmt.Errorf("%s: %w", name, err)
	}
	j, err := json.Marshal(raw)
	if err != nil {
		return access.Desk{}, fmt.Errorf("%s: %w", name, err)
	}
	var desk access.Desk
	if err := json.Unmarshal(j, &desk); err != nil {
		return access.Desk{}, fmt.Errorf("%s: %w", name, err)
	}
	if len(desk.Tabs) == 0 {
		return access.Desk{}, fmt.Errorf("%s: у стола нет вкладок", name)
	}
	return desk, nil
}
