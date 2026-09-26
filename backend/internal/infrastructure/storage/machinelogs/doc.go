// Пакет machinelogs — хранение модуля machinelogs (зона storage, AD-1).
// Проекции модуля (machinelogs.item_runs, run_index, equipment, timeline,
// violations; писатель — machinelogs, AD-45) хранятся каркасом проекций
// движка: пишутся только эффектами в транзакции Append (воркер, проектор) и
// читаются через порт application/engine.ProjectionStore — своей схемы
// Postgres модулю пока не нужно; она появится вместе с миграциями goose в
// migrations/, если проекциям понадобятся свои индексы. Чужих таблиц не
// читает; в журнал пишет только через порт journal (AD-44).
//
// Тесты пакета — сквозной путь модуля на своей БД агента (make dev-db):
// журнал, стадия, воркер с настоящей свёрткой, проектор, операции чтения.
//
// Слой: infrastructure/storage. Владелец: эпик 23 (MachineLogs).
package machinelogs
