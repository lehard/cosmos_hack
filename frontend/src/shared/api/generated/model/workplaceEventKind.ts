/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Вид события поста; zone_in, zone_out — проход назначенного на пост через точку СКУД зоны поста (эпик 37).
 */
export type WorkplaceEventKind = typeof WorkplaceEventKind[keyof typeof WorkplaceEventKind];


export const WorkplaceEventKind = {
  assigned: 'assigned',
  cleared: 'cleared',
  token_in: 'token_in',
  token_out: 'token_out',
  admitted: 'admitted',
  released: 'released',
  revoked: 'revoked',
  presence_deviation: 'presence_deviation',
  zone_in: 'zone_in',
  zone_out: 'zone_out',
} as const;
