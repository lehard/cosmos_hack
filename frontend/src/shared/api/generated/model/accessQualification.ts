/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AccessQualificationStatus } from './accessQualificationStatus';

export interface AccessQualification {
  certificate_ref?: string;
  person_id: string;
  qualification_id: string;
  scope?: string;
  status: AccessQualificationStatus;
  valid_from: string;
  valid_until?: string;
}
