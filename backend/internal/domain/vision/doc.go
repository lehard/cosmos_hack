// Пакет vision — бизнес-правила модуля vision: VisionQC и OperatorVision: порты сигналов, карты контроля, паспорта допуска анализаторов, откат.
//
// Слой: domain — чистые детерминированные функции без часов, случайности,
// float и ввода-вывода (AD-4); модуль композиции свёртки изделия (AD-40): импортирует только доменные модули раньше себя в порядке kernel → reference → signing → access → item → process → vision → quality → machinelogs → documents → nonconformity → analysis → notifications → crossitem → engine.
// Связи: вызывается из application/vision, из свёртки (domain/engine) и
// независимым верификатором (AD-9).
//
// Требования: FR-97…FR-103, FR-126, AD-18, AD-29.
// Компонент кейса §3.2: VisionQC, OperatorVision.
// Владелец после волны 1: эпик 33 (порты и эмуляторы), 40 (адаптация).
package vision
