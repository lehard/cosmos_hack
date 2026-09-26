package access

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
	// псевдоним → id полномочий. Рамки (scope, limits) — эпик 26.
	Authorities map[string][]string
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

// DeskFor — стол роли: свой файл или стол ближайшей базовой роли (обход
// наследования в ширину, AD-21). Возвращается копия верхнего уровня.
func (d *Directory) DeskFor(role string) (Desk, bool) {
	seen := map[string]bool{}
	queue := []string{role}
	for len(queue) > 0 {
		r := queue[0]
		queue = queue[1:]
		if seen[r] {
			continue
		}
		seen[r] = true
		if desk, ok := d.Desks[r]; ok {
			return desk, true
		}
		if rr, ok := d.Role(r); ok {
			queue = append(queue, rr.Inherits...)
		}
	}
	return Desk{}, false
}
