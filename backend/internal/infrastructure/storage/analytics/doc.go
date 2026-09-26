// Пакет analytics — хранение модуля analytics (зона storage, AD-1, AD-45):
// чтение строк вклада изделий и глобальных проекций показателей (ведомый порт
// application/analytics.Store). Строки и проекции пишет только движок
// эффектами в транзакции journal.Append (ContributionsReplace, ProjectionPut)
// в таблицы каркаса проекций engine.contributions и engine.projections
// (имена analytics.*); своей схемы и миграций у модуля нет — агрегаты
// считаются запросом над строками вклада, материализаций нет.
//
// Слой: infrastructure/storage; реализует ведомый порт application/analytics;
// не импортирует другие зоны.
// Владелец: эпик 25 (аналитика).
package analytics
