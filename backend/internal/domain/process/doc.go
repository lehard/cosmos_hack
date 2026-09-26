// Пакет process — бизнес-правила модуля process: нормативный слой: версии процесса BPMN, кворум, исполнитель BPMN, операции и перемещения.
//
// Слой: domain — чистые детерминированные функции без часов, случайности,
// float и ввода-вывода (AD-4); модуль композиции свёртки изделия (AD-40): импортирует только доменные модули раньше себя в порядке kernel → reference → signing → access → item → process → vision → quality → machinelogs → documents → nonconformity → analysis → notifications → crossitem → engine.
// Связи: вызывается из application/process, из свёртки (domain/engine) и
// независимым верификатором (AD-9).
//
// Требования: FR-10…FR-25, FR-44, FR-47, FR-125, FR-154…FR-156, AD-17, AD-13, AD-43.
// Компонент кейса §3.2: Централизованное ядро.
// Владелец после волны 1: эпик 17 (процесс), 39 (редактор и кворум).
package process
