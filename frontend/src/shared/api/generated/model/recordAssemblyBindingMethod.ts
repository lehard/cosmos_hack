/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type RecordAssemblyBindingMethod = typeof RecordAssemblyBindingMethod[keyof typeof RecordAssemblyBindingMethod];


export const RecordAssemblyBindingMethod = {
  dpm_datamatrix: 'dpm_datamatrix',
  tag_qr: 'tag_qr',
  container_cell: 'container_cell',
  route_card: 'route_card',
  post_context: 'post_context',
  manual_entry: 'manual_entry',
  internal_id: 'internal_id',
} as const;
