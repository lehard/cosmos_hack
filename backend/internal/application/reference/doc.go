// Пакет reference — производственные сценарии модуля reference слоя application: справочники с версиями: номенклатура, места, оборудование и поверка, календарь, смены, соответствия внешних ID.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/reference,
// AD-36) и ведомые порты модуля; вызывает domain/reference. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/reference,
// хранение — infrastructure/storage/reference.
//
// Требования: FR-17, FR-80, FR-81, FR-95, AD-31, AD-18.
// Владелец после волны 1: эпик 19 (справочники, календарь, смены).
package reference
