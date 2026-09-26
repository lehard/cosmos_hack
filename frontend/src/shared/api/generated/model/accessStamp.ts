/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { AccessStampStatus } from './accessStampStatus';

export interface AccessStamp {
  /** Вид контроля (stamp_kinds политики). */
  inspection_kind: string;
  /** Приказ. */
  order_ref: string;
  person_id: string;
  scope: string;
  stamp_id: string;
  status: AccessStampStatus;
  valid_from: string;
  valid_until?: string;
}
