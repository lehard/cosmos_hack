// Пакет security — хранение модуля security (зона storage, AD-1): своя схема Postgres
// «security», свои миграции goose (migrations/, версии — метки времени), своя
// конфигурация sqlc (sqlc.yaml рядом, make generate) и проекции модуля — один
// писатель на проекцию (AD-45). Чужих таблиц не читает; в журнал пишет только
// через порт journal (AD-44).
// Инфраструктура модуля security по AD-1 лежит здесь и в transport/security
// (infrastructure/security — только технические механизмы без модульных папок).
//
// Слой: infrastructure/storage; реализует ведомые порты application/security;
// не импортирует другие зоны.
// Владелец после волны 1: эпик 29 (доверие).
package security
