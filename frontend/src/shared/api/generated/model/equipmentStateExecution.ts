/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type EquipmentStateExecution = typeof EquipmentStateExecution[keyof typeof EquipmentStateExecution];


export const EquipmentStateExecution = {
  running: 'running',
  idle: 'idle',
  stopped: 'stopped',
  setup: 'setup',
  interrupted: 'interrupted',
  unknown: 'unknown',
} as const;
