// Пакет quality — бизнес-правила модуля quality: качество: результаты контроля любого метода, сигналы, дефекты, полнота контроля, карта реакций, методы и покрытие.
//
// Слой: domain — чистые детерминированные функции без часов, случайности,
// float и ввода-вывода (AD-4); модуль композиции свёртки изделия (AD-40): импортирует только доменные модули раньше себя в порядке kernel → reference → signing → access → item → process → vision → quality → machinelogs → documents → nonconformity → analysis → notifications → crossitem → engine.
// Связи: вызывается из application/quality, из свёртки (domain/engine) и
// независимым верификатором (AD-9).
//
// Требования: FR-14, FR-35…FR-38, FR-48, FR-125, FR-144, AD-3, AD-27, AD-29, AD-30.
// Компонент кейса §3.2: VisionQC (границы).
// Владелец после волны 1: эпик 20 (качество).
package quality
