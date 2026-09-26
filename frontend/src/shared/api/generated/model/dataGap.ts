/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface DataGap {
  from: string;
  /** Источник (id). */
  source: string;
  /** Что пропало словами; пропуск — «исключать нельзя». */
  text: string;
  to: string;
}
