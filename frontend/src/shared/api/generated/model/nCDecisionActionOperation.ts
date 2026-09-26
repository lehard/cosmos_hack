/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

/**
 * Операция API.
 */
export type NCDecisionActionOperation = typeof NCDecisionActionOperation[keyof typeof NCDecisionActionOperation];


export const NCDecisionActionOperation = {
  nonconformitynonconformityconfirm: 'nonconformity.nonconformity.confirm',
  nonconformitysignalreject: 'nonconformity.signal.reject',
  nonconformityrecheckrequest: 'nonconformity.recheck.request',
  nonconformityitemisolate: 'nonconformity.item.isolate',
  nonconformitypresentationresolve: 'nonconformity.presentation.resolve',
  nonconformitydispositionset: 'nonconformity.disposition.set',
  nonconformitydispositionverify: 'nonconformity.disposition.verify',
  nonconformitynonconformityclose: 'nonconformity.nonconformity.close',
  nonconformitycontainmentset: 'nonconformity.containment.set',
  nonconformitycontainmentrelease: 'nonconformity.containment.release',
} as const;
