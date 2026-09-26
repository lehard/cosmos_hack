// Пакет process — хранение модуля process (зона storage, AD-1): своя схема
// Postgres «process» с версиями процесса BPMN (байты XML как загружены, хеш
// при загрузке, статус и подписи кворума; AD-17, FR-22, FR-23) и свои
// миграции goose (migrations/, версии — метки времени). Запросы — на pgx без
// sqlc (как у ingest, Д-21). Проекции модуля (process.item, process.index,
// process.runs) пишет движок эффектами в engine.projections (AD-45). Чужих
// таблиц не читает; в журнал пишет только через порт journal (AD-44).
//
// Слой: infrastructure/storage; реализует ведомый порт
// application/process.VersionStore; не импортирует другие зоны.
// Владелец: эпик 17 (процесс), дальше 39 (редактор и кворум).
package process
