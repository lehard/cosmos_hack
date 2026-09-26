package notifications

import (
	"ant/internal/application/platform"
	accessdom "ant/internal/domain/access"
)

// Область задач (решение пользователя, показ): список задач сужается на
// сервере по области сеанса — задача с местом видна, только если её место
// входит в область персоны. Мастер сварочного цеха (ent01/b1/wc) видит
// «Принять в цех» на WS-WC и не видит «Отправить …» механического цеха
// (WS-MC); технолог и начальник ОТК (ent01) видят всё. Сопоставление путей —
// одна функция access (accessdom.ScopeCovers, AD-15), место → путь области —
// справочник мест (normative/reference, порт Places).

// Places — ведомый порт мест: id места (цех, участок, рабочее место) → путь
// области `здание/цех/участок/рабочее место` (справочник мест; реализация —
// identity.Places, та же, что у барьера 3 access).
type Places interface {
	// ScopeOf — область места по id; нет в справочнике — false.
	ScopeOf(id string) (string, bool)
}

// InScope — задача с местом locationID видна субъекту p по области (FR-78,
// FR-85): место входит в область его роли (accessdom.ScopeCovers). Не сужает:
// анонимный вызов, задача без места (общие задачи — по правилам адресности),
// справочника мест нет, место не из справочника (адресность решает сама),
// область сеанса пустая или «*» — всё предприятие.
func InScope(p platform.Principal, locationID string, places Places) bool {
	if p.Anonymous() || locationID == "" || places == nil {
		return true
	}
	scope, ok := places.ScopeOf(locationID)
	if !ok {
		return true
	}
	return accessdom.ScopeCovers(p.Scope, scope)
}
