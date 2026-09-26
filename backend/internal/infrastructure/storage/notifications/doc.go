// Пакет notifications — хранение модуля notifications (зона storage, AD-1): своя схема Postgres
// «notifications», свои миграции goose (migrations/, версии — метки времени), своя
// конфигурация sqlc (sqlc.yaml рядом, make generate) и проекции модуля — один
// писатель на проекцию (AD-45). Чужих таблиц не читает; в журнал пишет только
// через порт journal (AD-44).
//
// Слой: infrastructure/storage; реализует ведомые порты application/notifications;
// не импортирует другие зоны.
// Владелец после волны 1: эпик 24 (уведомления).
package notifications
