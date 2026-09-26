// Пакет analytics — производственные сценарии модуля analytics слоя application: аналитика: вклады изделий и агрегаты показателей, контрольные карты, счётчики узлов, ограничение линии.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/analytics,
// AD-36) и ведомые порты модуля; вызывает domain/analytics. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/analytics,
// хранение — infrastructure/storage/analytics.
//
// Требования: FR-3, FR-5, FR-86…FR-89, AD-45.
// Владелец после волны 1: эпик 25 (аналитика).
package analytics
