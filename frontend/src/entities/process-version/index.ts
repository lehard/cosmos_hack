/**
 * Сущность «process_version» — версии описания процесса (нормативный слой).
 * Ключи кэша — по соглашению shared/api/keys.ts (каркас эпика 03).
 *
 * Эпик 12: читаемое представление версии и разница с действующей (FR-24);
 * чтение — api.ts поверх сгенерированного клиента (операции `process.version.*`).
 */
export * from './model/types'
export * from './model/diff'
export * from './api'
