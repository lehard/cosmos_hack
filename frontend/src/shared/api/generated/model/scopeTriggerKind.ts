/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Опоздавшие данные, окно нарушения режима, решение человека, правило системы.
 */
export type ScopeTriggerKind = typeof ScopeTriggerKind[keyof typeof ScopeTriggerKind];


export const ScopeTriggerKind = {
  late_event: 'late_event',
  violation_window: 'violation_window',
  human: 'human',
  computed: 'computed',
} as const;
