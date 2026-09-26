package access

import "strings"

// ReadClass — класс операции «чтение» (AD-27, x-ant-action class).
const ReadClass = "read"

// ActionMatches — действие id (`‹модуль›.‹объект›.‹действие›`, x-ant-action,
// AD-40) подходит под шаблон политики pattern: три сегмента, `*` — любой
// сегмент (схема policy.role.defined).
func ActionMatches(id, pattern string) bool {
	a := strings.Split(id, ".")
	p := strings.Split(pattern, ".")
	if len(a) != 3 || len(p) != 3 {
		return false
	}
	for i := range a {
		if a[i] == "" || (p[i] != "*" && p[i] != a[i]) {
			return false
		}
	}
	return true
}

// ReadAlias — второе имя операции чтения для политики: `‹модуль›.*.read`.
// Шаблон `quality.*.read` в роли разрешает все операции чтения модуля
// quality — и `quality.signal.read`, и `quality.signal.list`: класс операции
// (AD-27) — атрибут запроса. У операций других классов второго имени нет.
func ReadAlias(id, class string) string {
	if class != ReadClass {
		return ""
	}
	module, _, ok := strings.Cut(id, ".")
	if !ok || module == "" {
		return ""
	}
	return module + ".*." + ReadClass
}
