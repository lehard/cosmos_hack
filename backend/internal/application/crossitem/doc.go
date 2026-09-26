// Пакет crossitem — производственные сценарии модуля crossitem слоя application: межизделийная стадия: генеалогия, партии, садки, привязка событий без изделия, временная линия оборудования, адресованные записи, документы вне изделия.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/crossitem,
// AD-36) и ведомые порты модуля; вызывает domain/crossitem. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/crossitem,
// хранение — infrastructure/storage/crossitem.
//
// Требования: FR-15, FR-45, FR-121, FR-151, AD-41, AD-42.
// Владелец после волны 1: эпик 07 (рамка), 18 (генеалогия).
package crossitem
