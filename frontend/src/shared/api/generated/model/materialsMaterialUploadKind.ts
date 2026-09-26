/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type MaterialsMaterialUploadKind = typeof MaterialsMaterialUploadKind[keyof typeof MaterialsMaterialUploadKind];


export const MaterialsMaterialUploadKind = {
  photo: 'photo',
  video: 'video',
  illustration: 'illustration',
  protocol: 'protocol',
  log_excerpt: 'log_excerpt',
  scan: 'scan',
  other: 'other',
} as const;
