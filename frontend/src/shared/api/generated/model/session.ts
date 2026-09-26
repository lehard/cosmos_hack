/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { RoleRef } from './roleRef';
import type { SessionShift } from './sessionShift';
import type { SessionUser } from './sessionUser';
import type { SessionWorkplace } from './sessionWorkplace';

export interface Session {
  user: SessionUser;
  role: RoleRef;
  scope?: string;
  shift?: SessionShift;
  workplace?: SessionWorkplace;
  /**
     * Версия политики
     * @minimum 0
     */
  policy_seq: number;
  /** Вход демо-персоной без пароля */
  demo: boolean;
}
