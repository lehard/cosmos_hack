package access

import (
	"slices"
	"strings"
	"time"
)

// Эффективные права сотрудника (AD-11, AD-15): роли с наследованием (одна
// функция Roles.Closure) в областях назначений, полномочия и цифровые клейма
// с областью и сроком. Casbin решает доступ к операции только по ролям;
// полномочия и клейма — рамки доменных гардов (FR-50, FR-145): гард
// модуля-владельца спрашивает HasAuthority / StampFor у политики на basis_seq.
// «Выдача себе» и «расширение прав администратора» проверяются по этим же
// эффективным правам (Expands), а не по названию роли.

// RoleTraitsOf — свойства роли с наследованием: сфера — самая строгая среди
// роли и её базовых (admin важнее qc); «только карточка» — своя, если роль
// есть в каталоге нормативного слоя (метролог наследует согласующего, но
// работает и со справочниками), иначе — ближайшей базовой роли каталога
// (новая роль-наследник согласующего остаётся редким подписантом).
func (p Policy) RoleTraitsOf(role string) RoleTraits {
	var out RoleTraits
	cardSet := false
	for _, r := range p.Hierarchy().Closure(role) {
		t, ok := p.Catalog.Roles[r]
		switch {
		case t.Domain == DomainAdmin:
			out.Domain = DomainAdmin
		case t.Domain != "" && out.Domain == "":
			out.Domain = t.Domain
		}
		if ok && !cardSet {
			out.CardOnly, cardSet = t.CardOnly, true
		}
	}
	return out
}

// PersonDomain — сфера сотрудника для второй подписи (AD-11): admin, если
// хоть одна действующая роль — администратора или аудита; qc — роль ОТК;
// иначе production.
func (p Policy) PersonDomain(personID string, at time.Time) string {
	d := DomainProduction
	for _, a := range p.AssignmentsOf(personID, at) {
		switch p.RoleTraitsOf(a.RoleID).Domain {
		case DomainAdmin:
			return DomainAdmin
		case DomainQC:
			d = DomainQC
		}
	}
	return d
}

// CardOnly — сотрудник — редкий подписант (FR-136): все его действующие роли
// «только карточка». Без ролей — false (права решает политика: их нет).
func (p Policy) CardOnly(personID string, at time.Time) bool {
	as := p.AssignmentsOf(personID, at)
	if len(as) == 0 {
		return false
	}
	for _, a := range as {
		if !p.RoleTraitsOf(a.RoleID).CardOnly {
			return false
		}
	}
	return true
}

// HasRole — у сотрудника в момент at действует роль role (сама или как
// базовая роль назначенной) в области, включающей место scope (пусто — любое).
func (p Policy) HasRole(personID, role, scope string, at time.Time) bool {
	h := p.Hierarchy()
	for _, a := range p.AssignmentsOf(personID, at) {
		if (scope == "" || ScopeCovers(a.Scope, scope)) && h.Covers(a.RoleID, role) {
			return true
		}
	}
	return false
}

// HasAuthority — у сотрудника в момент at действует полномочие authority в
// области, включающей место scope (пусто — любое место). Срок — по доменному
// времени (AD-37).
func (p Policy) HasAuthority(personID, authority, scope string, at time.Time) bool {
	for _, a := range p.Authorities {
		if a.PersonID == personID && a.AuthorityID == authority && ValidAt(at, a.ValidFrom, a.ValidUntil) &&
			(scope == "" || ScopeCovers(a.Scope, scope)) {
			return true
		}
	}
	return false
}

// AuthoritiesOf — действующие полномочия сотрудника в момент at.
func (p Policy) AuthoritiesOf(personID string, at time.Time) []Authority {
	var out []Authority
	for _, a := range p.Authorities {
		if a.PersonID == personID && ValidAt(at, a.ValidFrom, a.ValidUntil) {
			out = append(out, a)
		}
	}
	return out
}

// StampFor — действующее цифровое клеймо сотрудника по виду контроля kind в
// области, включающей место scope (FR-145: одно клеймо на вид контроля).
func (p Policy) StampFor(personID, kind, scope string, at time.Time) (Stamp, bool) {
	for _, s := range p.Stamps {
		if s.PersonID == personID && s.Kind == kind && ValidAt(at, s.ValidFrom, s.ValidUntil) && (scope == "" || ScopeCovers(s.Scope, scope)) {
			return s, true
		}
	}
	return Stamp{}, false
}

// StampsOf — действующие клейма сотрудника в момент at.
func (p Policy) StampsOf(personID string, at time.Time) []Stamp {
	var out []Stamp
	for _, s := range p.Stamps {
		if s.PersonID == personID && ValidAt(at, s.ValidFrom, s.ValidUntil) {
			out = append(out, s)
		}
	}
	return out
}

// Виды права в Rights.
const (
	RightAction    = "action"
	RightAuthority = "authority"
	RightStamp     = "stamp"
)

// Right — одно эффективное право: действие (шаблон x-ant-action), полномочие
// или клеймо по виду контроля — в области.
type Right struct {
	Kind  string
	Value string
	Scope string
}

// Rights — эффективные права сотрудника.
type Rights []Right

// EffectiveRights — права сотрудника в момент at: действия всех ролей его
// назначений с наследованием (в области назначения), полномочия и клейма.
func (p Policy) EffectiveRights(personID string, at time.Time) Rights {
	var out Rights
	h := p.Hierarchy()
	for _, a := range p.AssignmentsOf(personID, at) {
		for _, rid := range h.Closure(a.RoleID) {
			if r, ok := p.Role(rid); ok {
				for _, act := range r.Actions {
					out = append(out, Right{Kind: RightAction, Value: act, Scope: a.Scope})
				}
			}
		}
	}
	for _, a := range p.AuthoritiesOf(personID, at) {
		out = append(out, Right{Kind: RightAuthority, Value: a.AuthorityID, Scope: a.Scope})
	}
	for _, s := range p.StampsOf(personID, at) {
		out = append(out, Right{Kind: RightStamp, Value: s.Kind, Scope: s.Scope})
	}
	return out
}

// Covers — право x уже есть в r: то же или более общее действие (шаблон с `*`)
// или то же полномочие / клеймо — в области, включающей область x.
func (r Rights) Covers(x Right) bool {
	for _, b := range r {
		if b.Kind != x.Kind || !ScopeCovers(b.Scope, x.Scope) {
			continue
		}
		if b.Value == x.Value || (b.Kind == RightAction && patternCovers(b.Value, x.Value)) {
			return true
		}
	}
	return false
}

// Expands — права after, которых нет в before (расширение прав), в порядке after без повторов.
func Expands(before, after Rights) Rights {
	var out Rights
	for _, x := range after {
		if !before.Covers(x) && !slices.Contains(out, x) {
			out = append(out, x)
		}
	}
	return out
}

// patternCovers — шаблон действия a включает шаблон b: в каждом сегменте `*` или совпадение.
func patternCovers(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	if len(as) != 3 || len(bs) != 3 {
		return false
	}
	for i := range as {
		if as[i] != "*" && as[i] != bs[i] {
			return false
		}
	}
	return true
}

// Allows — в правах есть действие id (x-ant-action) в какой-либо области.
func (r Rights) Allows(id string) bool {
	for _, b := range r {
		if b.Kind == RightAction && patternCovers(b.Value, id) {
			return true
		}
	}
	return false
}
