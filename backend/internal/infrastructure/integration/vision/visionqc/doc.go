// Пакет visionqc — адаптер внешней системы VisionQC («Визуальный контроль»):
// сигналы камеры как цепочка ступеней анализатора (FR-38, FR-97; зона
// integration, AD-18, AD-35). Реализует порт application/vision.SignalAdapter:
// результат системы по протоколу contracts/integrations/vision/visionqc/result.v1.json
// (по образцу ResultDataType OPC UA for Machine Vision) → наблюдение на нашем
// языке с вектором версий. Работает на краю, в edge-агенте (POST
// /v1/vision/visionqc): в ядро уходят только события контракта через обычный
// приём. Протокол — закрытая схема; система другой мажорной версии контракта
// не принимается (CheckContract, AD-18). Здесь же — файлы открытого набора
// иллюстраций на краю (DirIllustrations) и хранилище материалов ядра через
// операцию materials.material.upload (HTTPMaterials).
//
// Замена stand-а реальной системой — смена адреса; другой системой — другой
// SignalAdapter в cmd/edge-agent; ядро не меняется (NFR-TEST-2).
//
// Владелец: эпик 33 (порты и эмуляторы), 40 (адаптация).
package visionqc
