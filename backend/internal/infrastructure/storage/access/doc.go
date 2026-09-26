// Пакет access — хранение модуля access (зона storage, AD-1): своя схема Postgres
// «access», свои миграции goose (migrations/, версии — метки времени), своя
// конфигурация sqlc (sqlc.yaml рядом, make generate) и проекции модуля — один
// писатель на проекцию (AD-45). Чужих таблиц не читает; в журнал пишет только
// через порт journal (AD-44).
//
// Слой: infrastructure/storage; реализует ведомые порты application/access;
// не импортирует другие зоны.
// Владелец после волны 1: эпик 08, 26, 37.
package access
