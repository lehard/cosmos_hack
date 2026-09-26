/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { PartnerSignatureVerification } from './partnerSignatureVerification';

export interface PartnerSignature {
  /** Ключ сотрудника по акту, подписанному корнем партнёра; false — корень или шлюз предприятия. */
  human?: boolean;
  key_ref: string;
  /** Подписант у партнёра (из выписки). */
  name?: string;
  signer_role?: string;
  verification: PartnerSignatureVerification;
}
