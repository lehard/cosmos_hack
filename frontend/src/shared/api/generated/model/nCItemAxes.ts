/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { NCItemAxesContainment } from './nCItemAxesContainment';
import type { NCItemAxesDisposition } from './nCItemAxesDisposition';
import type { NCItemAxesErpAccounting } from './nCItemAxesErpAccounting';
import type { NCItemAxesPosition } from './nCItemAxesPosition';
import type { NCItemAxesQuality } from './nCItemAxesQuality';

export interface NCItemAxes {
  /** Сдерживание (nonconformity). */
  containment: NCItemAxesContainment;
  /** Решение по изделию (nonconformity). */
  disposition: NCItemAxesDisposition;
  /** Учёт в 1С (erp) — только по квитанции. */
  erp_accounting: NCItemAxesErpAccounting;
  /** Положение в процессе (process). */
  position: NCItemAxesPosition;
  /** Состояние качества (quality). */
  quality: NCItemAxesQuality;
}
