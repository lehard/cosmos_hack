/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface HypothesisChange {
  /** Когда. */
  at: string;
  /**
     * Уверенность после изменения; нет — не оценить.
     * @minimum 0
     * @maximum 10000
     */
  confidence_bp?: number;
  /** Запись, изменившая уверенность. */
  event_id?: string;
  /** Что изменилось словами. */
  text: string;
}
