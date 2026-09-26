// Пакет documents — бизнес-правила модуля documents: документы: шаблоны, отрисовка, отпечаток, жизненный цикл, маршрут подписей и его закрытие, бумага с заверением.
//
// Слой: domain — чистые детерминированные функции без часов, случайности,
// float и ввода-вывода (AD-4); модуль композиции свёртки изделия (AD-40): импортирует только доменные модули раньше себя в порядке kernel → reference → signing → access → item → process → vision → quality → machinelogs → documents → nonconformity → analysis → notifications → crossitem → engine.
// Связи: вызывается из application/documents, из свёртки (domain/engine) и
// независимым верификатором (AD-9).
//
// Требования: FR-65, FR-136, FR-139, AD-12, AD-13, AD-43.
// Компонент кейса §3.2: CriticalActions.
// Владелец после волны 1: эпик 28 (документы), 44 (каталог).
package documents
