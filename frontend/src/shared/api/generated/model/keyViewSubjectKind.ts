/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type KeyViewSubjectKind = typeof KeyViewSubjectKind[keyof typeof KeyViewSubjectKind];


export const KeyViewSubjectKind = {
  person: 'person',
  device: 'device',
  engine: 'engine',
  gateway: 'gateway',
  enterprise_gateway: 'enterprise_gateway',
  keeper: 'keeper',
  verifier: 'verifier',
  demo_persona: 'demo_persona',
  partner_root: 'partner_root',
} as const;
