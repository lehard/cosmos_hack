/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AccessAssignmentAssigneeRole } from './accessAssignmentAssigneeRole';

export interface AccessAssignment {
  /** Допуск к рабочему месту действует (барьер 2). */
  admitted: boolean;
  /** Документ согласования начальника ОТК — для контролёра обязателен. */
  approval_document_id?: string;
  assignee_role: AccessAssignmentAssigneeRole;
  person_id: string;
  /** Квалификация действует на дату смены (FR-80). */
  qualification_ok: boolean;
  shift_id: string;
  workplace_id: string;
}
