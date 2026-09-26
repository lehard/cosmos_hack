// Пакет documents — хранение модуля documents (зона storage, AD-1): своя схема Postgres
// «documents», свои миграции goose (migrations/, версии — метки времени), своя
// конфигурация sqlc (sqlc.yaml рядом, make generate) и проекции модуля — один
// писатель на проекцию (AD-45). Чужих таблиц не читает; в журнал пишет только
// через порт journal (AD-44).
//
// Слой: infrastructure/storage; реализует ведомые порты application/documents;
// не импортирует другие зоны.
// Владелец после волны 1: эпик 28 (документы), 44 (каталог).
package documents
