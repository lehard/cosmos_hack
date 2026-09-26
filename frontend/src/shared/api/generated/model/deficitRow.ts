/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { DeficitEstimate } from './deficitEstimate';
import type { DeficitPlace } from './deficitPlace';
import type { DeficitRowKind } from './deficitRowKind';

export interface DeficitRow {
  /** Какая цифровизация закрыла бы пробел. */
  digitization: string;
  /** Нет — оценка невозможна: по этим расследованиям область ещё не сужали по основаниям. */
  estimate?: DeficitEstimate;
  incident_ids: string[];
  /** В скольких расследованиях не хватало. */
  investigations: number;
  kind: DeficitRowKind;
  nc_ids: string[];
  places: DeficitPlace[];
  /** Предложение цифровизации по этой строке, если записано. */
  suggestion_id?: string;
}
