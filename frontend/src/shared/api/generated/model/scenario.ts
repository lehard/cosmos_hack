/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface Scenario {
  /**
     * Число утверждений в scenarios/expected — строк табло.
     * @minimum 0
     */
  assertions: number;
  /** Ссылки на кейс: «§4.2 ситуация 3», «§5.1 проверка 7». */
  case_refs?: string[];
  /**
     * Сколько раз сценарий останавливается на решении человека.
     * @minimum 0
     */
  decisions: number;
  /**
     * Изделий в прогоне по умолчанию (плюс фоновые).
     * @minimum 0
     */
  default_items: number;
  /** @minimum 0 */
  default_seed: number;
  description?: string;
  /** situation — ситуация кейса §4.2; check — проверка §5.1; demo — демо-сценарий; failure — сбой каталога; extra — сверх кейса; run — прогон целиком. */
  kind?: string;
  /** Определение прогона, который запускается (scenarios/definitions/runs). */
  run_def?: string;
  /** Идентификатор сценария (S01…S13, демо, сбой). */
  scenario_id: string;
  title: string;
  /** Версия определения сценария. */
  version: string;
}
