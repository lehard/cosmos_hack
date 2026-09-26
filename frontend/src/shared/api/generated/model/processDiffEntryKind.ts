/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ProcessDiffEntryKind = typeof ProcessDiffEntryKind[keyof typeof ProcessDiffEntryKind];


export const ProcessDiffEntryKind = {
  elementAdded: 'elementAdded',
  elementRemoved: 'elementRemoved',
  presentationPointAdded: 'presentationPointAdded',
  thresholdChanged: 'thresholdChanged',
  propertyChanged: 'propertyChanged',
} as const;
