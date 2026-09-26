/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AccessAuthority } from './accessAuthority';
import type { AccessRole } from './accessRole';

export interface AccessRoleList {
  authorities: AccessAuthority[];
  items: AccessRole[];
  policy_seq: number;
  /** Виды контроля для цифровых клейм (FR-145). */
  stamp_kinds: string[];
}
