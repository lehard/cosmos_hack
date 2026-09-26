/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { WorkplaceCandidate } from './workplaceCandidate';

export interface WorkplaceCandidateList {
  basis_seq: number;
  items: WorkplaceCandidate[];
  /** Смена, на дату которой проверены квалификации; пусто — текущая. */
  shift_id?: string;
  workplace_id: string;
}
