/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { PersonPostAssigneeRole } from './personPostAssigneeRole';

export interface PersonPost {
  assignee_role?: PersonPostAssigneeRole;
  shift_id?: string;
  /** Пост — подпись. */
  station: string;
  workplace_id: string;
}
