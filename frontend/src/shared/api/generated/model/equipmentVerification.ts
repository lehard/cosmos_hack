/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { EquipmentVerificationStatus } from './equipmentVerificationStatus';

export interface EquipmentVerification {
  status: EquipmentVerificationStatus;
  /** @nullable */
  valid_till?: string | null;
}
