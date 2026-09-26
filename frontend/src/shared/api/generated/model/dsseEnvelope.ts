/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DsseEnvelopeSignaturesElem } from './dsseEnvelopeSignaturesElem';

export interface DsseEnvelope {
  payload: string;
  payloadType: string;
  signatures: DsseEnvelopeSignaturesElem[];
}
