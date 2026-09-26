/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Происхождение подтверждено / подтверждено только сервером отправителя / не подтверждено (AD-19); у исходящей — not_applicable.
 */
export type PassportExtractOriginStatus = typeof PassportExtractOriginStatus[keyof typeof PassportExtractOriginStatus];


export const PassportExtractOriginStatus = {
  verified: 'verified',
  server_confirmed_only: 'server_confirmed_only',
  unverified: 'unverified',
  not_applicable: 'not_applicable',
} as const;
