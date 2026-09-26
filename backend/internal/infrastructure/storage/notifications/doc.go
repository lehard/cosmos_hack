// Пакет notifications — хранение модуля notifications (зона storage, AD-1):
// чтение проекций модуля — сроков (единственная проекция сроков, AD-4),
// задач и уведомлений. Проекции пишет только движок (роль projector)
// эффектами в транзакции journal.Append (AD-45) в таблицу каркаса
// engine.projections под именами notifications.*; своей схемы и миграций у
// модуля пока нет (как у analytics, Д-44). В журнал пишет только через порт
// journal (AD-44).
//
// Слой: infrastructure/storage; реализует ведомые порты application/notifications;
// не импортирует другие зоны.
// Требования: FR-8, FR-55, FR-57, AD-4, AD-45.
// Владелец: эпик 24 (уведомления).
package notifications
