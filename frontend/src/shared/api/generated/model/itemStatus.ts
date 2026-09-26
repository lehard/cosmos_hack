/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ItemStatusContainment } from './itemStatusContainment';
import type { ItemStatusDisposition } from './itemStatusDisposition';
import type { ItemStatusErpAccounting } from './itemStatusErpAccounting';
import type { ItemStatusPosition } from './itemStatusPosition';
import type { ItemStatusQuality } from './itemStatusQuality';
import type { ItemStatusSummary } from './itemStatusSummary';

export interface ItemStatus {
  /** Сдерживание (nonconformity); снятие блока ≠ годность. */
  containment: ItemStatusContainment;
  /** Решение по изделию (nonconformity). */
  disposition: ItemStatusDisposition;
  /** Учёт в 1С (erp) — только по квитанции. */
  erp_accounting: ItemStatusErpAccounting;
  /** Положение в процессе (process). */
  position: ItemStatusPosition;
  /** Состояние качества (quality). */
  quality: ItemStatusQuality;
  /** Сводный статус для списков и карты (словарь item_summary). */
  summary: ItemStatusSummary;
}
