/**
 * Сущность «incident» — инциденты, разбор обстоятельств, гипотезы, область риска
 * (семейство `incident`, владелец схемы — модуль analysis). Ключи кэша — по
 * соглашению shared/api/keys.ts (каркас эпика 03).
 *
 * Эпик 12: входные данные экранов разбора (model/types.ts) и чистые правила над
 * ними. Обёртки над сгенерированными запросами добавляются, когда операции
 * чтения analysis появятся в contracts/openapi.yaml (эпик 02).
 */
import { entityKeys } from '@/shared/api/keys'

export const incidentKeys = entityKeys('incident')

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
