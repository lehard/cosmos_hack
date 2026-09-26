/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface ItemZone {
  /** Доступ к зоне закрыт (FR-20). */
  closed: boolean;
  /** Шаг, закрывший доступ. */
  closed_by?: string;
  /** Открытое вмешательство (FR-21). */
  open_intervention?: string;
  title: string;
  zone_id: string;
}
