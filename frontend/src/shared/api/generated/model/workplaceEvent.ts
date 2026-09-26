/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { WorkplaceEventKind } from './workplaceEventKind';

export interface WorkplaceEvent {
  /** Когда произошло (occurred_at). */
  at: string;
  /** Тип записи журнала. */
  event_type: string;
  /** Вид события поста. */
  kind: WorkplaceEventKind;
  /** Отображаемое имя сотрудника. */
  person_display?: string;
  /** Сотрудник, если известен. */
  person_id?: string;
  /** Основание: текст снятия назначения, причина снятия допуска, вид отклонения присутствия. */
  reason?: string;
  /**
     * seq записи журнала.
     * @minimum 0
     */
  seq: number;
  shift_id?: string;
}
