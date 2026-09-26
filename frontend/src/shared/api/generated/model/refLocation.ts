/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { RefLocationKind } from './refLocationKind';

export interface RefLocation {
  /** Зона СКУД. */
  access_zone_id?: string;
  kind: RefLocationKind;
  location_id: string;
  name: string;
  parent_id?: string;
  /** Путь области для прав (AD-15). */
  scope: string;
  valid_from: string;
  valid_until?: string;
  warehouse_id?: string;
}
