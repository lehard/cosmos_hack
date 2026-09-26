// Пакет item — производственные сценарии модуля item слоя application: изделие: паспорт, зоны, носители идентификатора, сборка, вмешательства, выпуск.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/item,
// AD-36) и ведомые порты модуля; вызывает domain/item. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/item,
// хранение — infrastructure/storage/item.
//
// Требования: FR-42…FR-46, FR-130, AD-16, AD-41.
// Владелец после волны 1: эпик 18 (изделие, генеалогия).
package item
