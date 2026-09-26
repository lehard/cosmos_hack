/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { Permission } from './permission';

export interface PermissionList {
  items: Permission[];
  /**
     * Версия политики, по которой вычислен список (AD-39).
     * @minimum 0
     */
  policy_seq: number;
}
