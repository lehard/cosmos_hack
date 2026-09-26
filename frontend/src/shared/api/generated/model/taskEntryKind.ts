/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type TaskEntryKind = typeof TaskEntryKind[keyof typeof TaskEntryKind];


export const TaskEntryKind = {
  physical_move: 'physical_move',
  isolate_move: 'isolate_move',
  recheck: 'recheck',
  decision_required: 'decision_required',
  review_after_new_data: 'review_after_new_data',
  protection_basis_changed: 'protection_basis_changed',
  resign: 'resign',
  remark_carrier: 'remark_carrier',
  remove_temporary_carrier: 'remove_temporary_carrier',
  inspection_missing: 'inspection_missing',
  admin_resend: 'admin_resend',
  process_step: 'process_step',
  other: 'other',
} as const;
