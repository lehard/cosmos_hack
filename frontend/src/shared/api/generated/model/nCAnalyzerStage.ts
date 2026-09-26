/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface NCAnalyzerStage {
  /**
     * Уверенность ступени, б. п. — не вероятность брака.
     * @minimum 0
     * @maximum 10000
     */
  confidence_bp?: number;
  output_note?: string;
  stage: string;
  version: string;
}
