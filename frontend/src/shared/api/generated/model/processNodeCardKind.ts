/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export type ProcessNodeCardKind = typeof ProcessNodeCardKind[keyof typeof ProcessNodeCardKind];


export const ProcessNodeCardKind = {
  automatedInspection: 'automatedInspection',
  operation: 'operation',
  humanInspection: 'humanInspection',
  parallelGateway: 'parallelGateway',
  inclusiveGateway: 'inclusiveGateway',
  startEvent: 'startEvent',
  endEvent: 'endEvent',
  intermediateEvent: 'intermediateEvent',
  timer: 'timer',
  subprocess: 'subprocess',
  callActivity: 'callActivity',
  conditionalFlow: 'conditionalFlow',
  lane: 'lane',
} as const;
