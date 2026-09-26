/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface ObservationStage {
  /**
     * @minimum 0
     * @maximum 10000
     */
  confidence_bp?: number;
  name: string;
  output?: string;
  version: string;
}
