// Пакет operatorvision — адаптер внешней системы OperatorVision («Контроль
// действий оператора», FR-126; зона integration, AD-18, AD-35). Реализует
// порт application/vision.SignalAdapter: гипотеза системы о действии у
// рабочего места по протоколу contracts/integrations/vision/operatorvision/action.v1.json
// → operator.action.observed. Результат — гипотеза, а не вина; распознавания
// лиц нет: закрытая схема протокола отвергает любые сведения о человеке,
// исполнитель определяется в ant по входу на рабочее место
// (domain/vision.ExecutorAt). Работает на краю, в edge-агенте
// (POST /v1/vision/operatorvision).
//
// Владелец: эпик 33 (порты и эмуляторы), 40 (адаптация).
package operatorvision
