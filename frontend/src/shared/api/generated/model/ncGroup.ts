/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NcGroupInvestigation } from './ncGroupInvestigation';

export interface NcGroup {
  defect_type: string;
  equipment: string;
  group_key: string;
  investigation: NcGroupInvestigation;
  last_found_at: string;
  /** @minimum 0 */
  nc_count: number;
  /** Несоответствия группы — вход разбора обстоятельств и гипотез (эпик 12). */
  nc_ids: string[];
  operation: string;
}
