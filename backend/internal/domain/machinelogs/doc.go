// Пакет machinelogs — бизнес-правила модуля machinelogs: журналы оборудования: состояние, программа, инструмент, сводки циклов, отклонения, профиль выполнения операции, специальный процесс.
//
// Слой: domain — чистые детерминированные функции без часов, случайности,
// float и ввода-вывода (AD-4); модуль композиции свёртки изделия (AD-40): импортирует только доменные модули раньше себя в порядке kernel → reference → signing → access → item → process → vision → quality → machinelogs → documents → nonconformity → analysis → notifications → crossitem → engine.
// Связи: вызывается из application/machinelogs, из свёртки (domain/engine) и
// независимым верификатором (AD-9).
//
// Требования: FR-121, FR-147…FR-149, FR-151, AD-29, AD-42.
// Компонент кейса §3.2: MachineLogs.
// Владелец после волны 1: эпик 23 (MachineLogs).
package machinelogs
