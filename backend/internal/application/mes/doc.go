// Пакет mes — производственные сценарии модуля mes слоя application: обмен с
// MES без повторного ручного ввода (FR-93, AD-18).
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/mes,
// AD-36) и ведомые порты: Channel — канал MES (адаптер
// infrastructure/integration/mes/b2mml, подмножество B2MML-JSON), Intake —
// обычный приём. Вызывает domain/mes.
//
//   - Gateway (роль outbox): задания и события операций MES → факты через
//     обычный приём (mes.job.received, operation.run.*), шаг — по коду
//     операции нормативного слоя, изделие и исполнитель — по соответствиям.
//   - HoldReactor (роль projector): сдерживание изделия или партии →
//     реакция mes.hold.requested (блок или снятие в MES).
//   - Sender (роль outbox): неподтверждённые блоки → MES; квитанция —
//     mes.hold.responded; повтор только при транспортных ошибках (FR-96).
//   - Проекции mes.block и mes.job — операции mes.block.list, mes.order.list.
//
// Требования: FR-93, FR-95, FR-96, FR-123; AD-18, AD-27, AD-30, AD-45.
// Владелец: эпик 31 (Галактика, MES, КОМПАС-3D).
package mes
