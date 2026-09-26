/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: frontend/dev/openapi.draft.yaml (черновик до появления contracts/openapi.yaml)
 */
import type { NotificationSummaryByKind } from './notificationSummaryByKind';

export interface NotificationSummary {
  /** @minimum 0 */
  unread: number;
  by_kind?: NotificationSummaryByKind;
}
