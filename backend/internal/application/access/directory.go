package access

import accessdom "ant/internal/domain/access"

// Directory — каталог стартовой политики для входа и столов (демо-трек эпика
// 08): роли с наследованием, демо-персоны (псевдонимы сотрудников с ролью и
// областью, кейс §4.6) и столы ролей (normative/desks, AD-21). Источник —
// normative/policy и normative/desks; после генезиса (эпик 05) и проекции
// политики (эпик 26) — журнал. Только чтение после сборки.
type Directory struct {
	// Roles — роли в порядке политики.
	Roles []RoleRef
	// Personas — демо-персоны в порядке политики; роль и область — первые у сотрудника.
	Personas []DemoPersona
	// Desks — столы по id роли, у которой есть свой файл.
	Desks map[string]Desk
	// Authorities — полномочия сотрудников (grants.authorities политики):
	// псевдоним → id полномочий. Рамки (scope, limits) и сроки — у проекции
	// политики (accessdom.Policy.HasAuthority, эпик 26).
	Authorities map[string][]string
	// Workplaces — рабочие места (посты) из справочника мест: панель «Посты»,
	// назначения на посты (FR-6, FR-81).
	Workplaces []WorkplaceRef
}

// WorkplaceRef — рабочее место (пост) справочника мест.
type WorkplaceRef struct {
	ID    string
	Name  string
	Scope string
	// Workshop — цех (location id WS-…), в котором пост.
	Workshop string
	// WorkshopName — имя цеха из справочника мест.
	WorkshopName string
}

// Workplace — рабочее место по id.
func (d *Directory) Workplace(id string) (WorkplaceRef, bool) {
	for _, w := range d.Workplaces {
		if w.ID == id {
			return w, true
		}
	}
	return WorkplaceRef{}, false
}

// HasAuthority — у сотрудника person есть полномочие authority (FR-19, FR-50).
func (d *Directory) HasAuthority(person, authority string) bool {
	for _, a := range d.Authorities[person] {
		if a == authority {
			return true
		}
	}
	return false
}

// Role — роль по id.
func (d *Directory) Role(id string) (RoleRef, bool) {
	for _, r := range d.Roles {
		if r.ID == id {
			return r, true
		}
	}
	return RoleRef{}, false
}

// Persona — демо-персона по псевдониму.
func (d *Directory) Persona(id string) (DemoPersona, bool) {
	for _, p := range d.Personas {
		if p.ID == id {
			return p, true
		}
	}
	return DemoPersona{}, false
}

// Hierarchy — иерархия ролей затравки для accessdom.Roles.Closure.
func (d *Directory) Hierarchy() accessdom.Roles {
	h := make(accessdom.Roles, len(d.Roles))
	for _, r := range d.Roles {
		h[r.ID] = r.Inherits
	}
	return h
}

// DeskFor — стол роли: свой файл или стол ближайшей базовой роли (AD-21);
// порядок базовых ролей — одна функция наследования accessdom.Roles.Closure.
// Возвращается копия верхнего уровня.
func (d *Directory) DeskFor(role string) (Desk, bool) {
	for _, r := range d.Hierarchy().Closure(role) {
		if desk, ok := d.Desks[r]; ok {
			return desk, true
		}
	}
	return Desk{}, false
}
