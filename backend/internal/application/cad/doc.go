// Пакет cad — производственные сценарии модуля cad слоя application: КОМПАС-3D: импорт условной сборки.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/cad,
// AD-36) и ведомые порты модуля; вызывает domain/cad. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/cad,
// хранение — infrastructure/storage/cad.
//
// Требования: FR-94, AD-18.
// Владелец после волны 1: эпик 31 (Галактика, MES, КОМПАС-3D).
package cad
