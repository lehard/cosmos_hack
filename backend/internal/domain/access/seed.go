package access

// Seed — стартовая политика (normative/policy/policy.v1.yaml) до генезиса
// (эпик 05): генезис записывает те же данные записями policy.* (AD-33), и
// тогда проекция начинается с пустой политики.
type Seed struct {
	Root            string
	Unauthenticated string
	Roles           []Role
	Persons         []SeedPerson
	Authorities     []Authority
	Stamps          []Stamp
	// Catalog — сферы ролей и полномочий, маршрут выдачи, параметры аудита по умолчанию.
	Catalog Catalog
}

// SeedPerson — сотрудник затравки с ролями в областях.
type SeedPerson struct {
	ID    string
	Name  string
	Roles []SeedGrant
}

// SeedGrant — роль в области (бессрочно, с начала журнала).
type SeedGrant struct {
	Role  string
	Scope string
}

// FromSeed — политика из затравки: назначения ролей бессрочные, учётных
// записей нет (вход демо-персоной — только профили fixtures и demo; учётную
// запись активирует администратор, FR-128).
func FromSeed(s Seed) Policy {
	p := Policy{Root: s.Root, Unauthenticated: s.Unauthenticated, Catalog: s.Catalog, Audit: s.Catalog.Audit}
	p.Roles = append(p.Roles, s.Roles...)
	for _, sp := range s.Persons {
		p.Persons = append(p.Persons, Person{ID: sp.ID, Name: sp.Name})
		for _, g := range sp.Roles {
			p.Assignments = append(p.Assignments, Assignment{PersonID: sp.ID, RoleID: g.Role, Scope: g.Scope})
		}
	}
	p.Authorities = append(p.Authorities, s.Authorities...)
	p.Stamps = append(p.Stamps, s.Stamps...)
	return p.Clone()
}
