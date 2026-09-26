/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { KeyActSignatureProfileId } from './keyActSignatureProfileId';

export interface KeyActSignature {
  profile_id: KeyActSignatureProfileId;
  /** Роль в акте: субъект, администратор безопасности, вторая подпись (AD-11). */
  role: string;
  signed_at: string;
  signer_person_id: string;
}
