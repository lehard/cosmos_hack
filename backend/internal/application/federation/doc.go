// Пакет federation — производственные сценарии модуля federation слоя application: федерация предприятий: партнёры, выписки паспорта, межзаводской обмен.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/federation,
// AD-36) и ведомые порты модуля; вызывает domain/federation. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/federation,
// хранение — infrastructure/storage/federation.
//
// Требования: FR-131…FR-134, AD-19.
// Владелец после волны 1: эпик 41 (федерация).
package federation
