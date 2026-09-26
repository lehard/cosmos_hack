package platform

// DrillRef — ссылка для проваливания в объект из любого виджета (FR-7): вид
// сущности и id; фронтенд открывает объект по ключу Vue Query [entity, id].
type DrillRef struct {
	Entity EntityKind `json:"entity"`
	ID     string     `json:"id" maxLength:"128"`
}
