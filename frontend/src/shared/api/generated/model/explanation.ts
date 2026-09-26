/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface Explanation {
  /** x-ant-action id. */
  action: string;
  allowed: boolean;
  /** Что можно сделать вместо (например, «Запросить решение»). */
  allowed_actions: string[];
  /** Код отказа из contracts/errors.yaml. */
  code?: string;
  /** Объяснение по-русски: роль, область, полномочие, клеймо, разделение обязанностей. */
  reason?: string;
}
