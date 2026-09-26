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
  /** Подписанный пакет выписки — конверт DSSE (JSON): его скачивают и проверяют у получателя без доступа к журналу отправителя (AD-19). */
  envelope?: string;
  extract: PassportExtract;
  /** Почему такой статус происхождения — для человека. */
  origin_reason?: string;
  signatures: PartnerSignature[];
}
