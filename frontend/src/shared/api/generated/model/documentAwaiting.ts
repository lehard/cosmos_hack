/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface DocumentAwaiting {
  /** Кто может подписать этап (псевдонимы). */
  candidates: string[];
  /** Роль этапа. */
  role?: string;
  /**
     * Номер этапа.
     * @minimum 1
     */
  stage: number;
  /** Кто подписывает этап — для людей. */
  title: string;
}
