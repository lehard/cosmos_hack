// Пакет notifications — производственные сценарии модуля notifications слоя application: сроки, задачи, уведомления, эскалации с ценой задержки.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/notifications,
// AD-36) и ведомые порты модуля; вызывает domain/notifications. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/notifications,
// хранение — infrastructure/storage/notifications.
//
// Требования: FR-8, FR-55, FR-57, AD-4, AD-45.
// Владелец после волны 1: эпик 24 (уведомления).
package notifications
