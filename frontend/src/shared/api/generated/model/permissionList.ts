/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { Permission } from './permission';

export interface PermissionList {
  /** @minimum 0 */
  policy_seq: number;
  items: Permission[];
}
