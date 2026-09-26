/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { WorkplaceEvent } from './workplaceEvent';

export interface WorkplaceHistory {
  items: WorkplaceEvent[];
  next_cursor?: string;
  workplace_id: string;
}
