// Пакет quality — производственные сценарии модуля quality слоя application: качество: результаты контроля любого метода, сигналы, дефекты, полнота контроля, карта реакций, методы и покрытие.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/quality,
// AD-36) и ведомые порты модуля; вызывает domain/quality. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/quality,
// хранение — infrastructure/storage/quality.
//
// Требования: FR-14, FR-35…FR-38, FR-48, FR-125, FR-144, AD-3, AD-27, AD-29, AD-30.
// Владелец после волны 1: эпик 20 (качество).
package quality
