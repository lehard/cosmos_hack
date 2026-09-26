/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type AlertEntryKind = typeof AlertEntryKind[keyof typeof AlertEntryKind];


export const AlertEntryKind = {
  overdue_isolation: 'overdue_isolation',
  gate_overdue: 'gate_overdue',
  not_moved_to_isolator: 'not_moved_to_isolator',
  anomaly: 'anomaly',
  escalation: 'escalation',
  integrity_violation: 'integrity_violation',
} as const;
