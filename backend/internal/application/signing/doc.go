// Пакет signing — производственные сценарии модуля signing слоя application: подписи: криптопрофили, пакеты DSSE, проверка подписей, реестр ключей, акты регистрации и отзыва, генезис.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/signing,
// AD-36) и ведомые порты модуля; вызывает domain/signing. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/signing,
// хранение — infrastructure/storage/signing.
//
// Требования: FR-66…FR-70, FR-76, FR-79, AD-10, AD-11, AD-32, AD-33.
// Владелец после волны 1: эпик 27 (подписи и ключи), 05 (криптоядро и генезис).
package signing
