/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AccessPersonAccountStatus } from './accessPersonAccountStatus';
import type { AccessRoleGrant } from './accessRoleGrant';

export interface AccessPerson {
  /** Учётная запись: нет, ждёт активации, действует, заблокирована (FR-128). */
  account_status: AccessPersonAccountStatus;
  display_name: string;
  login?: string;
  org_unit?: string;
  /** Псевдоним сотрудника. */
  person_id: string;
  /** Версия политики, на которой построен ответ (AD-39). */
  policy_seq: number;
  roles: AccessRoleGrant[];
}
