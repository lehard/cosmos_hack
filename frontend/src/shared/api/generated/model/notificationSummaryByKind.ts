/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */

export type NotificationSummaryByKind = {
  /** @minimum 0 */
  info?: number;
  /** @minimum 0 */
  alarm?: number;
  /** @minimum 0 */
  task?: number;
  /** @minimum 0 */
  decision_request?: number;
};
