/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface DeficitEstimate {
  from_tenths: number;
  /** Сколько инцидентов с сужением по основаниям легло в оценку. */
  incidents: number;
  /** «в среднем с 13 до 4 деталей». */
  text: string;
  to_tenths: number;
}
