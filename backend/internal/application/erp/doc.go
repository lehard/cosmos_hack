// Пакет erp — производственные сценарии модуля erp слоя application: учётные системы 1С и Галактика: входящие задания и справочники, исходящие учётные сообщения и квитанции.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/erp,
// AD-36) и ведомые порты модуля; вызывает domain/erp. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/erp,
// хранение — infrastructure/storage/erp.
//
// Требования: FR-90, FR-91, FR-95, FR-96, FR-111, AD-7, AD-18.
// Владелец после волны 1: эпик 30 (1С), 31 (Галактика).
package erp
