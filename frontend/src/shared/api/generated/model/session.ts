/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { RoleRef } from './roleRef';
import type { SessionShift } from './sessionShift';
import type { SessionUser } from './sessionUser';
import type { SessionWorkplace } from './sessionWorkplace';

export interface Session {
  /** Вход демо-персоной без пароля. */
  demo: boolean;
  /**
     * Версия политики, по которой вычислены права (AD-39).
     * @minimum 0
     */
  policy_seq: number;
  role: RoleRef;
  /** Область роли. */
  scope?: string;
  shift?: SessionShift;
  user: SessionUser;
  workplace?: SessionWorkplace;
}
