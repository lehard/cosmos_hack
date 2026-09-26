package access

import "strings"

// ScopeCovers — область granted (путь `здание/цех/участок/рабочее место`,
// AD-15) включает место target: совпадает с ним или является его предком
// («ent01/b1/wc» включает «ent01/b1/wc/weld/wp1», но не «ent01/b1/wcx»).
// Пустая или «*» область политики — всё предприятие.
//
// FR-78, FR-85: исполнитель, назначенный на пост А, не может действовать на
// посту Б — область его роли не включает место Б.
func ScopeCovers(granted, target string) bool {
	granted = strings.Trim(granted, "/")
	target = strings.Trim(target, "/")
	if granted == "" || granted == "*" {
		return true
	}
	return target == granted || strings.HasPrefix(target, granted+"/")
}
