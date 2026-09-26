/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { PartnerSignatureVerification } from './partnerSignatureVerification';

export interface PartnerSignature {
  key_ref: string;
  signer_role?: string;
  verification: PartnerSignatureVerification;
}
