// Пакет mes — производственные сценарии модуля mes слоя application: MES: задания, блокировки изделий и партий.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/mes,
// AD-36) и ведомые порты модуля; вызывает domain/mes. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/mes,
// хранение — infrastructure/storage/mes.
//
// Требования: FR-92, FR-93, AD-18.
// Владелец после волны 1: эпик 31 (Галактика, MES, КОМПАС-3D).
package mes
