/**
 * Сущность «equipment» — каркас слоя entities (эпик 03). Ключи кэша по соглашению
 * shared/api/keys.ts; обёртки над сгенерированными запросами и типы добавляют
 * эпики-владельцы экранов, когда операции появятся в contracts/openapi.yaml.
 */
import { entityKeys } from '@/shared/api/keys'

export const equipmentKeys = entityKeys('equipment')
