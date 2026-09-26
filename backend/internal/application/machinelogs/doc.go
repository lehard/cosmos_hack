// Пакет machinelogs — производственные сценарии модуля machinelogs слоя application: журналы оборудования: состояние, программа, инструмент, сводки циклов, отклонения, профиль выполнения операции, специальный процесс.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/machinelogs,
// AD-36) и ведомые порты модуля; вызывает domain/machinelogs. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/machinelogs,
// хранение — infrastructure/storage/machinelogs.
//
// Требования: FR-121, FR-147…FR-149, FR-151, AD-29, AD-42.
// Владелец после волны 1: эпик 23 (MachineLogs).
package machinelogs
