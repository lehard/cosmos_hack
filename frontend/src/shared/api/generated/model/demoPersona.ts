/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { RoleRef } from './roleRef';

export interface DemoPersona {
  /** Псевдоним сотрудника (кейс §4.6) */
  id: string;
  name: string;
  role: RoleRef;
  /** Область действия роли (здание → цех → участок → рабочее место) */
  scope?: string;
}
