// Пакет documents — производственные сценарии модуля documents слоя application: документы: шаблоны, отрисовка, отпечаток, жизненный цикл, маршрут подписей и его закрытие, бумага с заверением.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/documents,
// AD-36) и ведомые порты модуля; вызывает domain/documents. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/documents,
// хранение — infrastructure/storage/documents.
//
// Требования: FR-65, FR-136, FR-139, AD-12, AD-13, AD-43.
// Владелец после волны 1: эпик 28 (документы), 44 (каталог).
package documents
