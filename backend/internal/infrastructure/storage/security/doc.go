// Пакет security — хранение модуля security (зона storage, AD-1): своя схема Postgres
// «security», свои миграции goose (migrations/, версии — метки времени), своя
// конфигурация sqlc (sqlc.yaml рядом, make generate) и проекции модуля — один
// писатель на проекцию (AD-45). Чужих таблиц не читает; в журнал пишет только
// через порт journal (AD-44).
// Инфраструктура модуля security по AD-1 лежит здесь и в transport/security
// (infrastructure/security — только технические механизмы без модульных папок).
//
// Своих таблиц у модуля нет: журнал критических действий — цепочка ca журнала
// (пишет journal.Append), шина — записи семейства security, курсоры
// подписчиков — journal_state. Здесь — демо-инструмент подделки в обход
// системы (Tamperer, make tamper, AD-28; порт application/simulation.Tamperer)
// и интеграционные тесты доверия на своей БД.
//
// Слой: infrastructure/storage; реализует ведомые порты application/security;
// не импортирует другие зоны.
// Владелец после волны 1: эпик 29 (доверие).
package security
