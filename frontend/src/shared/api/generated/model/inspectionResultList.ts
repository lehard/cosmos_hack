/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { InspectionResult } from './inspectionResult';

export interface InspectionResultList {
  items: InspectionResult[];
  next_cursor?: string;
}
