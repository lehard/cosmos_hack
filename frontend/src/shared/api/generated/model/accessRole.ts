/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface AccessRole {
  /** Действия ‹модуль›.‹объект›.‹действие›; * — любой сегмент. */
  actions: string[];
  /** Одна из пяти ролей кейса. */
  case_role: boolean;
  id: string;
  inherits: string[];
  title: string;
}
