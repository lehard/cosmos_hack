/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface ItemLookup {
  /** Носитель, по которому найдено: ant:carrier:‹тип›:‹значение›. */
  carrier_ref?: string;
  /** Внутренний ID изделия: код_предприятия:локальный_id (AD-16). */
  item_id: string;
}
