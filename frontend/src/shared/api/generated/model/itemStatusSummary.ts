/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Сводный статус для списков и карты (словарь item_summary).
 */
export type ItemStatusSummary = typeof ItemStatusSummary[keyof typeof ItemStatusSummary];


export const ItemStatusSummary = {
  in_process: 'in_process',
  suspect: 'suspect',
  reinspection_required: 'reinspection_required',
  hold: 'hold',
  pending_decision: 'pending_decision',
  nonconforming: 'nonconforming',
  cleared: 'cleared',
  released: 'released',
  in_rework: 'in_rework',
  in_repair: 'in_repair',
  accepted_with_concession: 'accepted_with_concession',
  scrapped: 'scrapped',
  returned: 'returned',
} as const;
