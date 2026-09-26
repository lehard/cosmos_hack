/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { BoardRowMapping } from './boardRowMapping';
import type { BoardRowStatus } from './boardRowStatus';

export interface BoardRow {
  /**
     * Полученное значение (JSON); null — ещё не проверено.
     * @nullable
     */
  actual: string | null;
  assertion_id: string;
  /** Доменное время проверки. */
  at?: string;
  /** Момент проверки словами карточки. */
  checkpoint?: string;
  /** Пояснение: почему ожидает или не совпало. */
  detail?: string;
  /** Ожидаемое значение (JSON). */
  expected: string;
  /** exact — поле ответа контракта; draft — по смыслу, путь уточняет модуль-владелец; manual — видно на экране. */
  mapping?: BoardRowMapping;
  /** Утверждение «чего не должно случиться». */
  must_not?: boolean;
  /** Операция API, которой проверяется утверждение (те же Queries). */
  operation_id: string;
  /** JSON Pointer в ответе операции. */
  path: string;
  /** Карточка сценария, которой принадлежит утверждение. */
  scenario_id?: string;
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
