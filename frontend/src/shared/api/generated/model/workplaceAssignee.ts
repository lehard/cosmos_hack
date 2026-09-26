/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { WorkplaceAssigneeAssigneeRole } from './workplaceAssigneeAssigneeRole';

export interface WorkplaceAssignee {
  assignee_role: WorkplaceAssigneeAssigneeRole;
  /** Отображаемое имя (условное). */
  person_display: string;
  /** Псевдоним сотрудника. */
  person_id: string;
  /** Квалификация действует на дату (FR-80); для контролёра — всегда true. */
  qualification_ok: boolean;
  shift_id: string;
}
