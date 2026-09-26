/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface ExplainRole {
  role_id: string;
  /** Область назначения. */
  scope: string;
  title: string;
  valid_until?: string;
  /** Базовая роль, в которой объявлено действие. */
  via?: string;
}
