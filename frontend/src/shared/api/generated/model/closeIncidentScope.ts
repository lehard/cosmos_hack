/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * risk_scope — закрыть область риска (по умолчанию); investigation — закрыть расследование: 422 incident.cause_branch_open, если нет вывода по одной из двух причин, 422 incident.effectiveness_unchecked, если эффективность мер не проверена.
 */
export type CloseIncidentScope = typeof CloseIncidentScope[keyof typeof CloseIncidentScope];


export const CloseIncidentScope = {
  risk_scope: 'risk_scope',
  investigation: 'investigation',
} as const;
