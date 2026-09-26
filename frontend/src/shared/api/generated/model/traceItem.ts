/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface TraceItem {
  assembled: boolean;
  item_id: string;
  label: string;
  /** made_from_lot — из партии или плавки; grouped_with — в садке; assembled — собрано из них. */
  relation: string;
  /** Изделие, через которое сборка попала в список. */
  via?: string;
}
