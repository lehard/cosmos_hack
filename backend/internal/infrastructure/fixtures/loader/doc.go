// Пакет loader — общий загрузчик заготовок (AD-36, FR-150): читает
// scenarios/fixtures/‹сценарий›/steps/NN.yaml (шаг, часы, ответы по operationId
// и параметрам), отдаёт адаптерам infrastructure/fixtures/‹модуль› ответ на шаге
// курсора (порт platform.FixtureCursor) и добавляет префикс прогона к ID (AD-38).
//
// Слой: служебная зона fixtures; импортирует только порты application.
// Владелец: эпик 09 (мир заготовок).
package loader
