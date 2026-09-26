// Пакет identity — адаптеры порта IdentityProvider (барьер 1, AD-15, FR-128)
// и затравка политики из нормативного слоя.
//
//   - Local (identity_provider = local): логин и пароль (argon2id, Argon2id),
//     сеансы scs в Postgres (SessionStore, pgxstore; cookie ant_session),
//     ограничение частоты попыток (golang.org/x/time/rate), блокировка после
//     N неудач, события неудачного входа security.auth.failed; в профилях
//     fixtures и demo — ещё вход демо-персоной без пароля и заголовок
//     Ant-Demo-Persona. Субъект и роли — по действующей политике на каждый запрос.
//   - Demo (identity_provider = demo): только демо-персоны, токен HMAC без
//     базы (выгрузка OpenAPI, тесты, запуск без Postgres).
//   - LoadSeed, LoadDirectory, LoadPlaces — затравка политики (до генезиса,
//     эпик 05), демо-персоны и столы ролей, области мест из normative/.
//
// Слой: infrastructure/security — технический механизм (AD-1); реализует
// порты application/access (IdentityProvider, PasswordHasher, Places).
// LDAP / ALD Pro / FreeIPA — следующий адаптер того же порта.
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
	accessdom "ant/internal/domain/access"
)

// PolicyFile, DesksGlob, LocationsFile — пути нормативного слоя от корня fsys (AD-21).
const (
	PolicyFile    = "normative/policy/policy.v1.yaml"
	DesksGlob     = "normative/desks/*.yaml"
	LocationsFile = "normative/reference/flange/locations.yaml"
)

// policyFile — стартовая политика (затравка до генезиса, AD-33).
type policyFile struct {
	Scopes struct {
		Root string `yaml:"root"`
	} `yaml:"scopes"`
	UnauthenticatedRole string `yaml:"unauthenticated_role"`
	Roles               []struct {
		ID       string   `yaml:"id"`
		Title    string   `yaml:"title"`
		CaseRole bool     `yaml:"case_role"`
		Inherits []string `yaml:"inherits"`
		Actions  []string `yaml:"actions"`
	} `yaml:"roles"`
	Persons []struct {
		ID    string `yaml:"id"`
		Name  string `yaml:"name"`
		Roles []struct {
			Role  string `yaml:"role"`
			Scope string `yaml:"scope"`
		} `yaml:"roles"`
	} `yaml:"persons"`
	Grants struct {
		Authorities []struct {
			Person    string `yaml:"person"`
			Authority string `yaml:"authority"`
			Scope     string `yaml:"scope"`
		} `yaml:"authorities"`
		Stamps []struct {
			Person   string `yaml:"person"`
			StampID  string `yaml:"stamp_id"`
			Kind     string `yaml:"kind"`
			Scope    string `yaml:"scope"`
			OrderRef string `yaml:"order_ref"`
		} `yaml:"stamps"`
	} `yaml:"grants"`
}

func readPolicy(fsys fs.FS) (policyFile, error) {
	var pf policyFile
	b, err := fs.ReadFile(fsys, PolicyFile)
	if err != nil {
		return pf, fmt.Errorf("каталог политики: %w", err)
	}
	if err := yaml.Unmarshal(b, &pf); err != nil {
		return pf, fmt.Errorf("%s: %w", PolicyFile, err)
	}
	return pf, nil
}

// LoadSeed — затравка политики из normative/policy (до генезиса, эпик 05):
// роли с действиями и наследованием, сотрудники с ролями в областях,
// полномочия и клейма, роль субъекта без сеанса.
func LoadSeed(fsys fs.FS) (accessdom.Seed, error) {
	pf, err := readPolicy(fsys)
	if err != nil {
		return accessdom.Seed{}, err
	}
	s := accessdom.Seed{Root: pf.Scopes.Root, Unauthenticated: pf.UnauthenticatedRole}
	known := map[string]bool{}
	for _, r := range pf.Roles {
		known[r.ID] = true
		s.Roles = append(s.Roles, accessdom.Role{ID: r.ID, Title: r.Title, CaseRole: r.CaseRole, Inherits: r.Inherits, Actions: r.Actions})
	}
	for _, r := range pf.Roles {
		for _, b := range r.Inherits {
			if !known[b] {
				return s, fmt.Errorf("%s: роль %s наследует неизвестную роль %q", PolicyFile, r.ID, b)
			}
		}
	}
	if s.Unauthenticated != "" && !known[s.Unauthenticated] {
		return s, fmt.Errorf("%s: unauthenticated_role — неизвестная роль %q", PolicyFile, s.Unauthenticated)
	}
	for _, p := range pf.Persons {
		sp := accessdom.SeedPerson{ID: p.ID, Name: p.Name}
		for _, g := range p.Roles {
			if !known[g.Role] {
				return s, fmt.Errorf("%s: у сотрудника %s неизвестная роль %q", PolicyFile, p.ID, g.Role)
			}
			sp.Roles = append(sp.Roles, accessdom.SeedGrant{Role: g.Role, Scope: g.Scope})
		}
		s.Persons = append(s.Persons, sp)
	}
	for _, a := range pf.Grants.Authorities {
		s.Authorities = append(s.Authorities, accessdom.Authority{PersonID: a.Person, AuthorityID: a.Authority, Scope: a.Scope})
	}
	for _, st := range pf.Grants.Stamps {
		s.Stamps = append(s.Stamps, accessdom.Stamp{StampID: st.StampID, PersonID: st.Person, Kind: st.Kind, Scope: st.Scope, OrderRef: st.OrderRef})
	}
	return s, nil
}

// Places — области мест из справочника мест (normative/reference): id места
// или рабочего места → путь области (порт application/access.Places).
type Places map[string]string

// ScopeOf — область места по id.
func (p Places) ScopeOf(id string) (string, bool) {
	s, ok := p[id]
	return s, ok
}

// LoadPlaces — справочник мест (здание → цех → участок → рабочее место).
func LoadPlaces(fsys fs.FS) (Places, error) {
	b, err := fs.ReadFile(fsys, LocationsFile)
	if err != nil {
		return nil, fmt.Errorf("справочник мест: %w", err)
	}
	var f struct {
		Locations []struct {
			ID    string `yaml:"id"`
			Scope string `yaml:"scope"`
		} `yaml:"locations"`
	}
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", LocationsFile, err)
	}
	out := Places{}
	for _, l := range f.Locations {
		if l.ID != "" && l.Scope != "" {
			out[l.ID] = l.Scope
		}
	}
	return out, nil
}

// LoadDirectory читает стартовую политику и столы ролей из fsys (корень
// репозитория или встроенная копия нормативного слоя).
func LoadDirectory(fsys fs.FS) (*access.Directory, error) {
	pf, err := readPolicy(fsys)
	if err != nil {
		return nil, err
	}
	d := &access.Directory{Desks: map[string]access.Desk{}, Authorities: map[string][]string{}}
	for _, g := range pf.Grants.Authorities {
		d.Authorities[g.Person] = append(d.Authorities[g.Person], g.Authority)
	}
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
