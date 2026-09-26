// Пакет ops — производственные сценарии модуля ops слоя application: эксплуатация: состояние компонентов, очереди, карантин, интеграции, остановленные изделия, настройки источников и адаптеров.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/ops,
// AD-36) и ведомые порты модуля; вызывает domain/ops. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/ops,
// хранение — infrastructure/storage/ops.
//
// Требования: FR-109, FR-127, AD-25, AD-45.
// Владелец после волны 1: эпик 34 (эксплуатация).
package ops
