/**
 * Сущность «process_version» — версии описания процесса (нормативный слой).
 * Ключи кэша — по соглашению shared/api/keys.ts (каркас эпика 03).
 *
 * Эпик 12: читаемое представление версии и разница с действующей (FR-24).
 * Обёртки над сгенерированными запросами добавляются, когда операции чтения
 * версий появятся в contracts/openapi.yaml (эпик 02).
 */
import { entityKeys } from '@/shared/api/keys'

export const processVersionKeys = entityKeys('process_version')

export * from './model/types'
export * from './model/diff'
