// Пакет analysis — производственные сценарии модуля analysis слоя application: разбор обстоятельств, гипотезы, общие факторы, инциденты и область риска, похожие случаи, предложения, корректирующие меры, карта дефицита данных.
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/analysis,
// AD-36) и ведомые порты модуля; вызывает domain/analysis. HTTP-фреймворка и
// драйверов БД здесь нет: операции регистрирует infrastructure/transport/analysis,
// хранение — infrastructure/storage/analysis.
//
// Требования: FR-58…FR-64, FR-135, FR-138, FR-143, FR-153, AD-3, AD-29, AD-42.
// Владелец после волны 1: эпик 22 (разбор и область риска), 42 (предложения).
package analysis
