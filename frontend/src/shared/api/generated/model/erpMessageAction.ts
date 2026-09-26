/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ErpMessageAction = typeof ErpMessageAction[keyof typeof ErpMessageAction];


export const ErpMessageAction = {
  accept_into_work: 'accept_into_work',
  warehouse_transfer: 'warehouse_transfer',
  scrap_transfer_rework: 'scrap_transfer_rework',
  scrap_transfer_writeoff: 'scrap_transfer_writeoff',
  scrap_transfer_reprocess: 'scrap_transfer_reprocess',
  return_to_supplier: 'return_to_supplier',
  release: 'release',
  inspection_result: 'inspection_result',
} as const;
