package platform

import "context"

// Principal — субъект запроса после барьера 1 (вход, AD-15): сотрудник, его
// активная роль и область, смена и рабочее место (барьер 2), версия политики.
type Principal struct {
	// PersonID — псевдоним сотрудника (кейс §4.6).
	PersonID string
	// Name — отображаемое имя (условное).
	Name string
	// Role — активная роль из политики (normative/policy).
	Role string
	// Roles — активная роль и все её базовые роли (наследование ролей — одна
	// функция access, domain/access.Roles.Closure, AD-15); заполняет порт
	// входа. Пусто — только Role.
	Roles []string
	// Scope — область роли: здание → цех → участок → рабочее место.
	Scope string
	// ShiftID, WorkplaceID, WorkplaceSessionID — смена и допуск к рабочему месту (барьер 2).
	ShiftID            string
	WorkplaceID        string
	WorkplaceSessionID string
	// PolicySeq — версия политики, по которой открыт сеанс.
	PolicySeq int64
	// Demo — вход демо-персоной без пароля (демо-трек, эпик 08).
	Demo bool
	// SessionID — идентификатор сеанса.
	SessionID string
}

// Anonymous — сеанса нет.
func (p Principal) Anonymous() bool { return p.PersonID == "" }

// EffectiveRoles — активная роль и её базовые роли (наследование — access, AD-15).
func (p Principal) EffectiveRoles() []string {
	if len(p.Roles) > 0 {
		return p.Roles
	}
	if p.Role == "" {
		return nil
	}
	return []string{p.Role}
}

// HasRole — субъект действует в роли role: это его активная роль или её
// базовая роль (начальник ОТК — и контролёр).
func (p Principal) HasRole(role string) bool {
	for _, r := range p.EffectiveRoles() {
		if r == role {
			return true
		}
	}
	return false
}

type principalKey struct{}

// WithPrincipal кладёт субъекта в контекст (делает transport после входа).
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom достаёт субъекта из контекста; нет — анонимный.
func PrincipalFrom(ctx context.Context) Principal {
	p, _ := ctx.Value(principalKey{}).(Principal)
	return p
}
