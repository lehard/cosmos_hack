// Пакет nonconformity — бизнес-правила модуля nonconformity: несоответствия, решения по изделию, сдерживание, изоляция, разрешения на отклонение.
//
// Слой: domain — чистые детерминированные функции без часов, случайности,
// float и ввода-вывода (AD-4); модуль композиции свёртки изделия (AD-40): импортирует только доменные модули раньше себя в порядке kernel → reference → signing → access → item → process → vision → quality → machinelogs → documents → nonconformity → analysis → notifications → crossitem → engine.
// Связи: вызывается из application/nonconformity, из свёртки (domain/engine) и
// независимым верификатором (AD-9).
//
// Требования: FR-49…FR-56, FR-144, FR-151, AD-27, AD-30, AD-39, AD-43.
// Компонент кейса §3.2: ErrorAnalysis.
// Владелец после волны 1: эпик 21 (несоответствия, решения, сдерживание).
package nonconformity
