/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { RoleRef } from './roleRef';

export interface DemoPersona {
  /** Псевдоним сотрудника (кейс §4.6). */
  id: string;
  /** Отображаемое имя (условное). */
  name: string;
  role: RoleRef;
  /** Область действия роли (здание → цех → участок → рабочее место). */
  scope?: string;
}
