/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type MaterialInfoKind = typeof MaterialInfoKind[keyof typeof MaterialInfoKind];


export const MaterialInfoKind = {
  photo: 'photo',
  video: 'video',
  illustration: 'illustration',
  protocol: 'protocol',
  log_excerpt: 'log_excerpt',
  scan: 'scan',
  quarantine_payload: 'quarantine_payload',
  other: 'other',
} as const;
