/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { RefEquipmentKind } from './refEquipmentKind';
import type { RefEquipmentVerificationResult } from './refEquipmentVerificationResult';

export interface RefEquipment {
  certificate_ref?: string;
  equipment_id: string;
  is_measuring_instrument: boolean;
  kind: RefEquipmentKind;
  location_id: string;
  name: string;
  /** Источник событий (edge-агент). */
  source_id?: string;
  /** unknown — нет в справочнике; not_verified — средство измерений без поверки; verification_invalid — непригодно; verification_expired — срок поверки истёк. */
  unusable_reason?: string;
  /** FR-17: можно использовать на момент ответа (поверка действует). */
  usable: boolean;
  verification_result?: RefEquipmentVerificationResult;
  verified_until?: string;
}
