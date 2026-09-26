/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Вид сущности — первый элемент ключа Vue Query и объект прав (как в contracts/events/common/sse-entity-changed.v1.json).
 */
export type EntityKind = typeof EntityKind[keyof typeof EntityKind];


export const EntityKind = {
  item: 'item',
  lot: 'lot',
  nonconformity: 'nonconformity',
  incident: 'incident',
  document: 'document',
  task: 'task',
  notification: 'notification',
  workplace: 'workplace',
  equipment: 'equipment',
  process_version: 'process_version',
  analyzer_passport: 'analyzer_passport',
  erp_message: 'erp_message',
  run: 'run',
  integrity: 'integrity',
  live_map: 'live_map',
  policy: 'policy',
  quarantine: 'quarantine',
} as const;
