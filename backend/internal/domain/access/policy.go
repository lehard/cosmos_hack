package access

import (
	"slices"
	"time"
)

// AnonymousSubject — субъект запроса без сеанса. Ему политика назначает роль
// Policy.Unauthenticated (приём фактов от устройств: подлинность — подпись
// пакета, AD-10, а не сеанс). Псевдонимы сотрудников не начинаются с «@».
const AnonymousSubject = "@anonymous"

// Role — роль политики (policy.role.defined): действия — шаблоны x-ant-action.
type Role struct {
	ID       string
	Title    string
	CaseRole bool
	Inherits []string
	Actions  []string
}

// Assignment — роль сотрудника в области со сроком (policy.role.assigned, FR-78).
type Assignment struct {
	PersonID   string
	RoleID     string
	Scope      string
	ValidFrom  time.Time
	ValidUntil time.Time
}

// ActiveAt — назначение действует в момент at (доменное время, AD-37).
func (a Assignment) ActiveAt(at time.Time) bool { return ValidAt(at, a.ValidFrom, a.ValidUntil) }

// Authority — полномочие поверх роли с областью и сроком (policy.authority.granted, FR-50, FR-78).
type Authority struct {
	PersonID    string
	AuthorityID string
	Scope       string
	ValidFrom   time.Time
	ValidUntil  time.Time
}

// Stamp — цифровое клеймо контролёра (policy.stamp.issued, FR-145).
type Stamp struct {
	StampID    string
	PersonID   string
	Kind       string
	Scope      string
	OrderRef   string
	ValidFrom  time.Time
	ValidUntil time.Time
	Revoked    bool
}

// Person — сотрудник (псевдоним, кейс §4.6) и его учётная запись (FR-128).
type Person struct {
	ID      string
	Name    string
	OrgUnit string
	// Login — логин учётной записи; пусто — учётной записи нет.
	Login string
	// Active — учётная запись активирована администратором (access.account.activated).
	Active bool
}

// Policy — действующая политика «кто — что — над чем — где — когда» (AD-15):
// затравка или генезис плюс записи policy.* и access.* журнала. Значение
// неизменяемо для читателей: проекция применяет записи к копии (Clone).
type Policy struct {
	// Seq — seq последней применённой записи журнала (policy_seq, AD-39); 0 — только затравка.
	Seq int64
	// Root — корень областей (предприятие).
	Root  string
	Roles []Role
	// Persons — сотрудники в порядке появления.
	Persons     []Person
	Assignments []Assignment
	Authorities []Authority
	Stamps      []Stamp
	// Unauthenticated — роль субъекта без сеанса (AnonymousSubject); пусто — без сеанса ничего нельзя.
	Unauthenticated string
	// Catalog — справочная часть из нормативного слоя (сферы, маршрут выдачи):
	// переходит и в политику генезиса, как Root.
	Catalog Catalog
	// Audit — действующие параметры аудита policy.audit.* (затравка или
	// последняя запись policy.audit.parameters_set Аудитора ИБ).
	Audit AuditParameters
	// Qualifications — квалификации со сроками (FR-80); Posts — действующие
	// назначения на посты в сменах (FR-81).
	Qualifications []Qualification
	Posts          []PostAssignment
}

// Clone — глубокая копия для применения новых записей.
func (p Policy) Clone() Policy {
	c := p
	c.Roles = make([]Role, len(p.Roles))
	for i, r := range p.Roles {
		r.Inherits = slices.Clone(r.Inherits)
		r.Actions = slices.Clone(r.Actions)
		c.Roles[i] = r
	}
	c.Persons = slices.Clone(p.Persons)
	c.Assignments = slices.Clone(p.Assignments)
	c.Authorities = slices.Clone(p.Authorities)
	c.Stamps = slices.Clone(p.Stamps)
	c.Qualifications = slices.Clone(p.Qualifications)
	c.Posts = slices.Clone(p.Posts)
	c.Audit.CriticalTypes = slices.Clone(p.Audit.CriticalTypes)
	c.Audit.SecurityBusSubscribers = slices.Clone(p.Audit.SecurityBusSubscribers)
	return c
}

// Role — роль по id.
func (p Policy) Role(id string) (Role, bool) {
	for _, r := range p.Roles {
		if r.ID == id {
			return r, true
		}
	}
	return Role{}, false
}

// Person — сотрудник по псевдониму.
func (p Policy) Person(id string) (Person, bool) {
	for _, x := range p.Persons {
		if x.ID == id {
			return x, true
		}
	}
	return Person{}, false
}

// PersonByLogin — сотрудник по логину учётной записи.
func (p Policy) PersonByLogin(login string) (Person, bool) {
	if login == "" {
		return Person{}, false
	}
	for _, x := range p.Persons {
		if x.Login == login {
			return x, true
		}
	}
	return Person{}, false
}

// Hierarchy — иерархия ролей политики для Roles.Closure.
func (p Policy) Hierarchy() Roles {
	h := make(Roles, len(p.Roles))
	for _, r := range p.Roles {
		h[r.ID] = r.Inherits
	}
	return h
}

// AssignmentsOf — назначения сотрудника, действующие в момент at, в порядке политики.
func (p Policy) AssignmentsOf(personID string, at time.Time) []Assignment {
	var out []Assignment
	for _, a := range p.Assignments {
		if a.PersonID == personID && a.ActiveAt(at) {
			out = append(out, a)
		}
	}
	return out
}

// Rule — строка политики для вычислителя (Casbin, AD-15): PType «p» —
// роль → шаблон действия; «g» — субъект или роль → роль в домене
// (GrantPlace / AnyPlace).
type Rule struct {
	PType string
	V     []string
}

// Rules — политика строками вычислителя, в детерминированном порядке:
// разрешения ролей, наследование ролей (домен AnyPlace), назначения ролей в
// области со сроком, роль субъекта без сеанса.
func (p Policy) Rules() []Rule {
	var out []Rule
	for _, r := range p.Roles {
		for _, a := range r.Actions {
			out = append(out, Rule{PType: "p", V: []string{r.ID, a}})
		}
	}
	for _, r := range p.Roles {
		for _, base := range r.Inherits {
			out = append(out, Rule{PType: "g", V: []string{r.ID, base, AnyPlace}})
		}
	}
	for _, a := range p.Assignments {
		out = append(out, Rule{PType: "g", V: []string{a.PersonID, a.RoleID, GrantPlace(a.Scope, a.ValidFrom, a.ValidUntil)}})
	}
	if p.Unauthenticated != "" {
		out = append(out, Rule{PType: "g", V: []string{AnonymousSubject, p.Unauthenticated, GrantPlace(p.Root, time.Time{}, time.Time{})}})
	}
	return out
}
