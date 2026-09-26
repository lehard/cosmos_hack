// Пакет engine — производственные сценарии модуля engine слоя application: движок: композиция свёртки изделия в фиксированном порядке и сценарий воркера (вход через WorkFeed, сравнение реакций, запись через journal).
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/engine,
// AD-36) и ведомые порты модуля; вызывает domain/engine. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/engine,
// хранение — infrastructure/storage/engine.
//
// Требования: FR-2, FR-32, FR-40, FR-112, FR-124, AD-3, AD-4, AD-5, AD-39, AD-40.
// Владелец после волны 1: эпик 07 (движок, воркер, проекции).
package engine
