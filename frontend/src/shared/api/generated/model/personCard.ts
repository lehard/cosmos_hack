/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AccessQualification } from './accessQualification';
import type { AccessRoleGrant } from './accessRoleGrant';
import type { PersonPost } from './personPost';

export interface PersonCard {
  display_name: string;
  org_unit?: string;
  /** Псевдоним сотрудника. */
  person_id: string;
  /**
     * Версия политики, на которой построен ответ (AD-39).
     * @minimum 0
     */
  policy_seq: number;
  /** Текущие посты сотрудника. */
  posts: PersonPost[];
  qualifications: AccessQualification[];
  roles: AccessRoleGrant[];
}
