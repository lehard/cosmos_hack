/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Второй статус несоответствия — «системное расследование» (FR-51): закрытие по изделию его не закрывает.
 */
export type NCCardInvestigationStatus = typeof NCCardInvestigationStatus[keyof typeof NCCardInvestigationStatus];


export const NCCardInvestigationStatus = {
  none: 'none',
  open: 'open',
  closed: 'closed',
} as const;
