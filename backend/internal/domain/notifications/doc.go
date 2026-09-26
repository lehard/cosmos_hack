// Пакет notifications — бизнес-правила модуля notifications: сроки, задачи, уведомления, эскалации с ценой задержки.
//
// Слой: domain — чистые детерминированные функции без часов, случайности,
// float и ввода-вывода (AD-4); модуль композиции свёртки изделия (AD-40): импортирует только доменные модули раньше себя в порядке kernel → reference → signing → access → item → process → vision → quality → machinelogs → documents → nonconformity → analysis → notifications → crossitem → engine.
// Связи: вызывается из application/notifications, из свёртки (domain/engine) и
// независимым верификатором (AD-9).
//
// Требования: FR-8, FR-55, FR-57, AD-4, AD-45.
// Компонент кейса §3.2: Централизованное ядро.
// Владелец после волны 1: эпик 24 (уведомления).
package notifications
