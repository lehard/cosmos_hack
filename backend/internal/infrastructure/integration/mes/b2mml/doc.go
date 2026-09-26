// Пакет b2mml — адаптер канала MES модуля mes (зона integration, AD-18,
// AD-35): подмножество транзакций B2MML V7 (ISA-95 / IEC 62264, MESA
// International) в форме B2MML-JSON, контракт mes.isa95.v1
// (contracts/integrations/mes). Реализует ведомый порт application/mes.Channel.
//
//   - Внутрь: ProcessOperationsSchedule (задания) и NotifyOperationsEvent
//     (начало, конец, пауза, возобновление, прерывание операций) — с канала
//     MES (`GET ‹канал›/outbox`), каждое сообщение проверяется своей схемой;
//     разбор терпим к лишним элементам стандарта (перевод идёт через
//     подмножество). Дальше — шлюз application/mes и обычный приём.
//   - Наружу: SyncMaterialSubLot (экземпляр) и SyncMaterialLot (партия) с
//     Disposition = Restricted — блок, UnRestricted — снятие; ConfirmationCode =
//     Always: блок должен быть подтверждён ConfirmBOD. Каждое исходящее
//     сообщение до отправки проверяется схемой (FR-111).
//   - ConfirmBOD: успех — квитанция (повтор BODID — Duplicate), ошибка
//     ErrorType = Data — ошибка данных без автоповтора, Contract —
//     несовместимость (канал degraded); 5xx и таймаут — транспорт (повтор тем
//     же BODID).
//   - Сверка при старте: `GET ‹канал›/about` — релиз B2MML и версии mes.isa95.
//
// Stand MES — эпик 43 (подпакет stand); здесь — клиент и контрактный тест на
// эталонах. Формат — проектное предположение (docs/integrations/mes.md).
//
// Требования: FR-90, FR-93, FR-95, FR-96, FR-111; AD-18, AD-20, AD-35.
// Владелец: эпик 31.
package b2mml
