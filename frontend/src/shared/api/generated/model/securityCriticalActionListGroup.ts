/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type SecurityCriticalActionListGroup = typeof SecurityCriticalActionListGroup[keyof typeof SecurityCriticalActionListGroup];


export const SecurityCriticalActionListGroup = {
  product_decision: 'product_decision',
  nc_decision: 'nc_decision',
  cause: 'cause',
  risk_scope: 'risk_scope',
  control_change: 'control_change',
  authority: 'authority',
  protected_data: 'protected_data',
  admin_security: 'admin_security',
} as const;
