/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Учёт в 1С (erp) — только по квитанции.
 */
export type ItemStatusErpAccounting = typeof ItemStatusErpAccounting[keyof typeof ItemStatusErpAccounting];


export const ItemStatusErpAccounting = {
  not_sent: 'not_sent',
  accepted_into_work: 'accepted_into_work',
  moved: 'moved',
  transferred_to_scrap: 'transferred_to_scrap',
  returned_to_supplier: 'returned_to_supplier',
  released: 'released',
} as const;
