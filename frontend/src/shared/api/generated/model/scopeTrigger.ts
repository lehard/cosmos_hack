/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { ScopeTriggerKind } from './scopeTriggerKind';

export interface ScopeTrigger {
  /** Запись-повод (для окна записи). */
  event_id?: string;
  /** Опоздавшие данные, окно нарушения режима, решение человека, правило системы. */
  kind: ScopeTriggerKind;
  /** Повод словами: «пришёл журнал «Сварочный источник ИС-2»: ток 176 А при уставке 160 ± 10 А». */
  label: string;
  /** Когда это произошло (ось «как было»). */
  occurred_at?: string;
  /** Когда данные пришли (ось «что мы знали»). */
  received_at?: string;
}
