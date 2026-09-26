// Пакет nonconformity — производственные сценарии модуля nonconformity слоя application: несоответствия, решения по изделию, сдерживание, изоляция, разрешения на отклонение.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/nonconformity,
// AD-36) и ведомые порты модуля; вызывает domain/nonconformity. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/nonconformity,
// хранение — infrastructure/storage/nonconformity.
//
// Требования: FR-49…FR-56, FR-144, FR-151, AD-27, AD-30, AD-39, AD-43.
// Владелец после волны 1: эпик 21 (несоответствия, решения, сдерживание).
package nonconformity
