/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface QualityAnalyzerStage {
  /**
     * Уверенность ступени, б. п.
     * @minimum 0
     * @maximum 10000
     */
  confidence_bp?: number;
  output_note?: string;
  /** Ступень ансамбля. */
  stage: string;
  /** Версия анализатора ступени. */
  version: string;
}
