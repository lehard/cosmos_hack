/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface GrantApprovalStage {
  authority_id: string;
  /** Этап закрывается самим запросом (инициатор). */
  by_source: boolean;
  /** Кто может подписать (не инициатор и не получатель). */
  candidates: string[];
  stage: number;
  title: string;
}
