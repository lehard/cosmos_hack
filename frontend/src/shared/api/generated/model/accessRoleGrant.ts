/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface AccessRoleGrant {
  role_id: string;
  /** Область: здание → цех → участок → рабочее место. */
  scope: string;
  valid_from: string;
  valid_until?: string;
}
