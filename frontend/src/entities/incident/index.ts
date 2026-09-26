/**
 * Сущность «incident» — инциденты, разбор обстоятельств, гипотезы, область риска
 * (семейство `incident`, владелец схемы — модуль analysis). Ключи кэша — по
 * соглашению shared/api/keys.ts (каркас эпика 03).
 *
 * Эпик 12: входные данные экранов разбора (model/types.ts) и чистые правила над
 * ними. Чтение и команды — api.ts поверх сгенерированного клиента
 * (операции `analysis.*`), ответы приводятся к входным данным экранов (model/map.ts).
 */
export * from './model/types'
export * from './model/record-text'
export * from './model/timescale'
export * from './model/circumstances'
export * from './model/factors'
export * from './model/hypotheses'
export * from './model/scope'
export * from './model/groups'
export * from './model/focus'
export * from './model/source'
export * from './model/map'
export * from './api'
