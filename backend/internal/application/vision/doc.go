// Пакет vision — производственные сценарии модуля vision слоя application: VisionQC и OperatorVision: порты сигналов, карты контроля, паспорта допуска анализаторов, откат.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/vision,
// AD-36) и ведомые порты модуля; вызывает domain/vision. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/vision,
// хранение — infrastructure/storage/vision.
//
// Требования: FR-97…FR-103, FR-126, AD-18, AD-29.
// Владелец после волны 1: эпик 33 (порты и эмуляторы), 40 (адаптация).
package vision
