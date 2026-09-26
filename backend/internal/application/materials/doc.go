// Пакет materials — производственные сценарии модуля materials слоя application: хранилище материалов по адресу содержимого: сканы, кадры, вложения.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/materials,
// AD-36) и ведомые порты модуля; вызывает domain/materials. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/materials,
// хранение — infrastructure/storage/materials.
//
// Требования: FR-75, FR-102, FR-139, AD-23.
// Владелец после волны 1: эпик 04 (журнал и хранение).
package materials
