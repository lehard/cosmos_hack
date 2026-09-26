/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { WorkplaceCandidateQualificationVerdict } from './workplaceCandidateQualificationVerdict';
import type { WorkplaceCandidateRole } from './workplaceCandidateRole';

export interface WorkplaceCandidate {
  /** Назначение пройдёт гард access.assignment.set (контролёру — после документа согласования начальника ОТК). */
  allowed: boolean;
  /** Уже назначен в этой смене на другой пост (workplace_id). */
  assigned_elsewhere?: string;
  /** Уже назначен на этот пост в этой смене. */
  assigned_here?: boolean;
  /** Сотрудник словами (псевдоним кейса §4.6). */
  display: string;
  /** Контролёр: назначение — по документу «запрос мастера → согласование начальника ОТК» (PRD §11.18). */
  needs_approval?: boolean;
  person_id: string;
  /** Квалификация, по которой вердикт. */
  qualification_id?: string;
  /** Квалификация на дату смены: ok — действует, expiring — истекает в 30 дней, expired — истекла, missing — нет в области поста. Контролёру квалификация не нужна (ok) — нужен документ согласования. */
  qualification_verdict: WorkplaceCandidateQualificationVerdict;
  /** Роль назначения (assignee_role команды access.assignment.set). */
  role: WorkplaceCandidateRole;
  /** Срок квалификации. */
  valid_until?: string;
  /** Почему так — по-русски: срок, область, что нужно для назначения. */
  why: string;
}
