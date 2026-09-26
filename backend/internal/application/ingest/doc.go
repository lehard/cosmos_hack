// Пакет ingest — производственные сценарии модуля ingest слоя application: приём событий: подпись, схема, пять случаев изменения контракта, дедупликация, карантин, разрешение носителя, ручной ввод и импорт CSV/Excel.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/ingest,
// AD-36) и ведомые порты модуля; вызывает domain/ingest. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/ingest,
// хранение — infrastructure/storage/ingest.
//
// Требования: FR-26…FR-41, FR-123, FR-140, FR-141, AD-7, AD-18, AD-20, AD-41.
// Владелец после волны 1: эпик 06 (приём и edge-агент).
package ingest
