/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface CadDiscrepancy {
  designation: string;
  item_type_id: string;
  note?: string;
  /** not_in_erp — нет в номенклатуре; not_checked — сверка не выполнялась. */
  reason: string;
  /** onec, galaktika, other. */
  system: string;
}
