// Пакет journal — производственные сценарии модуля journal слоя application: журнал: единственная функция записи journal.Append, две хеш-цепочки, контрольные точки, чтение по оси и моменту, потребители, часы.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/journal,
// AD-36) и ведомые порты модуля; вызывает domain/journal. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/journal,
// хранение — infrastructure/storage/journal.
//
// Требования: FR-40, FR-71, FR-72, FR-122, AD-2, AD-8, AD-37, AD-44, AD-45.
// Владелец после волны 1: эпик 04 (журнал и хранение), 29 (хранитель).
package journal
