/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { BoardRowStatus } from './boardRowStatus';

export interface BoardRow {
  /**
     * Полученное значение (JSON); null — ещё не проверено.
     * @nullable
     */
  actual: string | null;
  assertion_id: string;
  /** Ожидаемое значение (JSON). */
  expected: string;
  /** Операция API, которой проверяется утверждение (те же Queries). */
  operation_id: string;
  /** JSON Pointer в ответе операции. */
  path: string;
  /** not_reached — сценарий не дошёл до шага. */
  status: BoardRowStatus;
  /**
     * Шаг сценария, после которого проверяется.
     * @minimum 0
     */
  step: number;
  /** Что проверяется, по-русски. */
  title: string;
}
