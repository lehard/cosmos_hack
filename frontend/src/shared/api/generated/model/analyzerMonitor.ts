/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface AnalyzerMonitor {
  /** Качество кадра ниже — «кадр не как при допуске». */
  quality_min_bp: number;
  /** Качество последних кадров (старые — первыми). */
  recent_quality_bp: number[];
  /** Сколько ответов анализатора учтено правилом. */
  seen: number;
  /** Сколько наблюдений подряд проверяется. */
  window: number;
}
