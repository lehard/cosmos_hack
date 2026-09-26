// Пакет materials — хранение модуля materials (зона storage, AD-1): своя схема Postgres
// «materials», свои миграции goose (migrations/, версии — метки времени), своя
// конфигурация sqlc (sqlc.yaml рядом, make generate) и проекции модуля — один
// писатель на проекцию (AD-45). Чужих таблиц не читает; в журнал пишет только
// через порт journal (AD-44).
//
// Слой: infrastructure/storage; реализует ведомые порты application/materials;
// не импортирует другие зоны.
// Владелец после волны 1: эпик 04 (журнал и хранение).
package materials
