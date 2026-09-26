// Пакет process — производственные сценарии модуля process слоя application: нормативный слой: версии процесса BPMN, кворум, исполнитель BPMN, операции и перемещения.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/process,
// AD-36) и ведомые порты модуля; вызывает domain/process. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/process,
// хранение — infrastructure/storage/process.
//
// Требования: FR-10…FR-25, FR-44, FR-47, FR-125, FR-154…FR-156, AD-17, AD-13, AD-43.
// Владелец после волны 1: эпик 17 (процесс), 39 (редактор и кворум).
package process
