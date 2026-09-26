/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface CloseBlocker {
  /** Код отказа contracts/errors.yaml: incident.cause_branch_open, incident.effectiveness_unchecked, incident.investigation_closed. */
  code: string;
  /** Почему словами. */
  text: string;
}
