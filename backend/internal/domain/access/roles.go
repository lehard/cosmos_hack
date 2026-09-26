package access

// Roles — иерархия ролей политики: роль → её базовые роли (inherits).
// Руководящие подписанты — наследники базовых ролей (AD-15): начальник ОТК
// наследует контролёра, начальник цеха — мастера.
type Roles map[string][]string

// Closure — роль и все её базовые роли транзитивно, в порядке обхода в ширину
// (сначала сама роль, затем ближайшие базовые). Единственная функция
// наследования ролей системы (AD-15): вычислитель прав получает те же связи
// из политики, стол роли-наследника без своего файла — стол ближайшей базовой,
// задачи по роли видят и наследники. Циклы политики не зацикливают обход.
func (r Roles) Closure(role string) []string {
	if role == "" {
		return nil
	}
	out := []string{}
	seen := map[string]bool{}
	queue := []string{role}
	for len(queue) > 0 {
		x := queue[0]
		queue = queue[1:]
		if seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
		queue = append(queue, r[x]...)
	}
	return out
}

// Covers — роль role включает base: это она сама или её базовая роль.
func (r Roles) Covers(role, base string) bool {
	for _, x := range r.Closure(role) {
		if x == base {
			return true
		}
	}
	return false
}

// CoversAny — одна из ролей roles включает base.
func (r Roles) CoversAny(roles []string, base string) bool {
	for _, x := range roles {
		if r.Covers(x, base) {
			return true
		}
	}
	return false
}
