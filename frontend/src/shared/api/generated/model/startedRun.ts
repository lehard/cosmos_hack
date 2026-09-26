/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */

export interface StartedRun {
  /** Номер критического действия CA-‹n› (AD-28). */
  ca_ref?: string;
  /** id команды. */
  command_id: string;
  /** Записанные записи журнала. */
  event_ids: string[];
  /** Доменное время записи (AD-37). */
  recorded_at?: string;
  /** Повтор с тем же command_id — возвращён прежний ответ (AD-7). */
  replayed: boolean;
  /** Новый прогон — отдельное пространство имён (AD-38). */
  run_id: string;
  /** Позиция записи-решения в журнале; для следующей команды по объекту — новый basis_seq. */
  seq: number;
}
