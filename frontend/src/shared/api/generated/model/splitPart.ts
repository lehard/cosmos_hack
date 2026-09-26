/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface SplitPart {
  /**
     * Внутренний ID части; пусто — ‹исходный›-‹n›.
     * @maxLength 128
     */
  item_id?: string;
  /** @maxLength 64 */
  item_revision?: string;
  /**
     * Тип части; пусто — тип исходного.
     * @maxLength 128
     */
  item_type_id?: string;
  /** Дополнительные партии части (партии исходного переносятся сами). */
  lot_ids?: string[];
}
