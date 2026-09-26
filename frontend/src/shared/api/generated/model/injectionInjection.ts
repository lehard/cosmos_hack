/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type InjectionInjection = typeof InjectionInjection[keyof typeof InjectionInjection];


export const InjectionInjection = {
  duplicate_event: 'duplicate_event',
  late_event: 'late_event',
  corrupt_frame: 'corrupt_frame',
  machine_fault: 'machine_fault',
  data_loss: 'data_loss',
  tamper_outside: 'tamper_outside',
} as const;
