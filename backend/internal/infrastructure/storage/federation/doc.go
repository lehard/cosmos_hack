// Пакет federation — хранение модуля federation (зона storage, AD-1): своя схема Postgres
// «federation», свои миграции goose (migrations/, версии — метки времени), своя
// конфигурация sqlc (sqlc.yaml рядом, make generate) и проекции модуля — один
// писатель на проекцию (AD-45). Чужих таблиц не читает; в журнал пишет только
// через порт journal (AD-44).
//
// Слой: infrastructure/storage; реализует ведомые порты application/federation;
// не импортирует другие зоны.
// Владелец после волны 1: эпик 41 (федерация).
package federation
