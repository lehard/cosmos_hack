/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { PartnerSignature } from './partnerSignature';
import type { PassportExtract } from './passportExtract';
import type { PassportExtractViewContent } from './passportExtractViewContent';

export interface PassportExtractView {
  /** Контрольная точка хранителя отправителя. */
  checkpoint?: string;
  /** Содержимое выписки. */
  content: PassportExtractViewContent;
  extract: PassportExtract;
  signatures: PartnerSignature[];
}
