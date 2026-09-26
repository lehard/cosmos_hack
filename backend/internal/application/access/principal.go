package access

import (
	"time"

	"ant/internal/application/platform"
	accessdom "ant/internal/domain/access"
)

// PrincipalOf — субъект сеанса по действующей политике (барьер 1 → 3, AD-15):
// имя сотрудника, активная роль — роль сеанса, если она ещё назначена, иначе
// первая действующая; область этой роли; активная роль и её базовые роли —
// одной функцией наследования (accessdom.Roles.Closure); версия политики.
// Сотрудника нет в политике — false. Роль снята — сеанс остаётся, но без
// ролей: права решает политика на каждый запрос, а не токен.
func PrincipalOf(pol accessdom.Policy, personID, role string, at time.Time) (platform.Principal, bool) {
	person, ok := pol.Person(personID)
	if !ok {
		return platform.Principal{}, false
	}
	p := platform.Principal{PersonID: person.ID, Name: person.Name, PolicySeq: pol.Seq}
	as := pol.AssignmentsOf(person.ID, at)
	var active *accessdom.Assignment
	for i := range as {
		if as[i].RoleID == role {
			active = &as[i]
			break
		}
	}
	if active == nil && len(as) > 0 {
		active = &as[0]
	}
	if active != nil {
		p.Role, p.Scope = active.RoleID, active.Scope
		p.Roles = pol.Hierarchy().Closure(active.RoleID)
	}
	return p, true
}

// RoleRefOf — роль политики для ответа API (название и базовые роли).
func RoleRefOf(pol accessdom.Policy, id string) RoleRef {
	r, ok := pol.Role(id)
	if !ok {
		return RoleRef{ID: id, Title: id}
	}
	return RoleRef{ID: r.ID, Title: r.Title, Inherits: r.Inherits}
}
