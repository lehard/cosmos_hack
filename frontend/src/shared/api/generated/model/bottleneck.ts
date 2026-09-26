/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface Bottleneck {
  step_key: string;
  /** Имя узла BPMN версии процесса; нет — показывать step_key. */
  step_name?: string;
  /** Среднее ожидание текстом с единицей («37 мин»). */
  wait?: string;
}
