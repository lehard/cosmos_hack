/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NotificationSummaryByKind } from './notificationSummaryByKind';

export interface NotificationSummary {
  /** По видам: информация, тревога, задача, запрос решения. */
  by_kind?: NotificationSummaryByKind;
  /**
     * Непрочитанных всего.
     * @minimum 0
     */
  unread: number;
}
