/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { CommonFactorRow } from './commonFactorRow';

export interface CommonFactors {
  /** Вид дефекта × операция × оборудование. */
  group_key: string;
  group_label: string;
  /** @minimum 0 */
  nc_count: number;
  rows: CommonFactorRow[];
}
