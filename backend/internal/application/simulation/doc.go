// Пакет simulation — производственные сценарии модуля simulation слоя application: генератор и тестовые сценарии: прогоны, виртуальные часы, пауза, скорость, автосверка «ожидалось → получилось», цифровой стенд.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/simulation,
// AD-36) и ведомые порты модуля; вызывает domain/simulation. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/simulation,
// хранение — infrastructure/storage/simulation.
//
// Требования: FR-104…FR-108, FR-119, FR-124, FR-129, FR-152, AD-26, AD-37, AD-38.
// Владелец после волны 1: эпик 32 (симуляция), 36 (цифровой стенд).
package simulation
