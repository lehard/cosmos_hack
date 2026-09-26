/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * На месте; по СКУД на месте, но ключ не вставлен; ключ вставлен, а владельца нет в зоне; нет ни в зоне, ни ключа; никто не назначен; неизвестно.
 */
export type PostRowPresence = typeof PostRowPresence[keyof typeof PostRowPresence];


export const PostRowPresence = {
  present: 'present',
  key_missing: 'key_missing',
  owner_absent: 'owner_absent',
  absent: 'absent',
  not_assigned: 'not_assigned',
  unknown: 'unknown',
} as const;
