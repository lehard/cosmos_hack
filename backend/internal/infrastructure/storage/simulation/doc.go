// Пакет simulation — хранение модуля simulation (зона storage, AD-1): своя схема Postgres
// «simulation», свои миграции goose (migrations/, версии — метки времени), своя
// конфигурация sqlc (sqlc.yaml рядом, make generate) и проекции модуля — один
// писатель на проекцию (AD-45). Чужих таблиц не читает; в журнал пишет только
// через порт journal (AD-44).
//
// Слой: infrastructure/storage; реализует ведомые порты application/simulation;
// не импортирует другие зоны.
// Владелец после волны 1: эпик 32 (симуляция), 36 (цифровой стенд).
package simulation
