// Пакет security — производственные сценарии модуля security слоя application: сервис доверенных решений (CriticalActions), журнал критических действий, шина безопасности, индикатор целостности.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/security,
// AD-36) и ведомые порты модуля; вызывает domain/security. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/security,
// хранение — infrastructure/storage/security.
//
// Требования: FR-72…FR-77, FR-118, FR-146, AD-8, AD-9, AD-24, AD-28, AD-46.
// Владелец после волны 1: эпик 29 (доверие).
package security
