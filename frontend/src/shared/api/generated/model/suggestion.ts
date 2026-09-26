/**
 * СГЕНЕРИРОВАНО orval (frontend/scripts/generate.mjs) — руками не править (AD-20).
 * Источник: contracts/openapi.yaml
 */
import type { SuggestionHistory } from './suggestionHistory';
import type { SuggestionKind } from './suggestionKind';
import type { SuggestionStatus } from './suggestionStatus';

export interface Suggestion {
  /** Основания — event_id записей журнала. */
  basis: string[];
  /** basis_seq для команд по предложению (AD-39). */
  basis_seq: number;
  /** Оценка эффекта словами с единицами. */
  estimate?: string;
  event_id: string;
  /** Генератор (порт + адаптер): rules.bottleneck, rules.risk_scope… */
  generator: string;
  history: SuggestionHistory[];
  incident_id?: string;
  kind: SuggestionKind;
  /** Вид недостающих сведений (карта дефицита). */
  missing_kind?: string;
  recorded_at: string;
  /** Кому передано. */
  responsible_id?: string;
  /** Роль ответственного. */
  responsible_role?: string;
  statement: string;
  /** Новое / передано ответственному / принято в работу / отклонено. «Принято» ничего не применяет само. */
  status: SuggestionStatus;
  /** Узел процесса — ссылка на карту процесса. */
  step_key?: string;
  suggestion_id: string;
  title: string;
}
