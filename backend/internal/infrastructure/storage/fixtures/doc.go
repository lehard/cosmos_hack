// Пакет fixtures — адаптер курсора сценария заготовок (порт
// platform.FixtureCursor, ключ fixture_cursor = postgres; AD-36): схема Postgres
// «fixtures», общий курсор для всех копий api — шаг, прогон, пауза, скорость,
// доменное «сейчас» шага.
//
// Слой: infrastructure/storage; не импортирует другие зоны.
// Владелец: эпик 09 (мир заготовок).
package fixtures
