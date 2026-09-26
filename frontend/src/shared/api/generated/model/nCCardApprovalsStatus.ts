/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Подписи маршрута решения (режим 4): pending — решение не исполняется; demo_stub — демо, подписи не проверялись.
 */
export type NCCardApprovalsStatus = typeof NCCardApprovalsStatus[keyof typeof NCCardApprovalsStatus];


export const NCCardApprovalsStatus = {
  route_closed: 'route_closed',
  pending: 'pending',
  demo_stub: 'demo_stub',
} as const;
